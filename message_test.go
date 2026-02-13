package main

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_Arg(t *testing.T) {
	m := MessageInput{Message: "hello"}
	got, err := m.Resolve()
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestResolve_EmptyNoStdin(t *testing.T) {
	m := MessageInput{}
	_, err := m.Resolve()
	require.Error(t, err)

	var cliErr *CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, "empty_message", cliErr.Code)
	assert.Equal(t, ExitInvalidInput, cliErr.ExitCode)
}

func TestResolve_StdinFlag(t *testing.T) {
	// Save and restore os.Stdin
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r

	_, _ = w.Write([]byte("piped message\n"))
	_ = w.Close()

	m := MessageInput{Stdin: true}
	got, err := m.Resolve()

	os.Stdin = oldStdin

	require.NoError(t, err)
	assert.Equal(t, "piped message", got) // trailing newline stripped
}

func TestResolve_ArgWithCodeFlag(t *testing.T) {
	// Code wrapping is applied in PostCmd.Run, not in Resolve.
	// Verify Resolve returns the raw message regardless of Code flag.
	m := MessageInput{Message: "test", Code: true}
	got, err := m.Resolve()
	require.NoError(t, err)
	assert.Equal(t, "test", got, "Resolve should return raw message; Code wrapping is PostCmd's job")
}
