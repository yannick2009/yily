package config

import (
	"errors"
	"os"
	"strings"
)

var (
	// ErrMissingMasterKey is returned when the YILY_MASTER_KEY environment variable is missing or empty.
	ErrMissingMasterKey = errors.New("YILY_MASTER_KEY is required (generate one with `openssl rand -base64 32`)")
)

// Config represents the configuration for the application.
type Config struct {
	// MasterKey is the master key used for encryption (YILY_MASTER_KEY).
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

// String implements fmt.Stringer and prevents the master key from being printed in logs.
func (c Config) String() string {
	return "Config{MasterKey: REDACTED}"
}

// getEnv retrieves the value of the environment variable named by key.
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return defaultValue
}
