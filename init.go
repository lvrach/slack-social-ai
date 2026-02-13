package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/lvrach/slack-social-ai/internal/keyring"
	"github.com/lvrach/slack-social-ai/internal/manifest"
	"github.com/lvrach/slack-social-ai/internal/slack"
)

// InitCmd configures the Slack webhook interactively or via argument.
type InitCmd struct {
	WebhookURL string `arg:"" optional:"" help:"Slack webhook URL (skips interactive prompt)."`
}

func (cmd *InitCmd) Run(globals *Globals) error {
	// Path C: non-interactive — webhook URL passed as argument.
	if cmd.WebhookURL != "" {
		return cmd.storeAndVerify(globals, cmd.WebhookURL)
	}

	// Check if already configured.
	existing, err := keyring.Get()
	if err == nil && existing != "" {
		return cmd.handleExisting(globals, existing)
	}

	// Interactive paths A/B.
	return cmd.interactive(globals)
}

func (cmd *InitCmd) handleExisting(globals *Globals, existing string) error {
	var choice string
	err := huh.NewSelect[string]().
		Title("Slack webhook is already configured.").
		Options(
			huh.NewOption("Test existing webhook", "test"),
			huh.NewOption("Overwrite with new URL", "overwrite"),
			huh.NewOption("Exit", "exit"),
		).
		Value(&choice).
		Run()
	if err != nil {
		return err
	}

	switch choice {
	case "test":
		return cmd.verifyWebhook(globals, existing)
	case "overwrite":
		return cmd.interactive(globals)
	default:
		return nil
	}
}

func (cmd *InitCmd) interactive(globals *Globals) error {
	fmt.Println("Welcome to slack-social-ai!")
	fmt.Println("Let's set up your Slack webhook.")
	fmt.Println()

	var hasWebhook bool
	err := huh.NewConfirm().
		Title("Do you already have a Slack webhook URL?").
		Affirmative("Yes").
		Negative("No, help me create one").
		Value(&hasWebhook).
		Run()
	if err != nil {
		return err
	}

	if !hasWebhook {
		// Path A: guided setup.
		if err := cmd.guidedSetup(); err != nil {
			return err
		}
	}

	// Path A (continued) or Path B: prompt for URL.
	var webhookURL string
	err = huh.NewInput().
		Title("Paste your Slack webhook URL:").
		Placeholder("https://hooks.slack.com/services/T.../B.../xxx").
		Validate(validateWebhookURL).
		Value(&webhookURL).
		Run()
	if err != nil {
		return err
	}

	return cmd.storeAndVerify(globals, webhookURL)
}

func (cmd *InitCmd) guidedSetup() error {
	var appName string
	err := huh.NewInput().
		Title("App name for Slack:").
		Placeholder("slack-social-ai").
		CharLimit(35).
		Value(&appName).
		Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(appName) == "" {
		appName = "slack-social-ai"
	}

	// Generate manifest file.
	manifestYAML := manifest.Generate(appName)
	if err := os.WriteFile("slack-app-manifest.yml", []byte(manifestYAML), 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	fmt.Println("\n✓ Created slack-app-manifest.yml")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Go to https://api.slack.com/apps?new_app=1")
	fmt.Println("  2. Select \"From a manifest\"")
	fmt.Println("  3. Choose your workspace")
	fmt.Println("  4. Paste the contents of slack-app-manifest.yml")
	fmt.Println("  5. Click \"Create\"")
	fmt.Println("  6. Go to \"Incoming Webhooks\" in the sidebar")
	fmt.Println("  7. Click \"Add New Webhook to Workspace\"")
	fmt.Println("  8. Pick a channel and authorize")
	fmt.Println("  9. Copy the webhook URL")
	fmt.Println()

	return nil
}

func (cmd *InitCmd) storeAndVerify(globals *Globals, webhookURL string) error {
	if err := validateWebhookURL(webhookURL); err != nil {
		return newCLIError(ExitInvalidInput, "invalid_url", err.Error())
	}

	// Verify before storing so a bad URL never persists.
	if err := cmd.verifyWebhook(globals, webhookURL); err != nil {
		return err
	}

	if err := keyring.Set(webhookURL); err != nil {
		return fmt.Errorf("store webhook in keychain: %w", err)
	}

	msg := "Slack webhook configured successfully."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		fmt.Println("\n" + msg)
		fmt.Println("\nTry it: slack-social-ai post \"Hello from the terminal!\"")
	}
	return nil
}

func (cmd *InitCmd) verifyWebhook(globals *Globals, webhookURL string) error {
	if !globals.JSON {
		fmt.Print("Verifying webhook... ")
	}
	if err := slack.SendWebhook(webhookURL, "👋 slack-social-ai is connected!"); err != nil {
		if !globals.JSON {
			fmt.Println("failed.")
		}
		return newCLIError(ExitRuntimeError, "webhook_failed",
			fmt.Sprintf("Webhook verification failed: %s", err))
	}
	if !globals.JSON {
		fmt.Println("ok!")
	}
	return nil
}

func validateWebhookURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}
	if !strings.HasPrefix(s, "https://hooks.slack.com/") {
		return fmt.Errorf("URL must start with https://hooks.slack.com/")
	}
	return nil
}
