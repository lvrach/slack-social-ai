package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxEntries = 200

// Entry represents a single history record.
type Entry struct {
	Timestamp string `json:"ts"`
	Message   string `json:"message"`
}

// Load reads the history file and returns all entries.
// Returns nil slice and nil error if the file does not exist.
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
	return entries, nil
}

// Append adds a new entry to the history file, capping at maxEntries.
// Swallows load errors on corrupt files (starts fresh).
func Append(message string) error {
	entries, err := Load()
	if err != nil {
		// Corrupt file — start fresh.
		entries = nil
	}

	entries = append(entries, Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   message,
	})

	// Cap at maxEntries (drop oldest).
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	return atomicWrite(entries)
}

// Clear removes all history entries.
func Clear() error {
	return os.Remove(historyPath())
}

func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "slack-social-ai", "history.json")
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
