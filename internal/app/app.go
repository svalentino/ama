package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/svalentino/ama/internal/config"
	"github.com/svalentino/ama/internal/history"
	"github.com/svalentino/ama/internal/llm"
)

type Application struct {
	providerNew  ProviderFactory
	configStore  *config.Store
	historyStore *history.Store
	output       io.Writer
	version      string
}

type ProviderFactory func(config.Config) (llm.Provider, error)

func New(providerNew ProviderFactory, output io.Writer, version string) (*Application, error) {
	configStore, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	historyStore, err := history.NewStore()
	if err != nil {
		return nil, err
	}
	return newWithFactoryAndStores(providerNew, output, configStore, historyStore, version), nil
}

func newWithStores(provider llm.Provider, output io.Writer, configStore *config.Store, historyStore *history.Store, version string) *Application {
	return newWithFactoryAndStores(func(config.Config) (llm.Provider, error) {
		return provider, nil
	}, output, configStore, historyStore, version)
}

func newWithFactoryAndStores(providerNew ProviderFactory, output io.Writer, configStore *config.Store, historyStore *history.Store, version string) *Application {
	return &Application{
		providerNew:  providerNew,
		configStore:  configStore,
		historyStore: historyStore,
		output:       output,
		version:      version,
	}
}

func (application *Application) Run(ctx context.Context, arguments []string) error {
	if application.configStore == nil || application.historyStore == nil {
		return errors.New("initialize application data directories")
	}
	command := application.Command()
	command.SetArgs(arguments)
	return command.ExecuteContext(ctx)
}

type askOptions struct {
	profile string
	model   string
	system  string
	verbose bool
}

func (application *Application) Command() *cobra.Command {
	options := askOptions{}
	root := &cobra.Command{
		Use:           "ama [flags] QUESTION",
		Short:         "Chat with an LLM from your terminal",
		Long:          "ama is a fast, local-first CLI for persistent conversations with LLMs.",
		Version:       application.version,
		Example:       "  ama \"Explain this error\"\n  ama history\n  ama config init --profile work",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				return command.Help()
			}
			return application.ask(command.Context(), arguments, options)
		},
	}
	root.SetOut(application.output)
	root.SetErr(io.Discard)
	addAskFlags(root, &options)
	root.AddCommand(
		application.askCommand(),
		application.clearCommand(),
		application.historyCommand(),
		application.configCommand(),
	)
	return root
}

func addAskFlags(command *cobra.Command, options *askOptions) {
	command.Flags().StringVar(&options.profile, "profile", "", "Override the provider profile")
	command.Flags().StringVar(&options.model, "model", "", "Override the model")
	command.Flags().StringVar(&options.system, "system", "", "Override the system directive")
	command.Flags().BoolVar(&options.verbose, "verbose", false, "Log raw provider requests and responses to stderr")
}

func (application *Application) askCommand() *cobra.Command {
	options := askOptions{}
	command := &cobra.Command{
		Use:     "ask QUESTION...",
		Aliases: []string{"a"},
		Short:   "Ask a question in the current conversation",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			return application.ask(command.Context(), arguments, options)
		},
	}
	addAskFlags(command, &options)
	return command
}

func (application *Application) ask(ctx context.Context, arguments []string, options askOptions) error {

	current, err := application.configStore.Load()
	if err != nil {
		return configurationError(err)
	}
	if options.profile != "" {
		current.SetProviderOption(config.ProviderOptionProfile, options.profile)
	}
	if options.model != "" {
		current.Model = options.model
	}
	if options.system != "" {
		current.SystemPrompt = options.system
	}
	if err := current.Validate(); err != nil {
		return err
	}
	provider, err := application.providerNew(current)
	if err != nil {
		return err
	}

	messages, err := application.historyStore.Load()
	if err != nil {
		return err
	}
	question := strings.Join(arguments, " ")
	messages = append(messages, llm.Message{Role: "user", Content: question})
	answer, err := provider.Chat(ctx, llm.Request{
		Model:        current.Model,
		SystemPrompt: current.SystemPrompt,
		Messages:     messages,
		Verbose:      options.verbose,
	})
	if err != nil {
		return err
	}
	messages = append(messages, llm.Message{Role: "assistant", Content: answer})
	if err := application.historyStore.Save(messages); err != nil {
		return err
	}
	if err := printResponse(application.output, answer); err != nil {
		return fmt.Errorf("render code blocks in response: %w", err)
	}
	_, err = fmt.Fprintln(application.output)
	return err
}

