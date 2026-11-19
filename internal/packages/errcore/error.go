package errcore

import "context"

// Error is an application error. Message is the registry text captured at
// construction. detail is an internal note and is never a client response.
// Cause and Metadata are for logs.
type Error struct {
	Code     Code
	Category Category
	Message  string
	Cause    error
	Metadata map[string]any

	detail string
	op     Operation
}

// Operation is the request identity copied from a context at construction.
// Frames is set only when debug is on. The context value itself is not stored.
type Operation struct {
	RequestID         string
	TraceID           string
	UserID            int64
	HasUserID         bool
	ConversationID    int64
	HasConversationID bool
	ConnectionID      string
	Frames            []string
}

// Error returns an internal description. It includes the cause so logs can
// show the failure chain. Client responses must use the registry message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := string(e.Code)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.detail != "" {
		msg += ": " + e.detail
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the wrapped cause.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is reports whether target is an application error with the same code.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil || e.Code == "" {
		return false
	}
	other, ok := target.(*Error)
	if !ok || other == nil || other.Code == "" {
		return false
	}
	return e.Code == other.Code
}

// WithMetadata returns a copy that includes kv. The receiver is left unchanged.
// Empty keys are skipped. Values are for structured logs, not for clients.
func (e *Error) WithMetadata(kv map[string]any) *Error {
	if e == nil {
		return nil
	}
	cp := *e
	cp.Metadata = make(map[string]any, len(e.Metadata)+len(kv))
	for k, v := range e.Metadata {
		cp.Metadata[k] = v
	}
	for k, v := range kv {
		if k == "" {
			continue
		}
		cp.Metadata[k] = v
	}
	if len(cp.Metadata) == 0 {
		cp.Metadata = nil
	}
	return &cp
}

// Operation returns the ids and, when debug is on, the caller frames captured at construction.
func (e *Error) Operation() Operation {
	if e == nil {
		return Operation{}
	}
	return e.op
}

func build(ctx context.Context, code Code, cause error, detail string) *Error {
	e := &Error{
		Code:     code,
		Category: CategoryInternal,
		Cause:    cause,
		detail:   detail,
		op:       captureOperation(ctx),
	}
	if Debug() {
		e.op.Frames = captureFrames()
	}
	if def, ok := Lookup(code); ok {
		e.Category = def.Category
		e.Message = def.Message
	}
	return e
}
