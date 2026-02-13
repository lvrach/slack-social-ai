package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	original := configDir
	configDir = func() string { return dir }
	t.Cleanup(func() { configDir = original })
}

func TestDefaultSchedule(t *testing.T) {
	s := DefaultSchedule()

	if s.PostEveryMinutes != 0 {
		t.Errorf("PostEveryMinutes = %d, want 0", s.PostEveryMinutes)
	}
	if s.StartHour != 9 {
		t.Errorf("StartHour = %d, want 9", s.StartHour)
	}
	if s.EndHour != 18 {
		t.Errorf("EndHour = %d, want 18", s.EndHour)
	}

	wantDays := []string{"mon", "tue", "wed", "thu", "fri"}
	if len(s.Weekdays) != len(wantDays) {
		t.Fatalf("Weekdays len = %d, want %d", len(s.Weekdays), len(wantDays))
	}
	for i, d := range s.Weekdays {
		if d != wantDays[i] {
			t.Errorf("Weekdays[%d] = %q, want %q", i, d, wantDays[i])
		}
	}
}

func TestIsActiveAt(t *testing.T) {
	s := DefaultSchedule() // 9-18, mon-fri

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{
			name: "Monday 10:00 is active",
			time: time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC), // Monday
			want: true,
		},
		{
			name: "Monday 08:59 is before start",
			time: time.Date(2025, 1, 6, 8, 59, 0, 0, time.UTC), // Monday
			want: false,
		},
		{
			name: "Monday 18:00 end hour is exclusive",
			time: time.Date(2025, 1, 6, 18, 0, 0, 0, time.UTC), // Monday
			want: false,
		},
		{
			name: "Saturday 12:00 is weekend",
			time: time.Date(2025, 1, 11, 12, 0, 0, 0, time.UTC), // Saturday
			want: false,
		},
		{
			name: "Sunday 10:00 is weekend",
			time: time.Date(2025, 1, 12, 10, 0, 0, 0, time.UTC), // Sunday
			want: false,
		},
		{
			name: "Friday 17:59 last minute of window",
			time: time.Date(2025, 1, 10, 17, 59, 0, 0, time.UTC), // Friday
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.IsActiveAt(tt.time)
			if got != tt.want {
				t.Errorf("IsActiveAt(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestIsActiveAt_CustomSchedule(t *testing.T) {
	s := Schedule{
		StartHour: 9,
		EndHour:   22,
		Weekdays:  []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}

	// Saturday 15:00 should be active with this custom schedule.
	sat := time.Date(2025, 1, 11, 15, 0, 0, 0, time.UTC) // Saturday
	if !s.IsActiveAt(sat) {
		t.Errorf("IsActiveAt(Saturday 15:00) = false, want true for custom schedule")
	}
}

func TestPostEvery(t *testing.T) {
	tests := []struct {
		minutes int
		want    time.Duration
	}{
		{0, 0},
		{180, 3 * time.Hour},
	}

	for _, tt := range tests {
		s := Schedule{PostEveryMinutes: tt.minutes}
		got := s.PostEvery()
		if got != tt.want {
			t.Errorf("PostEvery() with %d minutes = %v, want %v", tt.minutes, got, tt.want)
		}
	}
}

func TestParseHours(t *testing.T) {
	tests := []struct {
		input     string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{"9-22", 9, 22, false},
		{"0-24", 0, 24, false},
		{"25-10", 0, 0, true},  // start out of range
		{"abc", 0, 0, true},    // not START-END format
		{"10-5", 0, 0, true},   // start >= end
		{"9-25", 0, 0, true},   // end out of range
		{"-1-10", 0, 0, true},  // negative start
		{"abc-10", 0, 0, true}, // non-numeric start
		{"9-abc", 0, 0, true},  // non-numeric end
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			start, end, err := ParseHours(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseHours(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil {
				if start != tt.wantStart {
					t.Errorf("start = %d, want %d", start, tt.wantStart)
				}
				if end != tt.wantEnd {
					t.Errorf("end = %d, want %d", end, tt.wantEnd)
				}
			}
		})
	}
}

func TestParseWeekdays(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "range mon-fri",
			input: "mon-fri",
			want:  []string{"mon", "tue", "wed", "thu", "fri"},
		},
		{
			name:  "list mon,wed,fri",
			input: "mon,wed,fri",
			want:  []string{"mon", "wed", "fri"},
		},
		{
			name:  "full week range",
			input: "mon-sun",
			want:  []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
		},
		{
			name:    "invalid weekday",
			input:   "xyz",
			wantErr: true,
		},
		{
			name:    "invalid range start",
			input:   "xyz-fri",
			wantErr: true,
		},
		{
			name:    "invalid range end",
			input:   "mon-xyz",
			wantErr: true,
		},
		{
			name:    "reversed range",
			input:   "fri-mon",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWeekdays(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseWeekdays(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil {
				if len(got) != len(tt.want) {
					t.Fatalf("len = %d, want %d", len(got), len(tt.want))
				}
				for i, d := range got {
					if d != tt.want[i] {
						t.Errorf("got[%d] = %q, want %q", i, d, tt.want[i])
					}
				}
			}
		})
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	withTempConfigDir(t)

	original := Config{
		Schedule: Schedule{
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

	defaults := DefaultSchedule()
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
