package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lvrach/slack-social-ai/internal/history"
)

// HistoryCmd shows or manages post history.
type HistoryCmd struct {
	QueuedOnly bool   `name:"queued" help:"Show only queued messages."`
	Published  bool   `name:"published" help:"Show only published messages."`
	Remove     string `help:"Remove a specific entry by ID."`
	Clear      bool   `help:"Clear published history (keeps queue)."`
	ClearAll   bool   `name:"clear-all" help:"Clear everything (published + queued)."`
}

func (cmd *HistoryCmd) Run(globals *Globals) error {
	if cmd.ClearAll {
		return cmd.clearAll(globals)
	}
	if cmd.Clear {
		return cmd.clearPublished(globals)
	}
	if cmd.Remove != "" {
		return cmd.remove(globals, cmd.Remove)
	}
	return cmd.list(globals)
}

func (cmd *HistoryCmd) clearAll(globals *Globals) error {
	if err := history.ClearAll(); err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	msg := "All history cleared."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

func (cmd *HistoryCmd) clearPublished(globals *Globals) error {
	if err := history.ClearPublished(); err != nil {
		return fmt.Errorf("clear published: %w", err)
	}
	msg := "Published history cleared. Queued messages preserved."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

func (cmd *HistoryCmd) remove(globals *Globals, id string) error {
	found, err := history.Remove(id)
	if err != nil {
		return fmt.Errorf("remove entry: %w", err)
	}
	if !found {
		return newCLIError(ExitInvalidInput, "not_found",
			fmt.Sprintf("Entry %q not found.", id))
	}
	msg := fmt.Sprintf("Entry %s removed.", id)
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

func (cmd *HistoryCmd) list(globals *Globals) error {
	var entries []history.Entry
	var err error

	switch {
	case cmd.QueuedOnly:
		entries, err = history.Queued()
	case cmd.Published:
		entries, err = history.Published()
	default:
		entries, err = history.Load()
	}
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}

	// Reverse so most recent entries appear first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if globals.JSON {
		if entries == nil {
			entries = []history.Entry{}
		}
		return json.NewEncoder(os.Stdout).Encode(entries)
	}

	if len(entries) == 0 {
		fmt.Println("No history.")
		return nil
	}

	for _, e := range entries {
		ts := e.CreatedAt
		status := e.Status

		// Show scheduled time for queued entries with future scheduledAt.
		scheduledInfo := ""
		if e.ScheduledAt != "" && (e.Status == "queued" || e.Status == "publishing") {
			scheduledInfo = fmt.Sprintf(" [at %s]", formatShortTime(e.ScheduledAt))
		}

		// Show ID for queued/publishing entries (useful for --remove).
		idInfo := ""
		if e.Status == "queued" || e.Status == "publishing" {
			idInfo = fmt.Sprintf("  (id: %s)", e.ID)
		}

		fmt.Printf("[%s] [%s]%s %s%s\n", formatShortTime(ts), status, scheduledInfo, e.Message, idInfo)
	}
	return nil
}

// formatShortTime extracts HH:MM from an RFC3339 timestamp for display,
// or returns the raw string if parsing fails.
func formatShortTime(rfc3339 string) string {
	// Quick extraction: "2025-02-13T14:30:00Z" → "2025-02-13 14:30"
	if len(rfc3339) >= 16 {
		return rfc3339[:10] + " " + rfc3339[11:16]
	}
	return rfc3339
}
