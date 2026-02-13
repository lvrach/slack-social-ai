package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lvrach/slack-social-ai/internal/history"
)

// HistoryCmd shows or manages post history.
type HistoryCmd struct {
	Clear bool `help:"Clear all history."`
}

func (cmd *HistoryCmd) Run(globals *Globals) error {
	if cmd.Clear {
		return cmd.clear(globals)
	}
	return cmd.list(globals)
}

func (cmd *HistoryCmd) clear(globals *Globals) error {
	if err := history.Clear(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear history: %w", err)
	}
	msg := "History cleared."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

func (cmd *HistoryCmd) list(globals *Globals) error {
	entries, err := history.Load()
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
		fmt.Printf("[%s] %s\n", e.Timestamp, e.Message)
	}
	return nil
}
