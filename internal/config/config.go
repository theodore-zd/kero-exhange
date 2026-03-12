package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	LogLevel    string
}

func Load() (*Config, error) {
	godotenv.Load("postgres.env")
	godotenv.Load("exchange.env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		DatabaseURL: databaseURL,
		Port:        port,
		LogLevel:    logLevel,
	}, nil
}

func LoadForCLI() (*Config, error) {
	godotenv.Load("postgres.env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required (set in postgres.env)")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		DatabaseURL: databaseURL,
		LogLevel:    logLevel,
	}, nil
}
