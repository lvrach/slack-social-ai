package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lvrach/slack-social-ai/internal/history"
)

func TestQueueShow_Empty(t *testing.T) {
	withTempHome(t)

	cmd := &QueueShowCmd{}
	globals := &Globals{JSON: false}

	output := captureStdout(t, func() {
		err := cmd.Run(globals)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Queue is empty")
}

func TestQueueShow_WithEntries_Human(t *testing.T) {
	withTempHome(t)

	_, err := history.Append("First post about AI tools", "queued", time.Time{})
	require.NoError(t, err)
	_, err = history.Append("Second post about CLI design", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &QueueShowCmd{}
	globals := &Globals{JSON: false}

	output := captureStdout(t, func() {
		retErr := cmd.Run(globals)
		assert.NoError(t, retErr)
	})

	assert.Contains(t, output, "Queue (2 messages)")
	assert.Contains(t, output, "First post about AI tools")
	assert.Contains(t, output, "Second post about CLI design")
	assert.Contains(t, output, "#")
	assert.Contains(t, output, "Publish At")
	assert.Contains(t, output, "Message")
}

func TestQueueShow_WithEntries_JSON(t *testing.T) {
	withTempHome(t)

	entry1, err := history.Append("JSON test post", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &QueueShowCmd{}
	globals := &Globals{JSON: true}

	output := captureStdout(t, func() {
		retErr := cmd.Run(globals)
		assert.NoError(t, retErr)
	})

	var resp map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, float64(1), resp["count"])

	queue := resp["queue"].([]any)
	require.Len(t, queue, 1)
	item := queue[0].(map[string]any)
	assert.Equal(t, entry1.ID, item["id"])
	assert.Equal(t, "JSON test post", item["message"])
	assert.Equal(t, float64(1), item["position"])
}

func TestQueueRemove_Success(t *testing.T) {
	withTempHome(t)

	entry, err := history.Append("Will be removed", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &QueueRemoveCmd{ID: entry.ID}
	globals := &Globals{JSON: false}

	output := captureStdout(t, func() {
		retErr := cmd.Run(globals)
		assert.NoError(t, retErr)
	})

	assert.Contains(t, output, "Removed entry")
	assert.Contains(t, output, entry.ID)

	// Verify it's actually gone.
	queued, err := history.Queued()
	require.NoError(t, err)
	assert.Empty(t, queued)
}

func TestQueueRemove_NotFound(t *testing.T) {
	withTempHome(t)

	cmd := &QueueRemoveCmd{ID: "nonexist"}
	globals := &Globals{JSON: false}

	err := cmd.Run(globals)
	require.Error(t, err)

	var cliErr *CLIError
	require.True(t, asCLIError(err, &cliErr))
	assert.Equal(t, "not_found", cliErr.Code)
}

func TestFormatPredictedTime_Today(t *testing.T) {
	now := time.Now()
	todayAt1430 := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location())
	result := formatPredictedTime(todayAt1430)
	assert.Contains(t, result, "Today")
	assert.Contains(t, result, "14:30")
}

func TestFormatPredictedTime_Tomorrow(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrowAt9 := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, tomorrow.Location())
	result := formatPredictedTime(tomorrowAt9)
	assert.Contains(t, result, "Tomorrow")
	assert.Contains(t, result, "09:00")
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "hello", firstLine("hello"))
	assert.Equal(t, "first", firstLine("first\nsecond"))
	assert.Equal(t, "", firstLine(""))
}

func TestMessagePreview_Short(t *testing.T) {
	// ≤ 5 lines total, all shown
	lines := messagePreview("line1\nline2\nline3", 3, 2, 60)
	assert.Equal(t, []string{"line1", "line2", "line3"}, lines)
}

func TestMessagePreview_Exact(t *testing.T) {
	// Exactly 5 lines (headN+tailN), all shown
	lines := messagePreview("a\nb\nc\nd\ne", 3, 2, 60)
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, lines)
}

func TestMessagePreview_Long(t *testing.T) {
	// 7 lines, should show 3 head + "..." + 2 tail
	lines := messagePreview("a\nb\nc\nd\ne\nf\ng", 3, 2, 60)
	assert.Equal(t, []string{"a", "b", "c", "...", "f", "g"}, lines)
}

func TestMessagePreview_SingleLine(t *testing.T) {
	lines := messagePreview("just one line", 3, 2, 60)
	assert.Equal(t, []string{"just one line"}, lines)
}

func TestMessagePreview_Empty(t *testing.T) {
	lines := messagePreview("", 3, 2, 60)
	assert.Equal(t, []string{""}, lines)
}

func TestMessagePreview_Truncation(t *testing.T) {
	long := strings.Repeat("x", 100)
	lines := messagePreview(long, 3, 2, 20)
	assert.Len(t, lines, 1)
	assert.Equal(t, 20, len(lines[0]))
}

func TestQueueShow_MultiLine_Human(t *testing.T) {
	withTempHome(t)

	_, err := history.Append("Line one\nLine two\nLine three\nLine four\nLine five\nLine six\nLine seven", "queued", time.Time{})
	require.NoError(t, err)

	cmd := &QueueShowCmd{}
	globals := &Globals{JSON: false}

	output := captureStdout(t, func() {
		retErr := cmd.Run(globals)
		assert.NoError(t, retErr)
	})

	assert.Contains(t, output, "Line one")
	assert.Contains(t, output, "Line two")
	assert.Contains(t, output, "Line three")
	assert.Contains(t, output, "...")
	assert.Contains(t, output, "Line six")
	assert.Contains(t, output, "Line seven")
	// Line four and five should be hidden
	assert.NotContains(t, output, "Line four")
	assert.NotContains(t, output, "Line five")
}
