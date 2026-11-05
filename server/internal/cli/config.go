package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the structure of the configuration file for the CLI application.
type Config struct {
	BaseURL  string     `json:"base_url"`            // BaseURL is the base URL of the Yily server.
	Token    string     `json:"token"`               // Token is the authentication token used for API requests.
	Scope    string     `json:"scope"`               // Scope is the permission scope associated with the token.
	ExpireAt *time.Time `json:"expire_at,omitempty"` // ExpireAt is the token expiration time; nil means it never expires.
}

// LoadConfigFile loads the configuration from a JSON file.
func LoadConfigFile() (*Config, error) {
	configFilePath, err := configPath()
	if err != nil {
		return nil, err
	}

	// Check if the configuration file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, nil // Return nil if the config file does not exist
	}

	// Read the configuration file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Unmarshal the JSON data into a Config struct
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	return &config, nil
}

// WriteConfigFile writes the configuration to a JSON file with owner-only permissions.
func WriteConfigFile(config *Config) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	configFilePath, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	// MkdirAll leaves an existing directory's permissions untouched.
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}

	if err := os.WriteFile(configFilePath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	// WriteFile only applies perm on creation, so enforce it for existing files too.
	if err := os.Chmod(configFilePath, 0o600); err != nil {
		return fmt.Errorf("secure config file: %w", err)
	}

	return nil
}

// CleanupConfigFile removes the configuration file for the CLI application.
func CleanupConfigFile() error {
	configFilePath, err := configPath()
	if err != nil {
		return err
	}

	// Check if the configuration file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil // Return nil if the config file does not exist
	}

	// Remove the configuration file
	if err := os.Remove(configFilePath); err != nil {
		return fmt.Errorf("remove config file: %w", err)
	}

	return nil
}

// configPath returns the path to the configuration file for the CLI application.
func configPath() (string, error) {
	rootPath, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}

	return filepath.Join(rootPath, "yily", "config.json"), nil
}
