# slack-social-ai

Post messages to Slack from the terminal via incoming webhooks. Minimal, scriptable, and designed to work with AI coding agents ([Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Cursor](https://cursor.com), [OpenCode](https://opencode.ai), and others).

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

## Using with AI Coding Agents

The primary workflow for `slack-social-ai` is pairing it with an AI coding agent. The `guide` command outputs an LLM-optimized posting guide. When possible, load it directly into the agent's prompt via shell substitution `$(slack-social-ai guide)` so the agent has the full guide in context without needing to run a command.

### Non-interactive

The agent gets the guide inlined via shell substitution and posts autonomously:

```bash
# Claude Code
claude "Follow this guide: $(slack-social-ai guide). Post an insight about what we worked on"

# OpenCode
opencode "Follow this guide: $(slack-social-ai guide). Post an insight about what we worked on"
```

### Interactive

The agent proposes posts and asks you to pick before publishing:

```bash
# Claude Code
claude "Follow this guide: $(slack-social-ai guide). Show me the options, ask before you post"

# OpenCode
opencode "Follow this guide: $(slack-social-ai guide). Show me the options, ask before you post"
```

## Scheduling

By default, `post` adds messages to a queue. Use `schedule install` to set up automatic publishing:

```bash
slack-social-ai schedule install --post-every 3h --hours 9-22 --weekdays mon-fri
```

This installs a macOS launchd timer that wakes every 15 minutes and publishes the next queued message when conditions are met (within active hours, respecting the frequency limit).

```bash
slack-social-ai schedule status    # check schedule + queue depth
slack-social-ai schedule uninstall # remove the schedule
```

To bypass the queue entirely:

```bash
slack-social-ai post "urgent" --now
```

### What happens under the hood

1. `$(slack-social-ai guide)` inlines the full posting guide directly into the agent's prompt
2. The agent runs `slack-social-ai history` to check recent posts and avoid repeats
3. The agent gathers context from session history, memory, and recent skills
4. The agent drafts posts that fit the channel's voice -- concise, opinionated, technically precise
5. **Interactive**: the agent proposes options and waits for you to pick one
6. **Non-interactive**: the agent picks the best option and posts autonomously
7. The agent runs `slack-social-ai post "..."` to queue (or `--now` to publish immediately)

## CLI Reference

| Command | Description |
|---------|-------------|
| `init [<webhook-url>]` | Configure Slack webhook (interactive or direct) |
| `post [<message>]` | Queue a message for publishing (use `--now` to publish immediately) |
| `publish` | Publish the next queued message to Slack (typically run by scheduler) |
| `schedule install\|uninstall\|status` | Manage the automatic publishing schedule (macOS launchd) |
| `history [--queued\|--published\|--clear]` | Show or manage post history |
| `guide` | Print the posting guide (for LLM agents) |

```bash
# Queue a message (published automatically by the schedule)
slack-social-ai post "your insight here"

# Publish immediately
slack-social-ai post "urgent insight" --now

# Preview before posting
slack-social-ai post "draft insight" --dry-run

# Check history
slack-social-ai history

# JSON output for scripts
slack-social-ai post "deploy completed" --json
```

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--json` | `-j` | Global | JSON output for LLM/script consumption |
| `--now` | `-N` | `post` | Publish immediately, skip the queue |
| `--dry-run` | `-n` | `post` | Preview without publishing or queuing |
| `--at` | `-a` | `post` | Schedule for a future time (HH:MM, duration, RFC3339) |
| `--code` | `-c` | `post` | Wrap message in a code block |
| `--stdin` | | `post` | Force reading from stdin |
| `--queued` | | `history` | Show only queued messages |
| `--published` | | `history` | Show only published messages |
| `--remove` | | `history` | Remove a specific entry by ID |
| `--clear` | | `history` | Clear published history (keeps queue) |
| `--clear-all` | | `history` | Clear everything |
| `--post-every` | `-p` | `schedule install` | Minimum time between posts |
| `--hours` | | `schedule install` | Active hours range |
| `--weekdays` | | `schedule install` | Active weekdays |

## Coming Soon

- **Multi-channel support** -- post to different Slack channels from the same CLI
- **Block Kit formatting** -- rich message layouts with headers, dividers, and context blocks

## How It Works

- **Secrets**: Webhook URL is stored in macOS Keychain via [go-keyring](https://github.com/zalando/go-keyring)
- **Slack API**: Posts via [incoming webhooks](https://api.slack.com/messaging/webhooks) (no bot token needed)
- **CLI**: Built with [Kong](https://github.com/alecthomas/kong)
- **Interactive UI**: Powered by [huh](https://github.com/charmbracelet/huh)
- **Posting guide**: Embedded in the binary via `go:embed` -- no external files needed at runtime

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
