package websocket

import (
	"context"
	"encoding/json"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
)

// Context is one inbound frame plus the connection it arrived on.
// It implements context.Context through the embedded value.
type Context struct {
	context.Context
	Conn *Conn
	in   inbound
}

// Method returns the inbound type, such as ping or room.join.
func (c *Context) Method() string {
	if c == nil {
		return ""
	}
	return c.in.Type
}

// MessageID returns the client correlation id.
// An empty id means the frame is a notification and a successful handler writes nothing.
func (c *Context) MessageID() string {
	if c == nil {
		return ""
	}
	return c.in.ID
}

// Data returns the raw JSON data field.
func (c *Context) Data() json.RawMessage {
	if c == nil {
		return nil
	}
	return c.in.Data
}

// Bind decodes the data field into dst.
// Missing data leaves dst unchanged.
func (c *Context) Bind(dst any) error {
	if c == nil || dst == nil {
		return errcore.New(contextOf(c), errs.ValidationInvalidInput)
	}
	if len(c.in.Data) == 0 || string(c.in.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(c.in.Data, dst); err != nil {
		return errcore.Wrap(c.Context, errs.WebsocketInvalidMessage, err)
	}
	return nil
}

func contextOf(c *Context) context.Context {
	if c == nil || c.Context == nil {
		return context.Background()
	}
	return c.Context
}
