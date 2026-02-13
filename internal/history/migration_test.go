package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_LegacyFormat(t *testing.T) {
	withTempDataDir(t)

	// Write old format entries.
	legacy := []legacyEntry{
		{Timestamp: "2025-01-01T10:00:00Z", Message: "old post one"},
		{Timestamp: "2025-01-02T12:00:00Z", Message: "old post two"},
	}
	writeLegacy(t, legacy)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	for _, e := range entries {
		assert.NotEmpty(t, e.ID)
		assert.Len(t, e.ID, 8)
		assert.Equal(t, "published", e.Status)
		assert.NotEmpty(t, e.CreatedAt)
		assert.NotEmpty(t, e.PublishedAt)
		assert.NotEmpty(t, e.Message)
	}

	assert.Equal(t, "old post one", entries[0].Message)
	assert.Equal(t, "old post two", entries[1].Message)
	assert.Equal(t, "2025-01-01T10:00:00Z", entries[0].CreatedAt)
	assert.Equal(t, "2025-01-02T12:00:00Z", entries[1].CreatedAt)
}

func TestMigration_AlreadyNewFormat(t *testing.T) {
	withTempDataDir(t)

	// Write new format directly.
	newEntries := []Entry{
		{
			ID:        "abcdef01",
			Message:   "new format",
			Status:    "queued",
			CreatedAt: "2025-06-01T10:00:00Z",
		},
	}
	writeEntries(t, newEntries)

	entries, err := Load()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "abcdef01", entries[0].ID)
	assert.Equal(t, "new format", entries[0].Message)
	assert.Equal(t, "queued", entries[0].Status)
}

func TestMigration_EmptyFile(t *testing.T) {
	withTempDataDir(t)

	// Write empty JSON array.
	path := historyPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o600))

	entries, err := Load()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMigration_WritesBack(t *testing.T) {
	withTempDataDir(t)

	legacy := []legacyEntry{
		{Timestamp: "2025-03-01T09:00:00Z", Message: "migrated post"},
	}
	writeLegacy(t, legacy)

	// First Load triggers migration.
	_, err := Load()
	require.NoError(t, err)

	// Read raw file.
	data, err := os.ReadFile(historyPath())
	require.NoError(t, err)

	var entries []Entry
	err = json.Unmarshal(data, &entries)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.NotEmpty(t, entries[0].ID)
	assert.Equal(t, "published", entries[0].Status)
	assert.Equal(t, "migrated post", entries[0].Message)
}

func writeLegacy(t *testing.T, entries []legacyEntry) {
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
