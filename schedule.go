package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/launchd"
	"github.com/lvrach/slack-social-ai/internal/schedule"
)

// ScheduleCmd configures the publishing schedule (hours, weekdays, frequency).
type ScheduleCmd struct {
	Set       ScheduleSetCmd       `cmd:"" help:"Configure schedule (interactive, or use flags)."`
	Status    ScheduleStatusCmd    `cmd:"" help:"Show current schedule and queue."`
	Install   ScheduleInstallCmd   `cmd:"" help:"Install background timer for automatic publishing."`
	Uninstall ScheduleUninstallCmd `cmd:"" help:"Remove the background timer."`
}

// ScheduleSetCmd configures the schedule.
// With flags: saves config directly. Without flags: interactive setup.
type ScheduleSetCmd struct {
	PostEvery string `help:"Minimum time between posts (e.g. 3h, 30m)." short:"p"`
	Hours     string `help:"Active hours range (e.g. 9-17)." short:"H"`
	Weekdays  string `help:"Active weekdays (e.g. mon-fri)." short:"w"`
}

func (cmd *ScheduleSetCmd) Run(globals *Globals) error {
	// If flags provided, save directly.
	if cmd.PostEvery != "" || cmd.Hours != "" || cmd.Weekdays != "" {
		return cmd.saveFromFlags(globals)
	}

	// No flags — interactive setup.
	return cmd.interactive(globals)
}

func (cmd *ScheduleSetCmd) saveFromFlags(globals *Globals) error {
	// Start from existing config or defaults.
	sched := schedule.DefaultSchedule()
	if existing, err := config.Load(); err == nil {
		sched = existing.Schedule
	}

	if cmd.PostEvery != "" {
		dur, err := time.ParseDuration(cmd.PostEvery)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_post_every",
				fmt.Sprintf("Invalid --post-every value %q: %s", cmd.PostEvery, err))
		}
		if dur <= 0 {
			return newCLIError(ExitInvalidInput, "invalid_post_every",
				"The --post-every value must be positive.")
		}
		sched.PostEveryMinutes = int(dur.Minutes())
	}

	if cmd.Hours != "" {
		start, end, err := schedule.ParseHours(cmd.Hours)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_hours",
				fmt.Sprintf("Invalid --hours value: %s", err))
		}
		sched.StartHour = start
		sched.EndHour = end
	}

	if cmd.Weekdays != "" {
		days, err := schedule.ParseWeekdays(cmd.Weekdays)
		if err != nil {
			return newCLIError(ExitInvalidInput, "invalid_weekdays",
				fmt.Sprintf("Invalid --weekdays value: %s", err))
		}
		sched.Weekdays = days
	}

	return saveSchedule(globals, sched)
}

func (cmd *ScheduleSetCmd) interactive(globals *Globals) error {
	// Start from existing config or defaults.
	sched := schedule.DefaultSchedule()
	if existing, err := config.Load(); err == nil {
		sched = existing.Schedule
	}

	// Frequency: Select with current value moved to top (huh viewport bug workaround).
	freqOptions := buildFreqOptions(sched.PostEveryMinutes)
	frequency := sched.PostEveryMinutes

	if err := runField(
		huh.NewSelect[int]().
			Title("Minimum time between posts:").
			Height(len(freqOptions) + 2).
			Value(&frequency).
			Options(freqOptions...),
	); err != nil {
		return err
	}

	// Hours: Input fields (avoids huh Select viewport scroll bug with 24 items).
	hours := fmt.Sprintf("%d-%d", sched.StartHour, sched.EndHour)
	err := runField(
		huh.NewInput().
			Title("Active hours range:").
			Placeholder("9-17").
			Description(fmt.Sprintf("24-hour format START-END. Currently: %s–%s.",
				formatHourLabel(sched.StartHour), formatHourLabel(sched.EndHour))).
			Value(&hours),
	)
	if err != nil {
		return err
	}
	start, end, parseErr := schedule.ParseHours(hours)
	if parseErr != nil {
		return newCLIError(ExitInvalidInput, "invalid_hours",
			fmt.Sprintf("Invalid hours value: %s", parseErr))
	}

	// Weekdays: MultiSelect (7 items, all visible).
	weekdays := sched.Weekdays
	weekdayOptions := []huh.Option[string]{
		huh.NewOption("Monday", "mon"),
		huh.NewOption("Tuesday", "tue"),
		huh.NewOption("Wednesday", "wed"),
		huh.NewOption("Thursday", "thu"),
		huh.NewOption("Friday", "fri"),
		huh.NewOption("Saturday", "sat"),
		huh.NewOption("Sunday", "sun"),
	}

	if err := runField(
		huh.NewMultiSelect[string]().
			Title("Active weekdays:").
			Height(9).
			Options(weekdayOptions...).
			Value(&weekdays).
			Validate(func(v []string) error {
				if len(v) == 0 {
					return fmt.Errorf("select at least one day")
				}
				return nil
			}),
	); err != nil {
		return err
	}

	sched.PostEveryMinutes = frequency
	sched.StartHour = start
	sched.EndHour = end
	sched.Weekdays = weekdays

	return saveSchedule(globals, sched)
}

