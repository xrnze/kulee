package jobtypes

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// emailPayload is the expected JSON payload for the send_email job type.
type emailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SendEmail simulates an I/O-bound email send. It sleeps 2-5s and fails
// ~10% of the time to exercise retry/backoff.
func SendEmail(ctx context.Context, raw json.RawMessage) error {
	var p emailPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("send_email: invalid payload: %w", err)
	}

	delay := time.Duration(2000+rand.Intn(3000)) * time.Millisecond
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	if rand.Intn(10) == 0 {
		return fmt.Errorf("send_email: simulated failure for %s", p.To)
	}
	return nil
}
