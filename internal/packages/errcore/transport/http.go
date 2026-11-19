package transport

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	contextLoggerKey  = "rocket.transport.logger"
	contextHandledKey = "rocket.transport.handled"
)

// AbortWithError classifies err, logs it once, writes the public JSON body, and aborts.
// A second call on the same request does not log again and does not replace a written body.
func AbortWithError(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}
	class := classify.Classify(err)
	if !isHandled(c) {
		markHandled(c)
		classify.Log(loggerOf(c), class, requestContext(c), classify.Scope{
			Transport: "http",
			Component: "http",
		})
	}
	WriteHTTPError(c, class)
}

// WriteHTTPError writes the public JSON error and aborts the request.
// It does not log. Recovery uses it after the panic has already been logged.
// A response that has already been written is left unchanged.
func WriteHTTPError(c *gin.Context, class classify.Classification) {
	if c == nil {
		return
	}
	if c.Writer.Written() {
		c.Abort()
		return
	}
	class = prepare(class)
	c.AbortWithStatusJSON(class.HTTPStatus, HTTPBodyFrom(class))
}

func requestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	if c.Request.Context() == nil {
		return context.Background()
	}
	return c.Request.Context()
}

func loggerOf(c *gin.Context) zerolog.Logger {
	if c == nil {
		return zerolog.Nop()
	}
	v, ok := c.Get(contextLoggerKey)
	if !ok {
		return zerolog.Nop()
	}
	log, ok := v.(zerolog.Logger)
	if !ok {
		return zerolog.Nop()
	}
	return log
}

func isHandled(c *gin.Context) bool {
	v, ok := c.Get(contextHandledKey)
	if !ok {
		return false
	}
	handled, ok := v.(bool)
	return ok && handled
}

func markHandled(c *gin.Context) {
	c.Set(contextHandledKey, true)
}

// SetLogger stores the request logger used by AbortWithError.
// The HTTP middleware calls it before the handler runs.
func SetLogger(c *gin.Context, log zerolog.Logger) {
	if c == nil {
		return
	}
	c.Set(contextLoggerKey, log)
}
