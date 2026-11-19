package websocket

import (
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
	gorillaws "github.com/gorilla/websocket"
)

func (c *Conn) readLoop() {
	if err := c.raw.SetReadDeadline(time.Now().Add(c.cfg.PongWait)); err != nil {
		return
	}
	c.raw.SetReadLimit(c.cfg.MaxMessageSize)
	c.raw.SetPongHandler(func(string) error {
		return c.raw.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	})
	for {
		kind, data, err := c.raw.ReadMessage()
		if err != nil {
			return
		}
		metrics.WSFrameReceived()
		if kind != gorillaws.TextMessage {
			c.enqueueError(inbound{}, errcore.New(c.context(), errs.WebsocketInvalidMessage), false)
			continue
		}
		msg, err := decodeInbound(data)
		if err != nil {
			c.enqueueError(inbound{}, err, false)
			continue
		}
		select {
		case <-c.done:
			return
		case c.inbound <- msg:
		default:
			c.enqueueError(msg, errcore.New(c.context(), errs.RateLimitExceeded), false)
		}
	}
}

func (c *Conn) writeLoop() {
	ticker := time.NewTicker(c.cfg.PingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if c.expired() {
				frame := c.errorOutbound(inbound{}, errcore.New(c.context(), errs.AuthTokenExpired))
				_ = c.writeFrame(frame)
				return
			}
			if err := c.raw.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait)); err != nil {
				return
			}
			if err := c.raw.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		case frame := <-c.send:
			if c.expired() && frame.closeCode == 0 && !frame.normalClose {
				frame = c.errorOutbound(inbound{}, errcore.New(c.context(), errs.AuthTokenExpired))
			}
			if err := c.writeFrame(frame); err != nil {
				return
			}
			if frame.closeCode != 0 || frame.normalClose {
				return
			}
		}
	}
}

func (c *Conn) writeFrame(frame outbound) error {
	if len(frame.payload) > 0 {
		if err := c.raw.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait)); err != nil {
			return err
		}
		if err := c.raw.WriteMessage(gorillaws.TextMessage, frame.payload); err != nil {
			return err
		}
		metrics.WSFrameSent()
	}
	if frame.closeCode != 0 {
		return transport.WriteClose(c.raw, frame.closeCode, frame.closeReason)
	}
	if frame.normalClose {
		payload := gorillaws.FormatCloseMessage(gorillaws.CloseNormalClosure, closeShutdown)
		return c.raw.WriteControl(gorillaws.CloseMessage, payload, time.Now().Add(c.cfg.WriteWait))
	}
	return nil
}

func (c *Conn) dispatch() {
	for msg := range c.inbound {
		c.handle(msg)
	}
}

func (c *Conn) handle(msg inbound) {
	if c.expired() {
		c.enqueueError(msg, errcore.New(c.context(), errs.AuthTokenExpired), true)
		return
	}
	ctx := &Context{Context: c.context(), Conn: c, in: msg}
	if c.router == nil {
		c.reply(msg, nil, errcore.New(c.context(), errs.InternalUnexpectedError))
		return
	}
	result, err := c.router.Dispatch(ctx)
	c.reply(msg, result, err)
}

func (c *Conn) reply(msg inbound, result any, err error) {
	if err != nil {
		c.enqueueError(msg, err, true)
		return
	}
	if msg.ID == "" {
		return
	}
	payload, merr := marshalResult(msg.Type, msg.ID, result)
	if merr != nil {
		c.enqueueError(msg, errcore.Wrap(c.context(), errs.InternalUnexpectedError, merr), true)
		return
	}
	if !c.sendWait(outbound{payload: payload}) && !c.isClosed() {
		metrics.WSSendDropped()
	}
}
