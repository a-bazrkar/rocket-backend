package websocket

import (
	"encoding/json"
	"unicode"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
)

const (
	maxNameLen    = 128
	eventClosing  = "connection.closing"
	closeShutdown = "server shutting down"
)

// Event is a server push. Type is the name the client switches on, such as message.created.
type Event struct {
	Type string
	Data any
}

type inbound struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type resultFrame struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Data any    `json:"data,omitempty"`
}

type errorFrame struct {
	Type  string                `json:"type"`
	ID    string                `json:"id,omitempty"`
	Error transport.PublicError `json:"error"`
}

func decodeInbound(data []byte) (inbound, error) {
	if len(data) == 0 {
		return inbound{}, errcore.New(nil, errs.WebsocketInvalidMessage)
	}
	var msg inbound
	if err := json.Unmarshal(data, &msg); err != nil {
		return inbound{}, errcore.Wrap(nil, errs.WebsocketInvalidMessage, err)
	}
	if !validName(msg.Type) || !validID(msg.ID) {
		return inbound{}, errcore.New(nil, errs.WebsocketInvalidMessage)
	}
	return msg, nil
}

func validName(name string) bool {
	if name == "" || len(name) > maxNameLen {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validID(id string) bool {
	if len(id) > maxNameLen {
		return false
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func marshalEvent(ev Event) ([]byte, error) {
	if !validName(ev.Type) {
		return nil, errcore.New(nil, errs.WebsocketInvalidMessage)
	}
	return json.Marshal(resultFrame{Type: ev.Type, Data: ev.Data})
}

func marshalResult(method, id string, data any) ([]byte, error) {
	return json.Marshal(resultFrame{Type: method, ID: id, Data: data})
}

func encodeError(id string, class classify.Classification) ([]byte, error) {
	event := transport.EventFrom(class)
	return json.Marshal(errorFrame{Type: event.Type, ID: id, Error: event.Error})
}
