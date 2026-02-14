package schedule

import (
	"testing"
	"time"

	"github.com/lvrach/slack-social-ai/internal/history"
)

func TestPredictPublishTimes_EmptyQueue(t *testing.T) {
	sched := DefaultSchedule()
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC) // Monday

	predictions := PredictPublishTimes(nil, sched, time.Time{}, now)
	if len(predictions) != 0 {
		t.Errorf("expected 0 predictions, got %d", len(predictions))
	}
}

func TestPredictPublishTimes_SingleEntry_ActiveWindow(t *testing.T) {
	sched := DefaultSchedule()                          // 9-17 mon-fri, 180min
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC) // Monday 10:00

	entries := []history.Entry{
		{ID: "a1", Message: "Hello", Status: "queued", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(predictions))
	}

	p := predictions[0]
	if p.Position != 1 {
		t.Errorf("Position = %d, want 1", p.Position)
	}
	if !p.PublishAt.Equal(now) {
		t.Errorf("PublishAt = %v, want %v", p.PublishAt, now)
	}
	if p.Approximate {
		t.Error("first prediction should not be approximate")
	}
}

func TestPredictPublishTimes_SingleEntry_OutsideHours(t *testing.T) {
	sched := DefaultSchedule() // 9-17 mon-fri
	// Monday 18:00 — outside active hours
	now := time.Date(2026, 2, 9, 18, 0, 0, 0, time.UTC)

	entries := []history.Entry{
		{ID: "a1", Message: "Hello", Status: "queued", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(predictions))
	}

	// Should jump to next active window: Tuesday 09:00
	wantTime := time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC) // Tuesday 09:00
	p := predictions[0]
	if !p.PublishAt.Equal(wantTime) {
		t.Errorf("PublishAt = %v, want %v", p.PublishAt, wantTime)
	}
}

func TestPredictPublishTimes_MultipleEntries_PostEverySpacing(t *testing.T) {
	sched := DefaultSchedule()                         // 9-17 mon-fri, 180min (3h)
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC) // Monday 09:00

	entries := []history.Entry{
		{ID: "a1", Message: "First", Status: "queued", CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{ID: "a2", Message: "Second", Status: "queued", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339)},
		{ID: "a3", Message: "Third", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 3 {
		t.Fatalf("expected 3 predictions, got %d", len(predictions))
	}

	// First: now (Monday 09:00)
	if !predictions[0].PublishAt.Equal(now) {
		t.Errorf("predictions[0].PublishAt = %v, want %v", predictions[0].PublishAt, now)
	}
	if predictions[0].Approximate {
		t.Error("predictions[0] should not be approximate")
	}

	// Second: +3h = Monday 12:00
	want2 := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	if !predictions[1].PublishAt.Equal(want2) {
		t.Errorf("predictions[1].PublishAt = %v, want %v", predictions[1].PublishAt, want2)
	}
	if !predictions[1].Approximate {
		t.Error("predictions[1] should be approximate")
	}

	// Third: +3h = Monday 15:00
	want3 := time.Date(2026, 2, 9, 15, 0, 0, 0, time.UTC)
	if !predictions[2].PublishAt.Equal(want3) {
		t.Errorf("predictions[2].PublishAt = %v, want %v", predictions[2].PublishAt, want3)
	}
}

func TestPredictPublishTimes_ScheduledAt_PushesCursorForward(t *testing.T) {
	sched := DefaultSchedule()                         // 9-17 mon-fri, 180min
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC) // Monday 09:00

	// Entry with ScheduledAt at 14:00 — should push cursor forward
	entries := []history.Entry{
		{
			ID:          "a1",
			Message:     "Scheduled later",
			Status:      "queued",
			CreatedAt:   now.Format(time.RFC3339),
			ScheduledAt: time.Date(2026, 2, 9, 14, 0, 0, 0, time.UTC).Format(time.RFC3339),
		},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(predictions))
	}

	wantTime := time.Date(2026, 2, 9, 14, 0, 0, 0, time.UTC)
	if !predictions[0].PublishAt.Equal(wantTime) {
		t.Errorf("PublishAt = %v, want %v", predictions[0].PublishAt, wantTime)
	}
}

