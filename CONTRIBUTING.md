# Contributing to ama

Thanks for helping improve ama. Focused bug reports, documentation fixes, and small pull requests are welcome.

## Before You Start

- Search the existing issues before opening a new one.
- Discuss substantial features or behavior changes in an issue first.
- Never include AWS credentials, tokens, private prompts, conversation history, or raw `--verbose` output in an issue.

## Development Setup

You need Go 1.24 or later and [Task](https://taskfile.dev/installation/).

```sh
git clone https://github.com/svalentino/ama.git
cd ama
go mod download
task check
task build
```

The test suite uses fake providers and must not require live AWS access or an SSO login.

## Making Changes

- Keep changes focused and preserve existing command behavior unless the change is intentional.
- Add or update tests for behavior changes.
- Use standard Go formatting and return errors with useful context.
- Keep provider-specific behavior out of the application command layer.
- Update the README when user-facing commands, configuration, or requirements change.

Before opening a pull request, run:

```sh
task go.fmt
task check
task build
```

## Pull Requests

Describe the problem, the chosen solution, and how you tested it. Keep unrelated refactors in separate pull requests so each change is easy to review.
