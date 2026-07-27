#!/usr/bin/env bash
# Kulee load test — runs vegeta attacks against POST /api/jobs at
# worker pool sizes 1, 4, and 8, recording results in scripts/loadtest-results/.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/loadtest-results"
mkdir -p "$RESULTS_DIR"

if ! command -v vegeta &>/dev/null; then
  echo "ERROR: vegeta not found. Install it:"
  echo "  go install github.com/tsenart/vegeta/v12@latest"
  exit 1
fi

API_URL="${API_URL:-http://localhost:8080}"
DURATION="${DURATION:-10s}"
RATE="${RATE:-50}"

TARGETS_FILE=$(mktemp)
BODY_FILE=$(mktemp)
cleanup() { rm -f "$TARGETS_FILE" "$BODY_FILE"; }
trap cleanup EXIT

cat > "$BODY_FILE" <<'BODY'
{"type":"send_email","payload":{"to":"test@example.com","subject":"Load test","body":"Hello"},"priority":5}
BODY

cat > "$TARGETS_FILE" <<TARGETS
POST $API_URL/api/jobs
Content-Type: application/json
@$BODY_FILE
TARGETS

echo "=== Kulee Load Test ==="
echo "Target: $API_URL"
echo "Duration: $DURATION"
echo "Rate: $RATE req/s"
echo ""
echo "Job payload:"
echo "  $(cat "$BODY_FILE")"
echo ""

for workers in 1 4 8; do
  echo "--- Worker pool size: $workers ---"

  RESULT_FILE="$RESULTS_DIR/result-workers-$workers.bin"
  REPORT_FILE="$RESULTS_DIR/report-workers-$workers.txt"

  vegeta attack \
    -targets="$TARGETS_FILE" \
    -duration="$DURATION" \
    -rate="$RATE" \
    -workers="$workers" \
    -output="$RESULT_FILE"

  vegeta report -type=text "$RESULT_FILE" > "$REPORT_FILE"

  echo "Results written to $REPORT_FILE"
  echo ""
done

echo "=== Summary ==="
echo ""
for workers in 1 4 8; do
  REPORT_FILE="$RESULTS_DIR/report-workers-$workers.txt"
  if [ -f "$REPORT_FILE" ]; then
    echo "--- Workers: $workers ---"
    head -10 "$REPORT_FILE"
    echo ""
  fi
done

echo "All results saved to $RESULTS_DIR/"