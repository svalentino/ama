package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/svalentino/ama/internal/appdir"
	"gopkg.in/yaml.v3"
)

const (
	ProviderBedrock       = "bedrock"
	ProviderOptionProfile = "profile"
	ProviderOptionRegion  = "region"
	defaultModel          = "anthropic.claude-3-5-sonnet-20241022-v2:0"
)

var ErrNotFound = errors.New("configuration not found")

type Config struct {
	Provider     string            `json:"provider" yaml:"provider"`
	Model        string            `json:"model" yaml:"model"`
	SystemPrompt string            `json:"system_prompt" yaml:"system_prompt"`
	Options      map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

func Default(profile string) Config {
	config := Config{
		Provider:     ProviderBedrock,
		Model:        defaultModel,
		SystemPrompt: "You are a concise and helpful assistant.",
	}
	config.SetProviderOption(ProviderOptionProfile, profile)
	return config
}

func (config Config) ProviderName() string {
	if strings.TrimSpace(config.Provider) == "" {
		return ProviderBedrock
	}
	return config.Provider
}

func (config Config) withDefaults() Config {
	config.Provider = config.ProviderName()
	return config
}

func (config Config) ProviderOption(name string) string {
	return config.Options[name]
}

func (config *Config) SetProviderOption(name, value string) {
	if strings.TrimSpace(value) == "" {
		delete(config.Options, name)
		return
	}
	if config.Options == nil {
		config.Options = make(map[string]string)
	}
	config.Options[name] = value
}

func (config Config) Validate() error {
	if strings.TrimSpace(config.Model) == "" {
		return errors.New("model is required")
	}
	if strings.TrimSpace(config.SystemPrompt) == "" {
		return errors.New("system prompt is required")
	}
	return nil
}

type Store struct {
	path string
}

type persistedConfig struct {
	Provider      string            `json:"provider" yaml:"provider"`
	Model         string            `json:"model" yaml:"model"`
	SystemPrompt  string            `json:"system_prompt" yaml:"system_prompt"`
	Options       map[string]string `json:"options" yaml:"options"`
	LegacyProfile string            `json:"aws_profile" yaml:"aws_profile"`
	LegacyRegion  string            `json:"aws_region" yaml:"aws_region"`
}

func (persisted persistedConfig) config() Config {
	config := Config{
		Provider:     persisted.Provider,
		Model:        persisted.Model,
		SystemPrompt: persisted.SystemPrompt,
		Options:      persisted.Options,
	}
	if config.ProviderOption(ProviderOptionProfile) == "" {
		config.SetProviderOption(ProviderOptionProfile, persisted.LegacyProfile)
	}
	if config.ProviderOption(ProviderOptionRegion) == "" {
		config.SetProviderOption(ProviderOptionRegion, persisted.LegacyRegion)
	}
	return config.withDefaults()
}

func NewStore() (*Store, error) {
	directory, err := appdir.Config()
	if err != nil {
		return nil, fmt.Errorf("find config directory: %w", err)
	}
	store := NewStoreAt(filepath.Join(directory, "config.yaml"))
	if err := migrateJSONConfig(filepath.Join(directory, "config.json"), store); err != nil {
		return nil, fmt.Errorf("migrate configuration: %w", err)
	}
	legacyDirectory, err := appdir.LegacyConfig()
	if err != nil {
		return nil, err
	}
	if err := migrateJSONConfig(filepath.Join(legacyDirectory, "config.json"), store); err != nil {
		return nil, fmt.Errorf("migrate configuration: %w", err)
	}
	return store, nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (store *Store) Path() string {
	return store.path
}

func migrateJSONConfig(path string, store *Store) error {
	if path == store.path {
		return nil
	}
	if _, err := os.Stat(store.path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect YAML configuration: %w", err)
	}
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read JSON configuration: %w", err)
	}
	var persisted persistedConfig
	if err := json.Unmarshal(contents, &persisted); err != nil {
		return fmt.Errorf("parse JSON configuration: %w", err)
	}
	return store.Save(persisted.config())
}

func ClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func ImportClaudeSettings(path string, current Config) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read Claude settings: %w", err)
	}
	var settings struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(contents, &settings); err != nil {
		return Config{}, fmt.Errorf("parse Claude settings: %w", err)
	}

	profile := strings.TrimSpace(settings.Env["AWS_PROFILE"])
	if profile == "" {
		return Config{}, errors.New("Claude settings do not define env.AWS_PROFILE")
	}
	current.SetProviderOption(ProviderOptionProfile, profile)
	if region := strings.TrimSpace(settings.Env["AWS_REGION"]); region != "" {
		current.SetProviderOption(ProviderOptionRegion, region)
	}
	if model := claudeModel(settings.Env); model != "" {
		current.Model = model
	}
	if err := current.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid imported configuration: %w", err)
	}
	return current, nil
}

func claudeModel(environment map[string]string) string {
	for _, key := range []string{
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	} {
		if model := strings.TrimSpace(environment[key]); model != "" {
			return model
		}
	}
	return ""
}

func (store *Store) Load() (Config, error) {
	contents, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, ErrNotFound
	}
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	var persisted persistedConfig
	if err := yaml.Unmarshal(contents, &persisted); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	config := persisted.config()
	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return config, nil
}

func (store *Store) Save(config Config) error {
	config = config.withDefaults()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	contents, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	return writeFile(store.path, append(contents, '\n'))
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ama-*")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary configuration: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	return nil
}
