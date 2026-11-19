package errcore

import "context"

type ctxKey int

const (
	keyRequestID ctxKey = iota
	keyTraceID
	keyUserID
	keyConversationID
	keyConnectionID
)

// WithRequestID stores a request identifier. An empty id leaves ctx unchanged.
func WithRequestID(ctx context.Context, id string) context.Context {
	return withString(ctx, keyRequestID, id)
}

// RequestID returns the request identifier when one was stored.
func RequestID(ctx context.Context) (string, bool) {
	return stringValue(ctx, keyRequestID)
}

// WithTraceID stores a trace identifier. An empty id leaves ctx unchanged.
func WithTraceID(ctx context.Context, id string) context.Context {
	return withString(ctx, keyTraceID, id)
}

// TraceID returns the trace identifier when one was stored.
func TraceID(ctx context.Context) (string, bool) {
	return stringValue(ctx, keyTraceID)
}

// WithUserID stores the authenticated user id.
func WithUserID(ctx context.Context, id int64) context.Context {
	return withInt(ctx, keyUserID, id)
}

// UserID returns the user id when one was stored.
func UserID(ctx context.Context) (int64, bool) {
	return intValue(ctx, keyUserID)
}

// WithConversationID stores the conversation id for this operation.
func WithConversationID(ctx context.Context, id int64) context.Context {
	return withInt(ctx, keyConversationID, id)
}

// ConversationID returns the conversation id when one was stored.
func ConversationID(ctx context.Context) (int64, bool) {
	return intValue(ctx, keyConversationID)
}

// WithConnectionID stores a WebSocket connection id. An empty id leaves ctx unchanged.
func WithConnectionID(ctx context.Context, id string) context.Context {
	return withString(ctx, keyConnectionID, id)
}

// ConnectionID returns the connection id when one was stored.
func ConnectionID(ctx context.Context) (string, bool) {
	return stringValue(ctx, keyConnectionID)
}

func withString(ctx context.Context, k ctxKey, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, k, id)
}

func stringValue(ctx context.Context, k ctxKey) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(k).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func withInt(ctx context.Context, k ctxKey, id int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, k, id)
}

func intValue(ctx context.Context, k ctxKey) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	v, ok := ctx.Value(k).(int64)
	return v, ok
}
