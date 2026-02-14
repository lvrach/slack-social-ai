package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTempDataDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	original := dataDir
	dataDir = func() string { return dir }
	t.Cleanup(func() { dataDir = original })
}

func TestAppend_NewFile(t *testing.T) {
	withTempDataDir(t)

	entry, err := Append("hello world", "queued", time.Time{})
	require.NoError(t, err)

	assert.NotEmpty(t, entry.ID)
	assert.Len(t, entry.ID, 8)
	assert.Equal(t, "hello world", entry.Message)
	assert.Equal(t, "queued", entry.Status)
	assert.NotEmpty(t, entry.CreatedAt)
	assert.Empty(t, entry.ScheduledAt)

	// Verify the file was created.
	entries, err := Load()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestAppend_ExistingFile(t *testing.T) {
	withTempDataDir(t)

	e1, err := Append("first", "queued", time.Time{})
	require.NoError(t, err)

	e2, err := Append("second", "queued", time.Time{})
	require.NoError(t, err)

	assert.NotEqual(t, e1.ID, e2.ID)

	entries, err := Load()
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "first", entries[0].Message)
	assert.Equal(t, "second", entries[1].Message)
}

func TestAppend_WithScheduledAt(t *testing.T) {
	withTempDataDir(t)

	scheduled := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	entry, err := Append("scheduled post", "queued", scheduled)
	require.NoError(t, err)

	assert.Equal(t, scheduled.Format(time.RFC3339), entry.ScheduledAt)
}

func TestLoad_Empty(t *testing.T) {
	withTempDataDir(t)

	entries, err := Load()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLoad_RoundTrip(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("one", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("two", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("three", "queued", time.Time{})
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 3)

	assert.Equal(t, "one", entries[0].Message)
	assert.Equal(t, "two", entries[1].Message)
	assert.Equal(t, "three", entries[2].Message)

	for _, e := range entries {
		assert.NotEmpty(t, e.ID)
		assert.Equal(t, "queued", e.Status)
		assert.NotEmpty(t, e.CreatedAt)
	}
}

func TestClaimNextReady_FIFO(t *testing.T) {
	withTempDataDir(t)

	e1, err := Append("first", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("second", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("third", "queued", time.Time{})
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, e1.ID, claimed.ID)
	assert.Equal(t, "publishing", claimed.Status)
	assert.NotEmpty(t, claimed.UpdatedAt)
}

func TestClaimNextReady_SkipsFuture(t *testing.T) {
	withTempDataDir(t)

	future := time.Now().Add(1 * time.Hour)
	_, err := Append("future post", "queued", future)
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestClaimNextReady_FutureBecomesReady(t *testing.T) {
	withTempDataDir(t)

	past := time.Now().Add(-1 * time.Hour)
	_, err := Append("past post", "queued", past)
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, "past post", claimed.Message)
	assert.Equal(t, "publishing", claimed.Status)
}

func TestClaimNextReady_Empty(t *testing.T) {
	withTempDataDir(t)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestClaimNextReady_SkipsPublished(t *testing.T) {
	withTempDataDir(t)

	e, err := Append("published post", "queued", time.Time{})
	require.NoError(t, err)

	// Claim and then publish it.
	_, err = ClaimNextReady()
	require.NoError(t, err)
	err = MarkPublished(e.ID)
	require.NoError(t, err)

	// Now nothing should be claimable.
	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestMarkPublished(t *testing.T) {
	withTempDataDir(t)

	e, err := Append("to publish", "queued", time.Time{})
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)

	err = MarkPublished(e.ID)
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "published", entries[0].Status)
	assert.NotEmpty(t, entries[0].PublishedAt)
	assert.NotEmpty(t, entries[0].UpdatedAt)
}

func TestResetToQueued(t *testing.T) {
	withTempDataDir(t)

	e, err := Append("to reset", "queued", time.Time{})
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, "publishing", claimed.Status)

	err = ResetToQueued(e.ID)
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "queued", entries[0].Status)
	assert.NotEmpty(t, entries[0].UpdatedAt)
}

func TestRemove(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("first", "queued", time.Time{})
	require.NoError(t, err)
	e2, err := Append("second", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("third", "queued", time.Time{})
	require.NoError(t, err)

	found, err := Remove(e2.ID)
	require.NoError(t, err)
	assert.True(t, found)

	entries, err := Load()
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "first", entries[0].Message)
	assert.Equal(t, "third", entries[1].Message)
}

func TestRemove_NotFound(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("only", "queued", time.Time{})
	require.NoError(t, err)

	found, err := Remove("nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestClearPublished(t *testing.T) {
	withTempDataDir(t)

	// Create a mix of queued and published entries.
	e1, err := Append("will publish", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("stays queued", "queued", time.Time{})
	require.NoError(t, err)

	// Claim and publish the first.
	_, err = ClaimNextReady()
	require.NoError(t, err)
	err = MarkPublished(e1.ID)
	require.NoError(t, err)

	err = ClearPublished()
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "stays queued", entries[0].Message)
}

func TestClearAll(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("one", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("two", "queued", time.Time{})
	require.NoError(t, err)

	err = ClearAll()
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLastPublishedTime(t *testing.T) {
	withTempDataDir(t)

	// Create and publish three entries.
	for _, msg := range []string{"first", "second", "third"} {
		e, err := Append(msg, "queued", time.Time{})
		require.NoError(t, err)
		_, err = ClaimNextReady()
		require.NoError(t, err)
		err = MarkPublished(e.ID)
		require.NoError(t, err)
	}

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// The last published time should be the most recent publishedAt.
	lastPub, err := LastPublishedTime()
	require.NoError(t, err)
	assert.False(t, lastPub.IsZero())

	// Parse the most recent published entry's time.
	lastEntry := entries[len(entries)-1]
	expected, err := time.Parse(time.RFC3339, lastEntry.PublishedAt)
	require.NoError(t, err)
	assert.Equal(t, expected, lastPub)
}

func TestLastPublishedTime_NonePublished(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("just queued", "queued", time.Time{})
	require.NoError(t, err)

	lastPub, err := LastPublishedTime()
	require.NoError(t, err)
	assert.True(t, lastPub.IsZero())
}

func TestRecoverStuck(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("stuck", "queued", time.Time{})
	require.NoError(t, err)

	// Claim it to set it to publishing.
	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)

	// Manually set updatedAt to far in the past.
	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	entries[0].UpdatedAt = time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	writeEntries(t, entries)

	err = RecoverStuck(5 * time.Minute)
	require.NoError(t, err)

	entries, err = Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "queued", entries[0].Status)
}

func TestRecoverStuck_RecentPublishing(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("recent", "queued", time.Time{})
	require.NoError(t, err)

	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)

	// Recover with a generous timeout; entry is recent, so it should remain "publishing".
	err = RecoverStuck(5 * time.Minute)
	require.NoError(t, err)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "publishing", entries[0].Status)
}

