package main

import (
	"errors"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/huh"
)

// Globals holds flags shared across all commands.
type Globals struct {
	JSON bool `help:"Output JSON for LLM/script consumption." short:"j"`
}

// CLI is the root command structure for slack-social-ai.
type CLI struct {
	Globals

	Init    InitCmd    `cmd:"" help:"Configure Slack webhook (interactive setup)."`
	Post    PostCmd    `cmd:"" help:"Post a message to Slack."`
	History HistoryCmd `cmd:"" help:"Show or manage post history."`
	Guide   GuideCmd   `cmd:"" help:"Print the posting guide — designed for LLM agents to learn how to compose posts."`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("slack-social-ai"),
		kong.Description("Post messages to Slack from the terminal."),
		kong.UsageOnError(),
	)
	err := ctx.Run(&cli.Globals)
	if err != nil {
		// Ctrl+C / Ctrl+D — exit silently.
		if isUserAbort(err) {
			os.Exit(0)
		}

		var cliErr *CLIError
		if ok := asCLIError(err, &cliErr); ok {
			if cli.JSON {
				printErrorJSON(cliErr.Message, cliErr.Code)
			} else {
				printErrorHuman(cliErr.Message)
			}
			os.Exit(cliErr.ExitCode)
		}
		if cli.JSON {
			printErrorJSON(err.Error(), "runtime_error")
		} else {
			printErrorHuman(err.Error())
		}
		os.Exit(1)
	}
}

// isUserAbort returns true for errors caused by the user
// quitting an interactive prompt (Ctrl+C, Ctrl+D).
// It intentionally does NOT match io.EOF via errors.Is because
// EOF can originate from network failures (e.g. "send webhook: EOF"),
// which must surface as errors rather than silent exit 0.
func isUserAbort(err error) bool {
	if errors.Is(err, huh.ErrUserAborted) {
		return true
	}
	// huh wraps bubbletea errors as "huh: <err>"
	if strings.Contains(err.Error(), "user aborted") {
		return true
	}
	return false
}
