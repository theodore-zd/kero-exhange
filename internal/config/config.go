package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL                string
	Port                       string
	LogLevel                   string
	AdminPassword              string
	AdminPasswordHash          string
	DefaultCurrencyCode        string
	DefaultCurrencyName        string
	DefaultCurrencyDescription string
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

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminPasswordHash := os.Getenv("ADMIN_PASSWORD_HASH")
	if adminPassword == "" && adminPasswordHash == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD or ADMIN_PASSWORD_HASH is required")
	}

	defaultCurrencyCode := os.Getenv("DEFAULT_CURRENCY_CODE")
	if defaultCurrencyCode == "" {
		defaultCurrencyCode = "USD"
	}

	defaultCurrencyName := os.Getenv("DEFAULT_CURRENCY_NAME")
	if defaultCurrencyName == "" {
		defaultCurrencyName = "US Dollar"
	}

	defaultCurrencyDescription := os.Getenv("DEFAULT_CURRENCY_DESCRIPTION")

	return &Config{
		DatabaseURL:                databaseURL,
		Port:                       port,
		LogLevel:                   logLevel,
		AdminPassword:              adminPassword,
		AdminPasswordHash:          adminPasswordHash,
		DefaultCurrencyCode:        defaultCurrencyCode,
		DefaultCurrencyName:        defaultCurrencyName,
		DefaultCurrencyDescription: defaultCurrencyDescription,
	}, nil
}
