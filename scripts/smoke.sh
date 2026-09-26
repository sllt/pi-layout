#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN="$(mktemp -d "${TMPDIR:-/tmp}/pi-smoke.XXXXXX")"
PID=''
cleanup() {
  if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then kill -TERM "$PID"; wait "$PID" || true; fi
  rm -rf "$RUN"
}
trap cleanup EXIT
command -v jq >/dev/null
cd "$ROOT"
go build -mod=readonly -o "$RUN/server" ./cmd/server
go build -mod=readonly -o "$RUN/migration" ./cmd/migration
mkdir -p "$RUN/configs" "$RUN/storage"
touch "$RUN/configs/.env"
export DB_DIALECT=sqlite DB_NAME="$RUN/storage/app.db" PI_TELEMETRY=false
export HTTP_ADDR="127.0.0.1:${SMOKE_PORT:-18080}" HTTP_ENABLED=true GRPC_ENABLED=false METRICS_ENABLED=false
export JWT_SECRET=pi-smoke-ephemeral-key-with-at-least-32-bytes JWT_ISSUER=pi-layout JWT_AUDIENCE=pi-api JWT_TTL=1h
BASE_URL="http://$HTTP_ADDR"
cd "$RUN"
"$RUN/migration" up > "$RUN/migration.log" 2>&1
"$RUN/server" > "$RUN/server.log" 2>&1 &
PID=$!
for ((i=0;i<100;i++)); do
  kill -0 "$PID" 2>/dev/null || { cat "$RUN/server.log"; exit 1; }
  if curl --max-time 1 -fsS "$BASE_URL/" > "$RUN/root.json" 2>/dev/null; then break; fi
  sleep 0.1
done
jq -e '.code == 0 and .data[":)"] == "Thank you for using pi!"' "$RUN/root.json" >/dev/null
request() {
  local method="$1" path="$2" expected="$3" data="${4:-}" token="${5:-}"
  local args=(-sS --max-time 5 -o "$RUN/response.json" -w '%{http_code}' -X "$method" "$BASE_URL$path")
  [[ -z "$data" ]] || args+=(-H 'Content-Type: application/json' -d "$data")
  [[ -z "$token" ]] || args+=(-H "Authorization: Bearer $token")
  local actual
  actual="$(curl "${args[@]}")"
  [[ "$actual" == "$expected" ]] || { echo "$method $path: expected $expected, got $actual"; cat "$RUN/response.json"; exit 1; }
}
request POST /api/v1/register 201 '{"email":"smoke@example.com","password":"test-password"}'
jq -e '.code == 0 and .data == null' "$RUN/response.json" >/dev/null
request POST /api/v1/login 200 '{"email":"smoke@example.com","password":"test-password"}'
TOKEN="$(jq -er 'select(.code == 0) | .data.accessToken | select(length > 20)' "$RUN/response.json")"
request GET /api/v1/user 200 '' "$TOKEN"
USER_ID="$(jq -er 'select(.code == 0) | .data.userId | select(length > 0)' "$RUN/response.json")"
request PUT /api/v1/user 204 '{"email":"smoke@example.com","nickname":"smoke"}' "$TOKEN"
[[ ! -s "$RUN/response.json" ]]
request GET /api/v1/user 200 '' "$TOKEN"
jq -e --arg id "$USER_ID" '.code == 0 and .data.userId == $id and .data.nickname == "smoke"' "$RUN/response.json" >/dev/null
request GET /api/v1/user 401
jq -e '.code != 0 and .data == null' "$RUN/response.json" >/dev/null
kill -TERM "$PID"
for ((i=0;i<100;i++)); do kill -0 "$PID" 2>/dev/null || break; sleep 0.1; done
if kill -0 "$PID" 2>/dev/null; then echo 'server did not exit after SIGTERM'; exit 1; fi
wait "$PID"
PID=''
echo 'smoke passed: migration, identity, auth, exact statuses, profile, SIGTERM'
