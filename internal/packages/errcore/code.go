package errcore

// Code is a stable, lowercase, dot-separated error identifier.
// Clients may branch on a code. They must not branch on the public message.
type Code string

const (
	// CodeInternalUnexpectedError is the public code for an unrecognized failure.
	CodeInternalUnexpectedError Code = "internal.unexpected_error"
	// CodeInternalPanicRecovered is the public code for a recovered panic.
	CodeInternalPanicRecovered Code = "internal.panic_recovered"
)
