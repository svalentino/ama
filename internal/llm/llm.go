package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Verbose      bool
}

type Provider interface {
	Chat(context.Context, Request) (string, error)
}
