package websocket

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
)

const maxRoomLen = 128

// Room is an address chosen by the caller. This package does not interpret it.
type Room string

// RoomOf builds kind:id. An empty kind returns an empty room, which Join rejects.
func RoomOf(kind string, id int64) Room {
	if kind == "" {
		return ""
	}
	return Room(kind + ":" + strconv.FormatInt(id, 10))
}

// Hub tracks open connections, users, and rooms.
// Publish methods marshal once and do not wait for a slow socket.
type Hub struct {
	cfg    Config
	log    logger.Logger
	mu     sync.RWMutex
	conns  map[string]*Conn
	users  map[int64]map[string]*Conn
	rooms  map[Room]map[string]*Conn
	closed bool
}

// NewHub validates cfg and returns an empty hub.
func NewHub(cfg Config, log logger.Logger) (*Hub, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = logger.Nop()
	}
	cfg.AllowedOrigins = append([]string(nil), cfg.AllowedOrigins...)
	return &Hub{
		cfg:   cfg,
		log:   log,
		conns: map[string]*Conn{},
		users: map[int64]map[string]*Conn{},
		rooms: map[Room]map[string]*Conn{},
	}, nil
}

// Add inserts c. It fails when the hub is closed or a connection cap is reached.
func (h *Hub) Add(c *Conn) error {
	if h == nil || c == nil || c.id == "" {
		return errcore.New(nil, errs.WebsocketInvalidMessage)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.rejectLocked(c.userID); err != nil {
		return err
	}
	if _, exists := h.conns[c.id]; exists {
		return errcore.New(c.context(), errs.ExternalServiceUnavailable)
	}
	h.conns[c.id] = c
	if c.userID != 0 {
		set := h.users[c.userID]
		if set == nil {
			set = map[string]*Conn{}
			h.users[c.userID] = set
		}
		set[c.id] = c
	}
	metrics.WSConnectionOpened()
	h.log.WithLevel(logger.Info).
		Str("connection_id", c.id).
		Int64("user_id", c.userID).
		Str("platform", c.platform).
		Msg("websocket connected")
	return nil
}

func (h *Hub) admit(userID int64) error {
	if h == nil {
		return errcore.New(nil, errs.ExternalServiceUnavailable)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rejectLocked(userID)
}

func (h *Hub) rejectLocked(userID int64) error {
	if h.closed {
		return errcore.New(nil, errs.ExternalServiceUnavailable)
	}
	if h.cfg.MaxConnections > 0 && len(h.conns) >= h.cfg.MaxConnections {
		return errcore.New(nil, errs.WebsocketConnectionLimit)
	}
	if userID != 0 && h.cfg.MaxConnsPerUser > 0 && len(h.users[userID]) >= h.cfg.MaxConnsPerUser {
		return errcore.New(nil, errs.WebsocketConnectionLimit)
	}
	return nil
}

func (h *Hub) remove(c *Conn) {
	if h == nil || c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.conns[c.id]
	if !ok || current != c {
		return
	}
	delete(h.conns, c.id)
	if set := h.users[c.userID]; set != nil {
		delete(set, c.id)
		if len(set) == 0 {
			delete(h.users, c.userID)
		}
	}
	for room := range c.rooms {
		members := h.rooms[room]
		delete(members, c.id)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
	c.rooms = nil
	metrics.WSConnectionClosed()
	h.log.WithLevel(logger.Info).
		Str("connection_id", c.id).
		Int64("user_id", c.userID).
		Msg("websocket disconnected")
}

// Join adds the connection to room.
func (h *Hub) Join(connID string, room Room) error {
	if err := validRoom(room); err != nil {
		return err
	}
	if h == nil {
		return errcore.New(nil, errs.ExternalServiceUnavailable)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errcore.New(nil, errs.ExternalServiceUnavailable)
	}
	c := h.conns[connID]
	if c == nil {
		return errcore.New(nil, errs.ExternalServiceUnavailable)
	}
	members := h.rooms[room]
	if members == nil {
		members = map[string]*Conn{}
		h.rooms[room] = members
	}
	members[c.id] = c
	if c.rooms == nil {
		c.rooms = map[Room]struct{}{}
	}
	c.rooms[room] = struct{}{}
	return nil
}

// Leave removes the connection from room. A missing membership is ignored.
func (h *Hub) Leave(connID string, room Room) {
	if h == nil || room == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if c := h.conns[connID]; c != nil {
		delete(c.rooms, room)
	}
	members := h.rooms[room]
	if members == nil {
		return
	}
	delete(members, connID)
	if len(members) == 0 {
		delete(h.rooms, room)
	}
}

// PublishRoom sends ev to the current members of room.
func (h *Hub) PublishRoom(room Room, ev Event) {
	if h == nil {
		return
	}
	h.publish(h.roomTargets(room), ev)
}

// PublishUser sends ev to every open socket for userID.
func (h *Hub) PublishUser(userID int64, ev Event) {
	if h == nil {
		return
	}
	h.PublishUsers([]int64{userID}, ev)
}

// PublishUsers sends ev once to each open socket owned by the given users.
func (h *Hub) PublishUsers(userIDs []int64, ev Event) {
	if h == nil {
		return
	}
	h.mu.RLock()
	seen := map[string]struct{}{}
	targets := make([]*Conn, 0)
	for _, userID := range userIDs {
		for id, c := range h.users[userID] {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()
	h.publish(targets, ev)
}

func (h *Hub) roomTargets(room Room) []*Conn {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	members := h.rooms[room]
	targets := make([]*Conn, 0, len(members))
	for _, c := range members {
		targets = append(targets, c)
	}
	return targets
}

func (h *Hub) publish(targets []*Conn, ev Event) {
	if h == nil || len(targets) == 0 {
		return
	}
	payload, err := marshalEvent(ev)
	if err != nil {
		h.log.WithLevel(logger.Error).Err(err).Str("type", ev.Type).Msg("websocket publish failed")
		return
	}
	for _, c := range targets {
		if c.trySend(outbound{payload: payload}) {
			continue
		}
		if !c.isClosed() {
			metrics.WSSendDropped()
		}
	}
}

// Kick closes one connection. The close is immediate when the send buffer is full.
func (h *Hub) Kick(connID string) bool {
	c := h.conn(connID)
	if c == nil {
		return false
	}
	if !c.trySend(outbound{normalClose: true}) {
		c.force()
	}
	return true
}

// KickUser closes every connection for userID and returns how many it found.
func (h *Hub) KickUser(userID int64) int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	set := h.users[userID]
	conns := make([]*Conn, 0, len(set))
	for _, c := range set {
		conns = append(conns, c)
	}
	h.mu.RUnlock()
	for _, c := range conns {
		if !c.trySend(outbound{normalClose: true}) {
			c.force()
		}
	}
	return len(conns)
}

func (h *Hub) conn(id string) *Conn {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[id]
}

// Close stops new connections, tells open sockets the server is shutting down, and waits.
// A canceled ctx forces the remaining sockets closed and returns ctx.Err().
func (h *Hub) Close(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	h.closed = true
	conns := make([]*Conn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.shutdown()
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if h.count() == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			h.forceAll()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func (h *Hub) forceAll() {
	h.mu.RLock()
	conns := make([]*Conn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.RUnlock()
	for _, c := range conns {
		c.force()
	}
}

func validRoom(room Room) error {
	if room == "" || len(room) > maxRoomLen {
		return errcore.New(nil, errs.ValidationInvalidInput)
	}
	for _, r := range room {
		if r < 0x20 || r == 0x7f {
			return errcore.New(nil, errs.ValidationInvalidInput)
		}
	}
	return nil
}
