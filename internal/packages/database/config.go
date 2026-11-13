package database

import "time"

// Config selects the SQLite file and pool. Callers fill it; this package does not read the environment.
type Config struct {
	Path         string
	MaxOpenConns int
	MaxIdleConns int
	BusyTimeout  time.Duration
}
