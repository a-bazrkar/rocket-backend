package errcore

import (
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

const (
	codeUserNotFound         Code = "user.not_found"
	codeDatabaseQueryFailed  Code = "database.query_failed"
	codeConversationNotFound Code = "conversation.not_found"
	codeMessageNotFound      Code = "message.not_found"
	codeAuthInvalidToken     Code = "auth.invalid_token"
)

func TestMain(m *testing.M) {
	reg, err := NewRegistry([]Definition{
		{Code: codeUserNotFound, Category: CategoryNotFound, Message: "User not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeDatabaseQueryFailed, Category: CategoryDatabase, Message: "A database operation failed", HTTPStatus: 500, LogLevel: LogError},
		{Code: codeConversationNotFound, Category: CategoryNotFound, Message: "Conversation not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeMessageNotFound, Category: CategoryNotFound, Message: "Message not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeAuthInvalidToken, Category: CategoryAuthentication, Message: "Invalid token", HTTPStatus: 401, LogLevel: LogWarn, Socket: SocketClose, CloseCode: CloseCodeAuth},
	})
	if err != nil {
		panic(err)
	}
	SetDefault(reg)
	os.Exit(m.Run())
}

func TestNewUsesRegistry(t *testing.T) {
	err := New(nil, codeUserNotFound)
	if err.Code != codeUserNotFound || err.Category != CategoryNotFound || err.Message != "User not found" {
		t.Fatalf("error = %+v", err)
	}
	if err.Cause != nil || err.detail != "" {
		t.Fatalf("new error carries cause or detail: %+v", err)
	}
}

func TestWrapUnwrapAndIs(t *testing.T) {
	cause := stderrors.New("sql connection")
	err := Wrap(nil, codeDatabaseQueryFailed, cause)
	if !stderrors.Is(err, cause) {
		t.Fatal("cause was dropped")
	}
	if stderrors.Unwrap(err) != cause {
		t.Fatal("unwrap mismatch")
	}
	if !stderrors.Is(err, New(nil, codeDatabaseQueryFailed)) {
		t.Fatal("same code did not match")
	}
	if stderrors.Is(err, New(nil, codeUserNotFound)) {
		t.Fatal("different code matched")
	}

	wrapped := fmt.Errorf("load user: %w", New(nil, codeConversationNotFound))
	if !stderrors.Is(wrapped, New(nil, codeConversationNotFound)) {
		t.Fatal("wrapped application error was not found")
	}
	var target *Error
	if !stderrors.As(wrapped, &target) {
		t.Fatal("errors.As failed")
	}
	if target.Code != codeConversationNotFound || target.Message != "Conversation not found" {
		t.Fatalf("as = %+v", target)
	}
}

func TestWrapWithMessageKeepsPublicMessage(t *testing.T) {
	cause := io.EOF
	err := WrapWithMessage(nil, codeUserNotFound, cause, "SELECT secret FROM users")
	if err.Message != "User not found" {
		t.Fatalf("public message = %q", err.Message)
	}
	if err.detail != "SELECT secret FROM users" {
		t.Fatalf("detail = %q", err.detail)
	}
	text := err.Error()
	for _, part := range []string{"user.not_found", "User not found", "SELECT secret FROM users", "EOF"} {
		if !strings.Contains(text, part) {
			t.Fatalf("Error() = %q, missing %q", text, part)
		}
	}
}

func TestWithMetadataCopies(t *testing.T) {
	base := New(nil, codeMessageNotFound)
	first := base.WithMetadata(map[string]any{"id": 1, "": "skip"})
	second := first.WithMetadata(map[string]any{"id": 2, "room": "a"})
	if base.Metadata != nil {
		t.Fatalf("base metadata = %#v", base.Metadata)
	}
	if first.Metadata["id"] != 1 || len(first.Metadata) != 1 {
		t.Fatalf("first metadata = %#v", first.Metadata)
	}
	if _, ok := first.Metadata[""]; ok {
		t.Fatal("empty key was stored")
	}
	if second.Metadata["id"] != 2 || second.Metadata["room"] != "a" {
		t.Fatalf("second metadata = %#v", second.Metadata)
	}
}

func TestLookupAndFallback(t *testing.T) {
	def, ok := Lookup(codeAuthInvalidToken)
	if !ok || def.Message != "Invalid token" || def.HTTPStatus != 401 || def.Category != CategoryAuthentication {
		t.Fatalf("lookup = %+v ok=%v", def, ok)
	}
	if _, ok := Lookup(Code("missing.code")); ok {
		t.Fatal("unknown code was registered")
	}
	fb := Fallback()
	if fb.Code != CodeInternalUnexpectedError || fb.Message != "An unexpected error occurred" || fb.HTTPStatus != 500 || !fb.ShouldAlert {
		t.Fatalf("fallback = %+v", fb)
	}
}

func TestUnknownCodeStillBuilds(t *testing.T) {
	err := New(nil, Code("no.such_code"))
	if err.Code != "no.such_code" || err.Message != "" || err.Category != CategoryInternal {
		t.Fatalf("error = %+v", err)
	}
}

func TestNewRegistryRejectsDuplicate(t *testing.T) {
	def := Definition{Code: "test.duplicate", Category: CategoryNotFound, Message: "Duplicate", HTTPStatus: 404, LogLevel: LogInfo}
	if _, err := NewRegistry([]Definition{def, def}); err == nil {
		t.Fatal("duplicate code was accepted")
	}
}

func TestNewRegistryRejectsInvalidSocketPolicy(t *testing.T) {
	base := Definition{Code: "test.socket", Category: CategoryNotFound, Message: "Missing", HTTPStatus: 404, LogLevel: LogInfo}
	withCode := base
	withCode.CloseCode = CloseCodeAuth
	if _, err := NewRegistry([]Definition{withCode}); err == nil {
		t.Fatal("close code without a close policy was accepted")
	}
	closing := base
	closing.Socket = SocketClose
	if _, err := NewRegistry([]Definition{closing}); err == nil {
		t.Fatal("close policy without a private close code was accepted")
	}
	closing.CloseCode = 1000
	if _, err := NewRegistry([]Definition{closing}); err == nil {
		t.Fatal("protocol close code was accepted")
	}
}

func TestLookupIsConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				if _, ok := Lookup(codeUserNotFound); !ok {
					t.Error("missing user.not_found")
					return
				}
				if len(Codes()) == 0 {
					t.Error("empty code list")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestValidateRejectsDuplicatesAndInvalidDefinitions(t *testing.T) {
	valid := Definition{Code: codeUserNotFound, Category: CategoryNotFound, Message: "User not found", HTTPStatus: 404, LogLevel: LogInfo}
	if err := Validate([]Definition{valid, valid}); err == nil {
		t.Fatal("duplicate code was accepted")
	}
	cases := []Definition{
		{Code: "Bad.Code", Category: CategoryNotFound, Message: "User not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: "single", Category: CategoryNotFound, Message: "User not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeUserNotFound, Category: "NOPE", Message: "User not found", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeUserNotFound, Category: CategoryNotFound, Message: "", HTTPStatus: 404, LogLevel: LogInfo},
		{Code: codeUserNotFound, Category: CategoryNotFound, Message: "User not found", HTTPStatus: 200, LogLevel: LogInfo},
		{Code: codeUserNotFound, Category: CategoryNotFound, Message: "User not found", HTTPStatus: 404, LogLevel: "fatal"},
	}
	for _, def := range cases {
		if err := Validate([]Definition{def}); err == nil {
			t.Fatalf("accepted invalid definition %+v", def)
		}
	}
}
