package main

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

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
		return cmd.verifyWebhook(globals, existing)
	case "overwrite":
		return cmd.interactive(globals)
	default:
		return nil
	}
}

func (cmd *InitCmd) interactive(globals *Globals) error {
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
		// Path A: guided setup.
		if err := cmd.guidedSetup(); err != nil {
			return err
		}
	}

	// Path A (continued) or Path B: prompt for URL.
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

func (cmd *InitCmd) guidedSetup() error {
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

	// Try to copy to clipboard (macOS).
	clipCmd := exec.Command("pbcopy")
	clipCmd.Stdin = strings.NewReader(manifestJSON)
	copied := clipCmd.Run() == nil

	url := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")). // bright blue
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
		// Print manifest to stdout since clipboard is unavailable.
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

// runField wraps a single huh field in a form that supports
// Ctrl+C and Ctrl+D for quitting, with bottom margin styling.
func runField(field huh.Field) error {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "ctrl+d"))

	t := huh.ThemeBase()
	t.Focused.Base = t.Focused.Base.MarginBottom(1)
	t.Blurred.Base = t.Blurred.Base.MarginBottom(1)

	return huh.NewForm(huh.NewGroup(field)).
		WithShowHelp(false).
		WithKeyMap(km).
		WithTheme(t).
		Run()
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
