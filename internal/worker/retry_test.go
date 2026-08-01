package worker

import (
	"testing"
	"time"
)

func TestFullJitterDelay(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		base    time.Duration
		cap     time.Duration
	}{
		{"attempt 0", 0, 1 * time.Second, 60 * time.Second},
		{"attempt 1", 1, 1 * time.Second, 60 * time.Second},
		{"attempt 5", 5, 1 * time.Second, 60 * time.Second},
		{"attempt 10 (capped)", 10, 1 * time.Second, 60 * time.Second},
		{"small cap", 0, 100 * time.Millisecond, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mult := 1 << tt.attempt // 2^attempt
			upper := float64(tt.base) * float64(mult)
			if upper > float64(tt.cap) {
				upper = float64(tt.cap)
			}
			maxDelay := time.Duration(upper)

			for i := 0; i < 100; i++ {
				d := FullJitterDelay(tt.attempt, tt.base, tt.cap)
				if d < 0 {
					t.Errorf("delay is negative: %v", d)
				}
				if d > maxDelay {
					t.Errorf("delay %v exceeds upper bound %v", d, maxDelay)
				}
			}
		})
	}
}
