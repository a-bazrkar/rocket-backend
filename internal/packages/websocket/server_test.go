package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	gorillaws "github.com/gorilla/websocket"
)

func TestServerFramesRoomsAndShutdown(t *testing.T) {
	httpSrv, hub, srv := startServer(t, DefaultConfig(), nil, WithOnConnect(func(_ context.Context, conn *Conn) error {
		if conn.UserID() == 7 {
			return conn.hub.Join(conn.ID(), RoomOf("conversation", 5))
		}
		return nil
	}), WithAuthorizer(func(_ context.Context, userID int64, room Room) error {
		if userID == 8 && room == RoomOf("conversation", 5) {
			return nil
		}
		return errcore.New(nil, errs.AuthForbidden)
	}))
	srv.Router().Handle("echo", func(c *Context) (any, error) {
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return nil, err
		}
		return body, nil
	})
	srv.Router().Handle("fail", func(c *Context) (any, error) {
		return nil, errcore.New(c, errs.MessageSendFailed)
	})
	srv.Router().Handle("boom", func(*Context) (any, error) {
		panic("secret-boom")
	})
	srv.Router().Handle("kick", func(c *Context) (any, error) {
		return nil, errcore.New(c, errs.AuthUnauthorized)
	})
	srv.Router().Handle("tick", func(*Context) (any, error) {
		return map[string]bool{"ok": true}, nil
	}, RateLimit(Limit{Rate: 0.01, Burst: 1}))

	owner := dial(t, httpSrv.URL, "uid=7")
	ready(t, owner)
	member := dial(t, httpSrv.URL, "uid=8")
	ready(t, member)

	writeFrame(t, owner, `{"type":"echo","data":{"n":1}}`)
	writeFrame(t, owner, `{"type":"echo","id":"e1","data":{"n":1}}`)
	got := readFrame(t, owner)
	if got["type"] != "echo" || got["id"] != "e1" {
		t.Fatalf("echo = %#v", got)
	}

	writeFrame(t, owner, `{"type":"fail","id":"f1"}`)
	got = readFrame(t, owner)
	if got["type"] != "error" || got["id"] != "f1" {
		t.Fatalf("fail = %#v", got)
	}
	errObj, _ := got["error"].(map[string]any)
	if errObj["code"] != "message.send_failed" || errObj["message"] != "Unable to send message" {
		t.Fatalf("error = %#v", got["error"])
	}

	writeFrame(t, owner, `{"type":"not-a-real-method","id":"u1"}`)
	got = readFrame(t, owner)
	errObj, _ = got["error"].(map[string]any)
	if got["type"] != "error" || errObj["code"] != "websocket.method_not_found" {
		t.Fatalf("unknown = %#v", got)
	}

	writeFrame(t, owner, `{"type":"boom","id":"b1"}`)
	got = readFrame(t, owner)
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "secret-boom") {
		t.Fatalf("panic leaked: %s", raw)
	}
	errObj, _ = got["error"].(map[string]any)
	if errObj["code"] != "internal.panic_recovered" {
		t.Fatalf("panic = %#v", got)
	}
	writeFrame(t, owner, `{"type":"ping","id":"p1"}`)
	got = readFrame(t, owner)
	if got["type"] != "ping" || got["id"] != "p1" {
		t.Fatalf("ping after panic = %#v", got)
	}

	writeFrame(t, owner, `{"type":"tick","id":"t1"}`)
	if readFrame(t, owner)["type"] != "tick" {
		t.Fatal("first tick failed")
	}
	writeFrame(t, owner, `{"type":"tick","id":"t2"}`)
	got = readFrame(t, owner)
	errObj, _ = got["error"].(map[string]any)
	if errObj["code"] != "rate_limit.exceeded" {
		t.Fatalf("rate limit = %#v", got)
	}

	hub.PublishRoom(RoomOf("conversation", 5), Event{Type: "message.created", Data: map[string]int{"n": 9}})
	got = readFrame(t, owner)
	if got["type"] != "message.created" {
		t.Fatalf("owner push = %#v", got)
	}
	expectQuiet(t, member)

	writeFrame(t, member, `{"type":"room.join","id":"j1","data":{"room":"conversation:5"}}`)
	got = readFrame(t, member)
	if got["type"] != "room.join" || got["id"] != "j1" {
		t.Fatalf("join = %#v", got)
	}
	hub.PublishRoom(RoomOf("conversation", 5), Event{Type: "message.created", Data: map[string]int{"n": 2}})
	if readFrame(t, member)["type"] != "message.created" {
		t.Fatal("joined member missed the room")
	}
	if readFrame(t, owner)["type"] != "message.created" {
		t.Fatal("owner missed the room")
	}

	writeFrame(t, owner, `{"type":"room.join","id":"j2","data":{"room":"conversation:5"}}`)
	got = readFrame(t, owner)
	errObj, _ = got["error"].(map[string]any)
	if errObj["code"] != "auth.forbidden" {
		t.Fatalf("join denied = %#v", got)
	}

	hub.PublishUser(7, Event{Type: "typing.start"})
	if readFrame(t, owner)["type"] != "typing.start" {
		t.Fatal("typing missed the user")
	}
	expectQuiet(t, member)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := hub.Close(ctx); err != nil {
		t.Fatal(err)
	}
	got = readFrame(t, owner)
	if got["type"] != eventClosing {
		t.Fatalf("shutdown = %#v", got)
	}
	code := readClose(t, owner)
	if code != gorillaws.CloseNormalClosure {
		t.Fatalf("close code = %d", code)
	}
}

