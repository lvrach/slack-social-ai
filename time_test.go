package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAt_RFC3339(t *testing.T) {
	input := "2025-02-14T09:00:00Z"
	got, err := parseAtFrom(input, time.Now())
	require.NoError(t, err)

	expected, _ := time.Parse(time.RFC3339, input)
	assert.Equal(t, expected, got)
}

func TestParseAt_HH_MM(t *testing.T) {
	// now is 10:00, target is 14:30 -> should be today.
	now := time.Date(2025, 2, 14, 10, 0, 0, 0, time.Local)
	got, err := parseAtFrom("14:30", now)
	require.NoError(t, err)

	expected := time.Date(2025, 2, 14, 14, 30, 0, 0, time.Local)
	assert.Equal(t, expected, got)
}

func TestParseAt_HH_MM_Tomorrow(t *testing.T) {
	// now is 22:00, target is 09:00 -> should be tomorrow.
	now := time.Date(2025, 2, 14, 22, 0, 0, 0, time.Local)
	got, err := parseAtFrom("09:00", now)
	require.NoError(t, err)

	expected := time.Date(2025, 2, 15, 9, 0, 0, 0, time.Local)
	assert.Equal(t, expected, got)
}

func TestParseAt_SingleDigitHour(t *testing.T) {
	now := time.Date(2025, 2, 14, 6, 0, 0, 0, time.Local)
	got, err := parseAtFrom("9:00", now)
	require.NoError(t, err)

	expected := time.Date(2025, 2, 14, 9, 0, 0, 0, time.Local)
	assert.Equal(t, expected, got)
}

func TestParseAt_Duration(t *testing.T) {
	now := time.Date(2025, 2, 14, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "2h",
			input:    "2h",
			expected: now.Add(2 * time.Hour),
		},
		{
			name:     "30m",
			input:    "30m",
			expected: now.Add(30 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAtFrom(tt.input, now)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseAt_NegativeDuration(t *testing.T) {
	now := time.Date(2025, 2, 14, 10, 0, 0, 0, time.UTC)
	_, err := parseAtFrom("-1h", now)
	require.Error(t, err)

	var cliErr *CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, "invalid_time", cliErr.Code)
	assert.Contains(t, cliErr.Message, "positive")
}

func TestParseAt_Invalid(t *testing.T) {
	now := time.Date(2025, 2, 14, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input string
	}{
		{name: "natural language", input: "next tuesday"},
		{name: "random string", input: "abc"},
		{name: "empty string", input: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAtFrom(tt.input, now)
			require.Error(t, err)

			var cliErr *CLIError
			require.True(t, errors.As(err, &cliErr))
			assert.Equal(t, "invalid_time", cliErr.Code)
			assert.Contains(t, cliErr.Message, "Cannot parse")
		})
	}
}
