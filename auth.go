package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/lvrach/slack-social-ai/internal/keyring"
	"github.com/lvrach/slack-social-ai/internal/launchd"
	"github.com/lvrach/slack-social-ai/internal/manifest"
	"github.com/lvrach/slack-social-ai/internal/slack"
)

// AuthCmd manages webhook credentials.
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Configure Slack webhook (interactive or URL argument)."`
	Logout AuthLogoutCmd `cmd:"" help:"Remove webhook credentials from keychain."`
	Status AuthStatusCmd `cmd:"" default:"withargs" help:"Check webhook configuration status."`
}

// AuthLoginCmd configures the Slack webhook interactively or via argument.
type AuthLoginCmd struct {
	WebhookURL string `arg:"" optional:"" help:"Slack webhook URL (skips interactive prompt)."`
}

func (cmd *AuthLoginCmd) Run(globals *Globals) error {
	// Non-interactive — webhook URL passed as argument.
	if cmd.WebhookURL != "" {
		return cmd.storeAndVerify(globals, cmd.WebhookURL)
	}

	// Check if already configured.
	existing, err := keyring.Get()
	if err == nil && existing != "" {
		return cmd.handleExisting(globals, existing)
	}

	// Interactive paths.
	return cmd.interactive(globals)
}

func (cmd *AuthLoginCmd) handleExisting(globals *Globals, existing string) error {
	var choice string
	err := runField(
		huh.NewSelect[string]().
			Title("Slack webhook is already configured.").
			Options(
				huh.NewOption("Test existing webhook", "test"),
				huh.NewOption("Replace with a new URL", "overwrite"),
				huh.NewOption("Exit", "exit"),
			).
			Value(&choice),
	)
	if err != nil {
		return err
	}

	switch choice {
	case "test":
		return cmd.sendGreeting(globals, existing)
	case "overwrite":
		return cmd.interactive(globals)
	default:
		return nil
	}
}

func (cmd *AuthLoginCmd) interactive(globals *Globals) error {
	fmt.Println()
	fmt.Println("  Welcome to slack-social-ai!")
	fmt.Println("  Let's set up your Slack webhook.")
	fmt.Println()

	var hasWebhook bool
	err := runField(
		huh.NewConfirm().
			Title("Do you already have a Slack webhook URL?").
			Affirmative("Yes").
			Negative("No, guide me through setup").
			Value(&hasWebhook),
	)
	if err != nil {
		return err
	}

	if !hasWebhook {
		if err := cmd.guidedSetup(); err != nil {
			return err
		}
	}

	var webhookURL string
	err = runField(
		huh.NewInput().
			Title("Paste your Slack webhook URL:").
			Placeholder("https://hooks.slack.com/services/T.../B.../xxx").
			Validate(validateWebhookURL).
			Value(&webhookURL),
	)
	if err != nil {
		return err
	}

	return cmd.storeAndVerify(globals, webhookURL)
}

const maxAppNameLen = 35

func defaultAppName() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		name := u.Username + "'s Claude"
		if len(name) > maxAppNameLen {
			name = name[:maxAppNameLen]
		}
		return name
	}
	return "slack-social-ai"
}

func (cmd *AuthLoginCmd) guidedSetup() error {
	defName := defaultAppName()
	var appName string
	err := runField(
		huh.NewInput().
			Title("App name for Slack:").
			Placeholder(defName).
			CharLimit(35).
			Value(&appName),
	)
	if err != nil {
		return err
	}

	if strings.TrimSpace(appName) == "" {
		appName = defName
	}

	manifestJSON := manifest.Generate(appName)

	clipCmd := exec.Command("pbcopy")
	clipCmd.Stdin = strings.NewReader(manifestJSON)
	copied := clipCmd.Run() == nil

	url := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Underline(true).
		Render("https://api.slack.com/apps?new_app=1")

	var title string
	var desc string
	if copied {
		title = "Manifest copied to clipboard!"
		desc = "**Create the Slack app:**\n" +
			"1. Go to " + url + "\n" +
			"2. Select \"From a manifest\"\n" +
			"3. Choose your workspace\n" +
			"4. Switch to **JSON** tab and paste the manifest\n" +
			"5. Click \"Create\"\n\n" +
			"**Get the webhook URL:**\n" +
			"6. Go to \"Incoming Webhooks\" in the sidebar\n" +
			"7. Click \"Add New Webhook to Workspace\"\n" +
			"8. Pick a channel and authorize\n" +
			"9. Copy the webhook URL"
	} else {
		title = "Copy this manifest"
		fmt.Println(manifestJSON)
		desc = "**Create the Slack app:**\n" +
			"1. Copy the manifest printed above\n" +
			"2. Go to " + url + "\n" +
			"3. Select \"From a manifest\"\n" +
			"4. Choose your workspace\n" +
			"5. Switch to **JSON** tab and paste the manifest\n" +
			"6. Click \"Create\"\n\n" +
			"**Get the webhook URL:**\n" +
			"7. Go to \"Incoming Webhooks\" in the sidebar\n" +
			"8. Click \"Add New Webhook to Workspace\"\n" +
			"9. Pick a channel and authorize\n" +
			"10. Copy the webhook URL"
	}

	return runField(
		huh.NewNote().
			Title(title).
			Description(desc).
			Next(true).
			NextLabel("I have my webhook URL"),
	)
}

