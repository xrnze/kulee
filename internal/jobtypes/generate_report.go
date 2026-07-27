package jobtypes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// reportPayload is the expected JSON payload for generate_report.
type reportPayload struct {
	Rows         int    `json:"rows"`
	OutputFormat string `json:"output_format"`
}

// GenerateReport generates N rows of fake CSV data in a tight loop.
// CPU-bound: demonstrates the contrast with I/O-bound job types.
func GenerateReport(ctx context.Context, raw json.RawMessage) error {
	var p reportPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("generate_report: invalid payload: %w", err)
	}
	if p.Rows <= 0 {
		p.Rows = 100
	}
	if p.OutputFormat == "" {
		p.OutputFormat = "csv"
	}

	var buf bytes.Buffer
	buf.WriteString("id,name,email,amount\n")
	for i := 0; i < p.Rows; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strconv.Itoa(i+1) + "," +
			"user-" + strconv.Itoa(i+1) + "," +
			"user" + strconv.Itoa(i+1) + "@example.com," +
			strconv.FormatFloat(rand.Float64()*1000, 'f', 2, 64) + "\n"
		buf.WriteString(line)

		// Small pause so the CPU-bound loop can be interrupted by context
		// cancellation in a reasonable time frame.
		time.Sleep(time.Microsecond)
	}

	// In a real system we'd write buf to a file or return a URL.
	// Here we just discard the output -- the demo is the CPU work itself.
	_ = buf.Len()
	return nil
}
