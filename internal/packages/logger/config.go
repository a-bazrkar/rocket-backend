package logger

// Config controls logger output. Callers fill it; this package does not read the environment.
type Config struct {
	// Level is debug, info, warn, or error.
	Level string
	// Pretty writes a console line instead of JSON.
	Pretty bool
}
