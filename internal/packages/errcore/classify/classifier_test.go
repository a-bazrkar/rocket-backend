package classify

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
)

func TestMain(m *testing.M) {
	errs.MustLoad()
	os.Exit(m.Run())
}

func TestClassifyKnownAndWrapped(t *testing.T) {
	err := errcore.New(nil, errs.UserNotFound)
	class := Classify(err)
	if class.Code != errs.UserNotFound || class.Category != errcore.CategoryNotFound {
		t.Fatalf("class = %+v", class)
	}
	if class.Message != "User not found" || class.HTTPStatus != 404 || class.LogLevel != errcore.LogInfo {
		t.Fatalf("class = %+v", class)
	}
	if !class.ShouldLog || class.ShouldAlert || class.Retryable {
		t.Fatalf("flags = %+v", class)
	}

	cause := stderrors.New(`sql: SELECT password FROM users WHERE token = "secret"`)
	wrapped := fmt.Errorf("repo: %w", errcore.Wrap(nil, errs.DatabaseQueryFailed, cause))
	class = Classify(wrapped)
	if class.Code != errs.DatabaseQueryFailed || class.HTTPStatus != 500 || class.LogLevel != errcore.LogError {
		t.Fatalf("class = %+v", class)
	}
	if class.Message != "A database operation failed" {
		t.Fatalf("message = %q", class.Message)
	}
	if strings.Contains(class.Message, "SELECT") || strings.Contains(class.Message, "password") {
		t.Fatalf("public message leaked internals: %q", class.Message)
	}
	if !stderrors.Is(class.Cause, cause) {
		t.Fatal("original cause was dropped")
	}
}

func TestClassifyUnknownAndNil(t *testing.T) {
	if class := Classify(nil); class.ShouldLog || class.Code != "" || class.Message != "" || class.HTTPStatus != 0 {
		t.Fatalf("nil class = %+v", class)
	}

	secret := stderrors.New("dial tcp password=hunter2")
	class := Classify(secret)
	if class.Code != errcore.CodeInternalUnexpectedError || class.Category != errcore.CategoryInternal {
		t.Fatalf("class = %+v", class)
	}
	if class.Message != "An unexpected error occurred" || class.HTTPStatus != 500 || class.LogLevel != errcore.LogError {
		t.Fatalf("class = %+v", class)
	}
	if !class.ShouldLog || !class.ShouldAlert {
		t.Fatalf("flags = %+v", class)
	}
	if strings.Contains(class.Message, "hunter2") || class.Message == secret.Error() {
		t.Fatalf("public message = %q", class.Message)
	}

	unregistered := errcore.New(nil, "no.such_code")
	class = Classify(unregistered)
	if class.Code != errcore.CodeInternalUnexpectedError || class.Message != "An unexpected error occurred" {
		t.Fatalf("class = %+v", class)
	}
	if class.Metadata["unregistered_code"] != "no.such_code" {
		t.Fatalf("metadata = %#v", class.Metadata)
	}
}

func TestClassifyLevels(t *testing.T) {
	cases := []struct {
		code  errcore.Code
		level errcore.LogLevel
	}{
		{errs.ValidationInvalidInput, errcore.LogInfo},
		{errs.UserNotFound, errcore.LogInfo},
		{errs.UserInvalidCredentials, errcore.LogWarn},
		{errs.AuthUnauthorized, errcore.LogWarn},
		{errs.RateLimitExceeded, errcore.LogWarn},
		{errs.DatabaseConnectionFailed, errcore.LogError},
		{errs.ExternalServiceUnavailable, errcore.LogError},
		{errcore.CodeInternalUnexpectedError, errcore.LogError},
		{errcore.CodeInternalPanicRecovered, errcore.LogError},
	}
	for _, tc := range cases {
		class := Classify(errcore.New(nil, tc.code))
		if class.LogLevel != tc.level {
			t.Fatalf("%s level = %s, want %s", tc.code, class.LogLevel, tc.level)
		}
	}
	conn := Classify(errcore.New(nil, errs.DatabaseConnectionFailed))
	if !conn.ShouldAlert || !conn.Retryable {
		t.Fatalf("connection flags = %+v", conn)
	}
	rate := Classify(errcore.New(nil, errs.RateLimitExceeded))
	if !rate.Retryable || rate.ShouldAlert || rate.HTTPStatus != 429 {
		t.Fatalf("rate = %+v", rate)
	}

	credentials := Classify(errcore.New(nil, errs.UserInvalidCredentials))
	if credentials.Socket != errcore.SocketClose || credentials.CloseCode != errcore.CloseCodeAuth {
		t.Fatalf("credentials socket = %q code=%d", credentials.Socket, credentials.CloseCode)
	}
	forbidden := Classify(errcore.New(nil, errs.AuthForbidden))
	if forbidden.Socket != errcore.SocketEvent || forbidden.CloseCode != 0 {
		t.Fatalf("forbidden socket = %q code=%d", forbidden.Socket, forbidden.CloseCode)
	}
}

