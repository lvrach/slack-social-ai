package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/launchd"
)

// ScheduleCmd manages the automatic publishing schedule (macOS launchd).
type ScheduleCmd struct {
	Install   ScheduleInstallCmd   `cmd:"" help:"Install the macOS launchd publishing schedule."`
	Uninstall ScheduleUninstallCmd `cmd:"" help:"Remove the launchd publishing schedule."`
	Status    ScheduleStatusCmd    `cmd:"" help:"Show whether the schedule is active."`
}

// ScheduleInstallCmd installs the launchd timer with optional schedule config.
type ScheduleInstallCmd struct {
	PostEvery string `help:"Minimum time between posts (e.g. 3h, 30m). Default: no limit." short:"p"`
	Hours     string `help:"Active hours range (e.g. 9-22). Default: 9-18."`
	Weekdays  string `help:"Active weekdays (e.g. mon-fri). Default: mon-fri."`
}

func (cmd *ScheduleInstallCmd) Run(globals *Globals) error {
	// Build schedule from defaults, overriding with flags.
	sched := config.DefaultSchedule()

	if cmd.PostEvery != "" {
		dur, err := time.ParseDuration(cmd.PostEvery)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_post_every",
				fmt.Sprintf("Invalid --post-every value %q: %s", cmd.PostEvery, err))
		}
		sched.PostEveryMinutes = int(dur.Minutes())
	}

	if cmd.Hours != "" {
		start, end, err := config.ParseHours(cmd.Hours)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_hours",
				fmt.Sprintf("Invalid --hours value: %s", err))
		}
		sched.StartHour = start
		sched.EndHour = end
	}

	if cmd.Weekdays != "" {
		days, err := config.ParseWeekdays(cmd.Weekdays)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_weekdays",
				fmt.Sprintf("Invalid --weekdays value: %s", err))
		}
		sched.Weekdays = days
	}

	// Save config.
	cfg := config.Config{Schedule: sched}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Resolve binary path.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	// Install launchd plist.
	if err := launchd.Install(execPath); err != nil {
		return newCLIError(ExitRuntimeError, "install_failed",
			fmt.Sprintf("Failed to install schedule: %s", err))
	}

	// Format schedule summary.
	summary := formatScheduleSummary(sched)

	if globals.JSON {
		printSuccessJSON(fmt.Sprintf("Schedule installed. %s", summary))
	} else {
		fmt.Fprintf(os.Stdout, "Schedule installed. Waking every 15m. %s\n", summary)
		fmt.Fprintln(os.Stdout, "Note: If macOS asks for Keychain access, click 'Always Allow'.")
	}
	return nil
}

// ScheduleUninstallCmd removes the launchd timer.
type ScheduleUninstallCmd struct{}

func (cmd *ScheduleUninstallCmd) Run(globals *Globals) error {
	if err := launchd.Uninstall(); err != nil {
		return fmt.Errorf("uninstall schedule: %w", err)
	}
	msg := "Schedule removed."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

// ScheduleStatusCmd shows the current schedule status.
type ScheduleStatusCmd struct{}

func (cmd *ScheduleStatusCmd) Run(globals *Globals) error { //nolint:unparam // error required by Kong cmd interface
	installed := launchd.IsInstalled()

	if !installed {
		if globals.JSON {
			resp := map[string]string{"status": "not_configured"}
			b, _ := json.Marshal(resp)
			fmt.Fprintln(os.Stdout, string(b))
		} else {
			fmt.Fprintln(os.Stdout, "Not configured. Run `slack-social-ai schedule install` to set up.")
		}
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Config{Schedule: config.DefaultSchedule()}
	}

	queued, _ := history.Queued()
	lastPublished, _ := history.LastPublishedTime()

	if globals.JSON {
		resp := map[string]any{
			"status": "active",
			"schedule": map[string]any{
				"post_every_minutes": cfg.Schedule.PostEveryMinutes,
				"start_hour":         cfg.Schedule.StartHour,
				"end_hour":           cfg.Schedule.EndHour,
				"weekdays":           cfg.Schedule.Weekdays,
			},
			"queued_count": len(queued),
		}
		if !lastPublished.IsZero() {
			resp["last_published"] = lastPublished.UTC().Format(time.RFC3339)
		}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintln(os.Stdout, "Status: Active")
		fmt.Fprintf(os.Stdout, "Schedule: %s\n", formatScheduleSummary(cfg.Schedule))
		fmt.Fprintf(os.Stdout, "Queued messages: %d\n", len(queued))
		if !lastPublished.IsZero() {
			ago := time.Since(lastPublished).Truncate(time.Minute)
			fmt.Fprintf(os.Stdout, "Last published: %s (%s ago)\n",
				lastPublished.Local().Format("2006-01-02 15:04"), ago)
		} else {
			fmt.Fprintln(os.Stdout, "Last published: never")
		}
	}
	return nil
}

// formatScheduleSummary returns a human-readable schedule description.
func formatScheduleSummary(s config.Schedule) string {
	days := formatWeekdays(s.Weekdays)
	hours := fmt.Sprintf("%02d:00–%02d:00", s.StartHour, s.EndHour)

	summary := fmt.Sprintf("Publishing: %s %s", days, hours)
	if s.PostEveryMinutes > 0 {
		dur := time.Duration(s.PostEveryMinutes) * time.Minute
		summary += fmt.Sprintf(", max every %s", dur)
	}
	return summary
}

// formatWeekdays converts ["mon","tue","wed","thu","fri"] → "Mon–Fri".
func formatWeekdays(days []string) string {
	if len(days) == 0 {
		return "no days"
	}
	if len(days) == 7 {
		return "every day"
	}
	if len(days) == 1 {
		return capitalize(days[0])
	}
	return capitalize(days[0]) + "–" + capitalize(days[len(days)-1])
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
