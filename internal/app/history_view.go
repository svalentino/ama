package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/svalentino/ama/internal/llm"
)

var (
	historyTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	youStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	amaStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	separatorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	codeBlockStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
)

func renderHistory(output io.Writer, messages []llm.Message) error {
	if len(messages) == 0 {
		_, err := fmt.Fprintln(output, "No conversation history yet.")
		return err
	}
	if _, err := fmt.Fprintln(output, historyTitleStyle.Render("Conversation history")); err != nil {
		return err
	}
	for index, message := range messages {
		style := amaStyle
		speaker := "Ama"
		if message.Role == "user" {
			style = youStyle
			speaker = "You"
		}
		if _, err := fmt.Fprintln(output, style.Render(speaker)); err != nil {
			return err
		}
		if err := printResponse(output, message.Content); err != nil {
			return fmt.Errorf("render %s message: %w", strings.ToLower(speaker), err)
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return err
		}
		if index < len(messages)-1 {
			if _, err := fmt.Fprintln(output, separatorStyle.Render("----")); err != nil {
				return err
			}
		}
	}
	return nil
}

func printResponse(output io.Writer, response string) error {
	lines := strings.SplitAfter(response, "\n")
	var plain, code strings.Builder
	var language string
	inCodeBlock := false

	flushPlain := func() error {
		if plain.Len() == 0 {
			return nil
		}
		_, err := fmt.Fprint(output, plain.String())
		plain.Reset()
		return err
	}

	for _, line := range lines {
		if !inCodeBlock {
			if detectedLanguage, ok := codeFenceLanguage(line); ok {
				if err := flushPlain(); err != nil {
					return err
				}
				language = detectedLanguage
				code.Reset()
				inCodeBlock = true
				continue
			}
			plain.WriteString(line)
			continue
		}

		if isClosingCodeFence(line) {
			if err := printCodeBlock(output, language, code.String()); err != nil {
				return err
			}
			if strings.HasSuffix(line, "\n") {
				if _, err := fmt.Fprint(output, "\n"); err != nil {
					return err
				}
			}
			inCodeBlock = false
			continue
		}
		code.WriteString(line)
	}

	if inCodeBlock {
		plain.WriteString("```")
		if language != "" {
			plain.WriteString(language)
		}
		plain.WriteByte('\n')
		plain.WriteString(code.String())
	}
	return flushPlain()
}

func printCodeBlock(output io.Writer, language, code string) error {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	var highlighted strings.Builder
	if err := quick.Highlight(&highlighted, code, lexer.Config().Name, "terminal256", "monokai"); err != nil {
		return fmt.Errorf("highlight code: %w", err)
	}
	_, err := fmt.Fprint(output, codeBlockStyle.Render(strings.TrimRight(highlighted.String(), "\n")))
	return err
}

func codeFenceLanguage(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}
	return strings.TrimSpace(trimmed[3:]), true
}

func isClosingCodeFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= 3 && strings.Trim(trimmed, "`") == ""
}
