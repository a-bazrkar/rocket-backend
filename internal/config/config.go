package config

import (
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/storage"
	"github.com/a-bazrkar/rocket-backend/internal/packages/token"
	"github.com/a-bazrkar/rocket-backend/internal/transport/rest"
)

// Config is the process configuration. Load fills it from the environment.
type Config struct {
	Debug    bool
	Logger   logger.Config
	Database database.Config
	Storage  storage.Config
	HTTP     rest.Config
	Token    token.Config
}
