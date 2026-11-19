package recovery

import (
	"context"
	"runtime/debug"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/gin-gonic/gin"
)

// Gin returns Gin's official recovery middleware.
// The nil writer skips Gin's own logger. The panic is logged once through the application logger.
// A body that was already written is left unchanged.
func Gin(log logger.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		method, path := "", ""
		if c != nil && c.Request != nil {
			method = c.Request.Method
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
		}
		class := report(log, recovered, debug.Stack(), requestContext(c), classify.Scope{
			Transport: "http",
			Component: "gin",
			Method:    method,
			Path:      path,
		})
		if c.Writer.Written() {
			c.Abort()
			return
		}
		transport.WriteHTTPError(c, class)
	})
}

func requestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil || c.Request.Context() == nil {
		return context.Background()
	}
	return c.Request.Context()
}
