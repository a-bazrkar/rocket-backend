package main

import (
	"os"

	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
)

func main() {
	log, err := logger.New(logger.Config{
		Level:  "info",
		Pretty: true,
	})
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	db, err := database.Init(database.Config{
		Path: "rocket.db",
	})
	if err != nil {
		log.Error().Err(err).Msg("database init failed")
		os.Exit(1)
	}
	defer db.Close()

	log.Info().Str("path", "rocket.db").Msg("ready")
}
