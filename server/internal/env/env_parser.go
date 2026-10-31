package env

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidJSONFormat     = errors.New("invalid JSON format")
	ErrInvalidYAMLFormat     = errors.New("invalid YAML format")
	ErrInvalidEnvFileFormat  = errors.New("invalid .env file format")
	ErrDuplicateVariableName = errors.New("duplicate variable name in file")
)

// parseJSONVariables parses JSON content into a map of environment variables.
// It expects the JSON to be an object with string keys and string values.
// For example: {"KEY1": "value1", "KEY2": "value2"}
func ParseJSONVariables(data []byte) (map[string]string, error) {
	var vars map[string]string
	if err := json.Unmarshal(data, &vars); err != nil {
		return nil, ErrInvalidJSONFormat
	}
	return vars, nil
}

// parseYAMLVariables parses YAML content into a map of environment variables.
// It expects the YAML to be a mapping of string keys to string values.
// For example:
// KEY1: value1
// KEY2: value2
func ParseYAMLVariables(data []byte) (map[string]string, error) {
	var vars map[string]string
	if err := yaml.Unmarshal(data, &vars); err != nil {
		return nil, ErrInvalidYAMLFormat
	}
	return vars, nil
}

// ParseEnvFileVariables parses .env content into a map of environment variables.
// It expects the .env file to have lines in the format KEY=VALUE.
func ParseEnvFileVariables(data []byte) (map[string]string, error) {
	vars, err := godotenv.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidEnvFileFormat
	}

	return vars, nil
}

// FormatJSONVariables serializes the variables into an indented JSON object.
func FormatJSONVariables(variables map[string]string) ([]byte, error) {
	data, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return nil, ErrInvalidJSONFormat
	}
	return append(data, '\n'), nil
}

// FormatYAMLVariables serializes the variables into a YAML mapping.
func FormatYAMLVariables(variables map[string]string) ([]byte, error) {
	data, err := yaml.Marshal(variables)
	if err != nil {
		return nil, ErrInvalidYAMLFormat
	}
	return data, nil
}

// FormatEnvFileVariables serializes the variables into KEY=VALUE lines.
func FormatEnvFileVariables(variables map[string]string) ([]byte, error) {
	data, err := godotenv.Marshal(variables)
	if err != nil {
		return nil, ErrInvalidEnvFileFormat
	}
	return []byte(data), nil
}