func (cmd *AuthLoginCmd) storeAndVerify(globals *Globals, webhookURL string) error {
	if err := validateWebhookURL(webhookURL); err != nil {
		return newCLIError(ExitInvalidInput, "invalid_url", err.Error())
	}

	// Send a greeting to verify the webhook and confirm setup.
	if err := cmd.sendGreeting(globals, webhookURL); err != nil {
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

func (cmd *AuthLoginCmd) sendGreeting(globals *Globals, webhookURL string) error {
	if !globals.JSON {
		fmt.Print("Verifying webhook... ")
	}
	if err := slack.SendWebhook(webhookURL, "slack-social-ai is connected!"); err != nil {
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

// AuthLogoutCmd removes webhook credentials from keychain.
type AuthLogoutCmd struct{}

func (cmd *AuthLogoutCmd) Run(globals *Globals) error {
	// Check if credentials exist first.
	_, err := keyring.Get()
	if err != nil {
		if keyring.IsNotFound(err) {
			msg := "No webhook credentials found."
			if globals.JSON {
				printSuccessJSON(msg)
			} else {
				printSuccessHuman(msg)
			}
			return nil
		}
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to read keychain: %s", err))
	}

	// Warn if launchd timer is installed.
	if launchd.IsInstalled() {
		if !globals.JSON {
			fmt.Fprintln(os.Stderr, "Warning: background timer is installed. It will fail without credentials.")
			fmt.Fprintln(os.Stderr, "Run `slack-social-ai schedule uninstall` to remove the timer.")
		}
	}

	if err := keyring.Delete(); err != nil {
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to remove credentials: %s", err))
	}

	msg := "Webhook credentials removed from keychain."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

// AuthStatusCmd checks webhook configuration status.
type AuthStatusCmd struct {
	Verify bool `help:"Silently verify the webhook is working (no message sent)." short:"v"`
}

func (cmd *AuthStatusCmd) Run(globals *Globals) error {
	webhookURL, err := keyring.Get()
	if err != nil {
		if keyring.IsNotFound(err) {
			return cmd.printNotConfigured(globals)
		}
		return newCLIError(ExitRuntimeError, "keyring_error",
			fmt.Sprintf("Failed to read keychain: %s", err))
	}

	// Validate URL format.
	urlValid := validateWebhookURL(webhookURL) == nil

	// Mask the URL for display: show prefix only.
	urlPrefix := maskWebhookURL(webhookURL)

	// Optional: silent verify.
	var verified *bool
	if cmd.Verify {
		v := slack.VerifyWebhook(webhookURL) == nil
		verified = &v
	}

	if globals.JSON {
		return cmd.printJSON(urlPrefix, urlValid, verified)
	}
	return cmd.printHuman(urlPrefix, urlValid, verified)
}

func (cmd *AuthStatusCmd) printNotConfigured(globals *Globals) error {
	if globals.JSON {
		resp := map[string]any{"configured": false}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintln(os.Stdout, "Webhook: not configured")
		fmt.Fprintln(os.Stdout, "Run `slack-social-ai auth login` to set up.")
	}
	return nil
}

func (cmd *AuthStatusCmd) printJSON(urlPrefix string, urlValid bool, verified *bool) error {
	resp := map[string]any{
		"configured":         true,
		"webhook_url_prefix": urlPrefix,
		"url_valid":          urlValid,
	}
	if verified != nil {
		resp["verified"] = *verified
	}
	b, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

func (cmd *AuthStatusCmd) printHuman(urlPrefix string, urlValid bool, verified *bool) error {
	fmt.Fprintf(os.Stdout, "Webhook: configured (%s)\n", urlPrefix)
	if !urlValid {
		fmt.Fprintln(os.Stdout, "Warning: URL format is invalid.")
	}
	if verified != nil {
		if *verified {
			fmt.Fprintln(os.Stdout, "Verification: ok (no message sent)")
		} else {
			fmt.Fprintln(os.Stdout, "Verification: failed — webhook may be expired or revoked")
		}
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

// maskWebhookURL returns just the protocol + host + first path segment.
func maskWebhookURL(url string) string {
	// "https://hooks.slack.com/services/T.../B.../xxx" -> "https://hooks.slack.com/services/T..."
	parts := strings.SplitN(url, "/services/", 2)
	if len(parts) == 2 {
		// Show just the first segment (team ID prefix).
		service := parts[1]
		if idx := strings.Index(service, "/"); idx > 0 {
			return parts[0] + "/services/" + service[:idx] + "/..."
		}
	}
	return "https://hooks.slack.com/..."
}
