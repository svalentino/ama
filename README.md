<p align="center">
  <img src="docs/assets/ama-logo.svg" width="680" alt="ama - one shell, many models">
</p>

<h1 align="center">ama — Ask Me Anything</h1>

<p align="center">
  <strong>A fast, local-first command-line wrapper for LLM conversations.</strong><br>
  Keep the prompt in your shell, the context on your machine, and the browser tab count merely unreasonable.
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.24 or later"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-E85D4A?style=flat-square" alt="MIT license"></a>
</p>

> [!NOTE]
> Amazon Bedrock is the only provider adapter shipped today because it is the one currently needed. The command layer and LLM request contract are provider-neutral, and the `provider` config field selects the adapter. More APIs can be added without teaching the rest of ama their personal mythology. For now, `bedrock` is the only supported value.

## Why ama?

The name is the pitch: ask me anything, from your terminal.

- **One command:** ask with `ama "your question"`; no interactive shell or ceremonial clicking required.
- **Continuous context:** every successful exchange joins one persistent conversation.
- **Provider-ready core:** the application talks to a small `llm.Provider` interface, not directly to an SDK.
- **Local control:** configuration and conversation history stay in user-only files on your machine.
- **Readable output:** closed fenced code blocks receive terminal-friendly syntax highlighting.
- **No credential collection hobby:** providers use their native authentication; the Bedrock adapter uses your existing AWS profile.

## Install

```sh
go install github.com/svalentino/ama/cmd/ama@latest
```

Make sure Go's binary directory is on your `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Or build from source:

```sh
git clone https://github.com/svalentino/ama.git
cd ama
go build -o bin/ama ./cmd/ama
```

## Quick Start

The current adapter needs Linux or macOS, the [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html), and an AWS profile with access to an Amazon Bedrock model.

Initialize ama with that profile:

```sh
ama config init --provider bedrock --profile work
```

Then ask something:

```sh
ama "Explain the difference between an IAM role and an IAM user."
```

Follow-ups reuse the same conversation. ama remembers context so you do not have to paste the previous answer back into the machine that just produced it.

```sh
ama "Turn that into a comparison table."
ama history
```

Start fresh when the conversation has accumulated too much character development:

```sh
ama clear
```

## Configuration

Configuration is stored at `~/.config/ama/config.yaml`. A generated Bedrock configuration looks like this:

```yaml
provider: bedrock
model: anthropic.claude-3-5-sonnet-20241022-v2:0
system_prompt: You are a concise and helpful assistant.
options:
  profile: work
  region: us-east-1
```

`provider` chooses the adapter, while `options` belongs to that adapter. Existing configuration files without `provider`, or with the earlier `aws_profile` and `aws_region` keys, continue to load as Bedrock configuration. The region is optional; when omitted, the current adapter lets the AWS SDK resolve it from the profile and environment.

```sh
# Inspect the active settings
ama config show

# Select the adapter (only Bedrock ships today)
ama config set-provider bedrock

# Change the model ID or inference-profile ARN
ama config set-model anthropic.claude-3-7-sonnet-20250219-v1:0

# Set a persistent system directive
ama config set-system "Answer concisely and state assumptions."

# Override saved settings for one request
ama --profile sandbox --model MODEL_ID "Review this policy."
```

If you use Claude Code with Bedrock, ama can import its profile, region, and model selection:

```sh
ama config import-claude
ama config import-claude --file /path/to/settings.json
```

Run `ama --help` or `ama config --help` for the complete command reference. Run `ama --version` to inspect the installed version, or to settle an especially dull argument.

## Provider Architecture

The provider boundary is deliberately small:

```go
type Provider interface {
  Chat(context.Context, Request) (string, error)
}
```

`internal/app` owns commands, history, and presentation. `internal/llm` owns provider-neutral messages and requests. Provider packages own SDK translation, authentication, and transport details, while `internal/llm/factory` selects the configured implementation.

Adding another API means implementing `llm.Provider`, adding its configuration and validation, and registering it in the factory. It should not require provider-specific fields in `llm.Request` or SDK imports in the application layer. Cloud vendors already have enough surface area.

| Provider | Status | Authentication | Transport |
| --- | --- | --- | --- |
| Amazon Bedrock | Available | AWS shared config / SSO | Bedrock Converse API |
| Other APIs | Adapter welcome | Provider-native | Provider-owned |

## Data and Privacy

ama creates files with user-only permissions:

| Data | Location | Notes |
| --- | --- | --- |
| Configuration | `~/.config/ama/config.yaml` | Provider, model, adapter settings, and system directive |
| Conversation | `~/.config/ama/history.json` | Raw questions and responses for the current conversation |
| Provider credentials | Not stored by ama | Resolved by the active provider's native tooling |

The current conversation and system directive are sent to the selected provider with each request. History is plaintext on disk, so secrets remain a poor conversation topic. With Bedrock, `--verbose` writes raw AWS request and response bodies to stderr; use it only in a trusted terminal.

## Troubleshooting Bedrock

**`configuration not found`**

Run `ama config init --profile YOUR_PROFILE`.

**AWS SSO credentials are expired**

ama starts `aws sso login --profile YOUR_PROFILE` automatically and retries the request once after authentication.

**The model cannot be invoked**

Confirm that the selected model is available in the profile's region and that your AWS identity can call the Bedrock Converse API. Use `ama config show` to inspect the active provider, model, profile, and region.

## Contributing

Bug reports, focused pull requests, and new provider adapters are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

ama is available under the [MIT License](LICENSE).
