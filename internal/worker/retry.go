// Package worker provides the worker pool and retry helpers.
package worker

import (
	"math"
	"math/rand"
	"time"
)

// FullJitterDelay computes a full-jitter backoff delay for the given attempt.
// Returns random(0, min(cap, base * 2^attempt)). The first attempt (0) uses
// base delay; subsequent attempts double the upper bound until cap.
func FullJitterDelay(attempt int, base, cap time.Duration) time.Duration {
	upper := float64(base) * math.Pow(2, float64(attempt))
	if upper > float64(cap) {
		upper = float64(cap)
	}
	return time.Duration(rand.Int63n(int64(upper)))
}
