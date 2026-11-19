package websocket

import (
	"fmt"
	"time"
)

// Config controls timeouts, frame size, and connection caps.
// Zero connection caps mean unlimited. AllowedOrigins is exact string match.
// An empty Origin header is allowed. Any other Origin must be listed.
type Config struct {
	WriteWait        time.Duration
	PongWait         time.Duration
	PingPeriod       time.Duration
	MaxMessageSize   int64
	SendBuffer       int
	InboundBuffer    int
	MaxConnections   int
	MaxConnsPerUser  int
	AllowedOrigins   []string
	HandshakeTimeout time.Duration
}

// DefaultConfig returns limits that match a single-process chat server.
func DefaultConfig() Config {
	return Config{
		WriteWait:        10 * time.Second,
		PongWait:         60 * time.Second,
		PingPeriod:       54 * time.Second,
		MaxMessageSize:   64 * 1024,
		SendBuffer:       256,
		InboundBuffer:    32,
		HandshakeTimeout: 10 * time.Second,
	}
}

// Validate reports a config that would break the socket pumps.
func (c Config) Validate() error {
	if c.WriteWait <= 0 {
		return fmt.Errorf("websocket: WriteWait must be positive")
	}
	if c.PongWait <= 0 {
		return fmt.Errorf("websocket: PongWait must be positive")
	}
	if c.PingPeriod <= 0 {
		return fmt.Errorf("websocket: PingPeriod must be positive")
	}
	if c.PingPeriod >= c.PongWait {
		return fmt.Errorf("websocket: PingPeriod must be less than PongWait")
	}
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("websocket: MaxMessageSize must be positive")
	}
	if c.SendBuffer <= 0 {
		return fmt.Errorf("websocket: SendBuffer must be positive")
	}
	if c.InboundBuffer <= 0 {
		return fmt.Errorf("websocket: InboundBuffer must be positive")
	}
	if c.HandshakeTimeout <= 0 {
		return fmt.Errorf("websocket: HandshakeTimeout must be positive")
	}
	if c.MaxConnections < 0 {
		return fmt.Errorf("websocket: MaxConnections cannot be negative")
	}
	if c.MaxConnsPerUser < 0 {
		return fmt.Errorf("websocket: MaxConnsPerUser cannot be negative")
	}
	return nil
}
