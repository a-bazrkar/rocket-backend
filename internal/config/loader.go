package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

func loadString(envName string) string {
	validate(envName)

	return viper.GetString(envName)
}

func loadInt(envName string) int {
	validate(envName)

	return viper.GetInt(envName)
}

func loadInt64(envName string) int64 {
	validate(envName)

	return viper.GetInt64(envName)
}

func loadBool(envName string) bool {
	validate(envName)

	return viper.GetBool(envName)
}

func loadDuration(envName string) time.Duration {
	validate(envName)

	value := viper.GetString(envName)
	duration, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("environment variable [%s] is not a valid duration: %s", envName, err.Error()))
	}

	return duration
}

func validate(envName string) {
	if !viper.IsSet(envName) {
		panic(fmt.Sprintf("environment variable [%s] does not exist", envName))
	}
}