func TestMaxEntries_DropsOldestPublished(t *testing.T) {
	withTempDataDir(t)

	// Create maxEntries published entries.
	for i := range maxEntries {
		e, err := Append("published-"+string(rune('A'+i%26)), "queued", time.Time{})
		require.NoError(t, err)
		_, err = ClaimNextReady()
		require.NoError(t, err)
		err = MarkPublished(e.ID)
		require.NoError(t, err)
	}

	entries, err := Load()
	require.NoError(t, err)
	assert.Len(t, entries, maxEntries)

	// Add one more queued entry.
	newEntry, err := Append("new queued", "queued", time.Time{})
	require.NoError(t, err)

	entries, err = Load()
	require.NoError(t, err)
	assert.Len(t, entries, maxEntries)

	// The new queued entry should be present.
	found := false
	for _, e := range entries {
		if e.ID == newEntry.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "new queued entry should be preserved")

	// Should have exactly one queued entry.
	queuedCount := 0
	for _, e := range entries {
		if e.Status == "queued" {
			queuedCount++
		}
	}
	assert.Equal(t, 1, queuedCount)
}

func TestQueued(t *testing.T) {
	withTempDataDir(t)

	_, err := Append("queued1", "queued", time.Time{})
	require.NoError(t, err)
	e2, err := Append("queued2", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("queued3", "queued", time.Time{})
	require.NoError(t, err)

	// Claim the first to make it "publishing".
	_, err = ClaimNextReady()
	require.NoError(t, err)

	// Publish the second.
	// First we need it to be claimed.
	// Actually, let's claim next (should be queued2 now), then publish it.
	claimed, err := ClaimNextReady()
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, e2.ID, claimed.ID)
	err = MarkPublished(e2.ID)
	require.NoError(t, err)

	queued, err := Queued()
	require.NoError(t, err)
	// Should include "queued" and "publishing" entries.
	assert.Len(t, queued, 2) // queued1 (publishing) + queued3 (queued)
}

func TestPublished(t *testing.T) {
	withTempDataDir(t)

	e1, err := Append("first", "queued", time.Time{})
	require.NoError(t, err)
	_, err = Append("second", "queued", time.Time{})
	require.NoError(t, err)

	// Claim and publish the first.
	_, err = ClaimNextReady()
	require.NoError(t, err)
	err = MarkPublished(e1.ID)
	require.NoError(t, err)

	published, err := Published()
	require.NoError(t, err)
	assert.Len(t, published, 1)
	assert.Equal(t, "first", published[0].Message)
}

// writeEntries is a test helper that writes entries to disk directly.
func writeEntries(t *testing.T, entries []Entry) {
	t.Helper()
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
