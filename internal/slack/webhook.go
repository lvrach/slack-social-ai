package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type payload struct {
	Text string `json:"text"`
}

// SendWebhook posts a text message to the given Slack webhook URL.
func SendWebhook(webhookURL, message string) error {
	body, err := json.Marshal(payload{Text: message})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := httpClient.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// VerifyWebhook silently checks if a webhook URL is valid without posting a message.
// It POSTs an empty JSON object. Slack returns 400 with "no_text" or similar
// when auth + channel are valid but payload has no text. That means the webhook works.
func VerifyWebhook(webhookURL string) error {
	resp, err := httpClient.Post(webhookURL, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return fmt.Errorf("webhook unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		// 400 with "no_text" or "missing_text_or_fallback_or_attachments" means
		// webhook is valid — Slack validated auth + channel, just rejected empty payload.
		if strings.Contains(bodyStr, "no_text") || strings.Contains(bodyStr, "missing_text") {
			return nil
		}
		return fmt.Errorf("webhook returned 400: %s", bodyStr)
	case http.StatusForbidden:
		return fmt.Errorf("webhook forbidden (403) — token may be revoked")
	case http.StatusNotFound:
		return fmt.Errorf("webhook not found (404) — URL may be incorrect")
	case http.StatusGone:
		return fmt.Errorf("webhook gone (410) — webhook has been deleted")
	case http.StatusOK:
		// Unlikely with empty payload, but treat as valid.
		return nil
	default:
		return fmt.Errorf("webhook returned unexpected status %d: %s", resp.StatusCode, bodyStr)
	}
}