func TestAuthCloseAndHTTPRejections(t *testing.T) {
	httpSrv, _, srv := startServer(t, DefaultConfig(), func(r *http.Request) (Identity, error) {
		if r.URL.Query().Get("fail") == "1" {
			return Identity{}, errcore.New(r.Context(), errs.AuthUnauthorized)
		}
		return Identity{UserID: 3}, nil
	})
	srv.Router().Handle("kick", func(c *Context) (any, error) {
		return nil, errcore.New(c, errs.AuthUnauthorized)
	})

	resp := getOrigin(t, httpSrv.URL+"?fail=1", "")
	if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(resp.Body, `"code":"auth.unauthorized"`) {
		t.Fatalf("auth = %d %s", resp.StatusCode, resp.Body)
	}
	resp = getOrigin(t, httpSrv.URL, "https://evil.example")
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(resp.Body, `"code":"auth.forbidden"`) {
		t.Fatalf("origin = %d %s", resp.StatusCode, resp.Body)
	}

	conn := dial(t, httpSrv.URL, "")
	writeFrame(t, conn, `{"type":"kick","id":"k1"}`)
	if code := readClose(t, conn); code != 4001 {
		t.Fatalf("close code = %d", code)
	}
}

func TestConnectionLimitAndTokenExpiry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConnsPerUser = 1
	cfg.PingPeriod = 20 * time.Millisecond
	cfg.PongWait = time.Second
	cfg.WriteWait = 100 * time.Millisecond
	httpSrv, hub, _ := startServer(t, cfg, func(r *http.Request) (Identity, error) {
		id, _ := strconv.ParseInt(r.URL.Query().Get("uid"), 10, 64)
		if id == 0 {
			id = 4
		}
		identity := Identity{UserID: id}
		if r.URL.Query().Get("exp") == "1" {
			identity.ExpiresAt = time.Now().Add(30 * time.Millisecond)
		}
		return identity, nil
	})

	hold := dial(t, httpSrv.URL, "uid=4")
	waitCount(t, hub, 1)
	resp := getOrigin(t, httpSrv.URL+"?uid=4", "")
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(resp.Body, `"code":"websocket.connection_limit"`) {
		t.Fatalf("limit = %d %s", resp.StatusCode, resp.Body)
	}
	_ = hold.Close()

	expiring := dial(t, httpSrv.URL, "uid=5&exp=1")
	if code := readClose(t, expiring); code != 4001 {
		t.Fatalf("expiry close = %d", code)
	}
}

func TestAllowedOrigin(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowedOrigins = []string{"https://app.example"}
	httpSrv, _, _ := startServer(t, cfg, nil)

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL(httpSrv.URL), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	header := http.Header{"Origin": []string{"https://app.example"}}
	allowed, _, err := gorillaws.DefaultDialer.Dial(wsURL(httpSrv.URL), header)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = allowed.Close() })

	_, resp, err := gorillaws.DefaultDialer.Dial(wsURL(httpSrv.URL), http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		t.Fatal("evil origin was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("evil status = %v err = %v", resp, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "auth.forbidden") {
		t.Fatalf("body = %s", body)
	}
}

type httpResult struct {
	StatusCode int
	Body       string
}

func startServer(t *testing.T, cfg Config, auth Authenticator, opts ...Option) (*httptest.Server, *Hub, *Server) {
	t.Helper()
	if auth == nil {
		auth = func(r *http.Request) (Identity, error) {
			id := int64(1)
			if raw := r.URL.Query().Get("uid"); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 64)
				if err != nil || n == 0 {
					return Identity{}, errcore.New(r.Context(), errs.ValidationInvalidInput)
				}
				id = n
			}
			return Identity{UserID: id}, nil
		}
	}
	hub, err := NewHub(cfg, logger.Nop())
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(hub, logger.Nop(), auth, opts...)
	if err != nil {
		t.Fatal(err)
	}
	httpSrv := httptest.NewServer(srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = hub.Close(ctx)
		httpSrv.Close()
	})
	return httpSrv, hub, srv
}

func dial(t *testing.T, httpURL, query string) *gorillaws.Conn {
	t.Helper()
	url := wsURL(httpURL)
	if query != "" {
		url += "?" + query
	}
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func ready(t *testing.T, conn *gorillaws.Conn) {
	t.Helper()
	writeFrame(t, conn, `{"type":"ping","id":"ready"}`)
	got := readFrame(t, conn)
	if got["type"] != "ping" || got["id"] != "ready" {
		t.Fatalf("ready = %#v", got)
	}
}

func writeFrame(t *testing.T, conn *gorillaws.Conn, payload string) {
	t.Helper()
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(gorillaws.TextMessage, []byte(payload)); err != nil {
		t.Fatal(err)
	}
}

func readFrame(t *testing.T, conn *gorillaws.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("json %s: %v", data, err)
	}
	return body
}

func expectQuiet(t *testing.T, conn *gorillaws.Conn) {
	t.Helper()
	writeFrame(t, conn, `{"type":"ping","id":"quiet"}`)
	got := readFrame(t, conn)
	if got["type"] != "ping" || got["id"] != "quiet" {
		t.Fatalf("unexpected frame %#v", got)
	}
}

func readClose(t *testing.T, conn *gorillaws.Conn) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		var closed *gorillaws.CloseError
		if errors.As(err, &closed) {
			return closed.Code
		}
	}
	t.Fatal("connection did not close")
	return 0
}

func getOrigin(t *testing.T, url, origin string) httpResult {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return httpResult{StatusCode: resp.StatusCode, Body: string(body)}
}

func waitCount(t *testing.T, hub *Hub, n int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if hub.count() >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("connections = %d, want %d", hub.count(), n)
}
