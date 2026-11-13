package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger is a zerolog logger passed by value. It has no package-level default.
type Logger struct {
	base zerolog.Logger
}

// New writes to stderr. Pretty selects the console writer; otherwise the output is JSON.
func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return Logger{}, err
	}

	var out io.Writer = os.Stderr
	if cfg.Pretty {
		out = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}

	base := zerolog.New(out).Level(level).With().Timestamp().Logger()
	return Logger{base: base}, nil
}

func parseLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return 0, fmt.Errorf("logger: unknown level %q", level)
	}
}

func (l Logger) Debug() *Event { return &Event{e: l.base.Debug()} }
func (l Logger) Info() *Event  { return &Event{e: l.base.Info()} }
func (l Logger) Warn() *Event  { return &Event{e: l.base.Warn()} }
func (l Logger) Error() *Event { return &Event{e: l.base.Error()} }

// With starts a field chain. Call Logger to finish it.
func (l Logger) With() *Context {
	return &Context{c: l.base.With()}
}

// Event is one log line. Methods no-op when the level is disabled.
type Event struct {
	e *zerolog.Event
}

func (e *Event) Str(key, val string) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Str(key, val)
	return e
}

func (e *Event) Int(key string, val int) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Int(key, val)
	return e
}

func (e *Event) Int64(key string, val int64) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Int64(key, val)
	return e
}

func (e *Event) Bool(key string, val bool) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Bool(key, val)
	return e
}

func (e *Event) Dur(key string, val time.Duration) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Dur(key, val)
	return e
}

func (e *Event) Err(err error) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Err(err)
	return e
}

func (e *Event) Any(key string, val any) *Event {
	if e == nil || e.e == nil {
		return e
	}
	e.e.Any(key, val)
	return e
}

func (e *Event) Msg(msg string) {
	if e == nil || e.e == nil {
		return
	}
	e.e.Msg(msg)
}

// Context collects fields for a child logger.
type Context struct {
	c zerolog.Context
}

func (c *Context) Str(key, val string) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Str(key, val)
	return c
}

func (c *Context) Int(key string, val int) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Int(key, val)
	return c
}

func (c *Context) Int64(key string, val int64) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Int64(key, val)
	return c
}

func (c *Context) Bool(key string, val bool) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Bool(key, val)
	return c
}

func (c *Context) Dur(key string, val time.Duration) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Dur(key, val)
	return c
}

func (c *Context) Err(err error) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Err(err)
	return c
}

func (c *Context) Any(key string, val any) *Context {
	if c == nil {
		return c
	}
	c.c = c.c.Any(key, val)
	return c
}

// Logger finishes the field chain.
func (c *Context) Logger() Logger {
	if c == nil {
		return Logger{}
	}
	return Logger{base: c.c.Logger()}
}
