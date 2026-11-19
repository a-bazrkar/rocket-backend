package websocket

import (
	"sync"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
)

// Handler handles one inbound frame.
// A nil error and an empty message id produce no frame.
type Handler func(*Context) (any, error)

// Middleware wraps a handler. The first middleware passed to Use runs first.
type Middleware func(Handler) Handler

// Router maps an inbound type to a handler chain.
// Registration is safe with dispatch. Chains are rebuilt under the write lock.
type Router struct {
	mu       sync.RWMutex
	global   []Middleware
	handlers map[string]*handlerEntry
}

type handlerEntry struct {
	handler Handler
	local   []Middleware
	chain   Handler
}

// NewRouter returns an empty router.
func NewRouter() *Router {
	return &Router{handlers: map[string]*handlerEntry{}}
}

// Use appends middleware for every handler and rebuilds existing chains.
func (r *Router) Use(middlewares ...Middleware) *Router {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.global = append(r.global, middlewares...)
	for _, entry := range r.handlers {
		entry.chain = compile(r.global, entry.local, entry.handler)
	}
	return r
}

// Handle registers method. A later registration replaces an earlier one.
// Method middleware runs after the router's middleware.
func (r *Router) Handle(method string, handler Handler, middlewares ...Middleware) {
	if r == nil || method == "" || handler == nil {
		return
	}
	local := append([]Middleware(nil), middlewares...)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers == nil {
		r.handlers = map[string]*handlerEntry{}
	}
	r.handlers[method] = &handlerEntry{
		handler: handler,
		local:   local,
		chain:   compile(r.global, local, handler),
	}
}

// Group starts a chain that does not share later middleware with its parent.
func (r *Router) Group(middlewares ...Middleware) *Group {
	return &Group{router: r, middlewares: append([]Middleware(nil), middlewares...)}
}

// Dispatch runs the chain for the inbound type.
// An unregistered type returns websocket.method_not_found.
// Handler latency is labeled with the registered type, or unknown.
func (r *Router) Dispatch(c *Context) (any, error) {
	if r == nil || c == nil {
		return nil, errcore.New(nil, errs.WebsocketInvalidMessage)
	}
	start := time.Now()
	handler, label := r.lookup(c.Method())
	defer metrics.ObserveWSHandler(label, time.Since(start))
	return handler(c)
}

func (r *Router) lookup(method string) (Handler, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.handlers[method]
	if entry == nil || entry.chain == nil {
		return unknownMethod, "unknown"
	}
	return entry.chain, method
}

func unknownMethod(c *Context) (any, error) {
	return nil, errcore.New(contextOf(c), errs.WebsocketMethodNotFound)
}

// Chain composes middlewares so the first one is the outermost.
func Chain(middlewares ...Middleware) Middleware {
	return func(final Handler) Handler {
		if final == nil {
			final = unknownMethod
		}
		for i := len(middlewares) - 1; i >= 0; i-- {
			if middlewares[i] == nil {
				continue
			}
			final = middlewares[i](final)
		}
		return final
	}
}

func compile(global, local []Middleware, handler Handler) Handler {
	all := make([]Middleware, 0, len(global)+len(local))
	all = append(all, global...)
	all = append(all, local...)
	return Chain(all...)(handler)
}

// Group is a set of handlers that share extra middleware.
type Group struct {
	router      *Router
	middlewares []Middleware
}

// Use returns a new group. The receiver's middleware list is left as it was.
func (g *Group) Use(middlewares ...Middleware) *Group {
	if g == nil {
		return nil
	}
	next := make([]Middleware, 0, len(g.middlewares)+len(middlewares))
	next = append(next, g.middlewares...)
	next = append(next, middlewares...)
	return &Group{router: g.router, middlewares: next}
}

// Handle registers method on the group router.
func (g *Group) Handle(method string, handler Handler, middlewares ...Middleware) {
	if g == nil || g.router == nil {
		return
	}
	all := make([]Middleware, 0, len(g.middlewares)+len(middlewares))
	all = append(all, g.middlewares...)
	all = append(all, middlewares...)
	g.router.Handle(method, handler, all...)
}
