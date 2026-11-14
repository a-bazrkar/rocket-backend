package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New writes to stderr. Pretty selects the console writer; otherwise the output is JSON.
func New(cfg Config) (zerolog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return zerolog.Logger{}, err
	}

	var out io.Writer = os.Stderr
	if cfg.Pretty {
		out = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}

	return zerolog.New(out).Level(level).With().Timestamp().Logger(), nil
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
