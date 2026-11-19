package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestID stores a request id on the request context and echoes it.
// A usable incoming header is kept. Anything else is replaced with 16 random bytes.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := cleanRequestID(c.GetHeader(requestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		ctx := errcore.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func cleanRequestID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return ""
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return id
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}
