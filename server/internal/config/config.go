package config

import (
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

const (
	// MasterKeySize is the required length in bytes for the AES-256 master key.
	MasterKeySize = 32
)

var (
	// ErrMissingMasterKey is returned when the YILY_MASTER_KEY environment variable is empty.
	ErrMissingMasterKey = errors.New("YILY_MASTER_KEY is required (generate one with `openssl rand -base64 32`)")

	// ErrInvalidMasterKeyBase64 is returned when YILY_MASTER_KEY is not valid standard Base64.
	ErrInvalidMasterKeyBase64 = errors.New("YILY_MASTER_KEY must be a valid base64-encoded string")

	// ErrInvalidMasterKeyLength is returned when the decoded key length is not exactly 32 bytes.
	ErrInvalidMasterKeyLength = errors.New("YILY_MASTER_KEY decoded length must be exactly 32 bytes")
)

// config represents the application configuration.
type config struct {
	MasterKey  []byte // AES-256 master key (32 bytes)
	SetupToken string // setup token for initial configuration
}

// Load reads and validates the configuration from environment variables.
func Load() (*config, error) {
	rawKey := getEnv("YILY_MASTER_KEY", "")
	if rawKey == "" {
		return nil, ErrMissingMasterKey
	}

	decodedKey, err := base64.StdEncoding.DecodeString(rawKey)
	if err != nil {
		return nil, ErrInvalidMasterKeyBase64
	}

	if len(decodedKey) != MasterKeySize {
		return nil, ErrInvalidMasterKeyLength
	}

	setupToken := getEnv("YILY_SETUP_TOKEN", "")

	return &config{
		MasterKey:  decodedKey,
		SetupToken: setupToken,
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
