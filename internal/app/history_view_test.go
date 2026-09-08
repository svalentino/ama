package app

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/svalentino/ama/internal/llm"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func TestRenderHistoryIsCompactAndPreservesCodeBlocks(t *testing.T) {
	messages := []llm.Message{
		{Role: "user", Content: "Show a `Go` example."},
		{Role: "assistant", Content: "```go\nfmt.Println(\"hello\")\n```"},
	}
	var output bytes.Buffer
	if err := renderHistory(&output, messages); err != nil {
		t.Fatalf("renderHistory() error = %v", err)
	}
	got := stripANSI(output.String())
	for _, content := range []string{"Conversation history", "You", "Show a", "----", "Ama", "fmt.Println"} {
		if !strings.Contains(got, content) {
			t.Fatalf("history output lacks %q: %q", content, got)
		}
	}
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("history output has excessive blank lines: %q", got)
	}
}

func TestRenderHistoryEmptyConversation(t *testing.T) {
	var output bytes.Buffer
	if err := renderHistory(&output, nil); err != nil {
		t.Fatalf("renderHistory() error = %v", err)
	}
	if output.String() != "No conversation history yet.\n" {
		t.Fatalf("renderHistory() output = %q", output.String())
	}
}

func TestPrintResponseRendersFencedCode(t *testing.T) {
	var output bytes.Buffer
	if err := printResponse(&output, "Before\n```go\nfmt.Println(\"hello\")\n```\nAfter"); err != nil {
		t.Fatalf("printResponse() error = %v", err)
	}
	got := stripANSI(output.String())
	if strings.Contains(got, "```go") {
		t.Fatalf("code fence was printed literally: %q", got)
	}
	if !strings.Contains(got, "╭") || !strings.Contains(got, "╰") {
		t.Fatalf("code block is not visually bounded: %q", got)
	}
	for _, content := range []string{"Before", "fmt.Println", "After"} {
		if !strings.Contains(got, content) {
			t.Fatalf("rendered output lacks %q: %q", content, got)
		}
	}
	if !strings.Contains(got, "\n│fmt.Println") {
		t.Fatalf("code is not left-aligned inside its box: %q", got)
	}
}

func stripANSI(value string) string {
	return ansiEscape.ReplaceAllString(value, "")
}

func TestPrintResponsePreservesUnfencedMarkdown(t *testing.T) {
	const response = "# Heading\n\nA **bold** statement."
	var output bytes.Buffer
	if err := printResponse(&output, response); err != nil {
		t.Fatalf("printResponse() error = %v", err)
	}
	if output.String() != response {
		t.Fatalf("printResponse() = %q, want %q", output.String(), response)
	}
}
