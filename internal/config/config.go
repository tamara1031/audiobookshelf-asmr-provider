package config

import (
	"os"
)

// Config holds the application configuration.
type Config struct {
	Port     string
	LogLevel string
}

// Load initializes the configuration from environment variables.
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	return &Config{
		Port:     port,
		LogLevel: logLevel,
	}
}
