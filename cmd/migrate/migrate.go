package migrate

import (
	"context"
	"fmt"
	"os"

	"github.com/a-bazrkar/rocket-backend/internal/config"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/migration"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Move the schema forward or backward",
	}
	cmd.AddCommand(upCommand(), downCommand(), versionCommand())
	return cmd
}

func upCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply every pending migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(func(ctx context.Context, m *database.Migrator) error {
				return m.Up(ctx, 0)
			})
		},
	}
}

func downCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Roll back the current migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(func(ctx context.Context, m *database.Migrator) error {
				return m.Down(ctx, 1)
			})
		},
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the current schema version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(func(ctx context.Context, m *database.Migrator) error {
				return nil
			})
		},
	}
}

func run(fn func(context.Context, *database.Migrator) error) error {

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

	// create database
	db, _, err := database.New(cfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("database init failed")
		return err
	}
	defer db.Close()

	// create migrator
	ctx := context.Background()
	migrator := database.NewMigratorFS(db, migration.FS)

	// run migration
	if err := fn(ctx, migrator); err != nil {
		log.Error().Err(err).Msg("migration failed")
		return err
	}

	// print schema version
	version, dirty, err := migrator.Version(ctx)
	if err != nil {
		log.Error().Err(err).Msg("schema version")
		return err
	}
	log.Info().Int64("version", version).Bool("dirty", dirty).Msg("schema")
	return nil
}
