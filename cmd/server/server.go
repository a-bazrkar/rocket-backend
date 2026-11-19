package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/a-bazrkar/rocket-backend/internal/config"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/transport/rest"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Run the HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {

			// load configs
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			}

			// create logger
			log, err := logger.New(cfg.Logger)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			}

			// handle os signals
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// create HTTP server address
			addr := net.JoinHostPort(cfg.HTTP.Host, strconv.Itoa(cfg.HTTP.Port))
			log.Info().Str("addr", addr).Msg("server listening")

			// create HTTP server
			errcore.SetDebug(cfg.Debug)
			srv := rest.New(log)

			// setup routes
			srv.SetupRoutes()

			// run HTTP server
			if err := srv.Run(ctx, addr); err != nil {
				log.Error().Err(err).Msg("server stopped")
				return err
			}
			return nil
		},
	}
}
