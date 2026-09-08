package appdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigUsesHomeDotConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := Config()
	if err != nil {
		t.Fatalf("Config() error = %v", err)
	}
	want := filepath.Join(home, ".config", "ama")
	if got != want {
		t.Fatalf("Config() = %q, want %q", got, want)
	}
}

func TestMigrateFileCopiesOnlyWhenDestinationMissing(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy", "config.json")
	destination := filepath.Join(directory, "new", "config.json")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := MigrateFile(source, destination); err != nil {
		t.Fatalf("MigrateFile() error = %v", err)
	}
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(contents) != "original" {
		t.Fatalf("migrated contents = %q", contents)
	}
	if err := os.WriteFile(destination, []byte("current"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := MigrateFile(source, destination); err != nil {
		t.Fatalf("MigrateFile() error = %v", err)
	}
	contents, err = os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(contents) != "current" {
		t.Fatalf("existing destination was overwritten: %q", contents)
	}
}
