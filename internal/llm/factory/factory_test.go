package factory

import (
	"strings"
	"testing"

	"github.com/svalentino/ama/internal/config"
	"github.com/svalentino/ama/internal/llm/bedrock"
)

func TestNewSelectsConfiguredProvider(t *testing.T) {
	provider, err := New(config.Default("work"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := provider.(bedrock.Provider); !ok {
		t.Fatalf("New() provider = %T, want bedrock.Provider", provider)
	}
}

func TestNewRejectsUnsupportedProvider(t *testing.T) {
	current := config.Default("work")
	current.Provider = "future-api"
	_, err := New(current)
	if err == nil || !strings.Contains(err.Error(), `provider "future-api" is not supported`) {
		t.Fatalf("New() error = %v", err)
	}
}

func TestNewValidatesProviderConfiguration(t *testing.T) {
	_, err := New(config.Default(""))
	if err == nil || !strings.Contains(err.Error(), `initialize provider "bedrock": AWS profile is required`) {
		t.Fatalf("New() error = %v", err)
	}
}
