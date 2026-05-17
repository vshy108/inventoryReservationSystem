#!/usr/bin/env bash
# k6_load_test.sh — start IRS + Postgres + Redis, run k6 reservation load test,
# save a Markdown report to docs/k6-load-report.md, then tear down.
#
# Usage:
#   bash scripts/k6_load_test.sh
#
# Env overrides:
#   BASE_URL     target base URL (default: http://localhost:8090)
#   REPORT_FILE  output Markdown path (default: docs/k6-load-report.md)
#   KEEP_STACK   set to 1 to leave compose stack running after the test
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

BASE_URL="${BASE_URL:-http://localhost:8090}"
REPORT_FILE="${REPORT_FILE:-$REPO_ROOT/docs/k6-load-report.md}"
KEEP_STACK="${KEEP_STACK:-0}"

cd "$REPO_ROOT"

cleanup() {
  if [[ "$KEEP_STACK" != 1 ]]; then
    printf 'stopping compose stack...\n'
    docker compose -f docker-compose.yml -f docker-compose.loadtest.yml down -v \
      >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

printf 'starting IRS compose stack with load-test seed (sku-load:1000000)...\n'
docker compose -f docker-compose.yml -f docker-compose.loadtest.yml up -d --build --wait \
  >/dev/null 2>&1

printf 'waiting for /healthz to report postgres:ok and redis:ok...\n'
for _ in $(seq 1 30); do
  response="$(curl -fsS "${BASE_URL}/healthz" 2>/dev/null || true)"
  if echo "$response" | grep -q '"postgres":"ok"' && echo "$response" | grep -q '"redis":"ok"'; then
    printf 'ready: %s\n' "$response"
    break
  fi
  sleep 1
done

if ! echo "$response" | grep -q '"postgres":"ok"'; then
  printf 'FAIL: service not ready after 30s\n' >&2
  exit 1
fi

printf 'running k6 load test (50 VUs, 45s total)...\n'
K6_OUTPUT="$(k6 run \
  --env BASE_URL="$BASE_URL" \
  "$REPO_ROOT/k6/reservation_load.js" 2>&1)"
printf '%s\n' "$K6_OUTPUT"

mkdir -p "$(dirname "$REPORT_FILE")"
cat > "$REPORT_FILE" << EOF
# IRS k6 Load Test Report

Generated: $(date -u "+%Y-%m-%d %H:%M UTC")

## Setup

- Service: Go inventory reservation service + PostgreSQL 17 + Redis 8
- Seed: \`sku-load:1000000\` (1 M units — stock does not exhaust during test)
- Scenario: \`POST /reservations\` with unique \`userId\` per VU/iteration
- Stages: 10 s warmup (5 VUs) → 30 s load (50 VUs) → 5 s ramp-down
- Thresholds: p(95) < 500 ms, http error rate < 1 %

## k6 Output

\`\`\`
$K6_OUTPUT
\`\`\`
EOF

printf 'report saved to %s\n' "$REPORT_FILE"
