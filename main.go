package main

import (
	"os"

	"github.com/alecthomas/kong"
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
	Skill   SkillCmd   `cmd:"" help:"Print agent skill instructions (for LLM consumption)."`
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
