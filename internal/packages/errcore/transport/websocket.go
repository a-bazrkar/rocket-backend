package transport

import (
	stderrors "errors"
	"fmt"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/gorilla/websocket"
)

const (
	// CloseCodePanic is the private-use close code for a recovered panic.
	CloseCodePanic = errcore.CloseCodePanic
	// CloseCodeAuth is the private-use close code for an authentication failure
	// whose definition uses SocketClose.
	CloseCodeAuth = errcore.CloseCodeAuth
)

// JSONWriter is the single data-frame writer for an error event.
// The caller serializes writes. This package does not take a write lock.
type JSONWriter interface {
	WriteJSON(v any) error
}

// Close is the decision to end a WebSocket connection.
type Close struct {
	Code   int
	Reason string
}

// CloseDecision reports whether class ends the connection.
// The decision comes from the definition's socket policy, not from its category.
func CloseDecision(class classify.Classification) (Close, bool) {
	if class.Socket != errcore.SocketClose && class.Socket != errcore.SocketEventClose {
		return Close{}, false
	}
	public := Public(class)
	return Close{Code: class.CloseCode, Reason: truncateReason(public.Message)}, true
}

// Respond applies the definition's socket policy on the caller's goroutine.
// The caller must already be the only goroutine writing data frames on conn.
// SocketClose sends a close frame. SocketEventClose sends the error event, then the close frame.
// Any other policy sends the error event and leaves the connection open.
func Respond(conn *websocket.Conn, class classify.Classification) error {
	if conn == nil {
		return stderrors.New("transport: nil connection")
	}
	switch class.Socket {
	case errcore.SocketClose:
		decision, _ := CloseDecision(class)
		return WriteClose(conn, decision.Code, decision.Reason)
	case errcore.SocketEventClose:
		if err := WriteEvent(conn, class); err != nil {
			return err
		}
		decision, _ := CloseDecision(class)
		return WriteClose(conn, decision.Code, decision.Reason)
	default:
		return WriteEvent(conn, class)
	}
}

// WriteEvent writes one error event through the single data writer.
// A write error is returned to the caller. This function does not write again.
func WriteEvent(w JSONWriter, class classify.Classification) error {
	if w == nil {
		return stderrors.New("transport: nil writer")
	}
	return w.WriteJSON(EventFrom(class))
}

// WriteConn writes one recoverable error event.
// The caller must be the only goroutine writing data frames on conn.
func WriteConn(conn *websocket.Conn, class classify.Classification) error {
	if conn == nil {
		return stderrors.New("transport: nil connection")
	}
	return WriteEvent(conn, class)
}

// CloseConn sends a close frame when the classification's socket policy closes.
// WriteControl is safe to call while another goroutine writes data frames.
// Classifications that keep the connection open return an error and write nothing.
func CloseConn(conn *websocket.Conn, class classify.Classification) error {
	decision, ok := CloseDecision(class)
	if !ok {
		return stderrors.New("transport: classification does not close the connection")
	}
	return WriteClose(conn, decision.Code, decision.Reason)
}

// WriteClose sends a private-use close frame. code must be in 4000–4999.
// The reason is truncated to the protocol limit and is safe to show to the client.
func WriteClose(conn *websocket.Conn, code int, reason string) error {
	if conn == nil {
		return stderrors.New("transport: nil connection")
	}
	if code < 4000 || code > 4999 {
		return fmt.Errorf("transport: close code %d is outside 4000-4999", code)
	}
	reason = truncateReason(reason)
	if reason == "" {
		reason = errcore.Fallback().Message
	}
	payload := websocket.FormatCloseMessage(code, reason)
	return conn.WriteControl(websocket.CloseMessage, payload, time.Now().Add(time.Second))
}
