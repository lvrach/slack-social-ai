# CLAUDE.md

CLI tool that posts messages to Slack via incoming webhooks. Built with Go, Kong (CLI framework), and huh (interactive TUI forms). Designed for use by AI coding agents.

## Commands

```bash
make build        # compile to bin/slack-social-ai
make install      # go install to $GOPATH/bin
make test         # gotestsum with -race, -shuffle, coverage
make lint         # gofumpt + goimports + golangci-lint v2
make fmt          # format only (no lint)
make vulncheck    # govulncheck for known CVEs
make sec          # gitleaks (secrets) + govulncheck
```

## Architecture

```
main.go           CLI entrypoint, Kong parser, error handling, isUserAbort
auth.go           AuthCmd — login/logout/status, webhook validation, guided setup
init.go           InitCmd — first-run wizard (auth + schedule + timer), runField/runForm helpers
post.go           PostCmd — queue-first (default), --now, --dry-run, --at, --file
queue.go          QueueCmd — show/remove subcommands, predicted publish times, messagePreview
queue_inspect.go  QueueInspectCmd — Bubble Tea TUI, horizontal split (list|detail), pre-cached glamour
mrkdwn.go         mrkdwnToMarkdown preprocessor, renderMrkdwn via glamour (cached renderer, emoji)
message.go        MessageInput — shared message resolution (arg/file/stdin/pipe)
time.go           parseAt — time parsing for --at flag (HH:MM, duration, RFC3339)
publish.go        PublishCmd — scheduler processor, time/frequency guards, --ignore-schedule
schedule.go       ScheduleCmd — set/status/install/uninstall, interactive TUI (Select/MultiSelect)
history.go        HistoryCmd — list/clear/--queued/--published/--remove/--clear-all
guide.go          GuideCmd — prints go:embed'd posting guide
output.go         CLIError type, exit codes, JSON/human output helpers

internal/
  keyring/        Webhook URL storage via macOS Keychain (go-keyring)
  slack/          SendWebhook + VerifyWebhook (silent POST {}) — HTTP to Slack webhook
  history/        JSON file at ~/.local/share/slack-social-ai/history.json, file locking via gofrs/flock
  manifest/       Generates Slack app manifest JSON for guided setup
  config/         Thin persistence layer for config (~/.config/slack-social-ai/config.json)
  schedule/       Schedule struct, IsActiveAt, PostEvery, PredictPublishTimes, AdvanceToActive
  launchd/        Plist generation, bootstrap/bootout, IsInstalled, LogPath
```

Commands are Kong subcommands. Each is a struct with a `Run(globals *Globals) error` method. `Globals` carries the `--json` flag. Kong dispatches via `ctx.Run(&cli.Globals)`.

## Adding a New Command

1. Create `mycommand.go` with a struct and `Run(globals *Globals) error` method.
2. Add the field to `CLI` in `main.go`: `MyCmd MyCmd \`cmd:"" help:"Description."\``
3. Support dual output: check `globals.JSON`, use `printSuccessJSON`/`printSuccessHuman`.
4. Return `*CLIError` via `newCLIError` for structured errors with exit codes.

## Key Patterns

**`runField`/`runForm` wrappers** (init.go): Single `huh` fields use `runField()`; multi-field forms use `runForm()`. Both provide consistent Ctrl+C/D quit bindings and styling. Never call `huh.NewForm(...).Run()` directly.

**CLIError** (output.go): Structured errors with `ExitCode` (0-3), machine-readable `Code`, and `Message`. Exit codes: 0=OK, 1=runtime, 2=not configured, 3=invalid input.

**Dual output**: Every command must respect `globals.JSON`. Success to stdout, errors to stderr.

**`go:embed` posting guide** (guide.go): `slack-social-ai.guide.md` is embedded at compile time. The `guide` subcommand prints this guide. Changes require rebuild.

**Queue-first posting**: `post` queues by default. `post --now` publishes immediately. `publish` is the scheduler processor.

**Dumb timer + smart Go code**: launchd wakes every 10 min. All scheduling logic (hours, weekdays, frequency) lives in Go code (`internal/schedule`), not in the plist.

**File locking**: `gofrs/flock` with separate `.lock` file for history.json. ClaimNextReady is atomic (lock → read → update status → unlock).

**mrkdwn rendering**: `mrkdwn.go` preprocesses Slack mrkdwn to standard Markdown (bold, strike, links, mentions) then renders via glamour. Code blocks are preserved.

**Glamour renderer caching**: `renderMrkdwn` caches the `glamour.TermRenderer` by width. `WithAutoStyle()` performs OS I/O to detect terminal theme — must not be called in TUI hot paths. Cache invalidates on width change.

**Viewport keymap override**: Default bubbles viewport `HalfPageDown` binds to `d` — override to `ctrl+d` only when `d` is used as delete key. Also disable `Left`/`Right` bindings to prevent focus confusion.

## Code Style

- **Formatter**: gofumpt with `extra-rules: true` (stricter than gofmt)
- **Imports**: goimports with `-local=github.com/lvrach` (local imports in separate group)
- **Linter**: golangci-lint v2 — standard defaults plus gosec, misspell, unconvert, unparam, wastedassign
- Always run `make lint` before committing

## Gotchas

- Webhook URL is in **macOS Keychain**, not a config file. Use `internal/keyring`.
- History append is **best-effort** — write failures must never prevent a successful post.
- History caps at **200 entries**, drops oldest. Corrupt files silently discarded.
- History writes are **atomic** (write to `.tmp`, then rename).
- Slack uses **mrkdwn**, not Markdown: `*bold*` not `**bold**`, `<url|text>` not `[text](url)`.
- `isUserAbort` intentionally avoids `errors.Is(err, io.EOF)` — EOF can come from network failures.
- `defaultAppName()` truncates to 35 chars (Slack's app name limit).
- **macOS is the primary platform**: `pbcopy` for clipboard, Keychain for secrets.
- Schedule config is at `~/.config/slack-social-ai/config.json`, separate from history data.
- `ClaimNextReady` holds a file lock while reading + updating status to prevent double-claim.
- Legacy history format (`{ts, message}`) is auto-migrated on first Load() to new format with ID/status fields.
- launchd plist at `~/Library/LaunchAgents/com.slack-social-ai.publish.plist` — installed via `schedule install` or during `init` wizard.
- `config.Exists()` checks if a config file has been saved (used to distinguish "never configured" from "defaults").
- `VerifyWebhook` POSTs `{}` — Slack returns 400 "no_text" for valid webhooks (auth is checked, just no message body).
- **gofumpt rewrites** `for i := 0; i < N; i++` to `for i := range N` (Go 1.22+ range-over-integer). Don't fight it.

## Testing

```bash
make test
```

Uses gotestsum with: `-race`, `-shuffle=on`, `-failfast`, `-covermode=atomic`, `-coverpkg=./...`. Tests exist in root package and all internal packages. Use -race flag for concurrent history tests.
