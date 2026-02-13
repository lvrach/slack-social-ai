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
init.go           InitCmd — interactive webhook setup, runField helper, URL validation
post.go           PostCmd — message resolution (arg/stdin/pipe), send + history append
history.go        HistoryCmd — list/clear post history
skill.go          SkillCmd — prints go:embed'd skill guide
output.go         CLIError type, exit codes, JSON/human output helpers

internal/
  keyring/        Webhook URL storage via macOS Keychain (go-keyring)
  slack/          SendWebhook — HTTP POST to Slack incoming webhook
  history/        JSON file at ~/.local/share/slack-social-ai/history.json
  manifest/       Generates Slack app manifest JSON for guided setup
```

Commands are Kong subcommands. Each is a struct with a `Run(globals *Globals) error` method. `Globals` carries the `--json` flag. Kong dispatches via `ctx.Run(&cli.Globals)`.

## Adding a New Command

1. Create `mycommand.go` with a struct and `Run(globals *Globals) error` method.
2. Add the field to `CLI` in `main.go`: `MyCmd MyCmd \`cmd:"" help:"Description."\``
3. Support dual output: check `globals.JSON`, use `printSuccessJSON`/`printSuccessHuman`.
4. Return `*CLIError` via `newCLIError` for structured errors with exit codes.

## Key Patterns

**`runField` wrapper** (init.go): All `huh` form fields must use `runField()` for consistent Ctrl+C/D quit bindings and styling. Never call `huh.NewForm(...).Run()` directly.

**CLIError** (output.go): Structured errors with `ExitCode` (0-3), machine-readable `Code`, and `Message`. Exit codes: 0=OK, 1=runtime, 2=not configured, 3=invalid input.

**Dual output**: Every command must respect `globals.JSON`. Success to stdout, errors to stderr.

**`go:embed` skill guide** (skill.go): `slack-social-ai.skill.md` is embedded at compile time. Changes require rebuild.

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

## Testing

```bash
make test
```

Uses gotestsum with: `-race`, `-shuffle=on`, `-failfast`, `-covermode=atomic`, `-coverpkg=./...`. No test files exist yet.
