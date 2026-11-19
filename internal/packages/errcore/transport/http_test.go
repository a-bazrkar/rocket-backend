package transport

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	errs.MustLoad()
	os.Exit(m.Run())
}

func TestAbortWritesPublicJSON(t *testing.T) {
	w, c := newContext()
	AbortWithError(c, errcore.New(nil, errs.UserNotFound))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", w.Header().Get("Content-Type"))
	}
	body := decodeBody(t, w.Body.Bytes())
	if body.Error.Code != "user.not_found" || body.Error.Message != "User not found" {
		t.Fatalf("body = %+v", body)
	}
	if !c.IsAborted() {
		t.Fatal("context was not aborted")
	}
}

func TestAbortHidesUnknownError(t *testing.T) {
	w, c := newContext()
	AbortWithError(c, stderrors.New("sql: password=hunter2 host=/var/lib/db"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	raw := w.Body.String()
	if strings.Contains(raw, "hunter2") || strings.Contains(raw, "/var/lib") || strings.Contains(raw, "sql:") {
		t.Fatalf("body leaked internals: %s", raw)
	}
	body := decodeBody(t, w.Body.Bytes())
	if body.Error.Code != "internal.unexpected_error" || body.Error.Message != "An unexpected error occurred" {
		t.Fatalf("body = %+v", body)
	}
	assertOnlyPublicKeys(t, w.Body.Bytes())
}

func TestAbortDatabaseErrorUsesRegistryMessage(t *testing.T) {
	w, c := newContext()
	err := errcore.Wrap(nil, errs.DatabaseQueryFailed, stderrors.New("UNIQUE constraint failed: users.email"))
	AbortWithError(c, err)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	raw := w.Body.String()
	if strings.Contains(raw, "UNIQUE") || strings.Contains(raw, "users.email") {
		t.Fatalf("body leaked internals: %s", raw)
	}
	body := decodeBody(t, w.Body.Bytes())
	if body.Error.Code != "database.query_failed" || body.Error.Message != "A database operation failed" {
		t.Fatalf("body = %+v", body)
	}
}

func TestAbortDoesNotWriteTwice(t *testing.T) {
	w, c := newContext()
	AbortWithError(c, errcore.New(nil, errs.UserNotFound))
	AbortWithError(c, errcore.New(nil, errs.UserAlreadyExists))

	body := decodeBody(t, w.Body.Bytes())
	if body.Error.Code != "user.not_found" {
		t.Fatalf("body = %+v", body)
	}
	if strings.Count(w.Body.String(), `"code"`) != 1 {
		t.Fatalf("body written more than once: %s", w.Body.String())
	}
}

func TestWriteHTTPErrorLeavesWrittenResponse(t *testing.T) {
	w, c := newContext()
	c.String(http.StatusOK, "partial")
	WriteHTTPError(c, classify.Classify(errcore.New(nil, errs.UserNotFound)))

	if w.Code != http.StatusOK || w.Body.String() != "partial" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestAbortWithErrorStopsHandlers(t *testing.T) {
	r := gin.New()
	nextCalled := false
	r.GET("/", func(c *gin.Context) {
		AbortWithError(c, errcore.New(nil, errs.UserNotFound))
	}, func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if nextCalled {
		t.Fatal("handler after AbortWithError ran")
	}
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDebugStackIsOmittedUnlessEnabled(t *testing.T) {
	errcore.SetDebug(false)
	w, c := newContext()
	cause := stderrors.New("UNIQUE constraint failed: users.email")
	AbortWithError(c, errcore.Wrap(nil, errs.DatabaseQueryFailed, cause))
	if strings.Contains(w.Body.String(), `"stack"`) || strings.Contains(w.Body.String(), "UNIQUE") {
		t.Fatalf("body = %s", w.Body.String())
	}

	errcore.SetDebug(true)
	t.Cleanup(func() { errcore.SetDebug(false) })
	w, c = newContext()
	AbortWithError(c, errcore.Wrap(nil, errs.DatabaseQueryFailed, cause))
	raw := w.Body.String()
	if !strings.Contains(raw, `"stack"`) {
		t.Fatalf("body = %s", raw)
	}
	if strings.Contains(raw, "UNIQUE") || strings.Contains(raw, "users.email") {
		t.Fatalf("body leaked cause: %s", raw)
	}
	if !strings.Contains(raw, `"code":"database.query_failed"`) {
		t.Fatalf("body = %s", raw)
	}
}

func TestAbortWithErrorOmitsMetadata(t *testing.T) {
	w, c := newContext()
	err := errcore.New(nil, errs.UserNotFound).WithMetadata(map[string]any{
		"password": "s3cret",
		"items":    []any{map[string]any{"token": "raw-token"}},
	})
	AbortWithError(c, err)

	raw := w.Body.String()
	if strings.Contains(raw, "s3cret") || strings.Contains(raw, "raw-token") || strings.Contains(raw, "password") || strings.Contains(raw, "metadata") {
		t.Fatalf("body leaked metadata: %s", raw)
	}
	assertOnlyPublicKeys(t, w.Body.Bytes())
}

func newContext() (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return w, c
}

func decodeBody(t *testing.T, raw []byte) HTTPBody {
	t.Helper()
	var body HTTPBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return body
}

func assertOnlyPublicKeys(t *testing.T, raw []byte) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 {
		t.Fatalf("top-level keys = %#v", body)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok || len(errObj) != 2 {
		t.Fatalf("error object = %#v", body["error"])
	}
	if _, ok := errObj["code"]; !ok {
		t.Fatal("missing code")
	}
	if _, ok := errObj["message"]; !ok {
		t.Fatal("missing message")
	}
}
