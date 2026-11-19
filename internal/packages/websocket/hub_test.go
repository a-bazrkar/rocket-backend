package websocket

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
)

func TestHubRoomsUsersAndDrop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SendBuffer = 1
	hub, err := NewHub(cfg, logger.Nop())
	if err != nil {
		t.Fatal(err)
	}
	first := newConn(nil, hub, "a", Identity{UserID: 7, Platform: "web"}, context.Background(), nil)
	second := newConn(nil, hub, "b", Identity{UserID: 7, Platform: "ios"}, context.Background(), nil)
	other := newConn(nil, hub, "c", Identity{UserID: 8}, context.Background(), nil)
	for _, conn := range []*Conn{first, second, other} {
		if err := hub.Add(conn); err != nil {
			t.Fatal(err)
		}
	}
	if err := hub.Join(first.id, RoomOf("conversation", 3)); err != nil {
		t.Fatal(err)
	}
	if err := hub.Join(other.id, RoomOf("conversation", 9)); err != nil {
		t.Fatal(err)
	}

	hub.PublishRoom(RoomOf("conversation", 3), Event{Type: "message.created", Data: map[string]int{"n": 1}})
	if got := takePayload(t, first); !strings.Contains(string(got), "message.created") {
		t.Fatalf("member payload = %s", got)
	}
	if got := takePayload(t, other); got != nil {
		t.Fatalf("other room received %s", got)
	}
	if got := takePayload(t, second); got != nil {
		t.Fatalf("unjoined connection received %s", got)
	}

	hub.PublishUser(7, Event{Type: "typing.start"})
	if takePayload(t, first) == nil || takePayload(t, second) == nil {
		t.Fatal("user fanout missed a connection")
	}
	if got := takePayload(t, other); got != nil {
		t.Fatalf("other user received %s", got)
	}

	if !first.trySend(outbound{payload: []byte("full")}) {
		t.Fatal("buffer did not accept one frame")
	}
	before := metrics.DroppedSends()
	done := make(chan struct{})
	go func() {
		hub.PublishRoom(RoomOf("conversation", 3), Event{Type: "message.created"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on a full send buffer")
	}
	if metrics.DroppedSends() <= before {
		t.Fatal("full buffer was not counted as a drop")
	}
}

func TestHubCloseDrains(t *testing.T) {
	hub, err := NewHub(DefaultConfig(), logger.Nop())
	if err != nil {
		t.Fatal(err)
	}
	conn := newConn(nil, hub, "a", Identity{UserID: 1}, context.Background(), nil)
	if err := hub.Add(conn); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := hub.Close(ctx); err == nil {
		t.Fatal("close returned before the idle connection was forced")
	}
	if hub.count() != 0 {
		t.Fatalf("connections = %d", hub.count())
	}
	if err := hub.Add(newConn(nil, hub, "b", Identity{UserID: 2}, context.Background(), nil)); err == nil {
		t.Fatal("closed hub accepted a connection")
	}
}

func TestConfigRejectsPingPeriod(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PingPeriod = cfg.PongWait
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid config")
	}
	if _, err := NewHub(cfg, logger.Nop()); err == nil {
		t.Fatal("expected hub to reject the config")
	}
}

func takePayload(t *testing.T, conn *Conn) []byte {
	t.Helper()
	select {
	case frame := <-conn.send:
		return frame.payload
	default:
		return nil
	}
}
