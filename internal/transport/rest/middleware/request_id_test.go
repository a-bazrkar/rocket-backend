package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/gin-gonic/gin"
)

func TestRequestIDReusesHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		id, ok := errcore.RequestID(c.Request.Context())
		if !ok || id != "abc-123" {
			t.Errorf("id = %q ok=%v", id, ok)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-ID") != "abc-123" {
		t.Fatalf("header = %q", w.Header().Get("X-Request-ID"))
	}
}

func TestRequestIDReplacesUnsafeHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		id, ok := errcore.RequestID(c.Request.Context())
		if !ok || strings.Contains(id, "\n") || id == "" {
			t.Errorf("id = %q ok=%v", id, ok)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "bad\nid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := w.Header().Get("X-Request-ID")
	if got == "" || strings.Contains(got, "bad") || len(got) != 32 {
		t.Fatalf("header = %q", got)
	}
}
