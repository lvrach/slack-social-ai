# slack-social-ai

Post messages to Slack from the terminal via incoming webhooks. Minimal, scriptable, and designed to work with AI coding agents like [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

## Install

Requires Go 1.25+.

```bash
go install github.com/lvrach/slack-social-ai@latest
```

Or build from source:

```bash
git clone https://github.com/lvrach/slack-social-ai.git
cd slack-social-ai
make build
# binary at ./bin/slack-social-ai
```

> **Note:** macOS is the primary supported platform. Webhook credentials are stored securely in the macOS Keychain.

## Setup

Run the interactive setup to configure your Slack webhook:

```bash
slack-social-ai init
```

This walks you through creating a Slack app, generating a manifest, and storing the webhook URL in your macOS Keychain.

If you already have a webhook URL:

```bash
slack-social-ai init "https://hooks.slack.com/services/T.../B.../xxx"
```

## Usage

```bash
# Post a message
slack-social-ai post "deploy completed"

# Pipe from stdin
echo "hello from CI" | slack-social-ai post

# Wrap in a code block
git diff --stat | slack-social-ai post --code

# JSON output (for scripts and LLMs)
slack-social-ai post "build done" --json
```

### Commands

| Command | Description |
|---------|-------------|
| `init [<webhook-url>]` | Configure Slack webhook (interactive or direct) |
| `post [<message>]` | Post a message to Slack |
| `history [--clear]` | Show or manage post history |
| `skill` | Print agent skill guide (for LLM consumption) |

### Flags

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--json` | `-j` | Global | JSON output for LLM/script consumption |
| `--code` | `-c` | `post` | Wrap message in a code block |
| `--stdin` | | `post` | Force reading from stdin |
| `--clear` | | `history` | Clear all history |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error (network, Slack API, keychain) |
| 2 | Not configured (run `init` first) |
| 3 | Invalid input (empty message, bad args) |

## Agent Skill

The `skill` command prints an embedded guide that teaches AI agents how to compose high-quality engineering microblog posts:

```bash
slack-social-ai skill
```

The guide covers tone, post structure, Slack mrkdwn formatting (which differs from standard Markdown), topic and mood rotation, and concrete good/bad examples. It is embedded directly in the binary via `go:embed`, so it works offline with no network calls.

This is the bridge between the CLI and AI agent workflows -- the agent reads the guide, learns the conventions, and then uses the `post` and `history` commands to compose and publish.

## Using with Claude Code

The primary workflow for `slack-social-ai` is pairing it with an AI coding agent. Here is how to set it up with [Claude Code](https://docs.anthropic.com/en/docs/claude-code):

### Quick start

1. Install the tool and configure your webhook:

```bash
go install github.com/lvrach/slack-social-ai@latest
slack-social-ai init
```

2. In a Claude Code session, ask Claude to read the skill guide and post:

```
You: Run `slack-social-ai skill` to learn how to post, then share
     an insight about the refactoring we just did

Claude: [reads skill guide, checks history with `slack-social-ai history`,
         composes a post following the guide's conventions,
         runs `slack-social-ai post "..."`]
```

That's it. Claude reads the skill output, which teaches it the tone, structure, Slack mrkdwn formatting rules, and variety guidelines. It then checks recent history to avoid repeats and rotates topics and moods automatically.

### What happens under the hood

1. `slack-social-ai skill` prints the full LLM-optimized guide to stdout
2. Claude reads the guide and learns the posting conventions
3. Claude runs `slack-social-ai history` to see recent posts and avoid duplicate content or patterns
4. Claude composes a post that fits the channel's voice -- concise, opinionated, technically precise
5. Claude runs `slack-social-ai post "..."` to publish

### Making it easier to invoke

You can reduce the setup to a single prompt by adding the skill guide to your project's configuration:

- **CLAUDE.md**: Add `slack-social-ai skill` output (or a reference to run it) in your project's `CLAUDE.md` file so Claude always knows how to post.
- **Custom slash command**: Register `slack-social-ai skill` as a custom slash command in Claude Code for quick access during any session.

### Scripting and CI/CD

The `--json` flag makes every command machine-readable, so you can integrate posting into automated workflows:

```bash
slack-social-ai post "deploy to prod completed" --json
# {"status":"ok","message":"Message posted to Slack."}
```

```bash
slack-social-ai history --json
```

## JSON Output

When using `--json`, output is machine-readable:

```json
{"status":"ok","message":"Message posted to Slack."}
```

```json
{"status":"error","error":"not_configured","message":"Not configured. Run \"slack-social-ai init\" first."}
```

## How It Works

- **Secrets**: Webhook URL is stored in macOS Keychain via [go-keyring](https://github.com/zalando/go-keyring)
- **Slack API**: Posts via [incoming webhooks](https://api.slack.com/messaging/webhooks) (no bot token needed)
- **CLI**: Built with [Kong](https://github.com/alecthomas/kong)
- **Interactive UI**: Powered by [huh](https://github.com/charmbracelet/huh)
- **Skill guide**: Embedded in the binary via `go:embed` -- no external files needed at runtime

## Development

```bash
make help       # show all targets
make build      # compile to bin/slack-social-ai
make lint       # format + lint
make test       # run tests
make vulncheck  # check for vulnerabilities
```

## License

MIT
