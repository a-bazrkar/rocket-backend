package transport

import (
	"strings"
	"unicode/utf8"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
)

const closeReasonLimit = 123

// PublicError is the only error object returned to clients.
// Stack is present only when debug is on and a stack was captured.
type PublicError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Stack   []string `json:"stack,omitempty"`
}

// HTTPBody is the JSON body for an HTTP error response.
type HTTPBody struct {
	Error PublicError `json:"error"`
}

// Event is the application-level WebSocket error event.
type Event struct {
	Type  string      `json:"type"`
	Error PublicError `json:"error"`
}

const EventTypeError = "error"

// Public returns the client-visible code and message.
// A stack is included only when debug is on.
func Public(class classify.Classification) PublicError {
	class = prepare(class)
	out := PublicError{Code: string(class.Code), Message: class.Message}
	if !errcore.Debug() {
		return out
	}
	if class.Stack != "" {
		out.Stack = splitStack(class.Stack)
		return out
	}
	if len(class.Operation.Frames) > 0 {
		out.Stack = class.Operation.Frames
	}
	return out
}

func splitStack(stack string) []string {
	lines := strings.Split(strings.TrimRight(stack, "\n"), "\n")
	if len(lines) > 64 {
		return lines[:64]
	}
	return lines
}

// HTTPBodyFrom builds the HTTP JSON body.
func HTTPBodyFrom(class classify.Classification) HTTPBody {
	return HTTPBody{Error: Public(class)}
}

// EventFrom builds a recoverable WebSocket error event.
func EventFrom(class classify.Classification) Event {
	return Event{Type: EventTypeError, Error: Public(class)}
}

func prepare(class classify.Classification) classify.Classification {
	fb := errcore.Fallback()
	if class.Code == "" {
		class.Code = fb.Code
	}
	if class.Message == "" {
		class.Message = fb.Message
	}
	if class.HTTPStatus < 400 || class.HTTPStatus > 599 {
		class.HTTPStatus = fb.HTTPStatus
	}
	return class
}

func truncateReason(reason string) string {
	if len(reason) <= closeReasonLimit {
		return reason
	}
	cut := reason[:closeReasonLimit]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}
