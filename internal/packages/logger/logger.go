package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Level is a log severity. Callers pass these to WithLevel.
type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

// Logger writes structured events. The current implementation is Zerolog.
// Callers depend on this interface, not on Zerolog.
type Logger interface {
	WithLevel(level Level) Event
}

// Event is one log line. Msg finishes it.
type Event interface {
	Str(key, value string) Event
	Int64(key string, value int64) Event
	Err(err error) Event
	Interface(key string, value any) Event
	Bool(key string, value bool) Event
	Msg(msg string)
}

type zeroLogger struct {
	z zerolog.Logger
}

type zeroEvent struct {
	e *zerolog.Event
}

type nopEvent struct{}

// New writes to stderr. Pretty selects the console writer; otherwise the output is JSON.
func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	var out io.Writer = os.Stderr
	if cfg.Pretty {
		out = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}

	return zeroLogger{z: zerolog.New(out).Level(level).With().Timestamp().Caller().Logger()}, nil
}

// Nop discards every event.
func Nop() Logger {
	return zeroLogger{z: zerolog.Nop()}
}

// Writer writes JSON at debug level and above. Tests use it to read events.
func Writer(w io.Writer) Logger {
	if w == nil {
		return Nop()
	}
	return zeroLogger{z: zerolog.New(w).Level(zerolog.DebugLevel)}
}

func (l zeroLogger) WithLevel(level Level) Event {
	zl, ok := zerologLevel(level)
	if !ok || l.z.GetLevel() > zl {
		return nopEvent{}
	}
	return zeroEvent{e: l.z.WithLevel(zl)}
}

func (e zeroEvent) Str(key, value string) Event {
	if e.e == nil {
		return nopEvent{}
	}
	e.e = e.e.Str(key, value)
	return e
}

func (e zeroEvent) Int64(key string, value int64) Event {
	if e.e == nil {
		return nopEvent{}
	}
	e.e = e.e.Int64(key, value)
	return e
}

func (e zeroEvent) Err(err error) Event {
	if e.e == nil {
		return nopEvent{}
	}
	e.e = e.e.Err(err)
	return e
}

func (e zeroEvent) Interface(key string, value any) Event {
	if e.e == nil {
		return nopEvent{}
	}
	e.e = e.e.Interface(key, value)
	return e
}

func (e zeroEvent) Bool(key string, value bool) Event {
	if e.e == nil {
		return nopEvent{}
	}
	e.e = e.e.Bool(key, value)
	return e
}

func (e zeroEvent) Msg(msg string) {
	if e.e == nil {
		return
	}
	e.e.Msg(msg)
}

func (nopEvent) Str(string, string) Event    { return nopEvent{} }
func (nopEvent) Int64(string, int64) Event   { return nopEvent{} }
func (nopEvent) Err(error) Event             { return nopEvent{} }
func (nopEvent) Interface(string, any) Event { return nopEvent{} }
func (nopEvent) Bool(string, bool) Event     { return nopEvent{} }
func (nopEvent) Msg(string)                  {}

func parseLevel(level string) (zerolog.Level, error) {
	zl, ok := zerologLevel(Level(strings.ToLower(level)))
	if !ok {
		return 0, fmt.Errorf("logger: unknown level %q", level)
	}
	return zl, nil
}

func zerologLevel(level Level) (zerolog.Level, bool) {
	switch level {
	case Debug:
		return zerolog.DebugLevel, true
	case Info:
		return zerolog.InfoLevel, true
	case Warn:
		return zerolog.WarnLevel, true
	case Error:
		return zerolog.ErrorLevel, true
	default:
		return 0, false
	}
}
