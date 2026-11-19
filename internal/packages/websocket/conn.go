package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/recovery"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
	gorillaws "github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// Identity is the user attached to a connection at upgrade time.
type Identity struct {
	UserID    int64
	Platform  string
	ExpiresAt time.Time
}

type outbound struct {
	payload     []byte
	closeCode   int
	closeReason string
	normalClose bool
}

// Conn is one accepted socket. UserID and Platform do not change after creation.
type Conn struct {
	id        string
	userID    int64
	platform  string
	expiresAt time.Time
	raw       *gorillaws.Conn
	hub       *Hub
	router    *Router
	log       logger.Logger
	cfg       Config
	base      context.Context
	cancel    context.CancelFunc
	send      chan outbound
	inbound   chan inbound
	done      chan struct{}
	mu        sync.Mutex
	closed    bool
	limiters  map[string]*rate.Limiter
	rooms     map[Room]struct{}
	startOnce sync.Once
	exitOnce  sync.Once
	rawOnce   sync.Once
}

func newConn(raw *gorillaws.Conn, hub *Hub, id string, identity Identity, parent context.Context, router *Router) *Conn {
	if parent == nil {
		parent = context.Background()
	}
	parent = context.WithoutCancel(parent)
	parent = errcore.WithConnectionID(parent, id)
	if identity.UserID != 0 {
		parent = errcore.WithUserID(parent, identity.UserID)
	}
	ctx, cancel := context.WithCancel(parent)
	log := logger.Nop()
	cfg := DefaultConfig()
	if hub != nil {
		log = hub.log
		cfg = hub.cfg
	}
	if log == nil {
		log = logger.Nop()
	}
	return &Conn{
		id:        id,
		userID:    identity.UserID,
		platform:  identity.Platform,
		expiresAt: identity.ExpiresAt,
		raw:       raw,
		hub:       hub,
		router:    router,
		log:       log,
		cfg:       cfg,
		base:      ctx,
		cancel:    cancel,
		send:      make(chan outbound, cfg.SendBuffer),
		inbound:   make(chan inbound, cfg.InboundBuffer),
		done:      make(chan struct{}),
		rooms:     map[Room]struct{}{},
	}
}

func newID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

// ID returns the connection id.
func (c *Conn) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// UserID returns the authenticated user. Zero means the connection has no user.
func (c *Conn) UserID() int64 {
	if c == nil {
		return 0
	}
	return c.userID
}

// Platform returns the optional client platform captured at upgrade.
func (c *Conn) Platform() string {
	if c == nil {
		return ""
	}
	return c.platform
}

// ExpiresAt returns the access-token expiry captured at upgrade.
// The zero time means the server was not given an expiry.
func (c *Conn) ExpiresAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.expiresAt
}

func (c *Conn) context() context.Context {
	if c == nil || c.base == nil {
		return context.Background()
	}
	return c.base
}

func (c *Conn) allow(key string, limit Limit) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.limiters == nil {
		c.limiters = map[string]*rate.Limiter{}
	}
	limiter := c.limiters[key]
	if limiter == nil {
		burst := limit.Burst
		if burst < 1 {
			burst = 1
		}
		limiter = rate.NewLimiter(rate.Limit(limit.Rate), burst)
		c.limiters[key] = limiter
	}
	return limiter.Allow()
}

func (c *Conn) trySend(frame outbound) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- frame:
		return true
	default:
		return false
	}
}

func (c *Conn) sendWait(frame outbound) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	select {
	case c.send <- frame:
		c.mu.Unlock()
		return true
	default:
		c.mu.Unlock()
	}
	timer := time.NewTimer(c.cfg.WriteWait)
	defer timer.Stop()
	select {
	case c.send <- frame:
		return true
	case <-timer.C:
		return false
	case <-c.done:
		return false
	}
}

func (c *Conn) isClosed() bool {
	if c == nil {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Conn) expired() bool {
	if c == nil || c.expiresAt.IsZero() {
		return false
	}
	return !time.Now().Before(c.expiresAt)
}

func (c *Conn) stop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Conn) exit() {
	if c == nil {
		return
	}
	c.exitOnce.Do(func() {
		if c.hub != nil {
			c.hub.remove(c)
		}
		c.stop()
	})
}

func (c *Conn) closeRaw() {
	if c == nil {
		return
	}
	c.rawOnce.Do(func() {
		if c.raw != nil {
			_ = c.raw.Close()
		}
	})
}

func (c *Conn) force() {
	c.exit()
	c.closeRaw()
}

func (c *Conn) shutdown() {
	payload, err := marshalEvent(Event{
		Type: eventClosing,
		Data: map[string]string{"reason": closeShutdown},
	})
	if err != nil || !c.trySend(outbound{payload: payload, normalClose: true}) {
		c.force()
	}
}

func (c *Conn) start() {
	if c == nil || c.raw == nil {
		return
	}
	c.startOnce.Do(func() {
		cc := recovery.ConnContext{
			ConnectionID: c.id,
			UserID:       c.userID,
		}
		if id, ok := errcore.RequestID(c.base); ok {
			cc.RequestID = id
		}
		go func() {
			defer c.closeRaw()
			defer c.exit()
			defer close(c.inbound)
			recovery.Do(c.log, c.raw, cc, c.readLoop)
		}()
		go c.run(cc, c.writeLoop)
		go c.run(cc, c.dispatch)
	})
}

func (c *Conn) run(cc recovery.ConnContext, fn func()) {
	defer c.closeRaw()
	defer c.exit()
	recovery.Do(c.log, c.raw, cc, fn)
}

func (c *Conn) enqueueError(msg inbound, err error, wait bool) {
	frame := c.errorOutbound(msg, err)
	sent := false
	if wait {
		sent = c.sendWait(frame)
	} else {
		sent = c.trySend(frame)
	}
	if sent {
		return
	}
	if !c.isClosed() {
		metrics.WSSendDropped()
	}
	if frame.closeCode != 0 || frame.normalClose {
		c.force()
	}
}

func (c *Conn) errorOutbound(msg inbound, err error) outbound {
	class := classify.Classify(err)
	if !isLogged(err) {
		classify.Log(c.log, class, c.context(), classify.Scope{
			Transport: "websocket",
			Component: "websocket",
			Method:    msg.Type,
		})
	}
	frame := outbound{}
	if decision, ok := transport.CloseDecision(class); ok {
		frame.closeCode = decision.Code
		frame.closeReason = decision.Reason
	}
	if class.Socket == errcore.SocketClose {
		return frame
	}
	payload, merr := encodeError(msg.ID, class)
	if merr != nil {
		return frame
	}
	frame.payload = payload
	return frame
}
