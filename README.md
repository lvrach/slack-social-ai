# slack-social

Post messages to Slack from the terminal via incoming webhooks. Minimal, scriptable, LLM-friendly.

## Install

```bash
go install github.com/lvrach/slack-social@latest
```

Or build from source:

```bash
git clone https://github.com/lvrach/slack-social.git
cd slack-social
make build
# binary at ./bin/slack-social
```

## Setup

Run the interactive setup to configure your Slack webhook:

```bash
slack-social init
```

This walks you through creating a Slack app, generating a manifest, and storing the webhook URL in your macOS Keychain.

If you already have a webhook URL:

```bash
slack-social init "https://hooks.slack.com/services/T.../B.../xxx"
```

## Usage

```bash
# Post a message
slack-social post "deploy completed"

# Pipe from stdin
echo "hello from CI" | slack-social post

# Wrap in a code block
git diff --stat | slack-social post --code

# JSON output (for scripts and LLMs)
slack-social post "build done" --json
```

### Commands

| Command | Description |
|---------|-------------|
| `init [<webhook-url>]` | Configure Slack webhook (interactive or direct) |
| `post [<message>]` | Post a message to Slack |

### Flags

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--json` | `-j` | Global | JSON output for LLM/script consumption |
| `--code` | `-c` | `post` | Wrap message in a code block |
| `--stdin` | | `post` | Force reading from stdin |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error (network, Slack API, keychain) |
| 2 | Not configured (run `init` first) |
| 3 | Invalid input (empty message, bad args) |

## JSON Output

When using `--json`, output is machine-readable:

```json
{"status":"ok","message":"Message posted to Slack."}
```

```json
{"status":"error","error":"not_configured","message":"Not configured. Run \"slack-social init\" first."}
```

## How It Works

- **Secrets**: Webhook URL is stored in macOS Keychain via [go-keyring](https://github.com/zalando/go-keyring)
- **Slack API**: Posts via [incoming webhooks](https://api.slack.com/messaging/webhooks) (no bot token needed)
- **CLI**: Built with [Kong](https://github.com/alecthomas/kong)
- **Interactive UI**: Powered by [huh](https://github.com/charmbracelet/huh)

## Development

```bash
make help       # show all targets
make build      # compile to bin/slack-social
make lint       # format + lint
make test       # run tests
make vulncheck  # check for vulnerabilities
```

## License

MIT
