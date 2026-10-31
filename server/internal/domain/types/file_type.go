package types

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrUnsupportedFileType is returned when a file extension does not match any supported file type.
var ErrUnsupportedFileType = errors.New("unsupported file type")

// FileType identifies the format of an imported file.
type FileType string

const (
	FileTypeJSON FileType = "json" // FileTypeJSON identifies a JSON file.
	FileTypeYAML FileType = "yaml" // FileTypeYAML identifies a YAML file.
	FileTypeEnv  FileType = "env"  // FileTypeEnv identifies a .env file.
)

// FromFilename returns the FileType inferred from a file extension, case-insensitively.
// It supports .json, .yaml/.yml and .env, and returns ErrUnsupportedFileType otherwise.
func FromFilename(filename string) (FileType, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json":
		return FileTypeJSON, nil
	case ".yaml", ".yml":
		return FileTypeYAML, nil
	case ".env":
		return FileTypeEnv, nil
	default:
		return "", ErrUnsupportedFileType
	}
}

// FromString returns the FileType for a format name (e.g. "json", "yaml" or "env"), case-insensitively.
// It returns ErrUnsupportedFileType when the name does not match a supported format.
func FromString(s string) (FileType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FileTypeJSON, nil
	case "yaml", "yml":
		return FileTypeYAML, nil
	case "env":
		return FileTypeEnv, nil
	default:
		return "", ErrUnsupportedFileType
	}
}

// Extension returns the file extension for the file type, including the leading dot.
func (f FileType) Extension() string {
	switch f {
	case FileTypeJSON:
		return ".json"
	case FileTypeYAML:
		return ".yaml"
	case FileTypeEnv:
		return ".env"
	default:
		return ""
	}
}
