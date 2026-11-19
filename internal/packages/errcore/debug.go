package errcore

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
)

var debugOn atomic.Bool

// SetDebug records whether client responses may include a caller stack.
// Call it once at startup, before the server accepts traffic.
func SetDebug(on bool) {
	debugOn.Store(on)
}

// Debug reports whether client responses may include a caller stack.
func Debug() bool {
	return debugOn.Load()
}

func captureOperation(ctx context.Context) Operation {
	if ctx == nil {
		return Operation{}
	}
	var op Operation
	if id, ok := RequestID(ctx); ok {
		op.RequestID = id
	}
	if id, ok := TraceID(ctx); ok {
		op.TraceID = id
	}
	if id, ok := UserID(ctx); ok {
		op.UserID = id
		op.HasUserID = true
	}
	if id, ok := ConversationID(ctx); ok {
		op.ConversationID = id
		op.HasConversationID = true
	}
	if id, ok := ConnectionID(ctx); ok {
		op.ConnectionID = id
	}
	return op
}

func captureFrames() []string {
	var pcs [64]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	out := make([]string, 0, 16)
	for {
		frame, more := frames.Next()
		if !skipFrame(frame.Function) {
			out = append(out, formatFrame(frame))
			if len(out) == 32 {
				break
			}
		}
		if !more {
			break
		}
	}
	return out
}

func skipFrame(fn string) bool {
	return fn == "" || strings.HasPrefix(fn, "runtime.") || strings.Contains(fn, "/errcore.") || strings.Contains(fn, "/errcore/")
}

func formatFrame(frame runtime.Frame) string {
	file := frame.File
	if i := strings.Index(file, "rocket-backend/"); i >= 0 {
		file = file[i+len("rocket-backend/"):]
	}
	return fmt.Sprintf("%s:%d %s", file, frame.Line, shortFunc(frame.Function))
}

func shortFunc(fn string) string {
	if i := strings.LastIndex(fn, "/"); i >= 0 {
		fn = fn[i+1:]
	}
	if i := strings.Index(fn, "."); i >= 0 {
		fn = fn[i+1:]
	}
	fn = strings.TrimPrefix(fn, "(*")
	fn = strings.ReplaceAll(fn, ").", ".")
	return fn
}
