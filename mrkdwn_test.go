package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMrkdwnToMarkdown_Bold(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple bold",
			input:    "*bold*",
			expected: "**bold**",
		},
		{
			name:     "bold in sentence",
			input:    "this is *bold* text",
			expected: "this is **bold** text",
		},
		{
			name:     "multiple bolds",
			input:    "*one* and *two*",
			expected: "**one** and **two**",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mrkdwnToMarkdown(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMrkdwnToMarkdown_Strikethrough(t *testing.T) {
	got := mrkdwnToMarkdown("~strike~")
	assert.Equal(t, "~~strike~~", got)
}

func TestMrkdwnToMarkdown_Links(t *testing.T) {
	got := mrkdwnToMarkdown("<https://example.com|Example>")
	assert.Equal(t, "[Example](https://example.com)", got)
}

func TestMrkdwnToMarkdown_BareLinks(t *testing.T) {
	got := mrkdwnToMarkdown("<https://example.com>")
	assert.Equal(t, "https://example.com", got)
}

func TestMrkdwnToMarkdown_ChannelRefs(t *testing.T) {
	got := mrkdwnToMarkdown("<#C123ABC|general>")
	assert.Equal(t, "#general", got)
}

func TestMrkdwnToMarkdown_Mentions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<!here>", "@here"},
		{"<!channel>", "@channel"},
		{"<!everyone>", "@everyone"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mrkdwnToMarkdown(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMrkdwnToMarkdown_CodeBlocks(t *testing.T) {
	// Text inside code fences should NOT be transformed.
	input := "before *bold*\n```\n*not bold* ~not strike~\n```\nafter *bold*"
	got := mrkdwnToMarkdown(input)

	// The code block content must remain unchanged.
	assert.Contains(t, got, "\n*not bold* ~not strike~\n")
	// Text outside code blocks must be converted.
	assert.Contains(t, got, "before **bold**")
	assert.Contains(t, got, "after **bold**")
}

func TestMrkdwnToMarkdown_Mixed(t *testing.T) {
	input := "*Hello* from <#C99|random>! Check <https://example.com|this link> and ~old stuff~"
	got := mrkdwnToMarkdown(input)

	assert.Contains(t, got, "**Hello**")
	assert.Contains(t, got, "#random")
	assert.Contains(t, got, "[this link](https://example.com)")
	assert.Contains(t, got, "~~old stuff~~")
}

func TestMrkdwnToMarkdown_Passthrough(t *testing.T) {
	input := "Just plain text with no special formatting."
	got := mrkdwnToMarkdown(input)
	assert.Equal(t, input, got)
}

func TestRenderMrkdwn_Fallback(t *testing.T) {
	input := "*Hello* world"
	got := renderMrkdwn(input, 80)
	// Glamour output varies by theme, but it should return non-empty text.
	assert.NotEmpty(t, got)
	// The rendered output should contain "Hello" somewhere (in bold formatting).
	assert.Contains(t, got, "Hello")
}

func TestRenderMrkdwn_Emoji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"rocket", ":rocket: launch", "🚀"},
		{"brain", "big :brain: time", "🧠"},
		{"unknown shortcode", ":notarealshortcode: stays", ":notarealshortcode:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderMrkdwn(tt.input, 80)
			assert.Contains(t, got, tt.contains)
		})
	}
}

func TestRenderMrkdwn_EmojiInCodeBlock(t *testing.T) {
	// Emoji shortcodes inside code fences should NOT be converted.
	input := "```\n:rocket:\n```"
	got := renderMrkdwn(input, 80)
	assert.NotContains(t, got, "🚀")
	assert.Contains(t, got, ":rocket:")
}

func TestRenderMrkdwn_Width(t *testing.T) {
	// Create a long line that should wrap at a narrow width.
	words := strings.Repeat("word ", 30) // 150 chars
	got := renderMrkdwn(words, 40)

	// The output should contain line breaks from wrapping.
	lines := strings.Split(strings.TrimSpace(got), "\n")
	assert.Greater(t, len(lines), 1, "expected word wrap to produce multiple lines")
}