func (application *Application) clearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear the current conversation history",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := application.historyStore.Clear(); err != nil {
				return err
			}
			_, err := fmt.Fprintln(command.OutOrStdout(), "Conversation history cleared.")
			return err
		},
	}
}

func (application *Application) historyCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "history",
		Aliases: []string{"h"},
		Short:   "Render the current conversation history",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			messages, err := application.historyStore.Load()
			if err != nil {
				return err
			}
			return renderHistory(command.OutOrStdout(), messages)
		},
	}
}

func (application *Application) importClaudeCommand() *cobra.Command {
	var path string
	command := &cobra.Command{
		Use:   "import-claude",
		Short: "Import AWS and Bedrock settings from Claude Code",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if path == "" {
				var err error
				path, err = config.ClaudeSettingsPath()
				if err != nil {
					return err
				}
			}
			current, err := application.configStore.Load()
			if errors.Is(err, config.ErrNotFound) {
				current = config.Default("")
			} else if err != nil {
				return err
			}
			imported, err := config.ImportClaudeSettings(path, current)
			if err != nil {
				return err
			}
			if err := application.configStore.Save(imported); err != nil {
				return err
			}
			_, err = fmt.Fprintln(command.OutOrStdout(), "Claude settings imported.")
			return err
		},
	}
	command.Flags().StringVar(&path, "file", "", "Claude settings file")
	return command
}

func (application *Application) configCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Manage ama configuration",
	}
	command.AddCommand(
		application.configInitCommand(),
		application.configShowCommand(),
		application.importClaudeCommand(),
		application.configSetCommand("set-provider", "Set the LLM provider", func(current *config.Config, value string) { current.Provider = value }),
		application.configSetCommand("set-profile", "Set the provider profile", func(current *config.Config, value string) {
			current.SetProviderOption(config.ProviderOptionProfile, value)
		}),
		application.configSetCommand("set-region", "Set the provider region", func(current *config.Config, value string) {
			current.SetProviderOption(config.ProviderOptionRegion, value)
		}),
		application.configSetCommand("set-model", "Set the model", func(current *config.Config, value string) { current.Model = value }),
		application.configSetCommand("set-system", "Set the system directive", func(current *config.Config, value string) { current.SystemPrompt = value }),
	)
	return command
}

func (application *Application) configInitCommand() *cobra.Command {
	defaults := config.Default("")
	var provider, profile, region, model, system string
	command := &cobra.Command{
		Use:   "init",
		Short: "Create or replace configuration",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			newConfig := config.Config{Provider: provider, Model: model, SystemPrompt: system}
			newConfig.SetProviderOption(config.ProviderOptionProfile, profile)
			newConfig.SetProviderOption(config.ProviderOptionRegion, region)
			if err := application.configStore.Save(newConfig); err != nil {
				return err
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "Configuration saved: %s\n", application.configStore.Path())
			return err
		},
	}
	command.Flags().StringVar(&provider, "provider", defaults.ProviderName(), "LLM provider")
	command.Flags().StringVar(&profile, "profile", "", "Provider profile")
	command.Flags().StringVar(&region, "region", "", "Provider region")
	command.Flags().StringVar(&model, "model", defaults.Model, "Model")
	command.Flags().StringVar(&system, "system", defaults.SystemPrompt, "System directive")
	_ = command.MarkFlagRequired("profile")
	return command
}

func (application *Application) configShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show active configuration",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			current, err := application.configStore.Load()
			if err != nil {
				return configurationError(err)
			}
			_, err = fmt.Fprintf(command.OutOrStdout(), "Configuration file: %s\nProvider: %s\nProfile: %s\nRegion: %s\nModel: %s\nSystem directive: %s\n", application.configStore.Path(), current.ProviderName(), current.ProviderOption(config.ProviderOptionProfile), current.ProviderOption(config.ProviderOptionRegion), current.Model, current.SystemPrompt)
			return err
		},
	}
}

func (application *Application) configSetCommand(use, short string, setValue func(*config.Config, string)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " VALUE...",
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			current, err := application.configStore.Load()
			if err != nil {
				return configurationError(err)
			}
			setValue(&current, strings.Join(arguments, " "))
			if err := application.configStore.Save(current); err != nil {
				return err
			}
			_, err = fmt.Fprintln(command.OutOrStdout(), "Configuration saved.")
			return err
		},
	}
}

func configurationError(err error) error {
	if errors.Is(err, config.ErrNotFound) {
		return errors.New("configuration not found; run: ama config init --profile YOUR_PROFILE")
	}
	return err
}
