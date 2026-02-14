package schedule

import (
	"testing"
	"time"
)

func TestDefaultSchedule(t *testing.T) {
	s := DefaultSchedule()

	if s.PostEveryMinutes != 180 {
		t.Errorf("PostEveryMinutes = %d, want 180", s.PostEveryMinutes)
	}
	if s.StartHour != 9 {
		t.Errorf("StartHour = %d, want 9", s.StartHour)
	}
	if s.EndHour != 17 {
		t.Errorf("EndHour = %d, want 17", s.EndHour)
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
	s := DefaultSchedule() // 9-17, mon-fri

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
			name: "Monday 17:00 end hour is exclusive",
			time: time.Date(2025, 1, 6, 17, 0, 0, 0, time.UTC), // Monday
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
			name: "Friday 16:59 last minute of window",
			time: time.Date(2025, 1, 10, 16, 59, 0, 0, time.UTC), // Friday
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
