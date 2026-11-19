package classify

import (
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
)

// Classification is the transport-neutral result of inspecting an error.
// Message is safe for clients. Cause, Metadata, and Stack are for logs.
type Classification struct {
	Code       errcore.Code
	Category   errcore.Category
	Message    string
	HTTPStatus int

	LogLevel    errcore.LogLevel
	ShouldLog   bool
	ShouldAlert bool
	Retryable   bool

	// Socket and CloseCode are copied from the definition.
	// The WebSocket transport reads them. Classify does not open or close a connection.
	Socket    errcore.SocketPolicy
	CloseCode int

	Cause     error
	Metadata  map[string]any
	Stack     string
	Operation errcore.Operation
}

// Scope identifies where a log event came from.
// Method and Path are set for HTTP panics. Empty fields are omitted.
type Scope struct {
	Transport string
	Component string
	Method    string
	Path      string
}

// MetricKey is the low-cardinality set suitable for a counter label.
// User, request, and conversation identifiers are intentionally absent.
type MetricKey struct {
	Code      string
	Category  string
	Transport string
}

// MetricKey returns labels for a future counter. transport is supplied by the caller.
func (c Classification) MetricKey(transport string) MetricKey {
	return MetricKey{
		Code:      string(c.Code),
		Category:  string(c.Category),
		Transport: transport,
	}
}
