package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/transport"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	gorillaws "github.com/gorilla/websocket"
)

// Authenticator resolves the caller before the protocol upgrade.
// A non-nil error becomes an HTTP response and the socket is not opened.
type Authenticator func(*http.Request) (Identity, error)

// OnConnect runs after the connection is registered and before it reads frames.
// A non-nil error closes that socket. Realtime uses this to join the user's rooms.
type OnConnect func(context.Context, *Conn) error

// Authorizer decides whether a user may join or leave a room.
// room.join and room.leave are registered only when an authorizer is set.
type Authorizer func(context.Context, int64, Room) error

// Server accepts WebSocket upgrades.
type Server struct {
	hub        *Hub
	router     *Router
	auth       Authenticator
	onConnect  OnConnect
	authorizer Authorizer
	log        logger.Logger
	upgrader   gorillaws.Upgrader
}

// Option changes a server before it starts.
type Option func(*Server)

// WithOnConnect sets the hook invoked after a connection is registered.
func WithOnConnect(fn OnConnect) Option {
	return func(s *Server) {
		if s != nil {
			s.onConnect = fn
		}
	}
}

// WithAuthorizer registers room.join and room.leave, gated by fn.
func WithAuthorizer(fn Authorizer) Option {
	return func(s *Server) {
		if s != nil {
			s.authorizer = fn
		}
	}
}

// NewServer builds a server that requires auth, recovers handler panics, and answers ping.
func NewServer(hub *Hub, log logger.Logger, auth Authenticator, opts ...Option) (*Server, error) {
	if hub == nil {
		return nil, errors.New("websocket: nil hub")
	}
	if auth == nil {
		return nil, errors.New("websocket: nil authenticator")
	}
	if log == nil {
		log = logger.Nop()
	}
	s := &Server{
		hub:    hub,
		router: NewRouter(),
		auth:   auth,
		log:    log,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.router.Use(Recovery(s.log), Access(s.log), RequireUser)
	s.router.Handle("ping", ping)
	if s.authorizer != nil {
		s.router.Handle("room.join", s.joinRoom)
		s.router.Handle("room.leave", s.leaveRoom)
	}
	s.upgrader = gorillaws.Upgrader{
		HandshakeTimeout: hub.cfg.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			return allowOrigin(s.hub.cfg, r)
		},
	}
	return s, nil
}

// Router returns the method router. Handlers added later share the default middleware.
func (s *Server) Router() *Router {
	if s == nil {
		return nil
	}
	return s.router
}

// ServeHTTP authenticates the request and upgrades it.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.hub == nil || s.auth == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if !allowOrigin(s.hub.cfg, r) {
		writeHTTPError(s.log, w, r, errcore.New(requestContext(r), errs.AuthForbidden))
		return
	}
	identity, err := s.auth(r)
	if err != nil {
		writeHTTPError(s.log, w, r, err)
		return
	}
	if identity.UserID == 0 {
		writeHTTPError(s.log, w, r, errcore.New(requestContext(r), errs.AuthUnauthorized))
		return
	}
	if !validPlatform(identity.Platform) {
		writeHTTPError(s.log, w, r, errcore.New(requestContext(r), errs.ValidationInvalidInput))
		return
	}
	if !identity.ExpiresAt.IsZero() && !time.Now().Before(identity.ExpiresAt) {
		writeHTTPError(s.log, w, r, errcore.New(requestContext(r), errs.AuthTokenExpired))
		return
	}
	if err := s.hub.admit(identity.UserID); err != nil {
		writeHTTPError(s.log, w, r, err)
		return
	}
	raw, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithLevel(logger.Warn).Err(err).Msg("websocket upgrade failed")
		return
	}
	id, err := newID()
	if err != nil {
		s.failRaw(raw, errcore.Wrap(requestContext(r), errs.InternalUnexpectedError, err))
		return
	}
	conn := newConn(raw, s.hub, id, identity, requestContext(r), s.router)
	if err := s.hub.Add(conn); err != nil {
		s.failConn(conn, err)
		return
	}
	if s.onConnect != nil {
		if err := s.onConnect(conn.context(), conn); err != nil {
			s.failConn(conn, err)
			return
		}
	}
	conn.start()
}

func (s *Server) failRaw(raw *gorillaws.Conn, err error) {
	class := classify.Classify(err)
	classify.Log(s.log, class, context.Background(), classify.Scope{Transport: "websocket", Component: "websocket"})
	if raw != nil {
		_ = transport.Respond(raw, class)
		_ = raw.Close()
	}
}

func (s *Server) failConn(conn *Conn, err error) {
	class := classify.Classify(err)
	ctx := context.Background()
	if conn != nil {
		ctx = conn.context()
	}
	classify.Log(s.log, class, ctx, classify.Scope{Transport: "websocket", Component: "websocket"})
	if conn != nil && conn.raw != nil {
		_ = transport.Respond(conn.raw, class)
	}
	if conn != nil {
		conn.closeRaw()
		conn.exit()
	}
}

func (s *Server) joinRoom(c *Context) (any, error) {
	room, err := roomParam(c)
	if err != nil {
		return nil, err
	}
	if err := validRoom(room); err != nil {
		return nil, err
	}
	if s.authorizer != nil {
		if err := s.authorizer(c.Context, c.Conn.UserID(), room); err != nil {
			return nil, err
		}
	}
	if err := s.hub.Join(c.Conn.ID(), room); err != nil {
		return nil, err
	}
	return map[string]string{"room": string(room)}, nil
}

func (s *Server) leaveRoom(c *Context) (any, error) {
	room, err := roomParam(c)
	if err != nil {
		return nil, err
	}
	if err := validRoom(room); err != nil {
		return nil, err
	}
	if s.authorizer != nil {
		if err := s.authorizer(c.Context, c.Conn.UserID(), room); err != nil {
			return nil, err
		}
	}
	s.hub.Leave(c.Conn.ID(), room)
	return map[string]string{"room": string(room)}, nil
}

func roomParam(c *Context) (Room, error) {
	var body struct {
		Room string `json:"room"`
	}
	if err := c.Bind(&body); err != nil {
		return "", err
	}
	if body.Room == "" {
		return "", errcore.New(contextOf(c), errs.ValidationMissingField)
	}
	return Room(body.Room), nil
}

func ping(c *Context) (any, error) {
	return map[string]int64{"server_time": time.Now().Unix()}, nil
}

func allowOrigin(cfg Config, r *http.Request) bool {
	if r == nil {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range cfg.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func validPlatform(platform string) bool {
	if len(platform) > 32 {
		return false
	}
	for _, r := range platform {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func requestContext(r *http.Request) context.Context {
	if r == nil || r.Context() == nil {
		return context.Background()
	}
	return r.Context()
}

func writeHTTPError(log logger.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if log == nil {
		log = logger.Nop()
	}
	class := classify.Classify(err)
	classify.Log(log, class, requestContext(r), classify.Scope{Transport: "http", Component: "websocket"})
	if w == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	status := class.HTTPStatus
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(transport.HTTPBodyFrom(class))
}
