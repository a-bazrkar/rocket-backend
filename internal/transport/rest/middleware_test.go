package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestServerPanicStaysInternal(t *testing.T) {
	s := New(zerolog.Nop())
	s.engine.GET("/panic", func(c *gin.Context) {
		panic("server-panic-token")
	})

	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "server-panic-token") || strings.Contains(w.Body.String(), "goroutine") {
		t.Fatalf("body = %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"internal.panic_recovered"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
	if len(w.Header().Get("X-Request-ID")) != 32 {
		t.Fatalf("request id = %q", w.Header().Get("X-Request-ID"))
	}
}
