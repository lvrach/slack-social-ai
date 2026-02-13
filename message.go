package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// MessageInput provides shared message resolution (arg, stdin, pipe).
// Embedded in PostCmd and potentially other commands.
type MessageInput struct {
	Message string `arg:"" optional:"" help:"Message text."`
	Code    bool   `help:"Wrap message in a code block." short:"c"`
	Stdin   bool   `help:"Force reading message from stdin."`
}

// Resolve returns the message text, checking arg -> stdin flag -> piped stdin.
func (m *MessageInput) Resolve() (string, error) {
	// 1. Positional argument.
	if m.Message != "" {
		return m.Message, nil
	}

	// 2. --stdin flag.
	if m.Stdin {
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
