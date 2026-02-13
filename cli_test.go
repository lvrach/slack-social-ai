package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "slack-social-ai-test")
	if err != nil {
		panic(err)
	}
	testBinary = filepath.Join(dir, "slack-social-ai")
	cmd := exec.Command("go", "build", "-o", testBinary, ".") //nolint:gosec // test binary path is controlled by TestMain
	cmd.Dir = "/Users/leonidas/src/lvrach/slack-social-ai"
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("build failed: " + err.Error())
	}
	code := m.Run()
	_ = os.RemoveAll(dir) //nolint:gosec // best-effort cleanup
	os.Exit(code)
}

// runCLI executes the built binary with args in an isolated temp HOME directory.
// It returns stdout, stderr, and the process exit code.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	home := t.TempDir()

	cmd := exec.Command(testBinary, args...) //nolint:gosec // test binary path controlled by test setup
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
	)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// --- guide command ---

func TestCLI_Guide(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "guide")

	assert.Equal(t, 0, exitCode, "guide should exit 0")
	assert.NotEmpty(t, stdout, "guide output should not be empty")
	assert.Contains(t, stdout, "slack-social-ai", "guide should mention the tool name")
}

// --- history command ---

func TestCLI_HistoryEmpty(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history")

	assert.Equal(t, 0, exitCode, "history should exit 0 with no entries")
	assert.Contains(t, stdout, "No history", "empty history should say 'No history'")
}

func TestCLI_HistoryEmptyJSON(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history", "--json")

	assert.Equal(t, 0, exitCode, "history --json should exit 0")

	var entries []json.RawMessage
	err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries)
	require.NoError(t, err, "stdout should be valid JSON array")
	assert.Empty(t, entries, "empty history should return empty JSON array")
}

func TestCLI_HistoryQueuedEmpty(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history", "--queued")

	assert.Equal(t, 0, exitCode, "history --queued should exit 0")
	assert.Contains(t, stdout, "No history", "empty queued history should say 'No history'")
}

func TestCLI_HistoryPublishedEmpty(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history", "--published")

	assert.Equal(t, 0, exitCode, "history --published should exit 0")
	assert.Contains(t, stdout, "No history", "empty published history should say 'No history'")
}

func TestCLI_HistoryClearAllEmpty(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history", "--clear-all")

	assert.Equal(t, 0, exitCode, "history --clear-all should exit 0")
	assert.Contains(t, stdout, "cleared", "clear-all should confirm clearing")
}

func TestCLI_HistoryClearAllEmptyJSON(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "history", "--clear-all", "--json")

	assert.Equal(t, 0, exitCode, "history --clear-all --json should exit 0")

	var resp map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &resp)
	require.NoError(t, err, "stdout should be valid JSON")
	assert.Equal(t, "ok", resp["status"], "clear-all JSON should have status 'ok'")
}

// --- schedule status command ---

func TestCLI_ScheduleStatusNotConfigured(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "schedule", "status")

	assert.Equal(t, 0, exitCode, "schedule status should exit 0")
	assert.Contains(t, stdout, "Not configured", "should indicate schedule is not configured")
}

func TestCLI_ScheduleStatusNotConfiguredJSON(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "schedule", "status", "--json")

	assert.Equal(t, 0, exitCode, "schedule status --json should exit 0")

	var resp map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &resp)
	require.NoError(t, err, "stdout should be valid JSON")
	assert.Equal(t, "not_configured", resp["status"], "JSON status should be 'not_configured'")
}

// --- post command (not configured -- keyring is empty in temp HOME) ---

func TestCLI_PostNotConfigured(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "post", "test message")

	assert.Equal(t, ExitNotConfigured, exitCode, "post without webhook should exit with ExitNotConfigured")
	assert.Contains(t, stderr, "init", "error should mention running init")
}

func TestCLI_PostNotConfiguredJSON(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "post", "test message", "--json")

	assert.Equal(t, ExitNotConfigured, exitCode, "post --json without webhook should exit with ExitNotConfigured")

	var resp map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &resp)
	require.NoError(t, err, "stderr should be valid JSON error")
	assert.Equal(t, "error", resp["status"], "JSON status should be 'error'")
	assert.Equal(t, "not_configured", resp["error"], "JSON error code should be 'not_configured'")
}

func TestCLI_PostDryRunNotConfigured(t *testing.T) {
	// Even --dry-run needs the webhook URL check to pass first.
	_, stderr, exitCode := runCLI(t, "post", "test message", "--dry-run")

	assert.Equal(t, ExitNotConfigured, exitCode, "post --dry-run without webhook should exit with ExitNotConfigured")
	assert.Contains(t, stderr, "init", "error should mention running init")
}

// --- publish command (not configured) ---

func TestCLI_PublishNotConfigured(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "publish")

	assert.Equal(t, ExitNotConfigured, exitCode, "publish without webhook should exit with ExitNotConfigured")
	assert.Contains(t, stderr, "init", "error should mention running init")
}

func TestCLI_PublishNotConfiguredJSON(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "publish", "--json")

	assert.Equal(t, ExitNotConfigured, exitCode, "publish --json without webhook should exit with ExitNotConfigured")

	var resp map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &resp)
	require.NoError(t, err, "stderr should be valid JSON error")
	assert.Equal(t, "error", resp["status"], "JSON status should be 'error'")
}

// --- post command: flag mutual exclusion ---

func TestCLI_PostMutualExclusion_NowAndDryRun(t *testing.T) {
	_, _, exitCode := runCLI(t, "post", "test", "--now", "--dry-run")

	assert.NotEqual(t, 0, exitCode, "post --now --dry-run should fail due to xor constraint")
}

func TestCLI_PostMutualExclusion_NowAndAt(t *testing.T) {
	_, _, exitCode := runCLI(t, "post", "test", "--now", "--at", "14:00")

	assert.NotEqual(t, 0, exitCode, "post --now --at should fail due to xor constraint")
}

func TestCLI_PostMutualExclusion_DryRunAndAt(t *testing.T) {
	_, _, exitCode := runCLI(t, "post", "test", "--dry-run", "--at", "14:00")

	assert.NotEqual(t, 0, exitCode, "post --dry-run --at should fail due to xor constraint")
}

// --- post command: no message provided ---

func TestCLI_PostNoMessage(t *testing.T) {
	_, _, exitCode := runCLI(t, "post")

	// Without a message and without piped stdin, this should fail.
	// The exact exit code depends on whether keyring check or message
	// resolution fails first. With temp HOME, keyring fails first (exit 2).
	assert.NotEqual(t, 0, exitCode, "post with no message should fail")
}

// --- no arguments (should show help) ---

func TestCLI_NoArgs(t *testing.T) {
	_, stderr, exitCode := runCLI(t)

	assert.NotEqual(t, 0, exitCode, "running with no args should fail")
	// Kong prints an error listing available commands.
	assert.Contains(t, stderr, "expected one of", "should list available commands")
}

// --- help flag ---

func TestCLI_Help(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--help")

	assert.Equal(t, 0, exitCode, "--help should exit 0")
	assert.Contains(t, stdout, "post", "help should mention the post command")
	assert.Contains(t, stdout, "publish", "help should mention the publish command")
	assert.Contains(t, stdout, "schedule", "help should mention the schedule command")
	assert.Contains(t, stdout, "history", "help should mention the history command")
	assert.Contains(t, stdout, "guide", "help should mention the guide command")
	assert.Contains(t, stdout, "init", "help should mention the init command")
}
