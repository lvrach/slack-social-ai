package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
