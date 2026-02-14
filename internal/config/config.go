package config

import (
	"os"
)

// Config holds the application configuration.
type Config struct {
	Port     string
	LogLevel string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	return &Config{
		Port:     port,
		LogLevel: logLevel,
	}
}
