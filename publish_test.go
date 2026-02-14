package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/schedule"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			limit:    10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			limit:    5,
			expected: "hello",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "this is a very long message that should be truncated",
			limit:    20,
			expected: "this is a very lo...",
		},
		{
			name:     "truncate to minimum useful length",
			input:    "abcdef",
			limit:    4,
			expected: "a...",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			limit:    10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.limit)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPublishCmd_ExitOutsideSchedule_JSON(t *testing.T) {
	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	sched := schedule.DefaultSchedule()

	output := captureStdout(t, func() {
		retErr := cmd.exitOutsideSchedule(globals, sched)
		assert.NoError(t, retErr)
	})

	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "outside_schedule", resp["status"])
}

func TestPublishCmd_ExitOutsideSchedule_Human(t *testing.T) {
	cmd := &PublishCmd{}
	globals := &Globals{JSON: false}
	sched := schedule.DefaultSchedule()

	output := captureStdout(t, func() {
		retErr := cmd.exitOutsideSchedule(globals, sched)
		assert.NoError(t, retErr)
	})

	assert.Contains(t, output, "Skipped: outside active hours")
	assert.Contains(t, output, "09:00–17:00")
}

func TestPublishCmd_ExitNoQueued_Human(t *testing.T) {
	cmd := &PublishCmd{}
	globals := &Globals{JSON: false}

	output := captureStdout(t, func() {
		retErr := cmd.exitNoQueued(globals)
		assert.NoError(t, retErr)
	})

	assert.Contains(t, output, "Skipped: no messages queued.")
}

func TestPublishCmd_ExitTooSoon_JSON(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}

	nextEligible := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	retErr := cmd.exitTooSoon(globals, nextEligible)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf [1024]byte
	n, _ := r.Read(buf[:])
	_ = r.Close()
	output := string(buf[:n])

	assert.NoError(t, retErr)

	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "too_soon", resp["status"])
	assert.Equal(t, "2025-06-15T14:30:00Z", resp["next_eligible"])
}

// ---------------------------------------------------------------------------
// Integration tests for publishOne (the core publish flow).
// These use httptest.NewServer as a mock Slack webhook and override HOME
// so that history and config file paths resolve to a temp directory.
// ---------------------------------------------------------------------------

// alwaysActiveSchedule returns a schedule that is active at any time on any day.
func alwaysActiveSchedule() schedule.Schedule {
	return schedule.Schedule{
		PostEveryMinutes: 0,
		StartHour:        0,
		EndHour:          24,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}
}

// neverActiveSchedule returns a schedule that is never active (start == end).
func neverActiveSchedule() schedule.Schedule {
	return schedule.Schedule{
		PostEveryMinutes: 0,
		StartHour:        0,
		EndHour:          0,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}
}

// withTempHome sets HOME to a temporary directory so that history.dataDir and
// config.configDir resolve to paths inside the temp dir. It also pre-creates
// the required subdirectories.
func withTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Pre-create the data and config dirs that the packages will use.
	dataDir := filepath.Join(home, ".local", "share", "slack-social-ai")
	require.NoError(t, os.MkdirAll(dataDir, 0o700))
	configDir := filepath.Join(home, ".config", "slack-social-ai")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
}

// captureStdout redirects os.Stdout to a pipe for the duration of fn,
// then returns whatever was written to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Read from the pipe in a goroutine to avoid blocking if output exceeds
	// the pipe buffer size.
	var output string
	var wg sync.WaitGroup
	wg.Go(func() {
		data, _ := io.ReadAll(r)
		output = string(data)
	})

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	wg.Wait()
	_ = r.Close()
	return output
}

// writeHistoryEntries writes entries directly to the history JSON file in the
// temp HOME directory. This is used to set up test scenarios (e.g. stuck entries).
func writeHistoryEntries(t *testing.T, entries []history.Entry) {
	t.Helper()
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "slack-social-ai", "history.json")
	data, err := json.MarshalIndent(entries, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

// readHistoryEntries reads all entries from the history JSON file.
func readHistoryEntries(t *testing.T) []history.Entry {
	t.Helper()
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "slack-social-ai", "history.json")
	data, err := os.ReadFile(path) //nolint:gosec // test file path is controlled by test setup
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var entries []history.Entry
	require.NoError(t, json.Unmarshal(data, &entries))
	return entries
}

func TestPublish_HappyPath(t *testing.T) {
	withTempHome(t)

	// Queue a message via the history package.
	entry, err := history.Append("Hello Slack!", "queued", time.Time{})
	require.NoError(t, err)

	// Start a mock Slack webhook that returns 200.
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	cfg := config.Config{Schedule: alwaysActiveSchedule()}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne(srv.URL, cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Verify the mock server received the correct message.
	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(receivedBody), &payload))
	assert.Equal(t, "Hello Slack!", payload["text"])

	// Verify JSON output indicates success.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "Hello Slack!", resp["message"])
	assert.Equal(t, entry.ID, resp["id"])

	// Verify the entry is now marked as published.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 1)
	assert.Equal(t, "published", entries[0].Status)
	assert.NotEmpty(t, entries[0].PublishedAt)
}