// formatHourLabel converts a 24-hour int to a human-readable label.
func formatHourLabel(h int) string {
	switch {
	case h == 0:
		return "12am (midnight)"
	case h < 12:
		return fmt.Sprintf("%dam", h)
	case h == 12:
		return "12pm (noon)"
	default:
		return fmt.Sprintf("%dpm", h-12)
	}
}

func saveSchedule(globals *Globals, sched schedule.Schedule) error {
	cfg := config.Config{Schedule: sched}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	summary := formatScheduleSummary(sched)
	if globals.JSON {
		printSuccessJSON(fmt.Sprintf("Schedule updated. %s", summary))
	} else {
		fmt.Fprintf(os.Stdout, "Schedule updated. %s\n", summary)
	}

	// Hint about timer if not installed.
	if !globals.JSON && !launchd.IsInstalled() {
		fmt.Fprintln(os.Stdout, "Timer not installed. Run `slack-social-ai schedule install` to activate.")
	}

	return nil
}

// ScheduleStatusCmd shows the current schedule config and queue.
type ScheduleStatusCmd struct{}

func (cmd *ScheduleStatusCmd) Run(globals *Globals) error { //nolint:unparam // error required by Kong cmd interface
	if !config.Exists() {
		if globals.JSON {
			resp := map[string]string{"status": "not_configured"}
			b, _ := json.Marshal(resp)
			fmt.Fprintln(os.Stdout, string(b))
		} else {
			fmt.Fprintln(os.Stdout, "Not configured. Run `slack-social-ai schedule set` to set up.")
		}
		return nil
	}

	cfg, _ := config.Load()

	queued, _ := history.Queued()
	lastPublished, _ := history.LastPublishedTime()

	if globals.JSON {
		resp := map[string]any{
			"status": "configured",
			"schedule": map[string]any{
				"post_every_minutes": cfg.Schedule.PostEveryMinutes,
				"start_hour":         cfg.Schedule.StartHour,
				"end_hour":           cfg.Schedule.EndHour,
				"weekdays":           cfg.Schedule.Weekdays,
			},
			"queued_count":    len(queued),
			"timer_installed": launchd.IsInstalled(),
			"timer_loaded":    launchd.IsLoaded(),
			"plist_path":      launchd.PlistPath(),
			"log_path":        launchd.LogPath(),
		}
		if !lastPublished.IsZero() {
			resp["last_published"] = lastPublished.UTC().Format(time.RFC3339)
		}
		b, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stdout, string(b))
	} else {
		fmt.Fprintf(os.Stdout, "Schedule: %s\n", formatScheduleSummary(cfg.Schedule))
		fmt.Fprintf(os.Stdout, "Queued messages: %d\n", len(queued))
		if !lastPublished.IsZero() {
			ago := time.Since(lastPublished).Truncate(time.Minute)
			fmt.Fprintf(os.Stdout, "Last published: %s (%s ago)\n",
				lastPublished.Local().Format("2006-01-02 15:04"), ago)
		} else {
			fmt.Fprintln(os.Stdout, "Last published: never")
		}

		// Timer info.
		if launchd.IsInstalled() {
			loaded := "not loaded"
			if launchd.IsLoaded() {
				loaded = "loaded"
			}
			fmt.Fprintf(os.Stdout, "Timer: installed (%s)\n", loaded)
			fmt.Fprintf(os.Stdout, "  Plist: %s\n", launchd.PlistPath())
			fmt.Fprintf(os.Stdout, "  Logs:  %s\n", launchd.LogPath())
		} else {
			fmt.Fprintln(os.Stdout, "Timer: not installed")
			fmt.Fprintln(os.Stdout, "  Run `slack-social-ai schedule install` to activate.")
		}
	}
	return nil
}

// ScheduleInstallCmd installs the background timer for automatic publishing.
type ScheduleInstallCmd struct{}

