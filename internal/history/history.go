package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/svalentino/ama/internal/appdir"
	"github.com/svalentino/ama/internal/llm"
)

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	directory, err := appdir.Config()
	if err != nil {
		return nil, fmt.Errorf("find history directory: %w", err)
	}
	store := NewStoreAt(filepath.Join(directory, "history.json"))
	legacyDirectory, err := appdir.LegacyConfig()
	if err != nil {
		return nil, err
	}
	if err := appdir.MigrateFile(filepath.Join(legacyDirectory, "history.json"), store.path); err != nil {
		return nil, fmt.Errorf("migrate history: %w", err)
	}
	return store, nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (store *Store) Load() ([]llm.Message, error) {
	contents, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return []llm.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}

	var messages []llm.Message
	if err := json.Unmarshal(contents, &messages); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return messages, nil
}

func (store *Store) Save(messages []llm.Message) error {
	contents, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("encode history: %w", err)
	}
	return writeFile(store.path, append(contents, '\n'))
}

func (store *Store) Clear() error {
	if err := os.Remove(store.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ama-*")
	if err != nil {
		return fmt.Errorf("create temporary history: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary history: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary history: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary history: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}
