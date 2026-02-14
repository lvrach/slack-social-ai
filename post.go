package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/keyring"
	"github.com/lvrach/slack-social-ai/internal/slack"
)

// PostCmd queues a message for publishing (default) or publishes immediately.
type PostCmd struct {
	MessageInput `embed:""`
	Now          bool   `help:"Publish immediately, skip the queue." short:"N" xor:"mode"`
	DryRun       bool   `help:"Preview the message without publishing or queuing." short:"n" xor:"mode"`
	At           string `help:"Schedule for a future time (HH:MM, duration like 2h, or RFC3339)." short:"a" xor:"mode"`
}

func (cmd *PostCmd) Run(globals *Globals) error {
	// 1. Validate webhook exists.
	webhookURL, err := keyring.Get()
	if err != nil {
		if keyring.IsNotFound(err) {
			return newCLIError(ExitNotConfigured, "not_configured",
				"Not configured. Run \"slack-social-ai auth login\" first.")
		}
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to read keychain: %s", err))
	}

	// 2. Resolve message.
	message, err := cmd.Resolve()
	if err != nil {
		return err
	}

	// 3. Apply --code wrapping.
	if cmd.Code {
		message = "```\n" + message + "\n```"
	}

	// 4. Dry run — preview only.
	if cmd.DryRun {
		return cmd.dryRun(globals, message)
	}

	// 5. Publish immediately with --now.
	if cmd.Now {
		return cmd.publishNow(globals, webhookURL, message)
	}

	// 6. Parse --at if provided.
	var scheduledAt time.Time
	if cmd.At != "" {
		scheduledAt, err = parseAt(cmd.At)
		if err != nil {
			return err
		}
	}

	// 7. Queue the message.
	entry, err := history.Append(message, "queued", scheduledAt)
	if err != nil {
		return newCLIError(ExitRuntimeError, "queue_failed",
			fmt.Sprintf("Failed to queue message: %s", err))
	}

	// 8. Print confirmation.
	if globals.JSON {
		resp := map[string]any{
			"status": "queued",
			"id":     entry.ID,
		}
		if !scheduledAt.IsZero() {
			resp["scheduled_at"] = scheduledAt.UTC().Format(time.RFC3339)
		}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		if !scheduledAt.IsZero() {
			fmt.Fprintf(os.Stdout, "Message queued. Scheduled for: %s.\n",
				scheduledAt.Local().Format("2006-01-02 15:04"))
		} else {
			fmt.Fprintln(os.Stdout, "Message queued.")
		}
	}
	return nil
}

func (cmd *PostCmd) dryRun(globals *Globals, message string) error {
	if globals.JSON {
		resp := map[string]any{
			"status":     "dry_run",
			"message":    message,
			"char_count": len(message),
		}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintln(os.Stdout, "[dry-run] Message preview:")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, message)
		fmt.Fprintf(os.Stdout, "\n(%d characters)\n", len(message))
	}
	return nil
}

func (cmd *PostCmd) publishNow(globals *Globals, webhookURL, message string) error {
	if err := slack.SendWebhook(webhookURL, message); err != nil {
		return newCLIError(ExitRuntimeError, "send_failed",
			fmt.Sprintf("Failed to post message: %s", err))
	}

	_, _ = history.Append(message, "published", time.Time{}) // best-effort

	if globals.JSON {
		printSuccessJSON("Message posted to Slack.")
	} else {
		printSuccessHuman("Message posted to Slack.")
	}
	return nil
}
