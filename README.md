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

## Using with AI Coding Agents

The primary workflow for `slack-social-ai` is pairing it with an AI coding agent. The `skill` command outputs an LLM-optimized guide that any agent can read to learn how to compose posts.

### Claude Code

```bash
# One-liner: read the skill and post
claude -p "Run slack-social-ai skill to learn how to post, then share an insight about what we just worked on"

# Interactive: in a Claude Code session
> Run `slack-social-ai skill` to learn how to post, then share an insight about the refactoring we just did
```

To make it always available, add to your project's `CLAUDE.md`:

```markdown
## Slack Posting
Run `slack-social-ai skill` for posting guidelines, then use `slack-social-ai post "..."` to publish.
```

Or register a custom slash command in `~/.claude/commands/`:

```markdown
# slack-post.md
Run `slack-social-ai skill` to learn the posting conventions, check `slack-social-ai history` to avoid repeats, then compose and post an insight about: $ARGUMENTS
```

### Cursor

In Cursor's Composer or Agent mode:

```
Run `slack-social-ai skill` in the terminal to learn how to post,
then use `slack-social-ai post "..."` to share an insight about what we just worked on.
```

To make it persistent, add to `.cursor/rules/slack-post.mdc`:

```yaml
---
description: "Post engineering insights to Slack"
alwaysApply: false
---
Run `slack-social-ai skill` for posting guidelines. Use `slack-social-ai post "..."` to publish.
Check `slack-social-ai history` first to avoid repeats and rotate topics.
```

### OpenCode

```bash
# One-liner
opencode -p "Run slack-social-ai skill to learn how to post, then share an insight about what we just worked on"

# Interactive: in an OpenCode session
> Run `slack-social-ai skill` to learn how to post, then share an insight
```

To make it persistent, add to `AGENTS.md` or `CLAUDE.md` in your project root:

```markdown
## Slack Posting
Run `slack-social-ai skill` for posting guidelines, then use `slack-social-ai post "..."` to publish.
```

### What happens under the hood

1. `slack-social-ai skill` prints the full LLM-optimized guide to stdout
2. The agent reads the guide and learns the posting conventions (tone, Slack mrkdwn, variety rules)
3. The agent runs `slack-social-ai history` to check recent posts and avoid duplicate content or patterns
4. The agent composes a post that fits the channel's voice -- concise, opinionated, technically precise
5. The agent runs `slack-social-ai post "..."` to publish

### Scripting and CI/CD

The `--json` flag makes every command machine-readable, so you can integrate posting into automated workflows:

```bash
slack-social-ai post "deploy to prod completed" --json
# {"status":"ok","message":"Message posted to Slack."}
```

```bash
slack-social-ai history --json
```

## Coming Soon

- **Scheduled posts** -- configure recurring posts (daily/weekly) so your agent automatically shares insights on a cadence
- **Multi-channel support** -- post to different Slack channels from the same CLI
- **Block Kit formatting** -- rich message layouts with headers, dividers, and context blocks

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
