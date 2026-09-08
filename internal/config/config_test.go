package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "ama", "config.json"))
	want := Default("engineering")
	want.SystemPrompt = "Be precise."

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
	contents, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), "profile: engineering\n") {
		t.Fatalf("configuration is not YAML: %q", contents)
	}
	if !strings.Contains(string(contents), "provider: bedrock\n") {
		t.Fatalf("configuration does not select a provider: %q", contents)
	}
}

func TestStoreLoadMissingConfig(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "missing.json"))
	_, err := store.Load()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestValidateLeavesProviderSettingsToProvider(t *testing.T) {
	current := Default("")
	current.Provider = "future-api"
	if err := current.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestImportClaudeSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{
  "env": {
    "AWS_PROFILE": "work",
    "AWS_REGION": "us-east-1",
    "ANTHROPIC_MODEL": "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-5-20250929-v1:0"
  }
}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	current := Default("")
	current.SystemPrompt = "Be precise."
	got, err := ImportClaudeSettings(path, current)
	if err != nil {
		t.Fatalf("ImportClaudeSettings() error = %v", err)
	}
	if got.ProviderOption(ProviderOptionProfile) != "work" || got.ProviderOption(ProviderOptionRegion) != "us-east-1" {
		t.Fatalf("imported AWS configuration = %#v", got)
	}
	if got.Model != "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-5-20250929-v1:0" {
		t.Fatalf("imported model = %q", got.Model)
	}
	if got.SystemPrompt != "Be precise." {
		t.Fatalf("system prompt = %q, want preserved value", got.SystemPrompt)
	}
}

func TestImportClaudeSettingsRequiresProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"env": {}}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := ImportClaudeSettings(path, Default(""))
	if err == nil {
		t.Fatal("ImportClaudeSettings() error = nil, want missing profile error")
	}
}

func TestMigrateJSONConfigWritesYAML(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "config.json")
	yamlStore := NewStoreAt(filepath.Join(directory, "config.yaml"))
	if err := os.WriteFile(jsonPath, []byte(`{"aws_profile":"work","model":"model","system_prompt":"Be concise."}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := migrateJSONConfig(jsonPath, yamlStore); err != nil {
		t.Fatalf("migrateJSONConfig() error = %v", err)
	}
	got, err := yamlStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ProviderOption(ProviderOptionProfile) != "work" || got.Model != "model" {
		t.Fatalf("migrated config = %#v", got)
	}
	if got.Provider != ProviderBedrock {
		t.Fatalf("migrated provider = %q", got.Provider)
	}
	contents, err := os.ReadFile(yamlStore.Path())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.HasPrefix(string(contents), "{") {
		t.Fatalf("migrated configuration is JSON: %q", contents)
	}
}

func TestLoadMigratesLegacyYAMLOptions(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "config.yaml"))
	contents := []byte("aws_profile: work\naws_region: us-west-2\nmodel: model\nsystem_prompt: Be concise.\n")
	if err := os.WriteFile(store.Path(), contents, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ProviderName() != ProviderBedrock || got.ProviderOption(ProviderOptionProfile) != "work" || got.ProviderOption(ProviderOptionRegion) != "us-west-2" {
		t.Fatalf("Load() = %#v", got)
	}
}
