package appdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Config() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ama"), nil
}

func LegacyConfig() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find platform configuration directory: %w", err)
	}
	return filepath.Join(directory, "ama"), nil
}

func MigrateFile(source, destination string) error {
	if source == destination {
		return nil
	}
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect destination: %w", err)
	}

	contents, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read existing file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	if err := os.WriteFile(destination, contents, 0600); err != nil {
		return fmt.Errorf("write migrated file: %w", err)
	}
	return nil
}
