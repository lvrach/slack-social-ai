package schedule

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Schedule controls when automated posts are allowed.
type Schedule struct {
	PostEveryMinutes int      `json:"post_every_minutes"` // min time between posts (0 = no limit)
	StartHour        int      `json:"start_hour"`         // 0-23 (default: 9)
	EndHour          int      `json:"end_hour"`           // 0-23 (default: 18)
	Weekdays         []string `json:"weekdays"`           // ["mon","tue",...] (default: mon-fri)
}

// DefaultSchedule returns a schedule with sensible defaults:
// every 3h, 9-17, Monday through Friday.
func DefaultSchedule() Schedule {
	return Schedule{
		PostEveryMinutes: 180,
		StartHour:        9,
		EndHour:          17,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri"},
	}
}

// IsActiveAt checks if the schedule is active at the given time.
func (s Schedule) IsActiveAt(t time.Time) bool {
	weekday := strings.ToLower(t.Weekday().String()[:3])
	if !slices.Contains(s.Weekdays, weekday) {
		return false
	}
	hour := t.Hour()
	return hour >= s.StartHour && hour < s.EndHour
}

// IsActiveNow checks if the schedule is active right now.
func (s Schedule) IsActiveNow() bool {
	return s.IsActiveAt(time.Now())
}

// PostEvery returns the minimum interval between posts as a duration.
func (s Schedule) PostEvery() time.Duration {
	return time.Duration(s.PostEveryMinutes) * time.Minute
}

// ParseHours parses "9-22" into start and end hour.
func ParseHours(s string) (int, int, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid hours format %q, expected START-END (e.g. 9-22)", s)
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start hour: %w", err)
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end hour: %w", err)
	}

	if start < 0 || start > 23 {
		return 0, 0, fmt.Errorf("start hour %d out of range 0-23", start)
	}
	if end < 0 || end > 24 {
		return 0, 0, fmt.Errorf("end hour %d out of range 0-24", end)
	}
	if start >= end {
		return 0, 0, fmt.Errorf("start hour %d must be less than end hour %d", start, end)
	}

	return start, end, nil
}

// ParseWeekdays parses "mon-fri" or "mon,wed,fri" into a slice of weekday abbreviations.
func ParseWeekdays(s string) ([]string, error) {
	valid := []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

	// Range format: "mon-fri"
	if parts := strings.SplitN(s, "-", 2); len(parts) == 2 && !strings.Contains(s, ",") {
		start := strings.ToLower(strings.TrimSpace(parts[0]))
		end := strings.ToLower(strings.TrimSpace(parts[1]))

		startIdx := slices.Index(valid, start)
		endIdx := slices.Index(valid, end)

		if startIdx == -1 {
			return nil, fmt.Errorf("invalid weekday %q", start)
		}
		if endIdx == -1 {
			return nil, fmt.Errorf("invalid weekday %q", end)
		}
		if startIdx > endIdx {
			return nil, fmt.Errorf("start weekday %q must come before end %q", start, end)
		}

		return valid[startIdx : endIdx+1], nil
	}

	// List format: "mon,wed,fri"
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		day := strings.ToLower(strings.TrimSpace(p))
		if !slices.Contains(valid, day) {
			return nil, fmt.Errorf("invalid weekday %q", day)
		}
		result = append(result, day)
	}

	return result, nil
}
