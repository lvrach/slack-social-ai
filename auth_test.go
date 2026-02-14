package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskWebhookURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard webhook URL",
			input:    "https://hooks.slack.com/services/T12345/B67890/abcdefghijk",
			expected: "https://hooks.slack.com/services/T12345/...",
		},
		{
			name:     "short URL",
			input:    "https://hooks.slack.com/other",
			expected: "https://hooks.slack.com/...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskWebhookURL(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "https://hooks.slack.com/services/T12345/B67890/xxx", false},
		{"empty", "", true},
		{"wrong prefix", "https://example.com/webhook", true},
		{"http not https", "http://hooks.slack.com/services/T/B/x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
