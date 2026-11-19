package recovery

import (
	"context"
	"runtime/debug"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/gorilla/websocket"
)

// ConnContext is the connection data included in a panic log.
// UserID and ConversationID are included when they are non-zero.
type ConnContext struct {
	RequestID      string
	ConnectionID   string
	UserID         int64
	ConversationID int64
}

// Go runs fn in a new goroutine and recovers a panic in that goroutine.
// The panic is logged with cc. When conn is set, a close frame is sent with
// WriteControl so it does not share the data-frame writer. A nil conn is only logged.
func Go(log logger.Logger, conn *websocket.Conn, cc ConnContext, fn func()) {
	if fn == nil {
		return
	}
	go func() {
		defer func() {
			value := recover()
			if value == nil {
				return
			}
			class := report(log, value, debug.Stack(), cc.context(), classify.Scope{
				Transport: "websocket",
				Component: "websocket",
			})
			if conn == nil {
				return
			}
			if err := transport.WriteClose(conn, transport.CloseCodePanic, class.Message); err != nil && log != nil {
				log.WithLevel(logger.Error).Err(err).Str("connection_id", cc.ConnectionID).Msg("websocket close after panic failed")
			}
		}()
		fn()
	}()
}

func (cc ConnContext) context() context.Context {
	ctx := context.Background()
	ctx = errcore.WithRequestID(ctx, cc.RequestID)
	ctx = errcore.WithConnectionID(ctx, cc.ConnectionID)
	if cc.UserID != 0 {
		ctx = errcore.WithUserID(ctx, cc.UserID)
	}
	if cc.ConversationID != 0 {
		ctx = errcore.WithConversationID(ctx, cc.ConversationID)
	}
	return ctx
}
