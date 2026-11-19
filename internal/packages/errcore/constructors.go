package errcore

import "context"

// New returns an application error for a registered code.
// Unknown codes still produce an error. The classifier hides them from clients.
// ctx may be nil. Its request ids are copied. The context itself is not stored.
func New(ctx context.Context, code Code) *Error {
	return build(ctx, code, nil, "")
}

// Wrap attaches cause to a registered code. The cause stays on the unwrap chain.
// ctx may be nil. Its request ids are copied. The context itself is not stored.
func Wrap(ctx context.Context, code Code, cause error) *Error {
	return build(ctx, code, cause, "")
}

// WrapWithMessage attaches cause and an internal detail.
// detail is included in Error for logs. The client message still comes from the registry.
// ctx may be nil. Its request ids are copied. The context itself is not stored.
func WrapWithMessage(ctx context.Context, code Code, cause error, detail string) *Error {
	return build(ctx, code, cause, detail)
}
