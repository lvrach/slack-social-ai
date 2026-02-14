package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// MessageInput provides shared message resolution (arg, file, stdin, pipe).
// Embedded in PostCmd and potentially other commands.
type MessageInput struct {
	Message string `arg:"" optional:"" help:"Message text."`
	Code    bool   `help:"Wrap message in a code block." short:"c"`
	File    string `help:"Read message from a file." short:"F" type:"existingfile"`
	Stdin   bool   `help:"Force reading message from stdin."`
}

// Resolve returns the message text, checking arg -> file -> stdin flag -> piped stdin.
func (m *MessageInput) Resolve() (string, error) {
	// 1. Positional argument.
	if m.Message != "" {
		return m.Message, nil
	}

	// 2. --file flag.
	if m.File != "" {
		return readFile(m.File)
	}

	// 3. --stdin flag.
	if m.Stdin {
		return readStdin()
	}

	// 4. Detect piped stdin (not a terminal).
	fi, err := os.Stdin.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return readStdin()
	}

	// 5. No message provided.
	return "", newCLIError(ExitInvalidInput, "empty_message",
		"No message provided. Pass a message as an argument, --file, or pipe via stdin.")
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-provided path via CLI flag
	if err != nil {
		return "", newCLIError(ExitRuntimeError, "read_file_failed",
			fmt.Sprintf("Failed to read file %q: %s", path, err))
	}
	msg := strings.TrimRight(string(data), "\n")
	if msg == "" {
		return "", newCLIError(ExitInvalidInput, "empty_message",
			fmt.Sprintf("File %q is empty.", path))
	}
	return msg, nil
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
