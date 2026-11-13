package logger

import (
	"bytes"
	"testing"
	"time"
)

func TestNewRejectsUnknownLevel(t *testing.T) {
	_, err := New(Config{Level: "trace"})
	if err == nil {
		t.Fatal("expected unknown level to fail")
	}
}

func TestDisabledLevelDoesNotPanic(t *testing.T) {
	log, err := New(Config{Level: "error"})
	if err != nil {
		t.Fatal(err)
	}
	log.Debug().
		Str("k", "v").
		Int("n", 1).
		Int64("id", 2).
		Bool("ok", true).
		Dur("d", time.Second).
		Err(nil).
		Any("a", []int{1}).
		Msg("hidden")
}

func TestChainWritesMessage(t *testing.T) {
	var buf bytes.Buffer
	base, err := New(Config{Level: "info"})
	if err != nil {
		t.Fatal(err)
	}
	log := Logger{base: base.base.Output(&buf)}

	log.With().Str("request_id", "r1").Logger().Info().Str("user_id", "7").Msg("ready")

	got := buf.String()
	if got == "" {
		t.Fatal("expected a log line")
	}
	for _, part := range []string{`"request_id":"r1"`, `"user_id":"7"`, `"message":"ready"`} {
		if !bytes.Contains([]byte(got), []byte(part)) {
			t.Fatalf("log %q missing %s", got, part)
		}
	}
}
