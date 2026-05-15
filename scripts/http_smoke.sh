#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:18080}"
ADDR="${BASE_URL#http://}"
TMP_DIR="$(mktemp -d)"
SERVER_LOG="$TMP_DIR/server.log"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID"
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$ROOT_DIR"
go run ./cmd/server --addr="$ADDR" --seed="sku-1:1,sku-2:1" --hold=2m --expiry-interval=0 >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

ready=false
for _ in {1..200}; do
  if curl -fsS "$BASE_URL/healthz" >"$TMP_DIR/health.json" 2>/dev/null; then
    ready=true
    break
  fi
done

if [[ "$ready" != true ]]; then
  cat "$SERVER_LOG" >&2
  echo "server did not become ready at $BASE_URL" >&2
  exit 1
fi

request() {
  local expected_status="$1"
  local method="$2"
  local path="$3"
  local body="${4:-}"
  local output_file="$5"
  local status

  if [[ -n "$body" ]]; then
    status="$(curl -sS -o "$output_file" -w '%{http_code}' -X "$method" "$BASE_URL$path" -H 'Content-Type: application/json' -d "$body")"
  else
    status="$(curl -sS -o "$output_file" -w '%{http_code}' -X "$method" "$BASE_URL$path")"
  fi

  if [[ "$status" != "$expected_status" ]]; then
    echo "$method $path: expected $expected_status, got $status" >&2
    cat "$output_file" >&2
    exit 1
  fi
}

json_field() {
  local file="$1"
  local field="$2"
  python3 - "$file" "$field" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)[sys.argv[2]])
PY
}

request 200 GET /healthz "" "$TMP_DIR/health.json"
request 201 POST /reservations '{"productId":"sku-1","userId":"user-a"}' "$TMP_DIR/reserve-sku1.json"
sku1_reservation_id="$(json_field "$TMP_DIR/reserve-sku1.json" reservationId)"
request 200 GET /products/sku-1/stock "" "$TMP_DIR/stock-sku1.json"
request 409 POST /reservations '{"productId":"sku-1","userId":"user-b"}' "$TMP_DIR/out-of-stock.json"
request 200 POST "/reservations/$sku1_reservation_id/confirm" "" "$TMP_DIR/confirm-sku1.json"
request 409 POST "/reservations/$sku1_reservation_id/confirm" "" "$TMP_DIR/double-confirm.json"

request 201 POST /reservations '{"productId":"sku-2","userId":"user-c"}' "$TMP_DIR/reserve-sku2.json"
sku2_reservation_id="$(json_field "$TMP_DIR/reserve-sku2.json" reservationId)"
request 200 POST "/reservations/$sku2_reservation_id/cancel" "" "$TMP_DIR/cancel-sku2.json"
request 200 GET /products/sku-2/stock "" "$TMP_DIR/stock-sku2.json"

request 404 GET /products/ghost/stock "" "$TMP_DIR/unknown-stock.json"
request 404 POST /reservations '{"productId":"ghost","userId":"user-d"}' "$TMP_DIR/unknown-reserve.json"
request 400 POST /reservations 'not-json' "$TMP_DIR/bad-json.json"
request 400 POST /reservations '{"productId":"","userId":""}' "$TMP_DIR/missing-fields.json"

echo "HTTP smoke passed for $BASE_URL"