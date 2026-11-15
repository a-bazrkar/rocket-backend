package token

import "time"

// Config controls access-token signing. Callers fill it; this package does not read the environment.
type Config struct {
	Secret    string
	Issuer    string
	AccessTTL time.Duration
}
