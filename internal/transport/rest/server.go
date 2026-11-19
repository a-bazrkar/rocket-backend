package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/recovery"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/metrics"
	"github.com/a-bazrkar/rocket-backend/internal/transport/rest/middleware"
)

type Server struct {
	engine *gin.Engine
	log    logger.Logger
}

// New creates the engine and the base middleware. Routes are added with SetupRoutes.
func New(log logger.Logger) *Server {
	if log == nil {
		log = logger.Nop()
	}
	errs.MustLoad()

	engine := gin.New()
	engine.Use(recovery.Gin(log), middleware.RequestID(), middleware.Access(log), middleware.Errors(log))

	return &Server{log: log, engine: engine}
}

// MountMetrics registers the Prometheus collectors, the Gin and route middleware, and GET /metrics.
// Call it once, before SetupRoutes, and only when metrics are enabled.
func (s *Server) MountMetrics() {
	if s == nil {
		return
	}
	metrics.Mount(s.engine)
}

// Run serves until ctx is canceled.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
