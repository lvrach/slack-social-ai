package history

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const maxEntries = 200

// Entry represents a single history record with scheduling and status tracking.
type Entry struct {
	ID          string `json:"id"`
	Message     string `json:"message"`
	Status      string `json:"status"`                 // "queued" | "publishing" | "published" | "failed"
	CreatedAt   string `json:"created_at"`             // RFC3339
	ScheduledAt string `json:"scheduled_at,omitempty"` // RFC3339; empty = ready now
	PublishedAt string `json:"published_at,omitempty"` // RFC3339; set when published
	UpdatedAt   string `json:"updated_at,omitempty"`   // RFC3339; tracks last status change
}

// legacyEntry is the old format used before the migration.
type legacyEntry struct {
	Timestamp string `json:"ts"`
	Message   string `json:"message"`
}

// dataDir is a var for test overrides.
var dataDir = defaultDataDir

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "slack-social-ai")
}

func historyPath() string { return filepath.Join(dataDir(), "history.json") }
func lockPath() string    { return filepath.Join(dataDir(), "history.lock") }

// withLock acquires an exclusive file lock for the duration of fn.
func withLock(fn func() error) error {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	fileLock := flock.New(lockPath())
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() { _ = fileLock.Unlock() }()
	return fn()
}

// generateID returns a random 8 hex-char identifier.
func generateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based.
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return hex.EncodeToString(b)
}

// Load reads the history file and returns all entries.
// If old format is detected (entries with empty ID but non-empty message),
// it runs migration under the file lock and writes back immediately.
func Load() ([]Entry, error) {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	// Check if migration is needed: all entries have empty ID but non-empty message.
	if needsMigration(entries, data) {
		// Perform migration under the file lock to prevent concurrent writes.
		var migrated []Entry
		lockErr := withLock(func() error {
			// Re-read under lock in case another process migrated first.
			freshData, err := os.ReadFile(historyPath())
			if err != nil {
				return err
			}
			var freshEntries []Entry
			if err := json.Unmarshal(freshData, &freshEntries); err != nil {
				return err
			}
			if !needsMigration(freshEntries, freshData) {
				// Already migrated by another process.
				migrated = freshEntries
				return nil
			}
			var migrateErr error
			migrated, migrateErr = migrateFromLegacy(freshData)
			if migrateErr != nil {
				return migrateErr
			}
			return atomicWrite(migrated)
		})
		if lockErr != nil {
			return nil, lockErr
		}
		return migrated, nil
	}

	return entries, nil
}

// needsMigration checks whether the data looks like legacy format.
func needsMigration(entries []Entry, raw []byte) bool {
	if len(entries) == 0 {
		return false
	}
	// If all entries have empty ID but non-empty message, it's legacy format.
	allEmptyID := true
	for _, e := range entries {
		if e.ID != "" {
			allEmptyID = false
			break
		}
	}
	if !allEmptyID {
		return false
	}
	// Verify it actually has the legacy "ts" field.
	var legacy []legacyEntry
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return false
	}
	for _, l := range legacy {
		if l.Message != "" && l.Timestamp != "" {
			return true
		}
	}
	return false
}

// migrateFromLegacy converts legacy entries to the new format.
func migrateFromLegacy(raw []byte) ([]Entry, error) {
	var legacy []legacyEntry
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(legacy))
	for i, l := range legacy {
		h := sha256.Sum256(fmt.Appendf(nil, "%s:%d:%s", l.Timestamp, i, l.Message))
		id := hex.EncodeToString(h[:])[:8]
		entries = append(entries, Entry{
			ID:          id,
			Message:     l.Message,
			Status:      "published",
			CreatedAt:   l.Timestamp,
			PublishedAt: l.Timestamp,
		})
	}
	return entries, nil
}

