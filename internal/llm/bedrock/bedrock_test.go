package bedrock

import (
	"errors"
	"testing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/svalentino/ama/internal/llm"
)

func TestRequiresSSOLogin(t *testing.T) {
	tokenErr := &ssocreds.InvalidTokenError{Err: errors.New("expired")}
	if !requiresSSOLogin(tokenErr) {
		t.Fatal("requiresSSOLogin() = false, want true")
	}
	if requiresSSOLogin(errors.New("network error")) {
		t.Fatal("requiresSSOLogin() = true for a non-SSO error")
	}
}

func TestConverseInputIncludesSystemPrompt(t *testing.T) {
	input := converseInput(llm.Request{
		Model:        "model-id",
		SystemPrompt: "Always answer in haiku.",
		Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
	})
	if len(input.System) != 1 {
		t.Fatalf("system blocks = %d, want 1", len(input.System))
	}
	block, ok := input.System[0].(*types.SystemContentBlockMemberText)
	if !ok {
		t.Fatalf("system block type = %T", input.System[0])
	}
	if block.Value != "Always answer in haiku." {
		t.Fatalf("system prompt = %q", block.Value)
	}
}

func TestLoadOptionsEnablesRequestAndResponseBodyLogging(t *testing.T) {
	provider, err := New(Config{Profile: "work"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	options := provider.loadOptions(llm.Request{Verbose: true})
	var loaded awsconfig.LoadOptions
	for _, option := range options {
		if err := option(&loaded); err != nil {
			t.Fatalf("apply load option: %v", err)
		}
	}
	if loaded.Logger == nil {
		t.Fatal("verbose load options do not configure a logger")
	}
	if loaded.ClientLogMode == nil {
		t.Fatal("verbose load options do not configure logging mode")
	}
	if !loaded.ClientLogMode.IsRequestWithBody() || !loaded.ClientLogMode.IsResponseWithBody() {
		t.Fatalf("verbose log mode = %v, want request and response bodies", *loaded.ClientLogMode)
	}
	if loaded.SharedConfigProfile != "work" {
		t.Fatalf("shared config profile = %q", loaded.SharedConfigProfile)
	}
}