func TestLogRedactsMetadataAndKeepsContext(t *testing.T) {
	var buf bytes.Buffer
	log := logger.Writer(&buf)
	err := errcore.New(nil, errs.AuthInvalidToken).WithMetadata(map[string]any{
		"password": "s3cret",
		"token":    "raw-token",
		"user_id":  "do-not-promote",
	})
	class := Classify(err)
	if class.Metadata["password"] != "[redacted]" || class.Metadata["token"] != "[redacted]" {
		t.Fatalf("metadata = %#v", class.Metadata)
	}
	if class.Metadata["user_id"] != "do-not-promote" {
		t.Fatalf("safe metadata = %#v", class.Metadata)
	}

	ctx := errcore.WithRequestID(context.Background(), "req-1")
	ctx = errcore.WithTraceID(ctx, "trace-1")
	ctx = errcore.WithUserID(ctx, 42)
	ctx = errcore.WithConversationID(ctx, 7)
	ctx = errcore.WithConnectionID(ctx, "conn-1")
	Log(log, class, ctx, Scope{Transport: "http", Component: "auth"})

	line := buf.String()
	for _, part := range []string{
		`"error_code":"auth.invalid_token"`,
		`"error_category":"AUTHENTICATION"`,
		`"request_id":"req-1"`,
		`"trace_id":"trace-1"`,
		`"user_id":42`,
		`"conversation_id":7`,
		`"connection_id":"conn-1"`,
		`"transport":"http"`,
		`"component":"auth"`,
		`[redacted]`,
	} {
		if !strings.Contains(line, part) {
			t.Fatalf("log %q missing %s", line, part)
		}
	}
	if strings.Contains(line, "s3cret") || strings.Contains(line, "raw-token") {
		t.Fatalf("log leaked secrets: %s", line)
	}

	key := class.MetricKey("http")
	if key != (MetricKey{Code: "auth.invalid_token", Category: "AUTHENTICATION", Transport: "http"}) {
		t.Fatalf("metric key = %+v", key)
	}
}

func TestLogKeepsSnapshotWithoutLiveContext(t *testing.T) {
	ctx := errcore.WithRequestID(context.Background(), "req-snap")
	err := errcore.New(ctx, errs.UserNotFound)
	var buf bytes.Buffer
	Log(logger.Writer(&buf), Classify(err), context.Background(), Scope{Transport: "http", Component: "http"})
	if !strings.Contains(buf.String(), "req-snap") {
		t.Fatalf("log dropped snapshot: %s", buf.String())
	}
}

func TestLogFillsMissingIDFromLiveContext(t *testing.T) {
	built := errcore.WithRequestID(context.Background(), "req-snap")
	err := errcore.New(built, errs.UserNotFound)
	live := errcore.WithUserID(context.Background(), 9)
	var buf bytes.Buffer
	Log(logger.Writer(&buf), Classify(err), live, Scope{Transport: "http", Component: "http"})
	line := buf.String()
	if !strings.Contains(line, "req-snap") || !strings.Contains(line, `"user_id":9`) {
		t.Fatalf("log = %s", line)
	}
}

func TestLogSkipsNilClassification(t *testing.T) {
	var buf bytes.Buffer
	Log(logger.Writer(&buf), Classify(nil), context.Background(), Scope{Transport: "http", Component: "http"})
	if buf.Len() != 0 {
		t.Fatalf("logged nil error: %s", buf.String())
	}
}

func TestLogRedactsNestedMetadata(t *testing.T) {
	err := errcore.New(nil, errs.UserNotFound).WithMetadata(map[string]any{
		"items": []any{
			map[string]any{"password": "nested-secret", "id": 1},
		},
		"rows": []map[string]any{
			{"access_token": "raw-access", "name": "ada"},
		},
		"note": "visible",
	})
	class := Classify(err)
	items, ok := class.Metadata["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", class.Metadata["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["password"] != "[redacted]" || item["id"] != 1 {
		t.Fatalf("item = %#v", items[0])
	}
	rows, ok := class.Metadata["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows = %#v", class.Metadata["rows"])
	}
	row, ok := rows[0].(map[string]any)
	if !ok || row["access_token"] != "[redacted]" || row["name"] != "ada" {
		t.Fatalf("row = %#v", rows[0])
	}
	if class.Metadata["note"] != "visible" {
		t.Fatalf("note = %#v", class.Metadata["note"])
	}

	var buf bytes.Buffer
	Log(logger.Writer(&buf), class, context.Background(), Scope{Transport: "http", Component: "http"})
	line := buf.String()
	if strings.Contains(line, "nested-secret") || strings.Contains(line, "raw-access") {
		t.Fatalf("log leaked nested secrets: %s", line)
	}
	if !strings.Contains(line, "visible") || !strings.Contains(line, "[redacted]") {
		t.Fatalf("log = %s", line)
	}
}
