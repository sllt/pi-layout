#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:8000}"
START_TIMEOUT_SECONDS="${START_TIMEOUT_SECONDS:-20}"
SERVER_LOG="${SERVER_LOG:-/tmp/pi-server-smoke.log}"
MIGRATION_LOG="${MIGRATION_LOG:-/tmp/pi-migration-smoke.log}"

USE_EXISTING_SERVER="${USE_EXISTING_SERVER:-0}"
SKIP_MIGRATION="${SKIP_MIGRATION:-0}"

if ! command -v curl >/dev/null 2>&1; then
  echo "[smoke] error: curl not found"
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "[smoke] error: go not found"
  exit 1
fi

SERVER_PID=""
cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" >/dev/null 2>&1; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

run_migration() {
  echo "[smoke] running migration..."
  go run ./cmd/migration >"$MIGRATION_LOG" 2>&1
  echo "[smoke] migration done"
}

start_server() {
  echo "[smoke] starting server..."
  go run ./cmd/server >"$SERVER_LOG" 2>&1 &
  SERVER_PID=$!
}

wait_server_ready() {
  local elapsed=0
  while (( elapsed < START_TIMEOUT_SECONDS )); do
    if curl -sS "$BASE_URL/" >/dev/null 2>&1; then
      echo "[smoke] server ready: $BASE_URL"
      return 0
    fi
    sleep 1
    ((elapsed += 1))
  done

  echo "[smoke] server not ready within ${START_TIMEOUT_SECONDS}s"
  echo "[smoke] server log: $SERVER_LOG"
  if [[ -f "$SERVER_LOG" ]]; then
    tail -n 80 "$SERVER_LOG" || true
  fi
  return 1
}

RESP_STATUS=""
RESP_BODY=""
request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local token="${4:-}"

  local tmp_file
  tmp_file="$(mktemp)"
  local -a args
  args=(-sS -o "$tmp_file" -w "%{http_code}" -X "$method" "$BASE_URL$path")

  if [[ -n "$data" ]]; then
    args+=( -H "Content-Type: application/json" -d "$data" )
  fi
  if [[ -n "$token" ]]; then
    args+=( -H "Authorization: Bearer $token" )
  fi

  RESP_STATUS="$(curl "${args[@]}")"
  RESP_BODY="$(cat "$tmp_file")"
  rm -f "$tmp_file"

  echo "[smoke] $method $path -> $RESP_STATUS"
  echo "[smoke] body: $RESP_BODY"
}

assert_status() {
  local actual="$1"
  shift
  local expected
  for expected in "$@"; do
    if [[ "$actual" == "$expected" ]]; then
      return 0
    fi
  done
  echo "[smoke] unexpected status: got=$actual expected=$*"
  exit 1
}

if [[ "$SKIP_MIGRATION" != "1" ]]; then
  run_migration
fi

if [[ "$USE_EXISTING_SERVER" != "1" ]]; then
  start_server
fi

wait_server_ready

EMAIL="pi_smoke_$(date +%s)@example.com"
PASSWORD="pass123456"

request POST /api/v1/register "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
assert_status "$RESP_STATUS" 200 202

request POST /api/v1/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
assert_status "$RESP_STATUS" 200 201
TOKEN="$(printf '%s' "$RESP_BODY" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')"
if [[ -z "$TOKEN" ]]; then
  echo "[smoke] login succeeded but accessToken not found"
  exit 1
fi

request GET /api/v1/user "" "$TOKEN"
assert_status "$RESP_STATUS" 200

request PUT /api/v1/user "{\"nickname\":\"pi-smoke\",\"email\":\"$EMAIL\"}" "$TOKEN"
assert_status "$RESP_STATUS" 200

request GET /api/v1/user "" "$TOKEN"
assert_status "$RESP_STATUS" 200

request GET /api/v1/user
assert_status "$RESP_STATUS" 401

echo "[smoke] all checks passed ✅"
