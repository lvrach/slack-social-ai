package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/keyring"
	"github.com/lvrach/slack-social-ai/internal/launchd"
	"github.com/lvrach/slack-social-ai/internal/schedule"
)

// InitCmd is a first-run wizard that orchestrates auth, schedule, and timer setup.
type InitCmd struct{}

func (cmd *InitCmd) Run(globals *Globals) error {
	// Step 1 — Auth.
	if err := cmd.stepAuth(globals); err != nil {
		return err
	}

	// Step 2 — Schedule.
	if err := cmd.stepSchedule(globals); err != nil {
		return err
	}

	// Step 3 — Timer.
	if err := cmd.stepTimer(); err != nil {
		return err
	}

	// Summary.
	cmd.printSummary()
	return nil
}

func (cmd *InitCmd) stepAuth(globals *Globals) error {
	existing, err := keyring.Get()
	if err == nil && existing != "" {
		fmt.Fprintln(os.Stdout, "Webhook: configured")
		var reconfigure bool
		err := runField(
			huh.NewConfirm().
				Title("Reconfigure webhook?").
				Affirmative("Yes").
				Negative("Keep current").
				Value(&reconfigure),
		)
		if err != nil {
			return err
		}
		if !reconfigure {
			return nil
		}
	}

	// Delegate to auth login.
	login := &AuthLoginCmd{}
	return login.Run(globals)
}

func (cmd *InitCmd) stepSchedule(globals *Globals) error {
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			fmt.Fprintf(os.Stdout, "Schedule: %s\n", formatScheduleSummary(cfg.Schedule))
			var modify bool
			err := runField(
				huh.NewConfirm().
					Title("Modify schedule?").
					Affirmative("Yes").
					Negative("Keep current").
					Value(&modify),
			)
			if err != nil {
				return err
			}
			if !modify {
				return nil
			}
			// Delegate to interactive schedule setup.
			set := &ScheduleSetCmd{}
			return set.Run(globals)
		}
	}

	// No config exists — offer to set up automatic publishing.
	var enableSchedule bool
	err := runField(
		huh.NewConfirm().
			Title("Enable automatic publishing?").
			Description("Posts you queue will be published automatically during your active hours.\n" +
				"A background timer checks every 10 minutes and publishes the next queued\n" +
				"post if your schedule allows it (right day, right hours, enough time since\n" +
				"the last post).\n\n" +
				"  Active hours: 09:00–17:00 (customize with `schedule set -H`)\n" +
				"  Active days:  Mon–Fri     (customize with `schedule set -w`)\n" +
				"  Post spacing: every 3h    (customize with `schedule set -p`)").
			Affirmative("Yes").
			Negative("Not now").
			Value(&enableSchedule),
	)
	if err != nil {
		return err
	}
	if !enableSchedule {
		return nil
	}

	// Save default schedule.
	cfg := config.Config{Schedule: schedule.DefaultSchedule()}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	sched := schedule.DefaultSchedule()
	fmt.Fprintf(os.Stdout, "Schedule saved. %s\n", formatScheduleSummary(sched))
	return nil
}

func (cmd *InitCmd) stepTimer() error {
	if !config.Exists() {
		// No schedule configured — skip timer setup.
		return nil
	}

	if launchd.IsInstalled() {
		loaded := "not loaded"
		if launchd.IsLoaded() {
			loaded = "loaded"
		}
		fmt.Fprintf(os.Stdout, "Timer: installed (%s)\n", loaded)

		var reinstall bool
		err := runField(
			huh.NewConfirm().
				Title("Reinstall timer? (useful after `go install` changes the binary path)").
				Affirmative("Yes").
				Negative("Keep current").
				Value(&reinstall),
		)
		if err != nil {
			return err
		}
		if !reinstall {
			return nil
		}
	}

	// Resolve binary path.
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not resolve executable path: %s\n", err)
		return nil
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not resolve symlinks: %s\n", err)
		return nil
	}

	if err := launchd.Install(execPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not install timer: %s\n", err)
		return nil
	}

	fmt.Fprintln(os.Stdout, "Background timer installed.")
	fmt.Fprintf(os.Stdout, "  Plist: %s\n", launchd.PlistPath())
	fmt.Fprintf(os.Stdout, "  Logs:  %s\n", launchd.LogPath())
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "The timer wakes every 10 minutes. All scheduling logic (hours, weekdays,")
	fmt.Fprintln(os.Stdout, "frequency) is in the CLI — the timer is just a trigger.")
	return nil
}

func (cmd *InitCmd) printSummary() {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Setup complete.")

	// Webhook status.
	if _, err := keyring.Get(); err == nil {
		fmt.Fprintln(os.Stdout, "  Webhook:  configured")
	} else {
		fmt.Fprintln(os.Stdout, "  Webhook:  not configured")
	}

	// Schedule status.
	if cfg, err := config.Load(); err == nil && config.Exists() {
		sched := cfg.Schedule
		fmt.Fprintf(os.Stdout, "  Schedule: %s %02d:00–%02d:00, every %s\n",
			formatWeekdays(sched.Weekdays), sched.StartHour, sched.EndHour,
			sched.PostEvery())
	} else {
		fmt.Fprintln(os.Stdout, "  Schedule: not configured")
	}

	// Timer status.
	if launchd.IsInstalled() {
		fmt.Fprintln(os.Stdout, "  Timer:    installed")
	} else {
		fmt.Fprintln(os.Stdout, "  Timer:    not installed")
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Next: slack-social-ai post \"Hello from the terminal!\"")
}

// runField wraps a single huh field in a form that supports
// Ctrl+C and Ctrl+D for quitting, with bottom margin styling.
func runField(field huh.Field) error {
	return runForm(huh.NewForm(huh.NewGroup(field)))
}

// runForm applies consistent keybindings and theme to a multi-field form.
// Same Ctrl+C/D quit bindings and margin styling as runField.
func runForm(form *huh.Form) error {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "ctrl+d"))

	t := huh.ThemeBase()
	t.Focused.Base = t.Focused.Base.MarginBottom(1)
	t.Blurred.Base = t.Blurred.Base.MarginBottom(1)

	return form.
		WithShowHelp(true).
		WithKeyMap(km).
		WithTheme(t).
		Run()
}
