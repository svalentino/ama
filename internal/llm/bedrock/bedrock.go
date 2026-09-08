package bedrock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go/logging"
	"github.com/svalentino/ama/internal/llm"
)

type ssoLoginFunc func(context.Context, string) error

type Config struct {
	Profile string
	Region  string
}

type Provider struct {
	config Config
	login  ssoLoginFunc
	logger logging.Logger
}

func New(config Config) (Provider, error) {
	if strings.TrimSpace(config.Profile) == "" {
		return Provider{}, errors.New("AWS profile is required")
	}
	return Provider{config: config, login: runSSOLogin, logger: logging.NewStandardLogger(os.Stderr)}, nil
}

func (provider Provider) Chat(ctx context.Context, request llm.Request) (string, error) {
	answer, err := provider.chat(ctx, request)
	if err == nil || !requiresSSOLogin(err) {
		return answer, err
	}
	if provider.login == nil {
		return "", errors.New("AWS SSO authentication is unavailable")
	}
	if err := provider.login(ctx, provider.config.Profile); err != nil {
		return "", fmt.Errorf("authenticate AWS SSO profile %q: %w", provider.config.Profile, err)
	}
	return provider.chat(ctx, request)
}

func (provider Provider) chat(ctx context.Context, request llm.Request) (string, error) {
	config, err := awsconfig.LoadDefaultConfig(ctx, provider.loadOptions(request)...)
	if err != nil {
		return "", fmt.Errorf("load AWS profile: %w", err)
	}

	client := bedrockruntime.NewFromConfig(config)
	response, err := client.Converse(ctx, converseInput(request))
	if err != nil {
		return "", fmt.Errorf("call Bedrock Converse: %w", err)
	}

	message, ok := response.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected Bedrock response type %T", response.Output)
	}

	var answer strings.Builder
	for _, block := range message.Value.Content {
		if text, ok := block.(*types.ContentBlockMemberText); ok {
			answer.WriteString(text.Value)
		}
	}
	if answer.Len() == 0 {
		return "", fmt.Errorf("Bedrock returned no text content")
	}

	return answer.String(), nil
}

func (provider Provider) loadOptions(request llm.Request) []func(*awsconfig.LoadOptions) error {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithSharedConfigProfile(provider.config.Profile),
	}
	if provider.config.Region != "" {
		options = append(options, awsconfig.WithRegion(provider.config.Region))
	}
	if request.Verbose {
		logger := provider.logger
		if logger == nil {
			logger = logging.NewStandardLogger(os.Stderr)
		}
		options = append(options,
			awsconfig.WithLogger(logger),
			awsconfig.WithClientLogMode(aws.LogRequestWithBody|aws.LogResponseWithBody),
		)
	}
	return options
}

func converseInput(request llm.Request) *bedrockruntime.ConverseInput {
	return &bedrockruntime.ConverseInput{
		ModelId:  aws.String(request.Model),
		Messages: toBedrockMessages(request.Messages),
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: request.SystemPrompt},
		},
	}
}

func requiresSSOLogin(requestErr error) bool {
	var tokenErr *ssocreds.InvalidTokenError
	return errors.As(requestErr, &tokenErr)
}

func runSSOLogin(ctx context.Context, profile string) error {
	fmt.Fprintf(os.Stderr, "ama: AWS SSO credentials for profile %q are unavailable; signing in...\n", profile)
	command := exec.CommandContext(ctx, "aws", "sso", "login", "--profile", profile)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func toBedrockMessages(messages []llm.Message) []types.Message {
	converted := make([]types.Message, 0, len(messages))
	for _, message := range messages {
		role := types.ConversationRoleUser
		if message.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}
		converted = append(converted, types.Message{
			Role: role,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: message.Content},
			},
		})
	}
	return converted
}
