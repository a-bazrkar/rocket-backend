package transport

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/gorilla/websocket"
)

type failWriter struct {
	calls int
}

func (w *failWriter) WriteJSON(any) error {
	w.calls++
	return stderrors.New("write failed")
}

func TestWriteEventReturnsWriteError(t *testing.T) {
	w := &failWriter{}
	class := classify.Classify(errcore.New(nil, errs.MessageSendFailed))
	err := WriteEvent(w, class)
	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("err = %v", err)
	}
	if w.calls != 1 {
		t.Fatalf("calls = %d", w.calls)
	}
}

func TestRecoverableEventKeepsConnection(t *testing.T) {
	server, client := wsPair(t)
	class := classify.Classify(errcore.New(nil, errs.MessageSendFailed))
	if _, ok := CloseDecision(class); ok {
		t.Fatal("send failure closes the connection")
	}
	if err := WriteConn(server, class); err != nil {
		t.Fatal(err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != EventTypeError || event.Error.Code != "message.send_failed" || event.Error.Message != "Unable to send message" {
		t.Fatalf("event = %+v", event)
	}

	if err := server.WriteMessage(websocket.TextMessage, []byte("open")); err != nil {
		t.Fatal(err)
	}
	_, data, err = client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "open" {
		t.Fatalf("connection closed, data = %q", data)
	}
}

func TestAuthorizationStaysOpen(t *testing.T) {
	server, client := wsPair(t)
	class := classify.Classify(errcore.New(nil, errs.AuthForbidden))
	if _, ok := CloseDecision(class); ok {
		t.Fatal("authorization error closes the connection")
	}
	if err := Respond(server, class); err != nil {
		t.Fatal(err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != EventTypeError || event.Error.Code != "auth.forbidden" || event.Error.Message != "You do not have permission" {
		t.Fatalf("event = %+v", event)
	}
	if strings.Contains(string(data), "password") || strings.Contains(string(data), "token") {
		t.Fatalf("event leaked metadata: %s", data)
	}

	if err := server.WriteMessage(websocket.TextMessage, []byte("open")); err != nil {
		t.Fatal(err)
	}
	_, data, err = client.ReadMessage()
	if err != nil || string(data) != "open" {
		t.Fatalf("connection closed, data=%q err=%v", data, err)
	}
}

func TestEventCloseWritesThenCloses(t *testing.T) {
	server, client := wsPair(t)
	class := classify.Classification{
		Code:      errs.MessageSendFailed,
		Message:   "Unable to send message",
		Socket:    errcore.SocketEventClose,
		CloseCode: 4002,
	}
	if err := Respond(server, class); err != nil {
		t.Fatal(err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != EventTypeError || event.Error.Code != "message.send_failed" {
		t.Fatalf("event = %+v", event)
	}

	_, data, err = client.ReadMessage()
	if err == nil {
		t.Fatalf("received data frame %s", data)
	}
	var closed *websocket.CloseError
	if !stderrors.As(err, &closed) {
		t.Fatalf("err = %v", err)
	}
	if closed.Code != 4002 || closed.Text != "Unable to send message" {
		t.Fatalf("close = %d %q", closed.Code, closed.Text)
	}
}

func TestAuthCloseUsesPrivateCode(t *testing.T) {
	server, client := wsPair(t)
	class := classify.Classify(errcore.New(nil, errs.AuthInvalidToken))
	if err := CloseConn(server, class); err != nil {
		t.Fatal(err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err == nil {
		t.Fatalf("received data frame %s", data)
	}
	var closed *websocket.CloseError
	if !stderrors.As(err, &closed) {
		t.Fatalf("err = %v", err)
	}
	if closed.Code != CloseCodeAuth {
		t.Fatalf("close code = %d", closed.Code)
	}
	if closed.Text != "Invalid token" || strings.Contains(closed.Text, "token value") {
		t.Fatalf("close text = %q", closed.Text)
	}
}

func TestCloseConnIgnoresRecoverableError(t *testing.T) {
	server, client := wsPair(t)
	class := classify.Classify(errcore.New(nil, errs.MessageSendFailed))
	if err := CloseConn(server, class); err == nil {
		t.Fatal("recoverable error closed the connection")
	}
	if err := server.WriteMessage(websocket.TextMessage, []byte("open")); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "open" {
		t.Fatalf("data = %q", data)
	}
}

func TestWriteCloseRejectsProtocolCodes(t *testing.T) {
	server, _ := wsPair(t)
	if err := WriteClose(server, websocket.CloseInternalServerErr, "hidden"); err == nil {
		t.Fatal("protocol close code was accepted")
	}
}

func wsPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	accepted := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		accepted <- conn
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	server := <-accepted
	t.Cleanup(func() { _ = server.Close() })
	return server, client
}
