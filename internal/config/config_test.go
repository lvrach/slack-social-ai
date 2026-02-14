package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lvrach/slack-social-ai/internal/schedule"
)

func withTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	original := configDir
	configDir = func() string { return dir }
	t.Cleanup(func() { configDir = original })
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	withTempConfigDir(t)

	original := Config{
		Schedule: schedule.Schedule{
			PostEveryMinutes: 60,
			StartHour:        10,
			EndHour:          20,
			Weekdays:         []string{"mon", "wed", "fri"},
		},
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Schedule.PostEveryMinutes != original.Schedule.PostEveryMinutes {
		t.Errorf("PostEveryMinutes = %d, want %d", loaded.Schedule.PostEveryMinutes, original.Schedule.PostEveryMinutes)
	}
	if loaded.Schedule.StartHour != original.Schedule.StartHour {
		t.Errorf("StartHour = %d, want %d", loaded.Schedule.StartHour, original.Schedule.StartHour)
	}
	if loaded.Schedule.EndHour != original.Schedule.EndHour {
		t.Errorf("EndHour = %d, want %d", loaded.Schedule.EndHour, original.Schedule.EndHour)
	}
	if len(loaded.Schedule.Weekdays) != len(original.Schedule.Weekdays) {
		t.Fatalf("Weekdays len = %d, want %d", len(loaded.Schedule.Weekdays), len(original.Schedule.Weekdays))
	}
	for i, d := range loaded.Schedule.Weekdays {
		if d != original.Schedule.Weekdays[i] {
			t.Errorf("Weekdays[%d] = %q, want %q", i, d, original.Schedule.Weekdays[i])
		}
	}

	// Verify file was written with correct permissions.
	info, err := os.Stat(filepath.Join(configDir(), "config.json"))
	if err != nil {
		t.Fatalf("Stat config file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}
}

func TestLoad_Missing(t *testing.T) {
	withTempConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	defaults := schedule.DefaultSchedule()
	if cfg.Schedule.StartHour != defaults.StartHour {
		t.Errorf("StartHour = %d, want %d", cfg.Schedule.StartHour, defaults.StartHour)
	}
	if cfg.Schedule.EndHour != defaults.EndHour {
		t.Errorf("EndHour = %d, want %d", cfg.Schedule.EndHour, defaults.EndHour)
	}
	if cfg.Schedule.PostEveryMinutes != defaults.PostEveryMinutes {
		t.Errorf("PostEveryMinutes = %d, want %d", cfg.Schedule.PostEveryMinutes, defaults.PostEveryMinutes)
	}
	if len(cfg.Schedule.Weekdays) != len(defaults.Weekdays) {
		t.Fatalf("Weekdays len = %d, want %d", len(cfg.Schedule.Weekdays), len(defaults.Weekdays))
	}
	for i, d := range cfg.Schedule.Weekdays {
		if d != defaults.Weekdays[i] {
			t.Errorf("Weekdays[%d] = %q, want %q", i, d, defaults.Weekdays[i])
		}
	}
}

func TestLoad_Corrupt(t *testing.T) {
	withTempConfigDir(t)

	// Write invalid JSON to the config file.
	path := filepath.Join(configDir(), "config.json")
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json!!!"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() with corrupt JSON should return error")
	}
}
