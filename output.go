package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Exit codes.
const (
	ExitOK            = 0
	ExitRuntimeError  = 1
	ExitNotConfigured = 2
	ExitInvalidInput  = 3
)

// CLIError is a structured error with an exit code and machine-readable code.
type CLIError struct {
	ExitCode int
	Code     string
	Message  string
}

func (e *CLIError) Error() string { return e.Message }

// asCLIError unwraps err into a *CLIError.
func asCLIError(err error, target **CLIError) bool {
	return errors.As(err, target)
}

// newCLIError creates a new CLIError.
func newCLIError(exitCode int, code, message string) *CLIError {
	return &CLIError{ExitCode: exitCode, Code: code, Message: message}
}

// JSON response types.
type jsonResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func printSuccessJSON(message string) {
	resp := jsonResponse{Status: "ok", Message: message}
	b, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stdout, string(b))
}

func printErrorJSON(message, code string) {
	resp := jsonResponse{Status: "error", Error: code, Message: message}
	b, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stderr, string(b))
}

func printSuccessHuman(message string) {
	fmt.Fprintln(os.Stdout, message)
}

func printErrorHuman(message string) {
	fmt.Fprintln(os.Stderr, "Error: "+message)
}
