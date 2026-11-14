package config

import (
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/storage"
)

// Config is the process configuration. Load fills it from the environment.
type Config struct {
	Logger   logger.Config
	Database database.Config
	Storage  storage.Config
}
