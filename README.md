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

Run the interactive setup to configure your Slack webhook, schedule, and background timer:

```bash
slack-social-ai init
```

This walks you through creating a Slack app, generating a manifest, storing the webhook URL in your macOS Keychain, and optionally enabling automatic publishing.

If you already have a webhook URL:

```bash
slack-social-ai auth login "https://hooks.slack.com/services/T.../B.../xxx"
```

## Using with AI Coding Agents

The primary workflow for `slack-social-ai` is pairing it with an AI coding agent. Both modes below use the queue + schedule system: posts queue up, and the scheduler publishes them during your active hours.

The `guide` command outputs an LLM-optimized posting guide. Load it directly into the agent's prompt via shell substitution `$(slack-social-ai guide)` so the agent has the full guide in context without needing to run a command.

### Human-in-the-loop (recommended starting point)

You run an interactive session — the agent drafts options, you pick which to queue, and the scheduler publishes them gradually across the day. Best way to start: refine the agent's sense of your voice before going fully autonomous.

```bash
# Claude Code
claude "Follow this guide: $(slack-social-ai guide). Draft 3 options, let me pick."

# OpenCode
opencode "Follow this guide: $(slack-social-ai guide). Draft 3 options, let me pick."
```

### Autonomous / fully agentic

A cron job or daily session calls the agent. The agent creates a batch of posts based on recent work. Posts queue up, and the scheduler spreads them across the day.

```bash
# Claude Code (non-interactive)
claude -p "Follow this guide: $(slack-social-ai guide). Create 2-3 posts and queue them."

# OpenCode
opencode "Follow this guide: $(slack-social-ai guide). Create 2-3 posts and queue them."
```

> **Tip**: Start with human-in-the-loop to build posting history. Once happy with quality, graduate to autonomous.

## Scheduling

By default, `post` adds messages to a queue. Configure when they get published:

```bash
slack-social-ai schedule set    # interactive setup
slack-social-ai schedule status # check schedule + queue depth
```

To bypass the queue entirely:

```bash
slack-social-ai post "urgent" --now
```

## Automatic Publishing

Enable automatic publishing with the background timer:

```bash
slack-social-ai schedule install   # install macOS launchd timer
```

This installs a macOS launchd timer at:

    ~/Library/LaunchAgents/com.slack-social-ai.publish.plist

The timer wakes every 10 minutes and runs `slack-social-ai publish --json`.
All scheduling logic (hours, weekdays, frequency) is handled by Go code —
launchd is just a dumb timer.

### Logs

    tail -f ~/.local/share/slack-social-ai/publish.log

### Remove the timer

```bash
slack-social-ai schedule uninstall
```

### What happens under the hood

1. `$(slack-social-ai guide)` inlines the full posting guide directly into the agent's prompt
2. The agent runs `slack-social-ai history` to check recent posts and avoid repeats
3. The agent gathers context from session history, memory, and recent work
4. The agent drafts posts that fit the channel's voice — concise, opinionated, technically precise
5. **Human-in-the-loop**: the agent proposes options and waits for you to pick
6. **Autonomous**: the agent picks the best option and queues autonomously
7. The agent runs `slack-social-ai post "..."` to queue the message
8. The publish scheduler sends queued posts during active hours

## CLI Reference

```bash
# Setup
slack-social-ai init                   # first-run wizard (auth + schedule + timer)
slack-social-ai auth login             # configure webhook (interactive or URL argument)
slack-social-ai auth status            # check webhook configuration
slack-social-ai auth status --verify   # silently verify webhook (no message sent)
slack-social-ai auth logout            # remove webhook credentials

# Posting
slack-social-ai post "message"         # queue a message
slack-social-ai post "urgent" --now    # publish immediately
slack-social-ai post "draft" -n        # dry-run preview
slack-social-ai post "later" --at 2h   # schedule for a future time

# Queue management
slack-social-ai queue                  # show queue with predicted publish times
slack-social-ai queue inspect          # interactive queue browser with detail pane
slack-social-ai queue remove <id>      # remove a queued message

# Publishing
slack-social-ai publish                # publish next queued message (scheduler)

# Schedule
slack-social-ai schedule set           # configure schedule (interactive)
slack-social-ai schedule status        # show schedule + timer status + queue depth
slack-social-ai schedule install       # install background timer
slack-social-ai schedule uninstall     # remove background timer

# Other
slack-social-ai history                # show post history
slack-social-ai guide                  # print the posting guide (for LLM agents)
```

Use `--help` on any command for full flag details (e.g. `slack-social-ai post --help`).

All commands support `--json` / `-j` for machine-readable output.

## Coming Soon

- **Multi-channel support** -- post to different Slack channels from the same CLI
- **Block Kit formatting** -- rich message layouts with headers, dividers, and context blocks

## How It Works

- **Secrets**: Webhook URL is stored in macOS Keychain via [go-keyring](https://github.com/zalando/go-keyring)
- **Slack API**: Posts via [incoming webhooks](https://api.slack.com/messaging/webhooks) (no bot token needed)
- **CLI**: Built with [Kong](https://github.com/alecthomas/kong)
- **Interactive UI**: Powered by [huh](https://github.com/charmbracelet/huh) and [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Posting guide**: Embedded in the binary via `go:embed` -- no external files needed at runtime
- **mrkdwn rendering**: Queue inspect detail pane renders Slack mrkdwn via [glamour](https://github.com/charmbracelet/glamour)

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
