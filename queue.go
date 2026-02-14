package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/schedule"
)

// QueueCmd manages the post queue.
type QueueCmd struct {
	Show    QueueShowCmd    `cmd:"" default:"withargs" help:"Show queued messages with predicted publish times."`
	Inspect QueueInspectCmd `cmd:"" help:"Interactive queue editor — browse and delete items."`
	Remove  QueueRemoveCmd  `cmd:"" help:"Remove a queued message by ID."`
}

// QueueShowCmd displays the queue with predicted publish times.
type QueueShowCmd struct{}

func (cmd *QueueShowCmd) Run(globals *Globals) error {
	entries, err := history.Queued()
	if err != nil {
		return newCLIError(ExitRuntimeError, "load_queue",
			fmt.Sprintf("Failed to load queue: %s", err))
	}

	cfg, _ := config.Load()
	lastPublished, _ := history.LastPublishedTime()
	now := time.Now().UTC()

	predictions := schedule.PredictPublishTimes(entries, cfg.Schedule, lastPublished, now)

	if globals.JSON {
		return cmd.printJSON(predictions, cfg.Schedule)
	}
	return cmd.printHuman(predictions, cfg.Schedule)
}

func (cmd *QueueShowCmd) printJSON(predictions []schedule.Prediction, sched schedule.Schedule) error {
	type jsonPrediction struct {
		Position         int    `json:"position"`
		ID               string `json:"id"`
		Message          string `json:"message"`
		PredictedPublish string `json:"predicted_publish_at"`
		Approximate      bool   `json:"approximate"`
		CreatedAt        string `json:"created_at"`
		ScheduledAt      string `json:"scheduled_at,omitempty"`
	}

	items := make([]jsonPrediction, len(predictions))
	for i, p := range predictions {
		items[i] = jsonPrediction{
			Position:         p.Position,
			ID:               p.Entry.ID,
			Message:          p.Entry.Message,
			PredictedPublish: p.PublishAt.Format(time.RFC3339),
			Approximate:      p.Approximate,
			CreatedAt:        p.Entry.CreatedAt,
			ScheduledAt:      p.Entry.ScheduledAt,
		}
	}

	resp := map[string]any{
		"queue":    items,
		"count":    len(items),
		"schedule": formatScheduleSummary(sched),
	}

	return json.NewEncoder(os.Stdout).Encode(resp)
}

func (cmd *QueueShowCmd) printHuman(predictions []schedule.Prediction, sched schedule.Schedule) error {
	if len(predictions) == 0 {
		fmt.Fprintln(os.Stdout, "Queue is empty.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "Queue (%d messages):\n\n", len(predictions))
	fmt.Fprintf(os.Stdout, " %-4s %-19s %s\n", "#", "Publish At", "Message")
	fmt.Fprintf(os.Stdout, " %-4s %-19s %s\n", "\u2500", strings.Repeat("\u2500", 18), strings.Repeat("\u2500", 40))

	indent := strings.Repeat(" ", 26) // 1 space + 4 pos + 1 space + 19 time + 1 space
	for _, p := range predictions {
		timeStr := formatPredictedTime(p.PublishAt)
		if p.Approximate {
			timeStr = "~" + timeStr
		}

		preview := messagePreview(p.Entry.Message, 3, 2, 60)
		fmt.Fprintf(os.Stdout, " %-4d %-19s %s\n", p.Position, timeStr, preview[0])
		for _, line := range preview[1:] {
			fmt.Fprintf(os.Stdout, "%s%s\n", indent, line)
		}
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprintf(os.Stdout, "%s\n", formatScheduleSummary(sched))
	return nil
}

// messagePreview returns a multi-line preview of a message.
// If the message has more than headN+tailN lines, the middle is replaced with "...".
// Each line is truncated to maxWidth.
func messagePreview(s string, headN, tailN, maxWidth int) []string { //nolint:unparam // headN kept as parameter for flexibility
	lines := strings.Split(s, "\n")
	total := len(lines)

	if total <= headN+tailN {
		result := make([]string, total)
		for i, line := range lines {
			result[i] = truncate(line, maxWidth)
		}
		return result
	}

	result := make([]string, 0, headN+1+tailN)
	for _, line := range lines[:headN] {
		result = append(result, truncate(line, maxWidth))
	}
	result = append(result, "...")
	for _, line := range lines[total-tailN:] {
		result = append(result, truncate(line, maxWidth))
	}
	return result
}

// formatPredictedTime formats a predicted publish time as a human-friendly string.
func formatPredictedTime(t time.Time) string {
	now := time.Now()
	local := t.Local()

	// Same day: "Today 14:30"
	if local.Year() == now.Year() && local.YearDay() == now.YearDay() {
		return "Today " + local.Format("15:04")
	}

	// Tomorrow: "Tomorrow 09:00"
	tomorrow := now.AddDate(0, 0, 1)
	if local.Year() == tomorrow.Year() && local.YearDay() == tomorrow.YearDay() {
		return "Tomorrow " + local.Format("15:04")
	}

	// Within this week: "Mon 09:00"
	if local.Before(now.AddDate(0, 0, 7)) {
		return local.Format("Mon 15:04")
	}

	// Beyond: "2026-02-23 09:00"
	return local.Format("2006-01-02 15:04")
}

// firstLine returns the first line of a string.
func firstLine(s string) string {
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return before
	}
	return s
}

// QueueRemoveCmd removes a queued message by ID.
type QueueRemoveCmd struct {
	ID string `arg:"" help:"ID of the message to remove."`
}

func (cmd *QueueRemoveCmd) Run(globals *Globals) error {
	found, err := history.Remove(cmd.ID)
	if err != nil {
		return newCLIError(ExitRuntimeError, "remove_failed",
			fmt.Sprintf("Failed to remove entry: %s", err))
	}
	if !found {
		return newCLIError(ExitInvalidInput, "not_found",
			fmt.Sprintf("Entry %q not found in queue.", cmd.ID))
	}

	msg := fmt.Sprintf("Removed entry %s from queue.", cmd.ID)
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}