func TestPublish_WebhookFail(t *testing.T) {
	withTempHome(t)

	// Queue a message.
	_, err := history.Append("Will fail to send", "queued", time.Time{})
	require.NoError(t, err)

	// Mock returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	cmd := &PublishCmd{}
	globals := &Globals{JSON: false}
	cfg := config.Config{Schedule: alwaysActiveSchedule()}

	retErr := cmd.publishOne(srv.URL, cfg, globals, false)

	// Should return an error.
	require.Error(t, retErr)
	var cliErr *CLIError
	require.True(t, asCLIError(retErr, &cliErr))
	assert.Equal(t, "webhook_failed", cliErr.Code)

	// Verify the entry was reset to queued.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 1)
	assert.Equal(t, "queued", entries[0].Status)
}

func TestPublish_OutsideHours(t *testing.T) {
	withTempHome(t)

	// Queue a message (it should NOT be claimed because schedule is inactive).
	_, err := history.Append("Should not be published", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	cfg := config.Config{Schedule: neverActiveSchedule()}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne("http://unused", cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Should output outside_schedule status.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "outside_schedule", resp["status"])

	// Entry should remain queued (nothing claimed).
	entries := readHistoryEntries(t)
	require.Len(t, entries, 1)
	assert.Equal(t, "queued", entries[0].Status)
}

func TestPublish_TooSoon(t *testing.T) {
	withTempHome(t)

	// Create an already-published entry with a recent publish time.
	recentlyPublished := history.Entry{
		ID:          "pub00001",
		Message:     "Already published",
		Status:      "published",
		CreatedAt:   time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339),
		PublishedAt: time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339),
	}
	writeHistoryEntries(t, []history.Entry{recentlyPublished})

	// Queue a second message.
	_, err := history.Append("Should not be published yet", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	// PostEvery = 180 minutes (3 hours). Last published 30 min ago, so too soon.
	cfg := config.Config{Schedule: schedule.Schedule{
		PostEveryMinutes: 180,
		StartHour:        0,
		EndHour:          24,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne("http://unused", cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Should output too_soon status.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "too_soon", resp["status"])
	assert.NotEmpty(t, resp["next_eligible"])

	// The queued entry should remain queued.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 2)
	assert.Equal(t, "queued", entries[1].Status)
}

func TestPublish_FrequencyOK(t *testing.T) {
	withTempHome(t)

	// Create an old published entry (4 hours ago).
	oldPublished := history.Entry{
		ID:          "pub00002",
		Message:     "Published long ago",
		Status:      "published",
		CreatedAt:   time.Now().Add(-4 * time.Hour).UTC().Format(time.RFC3339),
		PublishedAt: time.Now().Add(-4 * time.Hour).UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().Add(-4 * time.Hour).UTC().Format(time.RFC3339),
	}
	writeHistoryEntries(t, []history.Entry{oldPublished})

	// Queue a new message.
	_, err := history.Append("Should be published", "queued", time.Time{})
	require.NoError(t, err)

	// Mock webhook returns 200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	// PostEvery = 180 minutes (3 hours). Last published 4 hours ago, so OK.
	cfg := config.Config{Schedule: schedule.Schedule{
		PostEveryMinutes: 180,
		StartHour:        0,
		EndHour:          24,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne(srv.URL, cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Should publish successfully.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "Should be published", resp["message"])

	// Verify the entry is now published.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 2)
	assert.Equal(t, "published", entries[1].Status)
}

func TestPublish_EmptyQueue(t *testing.T) {
	withTempHome(t)

	// No entries at all.
	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	cfg := config.Config{Schedule: alwaysActiveSchedule()}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne("http://unused", cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Should output no_queued status.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "no_queued", resp["status"])
}

func TestPublish_RespectsScheduledAt(t *testing.T) {
	withTempHome(t)

	// Queue a message scheduled far in the future.
	futureTime := time.Now().Add(24 * time.Hour)
	_, err := history.Append("Future post", "queued", futureTime)
	require.NoError(t, err)

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	cfg := config.Config{Schedule: alwaysActiveSchedule()}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne("http://unused", cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// Should output no_queued because the only entry is scheduled for the future.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "no_queued", resp["status"])

	// Entry should remain queued.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 1)
	assert.Equal(t, "queued", entries[0].Status)
}

func TestPublish_RecoverStuck(t *testing.T) {
	withTempHome(t)

	// Manually create a "stuck" entry: status=publishing, updatedAt=10 minutes ago.
	stuckEntry := history.Entry{
		ID:        "stuck001",
		Message:   "Stuck message",
		Status:    "publishing",
		CreatedAt: time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
	}
	writeHistoryEntries(t, []history.Entry{stuckEntry})

	// Mock webhook returns 200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := &PublishCmd{}
	globals := &Globals{JSON: true}
	cfg := config.Config{Schedule: alwaysActiveSchedule()}

	output := captureStdout(t, func() {
		retErr := cmd.publishOne(srv.URL, cfg, globals, false)
		assert.NoError(t, retErr)
	})

	// The stuck entry should have been recovered (reset to queued) then claimed
	// and published successfully.
	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "Stuck message", resp["message"])
	assert.Equal(t, "stuck001", resp["id"])

	// Verify the entry is now published.
	entries := readHistoryEntries(t)
	require.Len(t, entries, 1)
	assert.Equal(t, "published", entries[0].Status)
}
