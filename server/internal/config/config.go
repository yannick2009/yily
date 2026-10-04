package config

import (
	"errors"
	"os"
	"strings"
)

var (
	// ErrMissingMasterKey is returned when the YILY_MASTER_KEY environment variable is empty.
	ErrMissingMasterKey = errors.New("YILY_MASTER_KEY is required (generate one with `openssl rand -base64 32`)")
)

// Config represents the configuration for the application.
type Config struct {
	MasterKey string
}

// Load reads and validates the configuration from environment variables.
func Load() (*Config, error) {
	masterKey := getEnv("YILY_MASTER_KEY", "")
	if masterKey == "" {
		return nil, ErrMissingMasterKey
	}

	return &Config{
		MasterKey: masterKey,
	}, nil
}

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is not present in the environment, it returns the defaultValue.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return defaultValue
}
