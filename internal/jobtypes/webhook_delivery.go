package jobtypes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// webhookPayload is the expected JSON payload for webhook_delivery.
type webhookPayload struct {
	URL     string `json:"url"`
	Body    string `json:"body"`
	Timeout int    `json:"timeout_seconds"`
}

// WebhookDelivery performs a real HTTP POST to the target URL.
// Demonstrates real HTTP client patterns and context cancellation.
func WebhookDelivery(ctx context.Context, raw json.RawMessage) error {
	var p webhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("webhook_delivery: invalid payload: %w", err)
	}
	if p.URL == "" {
		return fmt.Errorf("webhook_delivery: url is required")
	}
	if p.Timeout <= 0 {
		p.Timeout = 10
	}

	client := &http.Client{Timeout: time.Duration(p.Timeout) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader([]byte(p.Body)))
	if err != nil {
		return fmt.Errorf("webhook_delivery: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook_delivery: %s: %w", p.URL, err)
	}
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook_delivery: %s returned %d", p.URL, resp.StatusCode)
	}
	return nil
}
