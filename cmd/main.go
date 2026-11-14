package main

import (
	"os"

	"github.com/a-bazrkar/rocket-backend/internal/config"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
)

func main() {

	// load configs
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	// init logger
	log, err := logger.New(cfg.Logger)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	// init gorm
	repo, _, err := database.New(cfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("database init failed")
		os.Exit(1)
	}
	defer repo.Close()

	log.Info().Str("path", cfg.Database.Path).Msg("ready")
}
