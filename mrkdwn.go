package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
)

// Precompiled patterns for Slack mrkdwn -> Markdown conversion.
var (
	// Bold: *text* -> **text** (word boundary aware).
	mrkdwnBold = regexp.MustCompile(`(^|[\s\n({\[])(\*([^*\n]+)\*)`)
	// Strikethrough: ~text~ -> ~~text~~.
	mrkdwnStrike = regexp.MustCompile(`~([^~\n]+)~`)
	// Links with display text: <url|text> -> [text](url).
	mrkdwnLinkText = regexp.MustCompile(`<(https?://[^|>]+)\|([^>]+)>`)
	// Bare links: <url> -> url.
	mrkdwnLinkBare = regexp.MustCompile(`<(https?://[^>]+)>`)
	// Channel references: <#C123|channel> -> #channel.
	mrkdwnChannel = regexp.MustCompile(`<#[A-Z0-9]+\|([^>]+)>`)
	// Special mentions: <!here>, <!channel>, <!everyone>.
	mrkdwnMention = regexp.MustCompile(`<!(\w+)>`)
)

// mrkdwnToMarkdown converts Slack mrkdwn syntax to standard Markdown.
// It splits input on code-fence boundaries so that text inside fenced
// code blocks is not transformed.
func mrkdwnToMarkdown(s string) string {
	// Split on ``` boundaries. Odd-indexed segments are inside code fences.
	parts := strings.Split(s, "```")
	for i, part := range parts {
		if i%2 == 0 {
			parts[i] = convertMrkdwnSegment(part)
		}
		// Odd segments (code blocks) are left untouched.
	}
	return strings.Join(parts, "```")
}

// convertMrkdwnSegment applies mrkdwn-to-Markdown transformations
// to a segment that is known to be outside code fences.
func convertMrkdwnSegment(s string) string {
	// Order matters: channel refs before bare links (both use angle brackets).
	s = mrkdwnChannel.ReplaceAllString(s, "#$1")
	s = mrkdwnMention.ReplaceAllStringFunc(s, func(match string) string {
		inner := mrkdwnMention.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		switch inner[1] {
		case "here", "channel", "everyone":
			return "@" + inner[1]
		default:
			return match
		}
	})
	s = mrkdwnLinkText.ReplaceAllString(s, "[$2]($1)")
	s = mrkdwnLinkBare.ReplaceAllString(s, "$1")
	s = mrkdwnStrike.ReplaceAllString(s, "~~$1~~")

	// Bold: *text* -> **text**. We replace per-line to avoid matching across
	// lines and to handle the leading-character lookbehind correctly.
	s = convertBold(s)

	return s
}

// convertBold replaces Slack bold (*text*) with Markdown bold (**text**).
func convertBold(s string) string {
	return mrkdwnBold.ReplaceAllString(s, "${1}**${3}**")
}

// Cached glamour renderer — avoids re-creating on every call.
// WithAutoStyle() performs OS I/O to detect dark/light theme; caching
// eliminates this from the hot path in interactive TUIs.
var (
	cachedRenderer      *glamour.TermRenderer
	cachedRendererWidth int
)

// renderMrkdwn renders Slack mrkdwn as terminal-formatted text using glamour.
// It first preprocesses the input via mrkdwnToMarkdown, then renders with
// glamour. If rendering fails, the raw input text is returned as a fallback.
func renderMrkdwn(s string, width int) string {
	md := mrkdwnToMarkdown(s)

	if cachedRenderer == nil || cachedRendererWidth != width {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
			glamour.WithEmoji(),
		)
		if err != nil {
			return s
		}
		cachedRenderer = r
		cachedRendererWidth = width
	}

	rendered, err := cachedRenderer.Render(md)
	if err != nil {
		return s
	}

	return rendered
}
