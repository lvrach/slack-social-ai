package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/keyring"
	"github.com/lvrach/slack-social-ai/internal/schedule"
	"github.com/lvrach/slack-social-ai/internal/slack"
)

// PublishCmd publishes the next queued message to Slack.
// Typically invoked by the launchd scheduler, not manually.
type PublishCmd struct {
	IgnoreSchedule bool `help:"Skip time and frequency guards." long:"ignore-schedule"`
}

func (cmd *PublishCmd) Run(globals *Globals) error {
	// 1. Get webhook URL from keyring.
	webhookURL, err := keyring.Get()
	if err != nil {
		if keyring.IsNotFound(err) {
			return cmd.jsonOrError(globals, "not_configured",
				"Not configured. Run \"slack-social-ai auth login\" first.", ExitNotConfigured)
		}
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to read keychain: %s", err))
	}

	// 2. Load config.
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Config{Schedule: schedule.DefaultSchedule()}
	}

	return cmd.publishOne(webhookURL, cfg, globals, cmd.IgnoreSchedule)
}

// publishOne contains the core publish logic: time guard, frequency guard,
// recover stuck, claim, send webhook, and mark published.
// Extracted from Run so it can be tested without the macOS keychain.
func (cmd *PublishCmd) publishOne(webhookURL string, cfg config.Config, globals *Globals, ignoreSchedule bool) error {
	if !ignoreSchedule {
		// 3. Time guard: check if we're in active hours.
		if !cfg.Schedule.IsActiveNow() {
			return cmd.exitOutsideSchedule(globals, cfg.Schedule)
		}

		// 4. Frequency guard: check minimum interval between posts.
		if postEvery := cfg.Schedule.PostEvery(); postEvery > 0 {
			lastPublished, err := history.LastPublishedTime()
			if err == nil && !lastPublished.IsZero() {
				elapsed := time.Since(lastPublished)
				if elapsed < postEvery {
					nextEligible := lastPublished.Add(postEvery)
					return cmd.exitTooSoon(globals, nextEligible)
				}
			}
		}
	}

	// 5. Recover stuck entries (publishing for > 5 minutes).
	_ = history.RecoverStuck(5 * time.Minute)

	// 6. Claim next ready entry.
	entry, err := history.ClaimNextReady()
	if err != nil {
		return newCLIError(ExitRuntimeError, "claim_error",
			fmt.Sprintf("Failed to claim entry: %s", err))
	}
	if entry == nil {
		return cmd.exitNoQueued(globals)
	}

	// 7. Send webhook.
	if err := slack.SendWebhook(webhookURL, entry.Message); err != nil {
		// Reset to queued on failure.
		_ = history.ResetToQueued(entry.ID)
		return newCLIError(ExitRuntimeError, "webhook_failed",
			fmt.Sprintf("Failed to publish message: %s", err))
	}

	// 8. Mark published.
	if err := history.MarkPublished(entry.ID); err != nil {
		// Webhook succeeded but marking failed -- log but don't fail.
		fmt.Fprintf(os.Stderr, "Warning: message sent but failed to mark as published: %s\n", err)
	}

	// 9. Success.
	if globals.JSON {
		resp := map[string]string{"status": "ok", "message": entry.Message, "id": entry.ID}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintf(os.Stdout, "Published: %s\n", truncate(entry.Message, 80))
	}
	return nil
}

// exitOutsideSchedule reports that we're outside the configured active hours.
func (cmd *PublishCmd) exitOutsideSchedule(globals *Globals, sched schedule.Schedule) error {
	if globals.JSON {
		resp := map[string]string{"status": "outside_schedule"}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintf(os.Stdout, "Skipped: outside active hours (%s %02d:00–%02d:00).\n",
			formatWeekdays(sched.Weekdays), sched.StartHour, sched.EndHour)
	}
	return nil
}

// exitTooSoon handles the "too soon since last post" case.
func (cmd *PublishCmd) exitTooSoon(globals *Globals, nextEligible time.Time) error {
	if globals.JSON {
		resp := map[string]string{
			"status":        "too_soon",
			"next_eligible": nextEligible.UTC().Format(time.RFC3339),
		}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintf(os.Stdout, "Skipped: too soon. Next eligible: %s.\n",
			nextEligible.Local().Format("3:04pm"))
	}
	return nil
}

// exitNoQueued reports that there are no messages in the queue.
func (cmd *PublishCmd) exitNoQueued(globals *Globals) error {
	if globals.JSON {
		resp := map[string]string{"status": "no_queued"}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintln(os.Stdout, "Skipped: no messages queued.")
	}
	return nil
}

// jsonOrError returns a CLIError for non-JSON mode, or prints JSON error and returns nil.
func (cmd *PublishCmd) jsonOrError(globals *Globals, code, message string, exitCode int) error {
	if globals.JSON {
		resp := map[string]string{"status": "error", "error": code, "message": message}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stderr, string(b))
		os.Exit(exitCode)
	}
	return newCLIError(exitCode, code, message)
}

// truncate shortens a string to n chars with "..." suffix.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
