package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lvrach/slack-social/internal/keyring"
	"github.com/lvrach/slack-social/internal/slack"
)

// PostCmd posts a message to Slack.
type PostCmd struct {
	Message string `arg:"" optional:"" help:"Message text to post."`
	Code    bool   `help:"Wrap message in a code block." short:"c"`
	Stdin   bool   `help:"Force reading message from stdin."`
}

func (cmd *PostCmd) Run(globals *Globals) error {
	webhookURL, err := keyring.Get()
	if err != nil {
		if keyring.IsNotFound(err) {
			return newCLIError(ExitNotConfigured, "not_configured",
				"Not configured. Run \"slack-social init\" first.")
		}
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to read keychain: %s", err))
	}

	message, err := cmd.resolveMessage()
	if err != nil {
		return err
	}

	if cmd.Code {
		message = "```\n" + message + "\n```"
	}

	if err := slack.SendWebhook(webhookURL, message); err != nil {
		return newCLIError(ExitRuntimeError, "send_failed",
			fmt.Sprintf("Failed to post message: %s", err))
	}

	if globals.JSON {
		printSuccessJSON("Message posted to Slack.")
	} else {
		printSuccessHuman("Message posted to Slack.")
	}
	return nil
}

func (cmd *PostCmd) resolveMessage() (string, error) {
	// 1. Positional argument.
	if cmd.Message != "" {
		return cmd.Message, nil
	}

	// 2. --stdin flag.
	if cmd.Stdin {
		return readStdin()
	}

	// 3. Detect piped stdin (not a terminal).
	fi, err := os.Stdin.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return readStdin()
	}

	// 4. No message provided.
	return "", newCLIError(ExitInvalidInput, "empty_message",
		"No message provided. Pass a message as an argument or pipe via stdin.")
}

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	msg := strings.TrimRight(string(data), "\n")
	if msg == "" {
		return "", newCLIError(ExitInvalidInput, "empty_message",
			"No message provided (stdin was empty).")
	}
	return msg, nil
}
