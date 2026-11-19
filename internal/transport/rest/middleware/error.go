package middleware

import (
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Errors writes a leftover Gin error after the handler returns.
// AbortWithError from a handler marks the error handled, so this does not log it again.
func Errors(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		transport.SetLogger(c, log)
		c.Next()
		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}
		last := c.Errors.Last()
		if last == nil || last.Err == nil {
			return
		}
		transport.AbortWithError(c, last.Err)
	}
}
