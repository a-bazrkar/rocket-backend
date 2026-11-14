package storage

// Config selects the file directory and the upload limit.
// Callers fill it; this package does not read the environment.
type Config struct {
	Root     string
	MaxBytes int64
}
