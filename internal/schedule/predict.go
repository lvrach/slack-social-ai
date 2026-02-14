package schedule

import (
	"time"

	"github.com/lvrach/slack-social-ai/internal/history"
)

// launchdInterval is the fallback minimum interval between publish runs.
const launchdInterval = 10 * time.Minute

// Prediction represents a predicted publish time for a queued entry.
type Prediction struct {
	Entry       history.Entry
	Position    int       // 1-based queue position
	PublishAt   time.Time // predicted publish time
	Approximate bool      // true for position > 1 (depends on earlier items)
}

// PredictPublishTimes calculates predicted publish times for queued entries
// based on the schedule, last published time, and current time.
func PredictPublishTimes(
	entries []history.Entry,
	sched Schedule,
	lastPublished time.Time,
	now time.Time,
) []Prediction {
	if len(entries) == 0 {
		return nil
	}

	interval := max(sched.PostEvery(), launchdInterval)

	cursor := now

	// If last published + PostEvery > cursor, wait for the frequency guard.
	if !lastPublished.IsZero() && sched.PostEvery() > 0 {
		nextEligible := lastPublished.Add(sched.PostEvery())
		if nextEligible.After(cursor) {
			cursor = nextEligible
		}
	}

	predictions := make([]Prediction, len(entries))
	for i, entry := range entries {
		// If entry has a ScheduledAt that's after cursor, jump to it.
		if entry.ScheduledAt != "" {
			if scheduled, err := time.Parse(time.RFC3339, entry.ScheduledAt); err == nil {
				if scheduled.After(cursor) {
					cursor = scheduled
				}
			}
		}

		// Advance cursor to the next active window.
		cursor = AdvanceToActive(cursor, sched)

		predictions[i] = Prediction{
			Entry:       entry,
			Position:    i + 1,
			PublishAt:   cursor,
			Approximate: i > 0,
		}

		// Advance cursor for the next entry.
		cursor = cursor.Add(interval)
	}

	return predictions
}

// AdvanceToActive advances t to the next time the schedule is active.
// If t is already in an active window, returns t unchanged.
// Scans up to 14 days forward to handle long inactive gaps.
func AdvanceToActive(t time.Time, sched Schedule) time.Time {
	if sched.IsActiveAt(t) {
		return t
	}

	// Scan up to 14 days forward, day by day.
	for range 15 {
		// Try start hour on the current day (if we haven't passed it yet).
		if t.Hour() < sched.StartHour {
			candidate := time.Date(t.Year(), t.Month(), t.Day(), sched.StartHour, 0, 0, 0, t.Location())
			if sched.IsActiveAt(candidate) {
				return candidate
			}
		}
		// Jump to start hour of the next day.
		t = time.Date(t.Year(), t.Month(), t.Day()+1, sched.StartHour, 0, 0, 0, t.Location())
		if sched.IsActiveAt(t) {
			return t
		}
	}

	return t
}
