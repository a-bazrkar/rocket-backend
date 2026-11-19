package middleware

import (
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	errs.MustLoad()
	os.Exit(m.Run())
}

func TestErrorsWritesLeftoverErrorOnce(t *testing.T) {
	r := gin.New()
	r.Use(Errors(zerolog.Nop()))
	r.GET("/", func(c *gin.Context) {
		_ = c.Error(errcore.New(nil, errs.MessageNotFound))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":"message.not_found"`) || !strings.Contains(w.Body.String(), "Message not found") {
		t.Fatalf("body = %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"stack"`) {
		t.Fatalf("debug stack leaked: %s", w.Body.String())
	}
}

func TestErrorsSkipsHandledAbort(t *testing.T) {
	var buf strings.Builder
	r := gin.New()
	r.Use(Errors(zerolog.New(&buf)))
	r.GET("/", func(c *gin.Context) {
		ctx := errcore.WithRequestID(c.Request.Context(), "req-7")
		c.Request = c.Request.WithContext(ctx)
		transport.AbortWithError(c, errcore.New(nil, errs.UserNotFound))
		_ = c.Error(stderrors.New("second internal failure"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(w.Body.String(), `"code":"user.not_found"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "second internal") {
		t.Fatalf("second error leaked: %s", w.Body.String())
	}
	if strings.Count(buf.String(), `"error_code":"user.not_found"`) != 1 {
		t.Fatalf("log count wrong: %s", buf.String())
	}
	if strings.Contains(buf.String(), "second internal") {
		t.Fatalf("second error was logged by the error middleware: %s", buf.String())
	}
}
