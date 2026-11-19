package classify

import (
	"context"
	"reflect"
	"strings"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/rs/zerolog"
)

func zerologLevel(level errcore.LogLevel) zerolog.Level {
	switch level {
	case errcore.LogDebug:
		return zerolog.DebugLevel
	case errcore.LogInfo:
		return zerolog.InfoLevel
	case errcore.LogWarn:
		return zerolog.WarnLevel
	case errcore.LogError:
		return zerolog.ErrorLevel
	default:
		return zerolog.ErrorLevel
	}
}

// Log writes one structured event for a classification.
// Context fields are read from ctx. scope identifies the caller.
// A classification that should not be logged, or a disabled level, writes nothing.
// Classify does not call Log.
func Log(log zerolog.Logger, class Classification, ctx context.Context, scope Scope) {
	if !class.ShouldLog {
		return
	}
	ev := log.WithLevel(zerologLevel(class.LogLevel))
	if ev == nil {
		return
	}
	ev.Str("error_code", string(class.Code))
	ev.Str("error_category", string(class.Category))
	if class.Cause != nil {
		ev.Err(class.Cause)
	}
	if id := firstString(class.Operation.RequestID, requestID(ctx)); id != "" {
		ev.Str("request_id", id)
	}
	if id := firstString(class.Operation.TraceID, traceID(ctx)); id != "" {
		ev.Str("trace_id", id)
	}
	if class.Operation.HasUserID {
		ev.Int64("user_id", class.Operation.UserID)
	} else if id, ok := userID(ctx); ok {
		ev.Int64("user_id", id)
	}
	if class.Operation.HasConversationID {
		ev.Int64("conversation_id", class.Operation.ConversationID)
	} else if id, ok := conversationID(ctx); ok {
		ev.Int64("conversation_id", id)
	}
	if id := firstString(class.Operation.ConnectionID, connectionID(ctx)); id != "" {
		ev.Str("connection_id", id)
	}
	if scope.Component != "" {
		ev.Str("component", scope.Component)
	}
	if scope.Transport != "" {
		ev.Str("transport", scope.Transport)
	}
	if scope.Method != "" {
		ev.Str("method", scope.Method)
	}
	if scope.Path != "" {
		ev.Str("path", scope.Path)
	}
	if meta := redact(class.Metadata); len(meta) > 0 {
		ev.Interface("metadata", meta)
	}
	if class.Stack != "" {
		ev.Str("stack", class.Stack)
	}
	ev.Bool("alert", class.ShouldAlert)
	ev.Bool("retryable", class.Retryable)
	ev.Msg("error")
}

func redact(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if isSensitive(k) {
			out[k] = "[redacted]"
			continue
		}
		out[k] = redactValue(v)
	}
	return out
}

func redactValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return redact(val)
	case []any:
		out := make([]any, len(val))
		for i := range val {
			out[i] = redactValue(val[i])
		}
		return out
	default:
		return redactComposite(v)
	}
}

func redactComposite(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return v
		}
		return redactValue(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return v
		}
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = redactValue(rv.Index(i).Interface())
		}
		return out
	case reflect.Map:
		if rv.IsNil() || rv.Type().Key().Kind() != reflect.String {
			return v
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			item := iter.Value().Interface()
			if isSensitive(k) {
				out[k] = "[redacted]"
				continue
			}
			out[k] = redactValue(item)
		}
		return out
	default:
		return v
	}
}

func isSensitive(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	k = strings.ReplaceAll(k, "-", "_")
	switch k {
	case "password", "password_hash", "passwd", "secret", "session",
		"token", "access_token", "refresh_token", "id_token",
		"authorization", "cookie", "set_cookie",
		"content", "message", "body":
		return true
	}
	return strings.Contains(k, "password") || strings.Contains(k, "secret") || strings.Contains(k, "token")
}

func firstString(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func requestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := errcore.RequestID(ctx)
	return id
}

func traceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := errcore.TraceID(ctx)
	return id
}

func connectionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := errcore.ConnectionID(ctx)
	return id
}

func userID(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	return errcore.UserID(ctx)
}

func conversationID(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	return errcore.ConversationID(ctx)
}
