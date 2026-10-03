package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Default values used when the corresponding environment variables is unset.
const (
	defaultAddr   = ":16529"
	defaultDBPath = "yily.db"
)

// Config represents the configuration for the application.
type Config struct {
	// Addr is the address the HTTP server listen on (YILY_ADDR).
	Addr string

	// DBPath is the path to the SQLite database file (YILY_DB_PATH).
	DBPath string

	// MasterKey is the master key used for encryption (YILY_MASTER_KEY).
	// Only its presence is checked here; its format (base64, 32 bytes) will be
	// validated by the encryption package so the rule lives in one place.
	MasterKey string
}

//	Load reads the configuration from environment and validates it.
//
// It returns an error describing every invalid setting at once.
func Load() (*Config, error) {
	cfg := &Config{
		Addr:      getEnv("YILY_ADDR", defaultAddr),
		DBPath:    getEnv("YILY_DB_PATH", defaultDBPath),
		MasterKey: getEnv("YILY_MASTER_KEY", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate collects all configuration errors instead of stopping at the first one.
// So the user can fix everything in a single pass/
func (cfg *Config) validate() error {
	var errs []error

	if cfg.MasterKey == "" {
		errs = append(errs, errors.New("YILY_MASTER_KEY is required (generate one with `openssl rand -base64 32`)"))
	}

	// errors.Join returns nil when errs is empty
	return errors.Join(errs...)
}

// String implements fmt.Stringer and never prints the master key, so the
// configuration can be logged safely.
func (c Config) String() string {
	return fmt.Sprintf("Config{Addr: %q, DBPath: %q, MasterKey: REDACTED}", c.Addr, c.DBPath)
}

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is not present in the environment, it returns the defaultValue.
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return defaultValue
}
