package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFreqOptions_PresetCurrentFirst(t *testing.T) {
	opts := buildFreqOptions(180) // Every 3 hours

	require.GreaterOrEqual(t, len(opts), 6, "should have at least 6 options")

	// First option should be the current value with arrow marker.
	assert.Equal(t, 180, opts[0].Value)
	assert.Contains(t, opts[0].Key, "←")

	// No other option should have value 180.
	for _, o := range opts[1:] {
		assert.NotEqual(t, 180, o.Value, "current value should not be duplicated")
	}
}

func TestBuildFreqOptions_NoLimitCurrentFirst(t *testing.T) {
	opts := buildFreqOptions(0) // No limit

	require.GreaterOrEqual(t, len(opts), 6)

	// First option should be "No limit" with arrow marker.
	assert.Equal(t, 0, opts[0].Value)
	assert.Contains(t, opts[0].Key, "←")
}

func TestBuildFreqOptions_CustomValue(t *testing.T) {
	opts := buildFreqOptions(135) // 2h15m — not a preset

	require.GreaterOrEqual(t, len(opts), 7, "should have custom + 6 presets")

	// First option should be the custom value.
	assert.Equal(t, 135, opts[0].Value)
	assert.Contains(t, opts[0].Key, "current")

	// All 6 presets should also be present.
	values := make(map[int]bool)
	for _, o := range opts {
		values[o.Value] = true
	}
	assert.True(t, values[30])
	assert.True(t, values[60])
	assert.True(t, values[180])
	assert.True(t, values[360])
	assert.True(t, values[1440])
	assert.True(t, values[0])
}

func TestFormatHourLabel(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{0, "12am (midnight)"},
		{1, "1am"},
		{9, "9am"},
		{11, "11am"},
		{12, "12pm (noon)"},
		{13, "1pm"},
		{17, "5pm"},
		{23, "11pm"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatHourLabel(tt.hour))
		})
	}
}

func TestFormatWeekdays(t *testing.T) {
	tests := []struct {
		name     string
		days     []string
		expected string
	}{
		{"empty", nil, "no days"},
		{"all days", []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}, "every day"},
		{"single", []string{"mon"}, "Mon"},
		{"consecutive", []string{"mon", "tue", "wed", "thu", "fri"}, "Mon–Fri"},
		{"non-consecutive", []string{"mon", "wed", "fri"}, "Mon, Wed, Fri"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatWeekdays(tt.days))
		})
	}
}
