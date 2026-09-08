# Security Policy

## Supported Versions

Security fixes are applied to the latest released version of ama.

## Reporting a Vulnerability

Please report vulnerabilities privately through [GitHub Security Advisories](https://github.com/svalentino/ama/security/advisories/new). If private reporting is unavailable, contact [@svalentino](https://github.com/svalentino) privately before disclosing the issue.

Do not open a public issue for a suspected vulnerability. Do not include AWS credentials, access tokens, private prompts, conversation history, or raw `--verbose` output in a report.

Please include the affected ama version, operating system, impact, and minimal reproduction steps. You can expect an acknowledgement within seven days.

## Sensitive Local Data

ama stores its configuration and conversation history as user-readable files under `~/.config/ama`. AWS credentials are not stored by ama. The `--verbose` option can print full request and response bodies to stderr; treat that output as sensitive.
