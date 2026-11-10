package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// executablePath returns the absolute path of the running binary.
func executablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return absolute, nil
}

// replaceExecutable atomically replaces exePath with the binary at source.
func replaceExecutable(source, exePath string) error {
	if err := os.Rename(source, exePath); err != nil {
		_ = os.Remove(source)
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("cannot replace %s (permission denied); reinstall with the install script or your package manager", exePath)
		}
		return fmt.Errorf("replace %s: %w", exePath, err)
	}
	return nil
}

// removeExecutable deletes the binary at path.
func removeExecutable(path string) error {
	if runtime.GOOS == "windows" {
		return errors.New("uninstall is not supported on Windows; delete the binary manually")
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("permission denied removing %s; try: sudo rm %s", path, path)
		}
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}
