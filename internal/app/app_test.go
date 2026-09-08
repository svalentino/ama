package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svalentino/ama/internal/config"
	"github.com/svalentino/ama/internal/history"
	"github.com/svalentino/ama/internal/llm"
)

type fakeProvider struct {
	request  llm.Request
	response string
}

func (provider *fakeProvider) Chat(_ context.Context, request llm.Request) (string, error) {
	provider.request = request
	return provider.response, nil
}

func TestAskSendsAndPersistsSingleConversation(t *testing.T) {
	directory := t.TempDir()
	configStore := config.NewStoreAt(filepath.Join(directory, "config.json"))
	if err := configStore.Save(config.Default("work")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	provider := &fakeProvider{response: "## Hello\n\nUse `ama history`."}
	var providerConfig config.Config
	var output bytes.Buffer
	application := newWithFactoryAndStores(func(current config.Config) (llm.Provider, error) {
		providerConfig = current
		return provider, nil
	}, &output, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"Say", "hello"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.request.Model != config.Default("work").Model {
		t.Fatalf("provider model = %q", provider.request.Model)
	}
	if providerConfig.ProviderName() != config.ProviderBedrock || providerConfig.ProviderOption(config.ProviderOptionProfile) != "work" {
		t.Fatalf("provider config = %#v", providerConfig)
	}
	if provider.request.SystemPrompt != config.Default("work").SystemPrompt {
		t.Fatalf("provider system prompt = %q", provider.request.SystemPrompt)
	}
	if len(provider.request.Messages) != 1 || provider.request.Messages[0].Content != "Say hello" {
		t.Fatalf("provider messages = %#v", provider.request.Messages)
	}
	if !strings.Contains(output.String(), "Hello") || !strings.Contains(output.String(), "ama history") {
		t.Fatalf("output = %q", output.String())
	}

	messages, err := historyStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(messages) != 2 || messages[0].Content != "Say hello" || messages[1].Content != "## Hello\n\nUse `ama history`." {
		t.Fatalf("history = %#v", messages)
	}
}

func TestAskPassesVerboseFlagToProvider(t *testing.T) {
	directory := t.TempDir()
	configStore := config.NewStoreAt(filepath.Join(directory, "config.yaml"))
	if err := configStore.Save(config.Default("work")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	provider := &fakeProvider{response: "Done."}
	application := newWithStores(provider, io.Discard, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"--verbose", "Status?"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !provider.request.Verbose {
		t.Fatal("provider request does not enable verbose logging")
	}
}

func TestAskTreatsCommandNameAfterDoubleDashAsQuestion(t *testing.T) {
	for _, commandName := range []string{"ask", "clear", "completion", "config", "help", "history"} {
		t.Run(commandName, func(t *testing.T) {
			directory := t.TempDir()
			configStore := config.NewStoreAt(filepath.Join(directory, "config.yaml"))
			if err := configStore.Save(config.Default("work")); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
			if err := historyStore.Save([]llm.Message{
				{Role: "user", Content: "Previous question"},
				{Role: "assistant", Content: "Previous answer"},
			}); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			provider := &fakeProvider{response: "Answer"}
			application := newWithStores(provider, io.Discard, configStore, historyStore, "dev")

			if err := application.Run(context.Background(), []string{"--", commandName, "question"}); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			messages := provider.request.Messages
			if len(messages) != 3 || messages[2].Content != commandName+" question" {
				t.Fatalf("provider messages = %#v", messages)
			}
		})
	}
}

func TestClearStartsFreshConversationWhenGivenQuestion(t *testing.T) {
	directory := t.TempDir()
	configStore := config.NewStoreAt(filepath.Join(directory, "config.yaml"))
	if err := configStore.Save(config.Default("work")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	if err := historyStore.Save([]llm.Message{
		{Role: "user", Content: "Old question"},
		{Role: "assistant", Content: "Old answer"},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	provider := &fakeProvider{response: "Fresh answer"}
	var output bytes.Buffer
	application := newWithStores(provider, &output, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"clear", "--", "clear", "sky", "today?"}); err != nil {
		t.Fatalf("Run(clear question) error = %v", err)
	}
	if len(provider.request.Messages) != 1 || provider.request.Messages[0].Content != "clear sky today?" {
		t.Fatalf("provider messages = %#v", provider.request.Messages)
	}
	if output.String() != "Fresh answer\n" {
		t.Fatalf("output = %q", output.String())
	}

	messages, err := historyStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(messages) != 2 || messages[0].Content != "clear sky today?" || messages[1].Content != "Fresh answer" {
		t.Fatalf("history = %#v", messages)
	}
}

func TestClearRemovesCurrentConversation(t *testing.T) {
	directory := t.TempDir()
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	if err := historyStore.Save([]llm.Message{{Role: "user", Content: "Old question"}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	var output bytes.Buffer
	application := newWithStores(
		&fakeProvider{},
		&output,
		config.NewStoreAt(filepath.Join(directory, "config.yaml")),
		historyStore,
		"dev",
	)

	if err := application.Run(context.Background(), []string{"clear"}); err != nil {
		t.Fatalf("Run(clear) error = %v", err)
	}
	if output.String() != "Conversation history cleared.\n" {
		t.Fatalf("output = %q", output.String())
	}

	messages, err := historyStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("history = %#v", messages)
	}
}

func TestVersionFlag(t *testing.T) {
	directory := t.TempDir()
	var output bytes.Buffer
	application := newWithStores(
		&fakeProvider{},
		&output,
		config.NewStoreAt(filepath.Join(directory, "config.yaml")),
		history.NewStoreAt(filepath.Join(directory, "history.json")),
		"v1.2.3",
	)

	if err := application.Run(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("Run(--version) error = %v", err)
	}
	if output.String() != "ama version v1.2.3\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

func TestImportClaudeCreatesConfiguration(t *testing.T) {
	directory := t.TempDir()
	settingsPath := filepath.Join(directory, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{
  "env": {
    "AWS_PROFILE": "work",
    "AWS_REGION": "us-west-2",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "us.anthropic.claude-sonnet-4-5-20250929-v1:0"
  }
}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	configStore := config.NewStoreAt(filepath.Join(directory, "config.json"))
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	var output bytes.Buffer
	application := newWithStores(&fakeProvider{}, &output, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"config", "import-claude", "--file", settingsPath}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got, err := configStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ProviderOption(config.ProviderOptionProfile) != "work" || got.ProviderOption(config.ProviderOptionRegion) != "us-west-2" || got.Model != "us.anthropic.claude-sonnet-4-5-20250929-v1:0" {
		t.Fatalf("imported config = %#v", got)
	}
	if output.String() != "Claude settings imported.\n" {
		t.Fatalf("output = %q", output.String())
	}

	output.Reset()
	if err := application.Run(context.Background(), []string{"config", "show"}); err != nil {
		t.Fatalf("Run(config show) error = %v", err)
	}
	if !strings.Contains(output.String(), "Region: us-west-2\n") {
		t.Fatalf("config show output = %q", output.String())
	}
}

func TestConfigInitAndShowReportConfigPath(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	configStore := config.NewStoreAt(configPath)
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	var output bytes.Buffer
	application := newWithStores(&fakeProvider{}, &output, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"config", "init", "--profile", "work"}); err != nil {
		t.Fatalf("Run(config init) error = %v", err)
	}
	if output.String() != "Configuration saved: "+configPath+"\n" {
		t.Fatalf("config init output = %q", output.String())
	}
	current, err := configStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if current.Provider != config.ProviderBedrock {
		t.Fatalf("provider = %q", current.Provider)
	}

	output.Reset()
	if err := application.Run(context.Background(), []string{"config", "show"}); err != nil {
		t.Fatalf("Run(config show) error = %v", err)
	}
	if !strings.Contains(output.String(), "Configuration file: "+configPath+"\n") {
		t.Fatalf("config show output = %q", output.String())
	}
}

func TestHistoryCommandRendersPersistedConversation(t *testing.T) {
	directory := t.TempDir()
	configStore := config.NewStoreAt(filepath.Join(directory, "config.json"))
	historyStore := history.NewStoreAt(filepath.Join(directory, "history.json"))
	if err := historyStore.Save([]llm.Message{
		{Role: "user", Content: "Show a **bold** example."},
		{Role: "assistant", Content: "Use `fmt.Println`."},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	var output bytes.Buffer
	application := newWithStores(&fakeProvider{}, &output, configStore, historyStore, "dev")

	if err := application.Run(context.Background(), []string{"history"}); err != nil {
		t.Fatalf("Run(history) error = %v", err)
	}
	for _, content := range []string{"Conversation history", "You", "Ama", "bold", "fmt.Println"} {
		if !strings.Contains(output.String(), content) {
			t.Fatalf("history output does not contain %q: %q", content, output.String())
		}
	}
}
