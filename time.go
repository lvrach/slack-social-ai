package main

import (
	"fmt"
	"regexp"
	"time"
)

// parseAt parses a time specification into an absolute time.
// Supports: RFC3339, HH:MM (24-hour, local time), Go durations (2h, 30m).
func parseAt(input string) (time.Time, error) {
	return parseAtFrom(input, time.Now())
}

// parseAtFrom is the testable version that accepts a reference time.
func parseAtFrom(input string, now time.Time) (time.Time, error) {
	// 1. RFC3339 (most specific -- check first).
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}

	// 2. HH:MM (24-hour format, local time).
	if matched, _ := regexp.MatchString(`^\d{1,2}:\d{2}$`, input); matched {
		t, err := time.Parse("15:04", input)
		if err != nil {
			return time.Time{}, newCLIError(ExitInvalidInput, "invalid_time",
				fmt.Sprintf("Invalid time %q: %s", input, err))
		}
		result := time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), 0, 0, now.Location())
		if result.Before(now) {
			// Use AddDate to preserve wall-clock time across DST transitions.
			result = time.Date(now.Year(), now.Month(), now.Day()+1,
				t.Hour(), t.Minute(), 0, 0, now.Location())
		}
		return result, nil
	}

	// 3. Go duration ("2h", "30m").
	if dur, err := time.ParseDuration(input); err == nil {
		if dur <= 0 {
			return time.Time{}, newCLIError(ExitInvalidInput, "invalid_time",
				"Duration must be positive.")
		}
		return now.Add(dur), nil
	}

	// Error with helpful message.
	return time.Time{}, newCLIError(ExitInvalidInput, "invalid_time",
		fmt.Sprintf("Cannot parse %q. Use HH:MM, a duration (2h, 30m), or RFC3339.", input))
}
