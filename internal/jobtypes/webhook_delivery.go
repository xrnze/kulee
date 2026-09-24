package jobtypes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// webhookPayload is the expected JSON payload for webhook_delivery.
type webhookPayload struct {
	URL     string `json:"url"`
	Body    string `json:"body"`
	Timeout int    `json:"timeout_seconds"`
}

// ValidateWebhookDelivery checks the payload required by WebhookDelivery.
func ValidateWebhookDelivery(raw json.RawMessage) error {
	var p webhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("webhook_delivery: invalid payload: %w", err)
	}
	parsed, err := url.ParseRequestURI(p.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("webhook_delivery: url must be absolute")
	}
	if p.Timeout < 0 {
		return fmt.Errorf("webhook_delivery: timeout_seconds cannot be negative")
	}
	return nil
}

// WebhookDelivery performs a real HTTP POST to the target URL.
// Demonstrates real HTTP client patterns and context cancellation.
func WebhookDelivery(ctx context.Context, raw json.RawMessage) error {
	if err := ValidateWebhookDelivery(raw); err != nil {
		return err
	}
	var p webhookPayload
	_ = json.Unmarshal(raw, &p)
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
