# slack-social: Post Updates to Slack

## Overview

Use `slack-social` CLI to post status updates, summaries, and notifications to Slack from within agent workflows.

## Prerequisites

- `slack-social` must be installed: `go install github.com/lvrach/slack-social@latest`
- Webhook must be configured: `slack-social init`

## Quick Reference

```bash
# Post a simple message
slack-social post "Deploy completed successfully"

# Post with JSON output (for parsing results)
slack-social post "Build passed" --json

# Pipe command output as a code block
git log --oneline -5 | slack-social post --code

# Post multi-line content
echo "Build: passed\nTests: 42/42\nCoverage: 87%" | slack-social post
```

## Collecting Data for Slack Updates

When asked to post updates to Slack, follow this pattern:

### 1. Gather context

Collect relevant data using available tools before composing the message:

- **Git status**: branch name, recent commits, diff stats
- **Build results**: success/failure, duration, output
- **Test results**: pass/fail counts, coverage percentage
- **Deploy info**: environment, version, timestamp

### 2. Compose a concise message

Slack messages should be scannable. Use this format:

```
*[Category]* — one-line summary

Key details:
• detail 1
• detail 2

_timestamp or context_
```

### 3. Post with appropriate flags

| Scenario | Command |
|----------|---------|
| Status update | `slack-social post "message"` |
| Command output | `command \| slack-social post --code` |
| Script integration | `slack-social post "message" --json` |

## Common Patterns

### Post git summary after commits
```bash
SUMMARY=$(git log --oneline -1)
slack-social post "*Commit* — $SUMMARY"
```

### Post test results
```bash
RESULT=$(make test 2>&1 | tail -5)
echo "$RESULT" | slack-social post --code
```

### Post deploy notification
```bash
slack-social post "*Deploy* — $(git describe --tags) deployed to production"
```

## Exit Codes

| Code | Meaning | Agent action |
|------|---------|--------------|
| 0 | Sent | Continue workflow |
| 1 | Network/API error | Retry once, then warn user |
| 2 | Not configured | Tell user to run `slack-social init` |
| 3 | Empty message | Check data collection step |

## JSON Output

When using `--json`, parse the response:

```json
{"status":"ok","message":"Message posted to Slack."}
{"status":"error","error":"not_configured","message":"Not configured. Run \"slack-social init\" first."}
```
