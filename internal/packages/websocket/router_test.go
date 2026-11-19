package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
)

func TestMain(m *testing.M) {
	errs.MustLoad()
	os.Exit(m.Run())
}

func TestMiddlewareOrderAndGroupIsolation(t *testing.T) {
	var got []string
	record := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(c *Context) (any, error) {
				got = append(got, name)
				return next(c)
			}
		}
	}
	router := NewRouter()
	router.Use(record("global"))
	parent := router.Group(record("parent"))
	child := parent.Use(record("child"))
	parent.Handle("parent", func(*Context) (any, error) {
		got = append(got, "parent-handler")
		return nil, nil
	})
	child.Handle("child", func(*Context) (any, error) {
		got = append(got, "child-handler")
		return nil, nil
	}, record("method"))

	if _, err := router.Dispatch(frame("parent")); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "global,parent,parent-handler" {
		t.Fatalf("parent order = %v", got)
	}
	got = nil
	if _, err := router.Dispatch(frame("child")); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "global,parent,child,method,child-handler" {
		t.Fatalf("child order = %v", got)
	}

	router.Use(record("later"))
	got = nil
	if _, err := router.Dispatch(frame("child")); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "global,later,parent,child,method,child-handler" {
		t.Fatalf("rebuilt order = %v", got)
	}
}

func TestUnknownMethodErrorEvent(t *testing.T) {
	router := NewRouter()
	c := frame("not-a-real-method")
	c.in.ID = "req-1"
	_, err := router.Dispatch(c)
	if !errors.Is(err, errcore.New(nil, errs.WebsocketMethodNotFound)) {
		t.Fatalf("err = %v", err)
	}
	body := clientError(t, "req-1", err)
	if body["type"] != "error" || body["id"] != "req-1" {
		t.Fatalf("body = %#v", body)
	}
	event, ok := body["error"].(map[string]any)
	if !ok || event["code"] != "websocket.method_not_found" || event["message"] != "Unknown method" {
		t.Fatalf("error = %#v", body["error"])
	}
	note := clientError(t, "", err)
	if _, ok := note["id"]; ok {
		t.Fatalf("empty id was encoded: %#v", note)
	}
	for _, label := range metrics.HandlerMethods() {
		if label == "not-a-real-method" {
			t.Fatal("client method became a metric label")
		}
	}
	found := false
	for _, label := range metrics.HandlerMethods() {
		if label == "unknown" {
			found = true
		}
	}
	if !found {
		t.Fatalf("labels = %v", metrics.HandlerMethods())
	}
}

func TestRecoveryHidesPanic(t *testing.T) {
	router := NewRouter()
	router.Use(Recovery(logger.Nop()))
	router.Handle("boom", func(*Context) (any, error) {
		panic("secret-boom")
	})
	_, err := router.Dispatch(frame("boom"))
	if !errors.Is(err, errcore.New(nil, errcore.CodeInternalPanicRecovered)) {
		t.Fatalf("err = %v", err)
	}
	raw, merr := encodeError("1", classify.Classify(err))
	if merr != nil {
		t.Fatal(merr)
	}
	if strings.Contains(string(raw), "secret-boom") || strings.Contains(string(raw), "goroutine") {
		t.Fatalf("body leaked panic: %s", raw)
	}
}

func TestRateLimitStacks(t *testing.T) {
	router := NewRouter()
	router.Use(RateLimit(Limit{Rate: 10, Burst: 10}))
	router.Handle("tick", func(*Context) (any, error) {
		return "ok", nil
	}, RateLimit(Limit{Rate: 0.01, Burst: 1}))
	router.Handle("other", func(*Context) (any, error) {
		return "ok", nil
	})
	conn := &Conn{}
	first := &Context{Context: context.Background(), Conn: conn, in: inbound{Type: "tick", ID: "1"}}
	if _, err := router.Dispatch(first); err != nil {
		t.Fatal(err)
	}
	second := &Context{Context: context.Background(), Conn: conn, in: inbound{Type: "tick", ID: "2"}}
	_, err := router.Dispatch(second)
	if !errors.Is(err, errcore.New(nil, errs.RateLimitExceeded)) {
		t.Fatalf("err = %v", err)
	}
	other := &Context{Context: context.Background(), Conn: conn, in: inbound{Type: "other", ID: "3"}}
	if _, err := router.Dispatch(other); err != nil {
		t.Fatal(err)
	}
}

func TestRequireUser(t *testing.T) {
	router := NewRouter()
	router.Use(RequireUser)
	router.Handle("ping", func(*Context) (any, error) { return "ok", nil })
	_, err := router.Dispatch(&Context{Context: context.Background(), Conn: &Conn{}, in: inbound{Type: "ping"}})
	if !errors.Is(err, errcore.New(nil, errs.AuthUnauthorized)) {
		t.Fatalf("err = %v", err)
	}
	authed := &Context{Context: context.Background(), Conn: &Conn{userID: 4}, in: inbound{Type: "ping", ID: "9"}}
	result, err := router.Dispatch(authed)
	if err != nil || result != "ok" {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func frame(method string) *Context {
	return &Context{Context: context.Background(), in: inbound{Type: method}}
}

func clientError(t *testing.T, id string, err error) map[string]any {
	t.Helper()
	raw, merr := encodeError(id, classify.Classify(err))
	if merr != nil {
		t.Fatal(merr)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body
}