func TestPredictPublishTimes_WeekendGap(t *testing.T) {
	sched := DefaultSchedule() // 9-17 mon-fri
	// Friday 16:00
	now := time.Date(2026, 2, 13, 16, 0, 0, 0, time.UTC)

	entries := []history.Entry{
		{ID: "a1", Message: "Friday post", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
		{ID: "a2", Message: "Monday post", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(predictions))
	}

	// First: Friday 16:00 (in active window)
	if !predictions[0].PublishAt.Equal(now) {
		t.Errorf("predictions[0].PublishAt = %v, want %v", predictions[0].PublishAt, now)
	}

	// Second: Friday 16:00 + 3h = Friday 19:00 → outside hours → next active = Monday 09:00
	wantMonday := time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC) // Monday 09:00
	if !predictions[1].PublishAt.Equal(wantMonday) {
		t.Errorf("predictions[1].PublishAt = %v, want %v", predictions[1].PublishAt, wantMonday)
	}
}

func TestPredictPublishTimes_ZeroPostEvery_UsesLaunchdInterval(t *testing.T) {
	sched := Schedule{
		PostEveryMinutes: 0,
		StartHour:        0,
		EndHour:          24,
		Weekdays:         []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC) // Monday 10:00

	entries := []history.Entry{
		{ID: "a1", Message: "First", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
		{ID: "a2", Message: "Second", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(predictions))
	}

	// With PostEvery=0, use 10min launchd fallback interval
	want2 := now.Add(10 * time.Minute)
	if !predictions[1].PublishAt.Equal(want2) {
		t.Errorf("predictions[1].PublishAt = %v, want %v", predictions[1].PublishAt, want2)
	}
}

func TestPredictPublishTimes_SingleActiveWeekday(t *testing.T) {
	sched := Schedule{
		PostEveryMinutes: 180,
		StartHour:        9,
		EndHour:          17,
		Weekdays:         []string{"wed"},
	}
	// Monday — not active
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC) // Monday

	entries := []history.Entry{
		{ID: "a1", Message: "Wed only", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, time.Time{}, now)
	if len(predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(predictions))
	}

	// Should jump to Wednesday 09:00
	wantWed := time.Date(2026, 2, 11, 9, 0, 0, 0, time.UTC)
	if !predictions[0].PublishAt.Equal(wantWed) {
		t.Errorf("PublishAt = %v, want %v", predictions[0].PublishAt, wantWed)
	}
}

func TestPredictPublishTimes_LastPublished_PushesCursor(t *testing.T) {
	sched := DefaultSchedule()                          // 9-17 mon-fri, 180min
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC) // Monday 10:00
	// Last published 30min ago — cursor should wait until lastPublished + 3h = 12:30
	lastPublished := now.Add(-30 * time.Minute) // 09:30

	entries := []history.Entry{
		{ID: "a1", Message: "Hello", Status: "queued", CreatedAt: now.Format(time.RFC3339)},
	}

	predictions := PredictPublishTimes(entries, sched, lastPublished, now)
	if len(predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(predictions))
	}

	wantTime := lastPublished.Add(3 * time.Hour) // 12:30
	if !predictions[0].PublishAt.Equal(wantTime) {
		t.Errorf("PublishAt = %v, want %v", predictions[0].PublishAt, wantTime)
	}
}

func TestAdvanceToActive(t *testing.T) {
	sched := DefaultSchedule() // 9-17 mon-fri

	tests := []struct {
		name string
		t    time.Time
		want time.Time
	}{
		{
			name: "already active",
			t:    time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), // Monday 10:00
			want: time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
		},
		{
			name: "before start on active day",
			t:    time.Date(2026, 2, 9, 7, 0, 0, 0, time.UTC), // Monday 07:00
			want: time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC), // Monday 09:00
		},
		{
			name: "after end on Friday",
			t:    time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC), // Friday 18:00
			want: time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC),  // Monday 09:00
		},
		{
			name: "Saturday noon",
			t:    time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC), // Saturday
			want: time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC),  // Monday 09:00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AdvanceToActive(tt.t, sched)
			if !got.Equal(tt.want) {
				t.Errorf("AdvanceToActive(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}
