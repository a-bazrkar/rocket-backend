package websocket

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
)

// Limit is a token bucket. Rate is events per second.
// A non-positive Rate does not limit. Burst below 1 is treated as 1.
type Limit struct {
	Rate  float64
	Burst int
}

type loggedError struct{ error }

func (e loggedError) Unwrap() error { return e.error }

func (loggedError) alreadyLogged() {}

type logMark interface{ alreadyLogged() }

func isLogged(err error) bool {
	var mark logMark
	return errors.As(err, &mark)
}

// Recovery turns a handler panic into an error event.
// The panic is logged once. The connection stays open.
func Recovery(log logger.Logger) Middleware {
	if log == nil {
		log = logger.Nop()
	}
	return func(next Handler) Handler {
		return func(c *Context) (result any, err error) {
			defer func() {
				value := recover()
				if value == nil {
					return
				}
				err = errcore.Wrap(contextOf(c), errcore.CodeInternalPanicRecovered, panicErr(value))
				class := classify.Classify(err)
				class.Stack = string(debug.Stack())
				classify.Log(log, class, contextOf(c), classify.Scope{
					Transport: "websocket",
					Component: "websocket",
					Method:    methodOf(c),
				})
				result = nil
				err = loggedError{err}
			}()
			return next(c)
		}
	}
}

func panicErr(value any) error {
	if err, ok := value.(error); ok && err != nil {
		return err
	}
	return fmt.Errorf("%v", value)
}

func methodOf(c *Context) string {
	if c == nil {
		return ""
	}
	return c.Method()
}

// Access writes one log line after the handler returns.
func Access(log logger.Logger) Middleware {
	if log == nil {
		log = logger.Nop()
	}
	return func(next Handler) Handler {
		return func(c *Context) (any, error) {
			start := time.Now()
			result, err := next(c)
			ev := log.WithLevel(logger.Info)
			if err != nil {
				ev = log.WithLevel(logger.Warn)
				ev = ev.Str("error_code", string(classify.Classify(err).Code))
			}
			ev = ev.Str("type", methodOf(c)).Str("latency", time.Since(start).String())
			if c != nil && c.Conn != nil {
				ev = ev.Str("connection_id", c.Conn.ID()).Int64("user_id", c.Conn.UserID())
			}
			ev.Msg("websocket request")
			return result, err
		}
	}
}

// RequireUser rejects a connection that has no user id.
// The auth.unauthorized definition closes the socket when the frame is written.
func RequireUser(next Handler) Handler {
	return func(c *Context) (any, error) {
		if c == nil || c.Conn == nil || c.Conn.UserID() == 0 {
			return nil, errcore.New(contextOf(c), errs.AuthUnauthorized)
		}
		return next(c)
	}
}

// RateLimit limits one connection.
// The bucket is per method, so two methods do not share a global rate.
// A router limit and a method limit both run when both are installed.
func RateLimit(limit Limit) Middleware {
	if limit.Rate <= 0 {
		return func(next Handler) Handler { return next }
	}
	return func(next Handler) Handler {
		return func(c *Context) (any, error) {
			if c == nil || c.Conn == nil {
				return nil, errcore.New(contextOf(c), errs.WebsocketInvalidMessage)
			}
			if !c.Conn.allow(limitKey(c.Method(), limit), limit) {
				return nil, errcore.New(c.Context, errs.RateLimitExceeded)
			}
			return next(c)
		}
	}
}

func limitKey(method string, limit Limit) string {
	return method + "|" + strconv.FormatFloat(limit.Rate, 'g', -1, 64) + "|" + strconv.Itoa(limit.Burst)
}
