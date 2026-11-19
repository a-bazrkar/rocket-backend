package middleware

import (
	"strconv"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/gin-gonic/gin"
)

// Access logs one line for each request after the handler returns.
func Access(log logger.Logger) gin.HandlerFunc {
	if log == nil {
		log = logger.Nop()
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.Request.URL.Path
		log.WithLevel(logger.Info).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("status", strconv.Itoa(c.Writer.Status())).
			Str("latency", time.Since(start).String()).
			Msg("request")
	}
}
