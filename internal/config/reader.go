package config

import (
	"errors"
	"fmt"

	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
	"github.com/a-bazrkar/rocket-backend/internal/packages/storage"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.AllowEmptyEnv(true)

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	return &Config{
		Logger: logger.Config{
			Level:  loadString("LOG_LEVEL"),
			Pretty: loadBool("LOG_PRETTY"),
		},
		Database: database.Config{
			Path:         loadString("DATABASE_PATH"),
			MaxOpenConns: loadInt("DATABASE_MAX_OPEN_CONNS"),
			MaxIdleConns: loadInt("DATABASE_MAX_IDLE_CONNS"),
			BusyTimeout:  loadDuration("DATABASE_BUSY_TIMEOUT"),
		},
		Storage: storage.Config{
			Root:     loadString("STORAGE_ROOT"),
			MaxBytes: loadInt64("STORAGE_MAX_BYTES"),
		},
	}, nil
}