// loadFromDisk reads and unmarshals the history file without migration.
func loadFromDisk() ([]Entry, error) {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Append creates a new Entry and persists it.
func Append(message, status string, scheduledAt time.Time) (Entry, error) {
	entry := Entry{
		ID:        generateID(),
		Message:   message,
		Status:    status,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if status == "published" {
		entry.PublishedAt = entry.CreatedAt
	}
	if !scheduledAt.IsZero() {
		entry.ScheduledAt = scheduledAt.UTC().Format(time.RFC3339)
	}

	var result Entry
	err := withLock(func() error {
		entries, loadErr := loadFromDisk()
		if loadErr != nil {
			return fmt.Errorf("load history: %w", loadErr)
		}
		// Migrate legacy entries if needed.
		raw, _ := os.ReadFile(historyPath())
		if raw != nil && needsMigration(entries, raw) {
			migrated, migrateErr := migrateFromLegacy(raw)
			if migrateErr != nil {
				return fmt.Errorf("migrate history: %w", migrateErr)
			}
			entries = migrated
		}
		entries = append(entries, entry)
		entries = enforceMaxEntries(entries)
		result = entry
		return atomicWrite(entries)
	})
	return result, err
}

// enforceMaxEntries trims the entries slice to maxEntries.
// It drops oldest published entries first, then oldest queued.
func enforceMaxEntries(entries []Entry) []Entry {
	if len(entries) <= maxEntries {
		return entries
	}

	// Drop oldest published first.
	for len(entries) > maxEntries {
		idx := -1
		for i, e := range entries {
			if e.Status == "published" {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		entries = append(entries[:idx], entries[idx+1:]...)
	}

	// If still over, drop oldest queued.
	for len(entries) > maxEntries {
		idx := -1
		for i, e := range entries {
			if e.Status == "queued" {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		entries = append(entries[:idx], entries[idx+1:]...)
	}

	// Last resort: drop from front.
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	return entries
}

// ClaimNextReady atomically claims the oldest ready-to-publish entry.
// An entry is ready if status=="queued" and (scheduledAt is empty or <= now).
// Returns nil, nil if nothing is ready.
func ClaimNextReady() (*Entry, error) {
	var result *Entry
	err := withLock(func() error {
		entries, loadErr := loadFromDisk()
		if loadErr != nil {
			return loadErr
		}

		now := time.Now().UTC()
		for i, e := range entries {
			if e.Status != "queued" {
				continue
			}
			if e.ScheduledAt != "" {
				scheduled, parseErr := time.Parse(time.RFC3339, e.ScheduledAt)
				if parseErr != nil {
					continue
				}
				if scheduled.After(now) {
					continue
				}
			}
			// Found a ready entry.
			entries[i].Status = "publishing"
			entries[i].UpdatedAt = now.Format(time.RFC3339)
			claimed := entries[i]
			result = &claimed
			return atomicWrite(entries)
		}
		return nil
	})
	return result, err
}

// MarkPublished sets an entry's status to "published" with a publishedAt timestamp.
func MarkPublished(id string) error {
	return withLock(func() error {
		entries, err := loadFromDisk()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		for i, e := range entries {
			if e.ID == id {
				entries[i].Status = "published"
				entries[i].PublishedAt = now
				entries[i].UpdatedAt = now
				return atomicWrite(entries)
			}
		}
		return fmt.Errorf("entry %q not found", id)
	})
}

// ResetToQueued resets an entry's status back to "queued".
func ResetToQueued(id string) error {
	return withLock(func() error {
		entries, err := loadFromDisk()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		for i, e := range entries {
			if e.ID == id {
				entries[i].Status = "queued"
				entries[i].UpdatedAt = now
				return atomicWrite(entries)
			}
		}
		return fmt.Errorf("entry %q not found", id)
	})
}

// Remove deletes an entry by ID. Returns (found, error).
func Remove(id string) (bool, error) {
	found := false
	err := withLock(func() error {
		entries, loadErr := loadFromDisk()
		if loadErr != nil {
			return loadErr
		}
		for i, e := range entries {
			if e.ID == id {
				entries = append(entries[:i], entries[i+1:]...)
				found = true
				return atomicWrite(entries)
			}
		}
		return nil
	})
	return found, err
}

// ClearPublished removes all entries with status "published".
func ClearPublished() error {
	return withLock(func() error {
		entries, err := loadFromDisk()
		if err != nil {
			return err
		}
		filtered := make([]Entry, 0, len(entries))
		for _, e := range entries {
			if e.Status != "published" {
				filtered = append(filtered, e)
			}
		}
		return atomicWrite(filtered)
	})
}

// ClearAll removes all entries.
func ClearAll() error {
	return withLock(func() error {
		return atomicWrite([]Entry{})
	})
}

// Queued returns entries with status "queued" or "publishing".
func Queued() ([]Entry, error) {
	entries, err := Load()
	if err != nil {
		return nil, err
	}
	var result []Entry
	for _, e := range entries {
		if e.Status == "queued" || e.Status == "publishing" {
			result = append(result, e)
		}
	}
	return result, nil
}

// Published returns entries with status "published".
func Published() ([]Entry, error) {
	entries, err := Load()
	if err != nil {
		return nil, err
	}
	var result []Entry
	for _, e := range entries {
		if e.Status == "published" {
			result = append(result, e)
		}
	}
	return result, nil
}

// LastPublishedTime returns the most recent publishedAt timestamp among published entries.
// Returns zero time if no entries are published.
func LastPublishedTime() (time.Time, error) {
	entries, err := Load()
	if err != nil {
		return time.Time{}, err
	}
	var latest time.Time
	for _, e := range entries {
		if e.Status != "published" || e.PublishedAt == "" {
			continue
		}
		t, parseErr := time.Parse(time.RFC3339, e.PublishedAt)
		if parseErr != nil {
			continue
		}
		if t.After(latest) {
			latest = t
		}
	}
	return latest, nil
}

// RecoverStuck resets entries stuck in "publishing" state back to "queued"
// if their updatedAt is older than the given timeout.
func RecoverStuck(timeout time.Duration) error {
	return withLock(func() error {
		entries, err := loadFromDisk()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		changed := false
		for i, e := range entries {
			if e.Status != "publishing" {
				continue
			}
			if e.UpdatedAt == "" {
				continue
			}
			updated, parseErr := time.Parse(time.RFC3339, e.UpdatedAt)
			if parseErr != nil {
				continue
			}
			if now.Sub(updated) > timeout {
				entries[i].Status = "queued"
				entries[i].UpdatedAt = now.Format(time.RFC3339)
				changed = true
			}
		}
		if changed {
			return atomicWrite(entries)
		}
		return nil
	})
}

func atomicWrite(entries []Entry) error {
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
