package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	ginLogger "github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Server struct {
	engine *gin.Engine
	log    zerolog.Logger
}

// New creates the engine and the base middleware. Routes are added with SetupRoutes.
func New(log zerolog.Logger) *Server {

	// create gin engine
	engine := gin.New()

	// create middlewares
	logMiddleware := ginLogger.SetLogger(ginLogger.WithLogger(func(*gin.Context, zerolog.Logger) zerolog.Logger { return log }))
	recoveryMiddleware := gin.Recovery()

	// use middlewares
	engine.Use(logMiddleware, recoveryMiddleware)

	// return server
	return &Server{log: log, engine: engine}
}

// Run serves until ctx is canceled.
func (s *Server) Run(ctx context.Context, addr string) error {

	// create HTTP server
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// create error channel
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {

	// wait for context to be done
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)

	// wait for error
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