func (cmd *ScheduleInstallCmd) Run(globals *Globals) error {
	// Resolve the binary path.
	execPath, err := os.Executable()
	if err != nil {
		return newCLIError(ExitRuntimeError, "exec_path",
			fmt.Sprintf("Could not resolve executable path: %s", err))
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return newCLIError(ExitRuntimeError, "exec_path",
			fmt.Sprintf("Could not resolve symlinks: %s", err))
	}

	// Save default schedule if no config exists.
	if !config.Exists() {
		cfg := config.Config{Schedule: schedule.DefaultSchedule()}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save default config: %w", err)
		}
	}

	if err := launchd.Install(execPath); err != nil {
		return newCLIError(ExitRuntimeError, "install_failed",
			fmt.Sprintf("Failed to install timer: %s", err))
	}

	if globals.JSON {
		printSuccessJSON("Background timer installed.")
	} else {
		fmt.Fprintln(os.Stdout, "Background timer installed.")
		fmt.Fprintf(os.Stdout, "  Plist: %s\n", launchd.PlistPath())
		fmt.Fprintf(os.Stdout, "  Logs:  %s\n", launchd.LogPath())
		fmt.Fprintln(os.Stdout, "\nThe timer wakes every 10 minutes. All scheduling logic (hours, weekdays,")
		fmt.Fprintln(os.Stdout, "frequency) is in the CLI — the timer is just a trigger.")
	}
	return nil
}

// ScheduleUninstallCmd removes the background timer.
type ScheduleUninstallCmd struct{}

func (cmd *ScheduleUninstallCmd) Run(globals *Globals) error {
	if !launchd.IsInstalled() {
		msg := "No timer installed."
		if globals.JSON {
			printSuccessJSON(msg)
		} else {
			printSuccessHuman(msg)
		}
		return nil
	}

	if err := launchd.Uninstall(); err != nil {
		return newCLIError(ExitRuntimeError, "uninstall_failed",
			fmt.Sprintf("Failed to remove timer: %s", err))
	}

	msg := "Background timer removed."
	if globals.JSON {
		printSuccessJSON(msg)
	} else {
		printSuccessHuman(msg)
	}
	return nil
}

// formatScheduleSummary returns a human-readable schedule description.
func formatScheduleSummary(s schedule.Schedule) string {
	days := formatWeekdays(s.Weekdays)
	hours := fmt.Sprintf("%02d:00–%02d:00", s.StartHour, s.EndHour)

	summary := fmt.Sprintf("Publishing: %s %s", days, hours)
	if s.PostEveryMinutes > 0 {
		dur := time.Duration(s.PostEveryMinutes) * time.Minute
		summary += fmt.Sprintf(", max every %s", dur)
	}
	return summary
}

// formatWeekdays converts a list of day abbreviations to a human-readable string.
// Consecutive days use range notation ("Mon–Fri"); non-consecutive days are listed ("Mon, Wed, Fri").
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
	if isConsecutiveWeekdays(days) {
		return capitalize(days[0]) + "–" + capitalize(days[len(days)-1])
	}
	// Non-consecutive: list individually.
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = capitalize(d)
	}
	return strings.Join(parts, ", ")
}

// isConsecutiveWeekdays checks if the given days form a contiguous block in week order.
func isConsecutiveWeekdays(days []string) bool {
	order := map[string]int{
		"mon": 0, "tue": 1, "wed": 2, "thu": 3, "fri": 4, "sat": 5, "sun": 6,
	}
	for i := 1; i < len(days); i++ {
		if order[days[i]] != order[days[i-1]]+1 {
			return false
		}
	}
	return true
}

// buildFreqOptions returns frequency options with the current value at index 0.
// This works around a huh Select bug where the viewport starts at the selected
// item, hiding options above it (github.com/charmbracelet/huh/issues/679).
func buildFreqOptions(currentMinutes int) []huh.Option[int] {
	type freqPreset struct {
		label   string
		minutes int
	}
	presets := []freqPreset{
		{"Every 30 minutes", 30},
		{"Every 1 hour", 60},
		{"Every 3 hours", 180},
		{"Every 6 hours", 360},
		{"Every 24 hours", 1440},
		{"No limit", 0},
	}

	var options []huh.Option[int]
	var currentFound bool

	// Put the current selection first and mark it Selected so the viewport
	// starts at index 0 (workaround for huh viewport.YOffset bug).
	for _, p := range presets {
		if p.minutes == currentMinutes {
			options = append(options, huh.NewOption(p.label+" ←", p.minutes).Selected(true))
			currentFound = true
			break
		}
	}
	// Custom value not in presets.
	if !currentFound {
		dur := time.Duration(currentMinutes) * time.Minute
		options = append(options, huh.NewOption(fmt.Sprintf("Every %s (current)", dur), currentMinutes).Selected(true))
	}

	// Add the rest.
	for _, p := range presets {
		if p.minutes != currentMinutes {
			options = append(options, huh.NewOption(p.label, p.minutes))
		}
	}
	return options
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
