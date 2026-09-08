package history

import (
	"path/filepath"
	"testing"

	"github.com/svalentino/ama/internal/llm"
)

func TestStoreSaveLoadAndClear(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "history.json"))
	want := []llm.Message{{Role: "user", Content: "hello"}}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	got, err = store.Load()
	if err != nil {
		t.Fatalf("Load() after Clear() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Load() after Clear() = %#v, want no messages", got)
	}
}
