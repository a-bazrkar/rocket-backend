// Package websocket is the connection, routing, and delivery core.
// A handler returns a result or an error. Errors use the same public event as the rest of the API.
// Who may join a room, and which events are published after a database write, belongs in internal/realtime.
// This package does not import Gin. Server implements http.Handler.
package websocket
