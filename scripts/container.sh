#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
IMAGE="${IMAGE:-pi-layout:verify}"
NAME="pi-layout-verify-$$"
docker build -t "$IMAGE" .
cleanup() { docker rm -f "$NAME" >/dev/null 2>&1 || true; docker volume rm "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker volume create "$NAME" >/dev/null
# Initialize volume ownership without granting the application root privileges.
docker run --rm --user 0 -v "$NAME:/app/storage" "$IMAGE" chown 10001:10001 /app/storage
docker run --rm -v "$NAME:/app/storage" "$IMAGE" migration up
docker run -d --name "$NAME" -p 127.0.0.1::8000 -v "$NAME:/app/storage" -e JWT_SECRET=container-test-ephemeral-key-at-least-32-bytes "$IMAGE" >/dev/null
PORT="$(docker inspect -f '{{(index (index .NetworkSettings.Ports "8000/tcp") 0).HostPort}}' "$NAME")"
for ((i=0;i<100;i++)); do
  if curl --max-time 1 -fsS "http://127.0.0.1:$PORT/" >/dev/null 2>&1; then break; fi
  sleep 0.1
done
curl --max-time 3 -fsS "http://127.0.0.1:$PORT/" | jq -e '.code == 0 and .data[":)"] == "Thank you for using pi!"' >/dev/null
[[ "$(docker exec "$NAME" id -u)" == 10001 ]]
docker exec "$NAME" test ! -e /app/configs/.env
docker exec "$NAME" test -f /etc/ssl/certs/ca-certificates.crt
docker stop --time 15 "$NAME" >/dev/null
[[ "$(docker inspect -f '{{.State.ExitCode}}' "$NAME")" == 0 ]]
echo 'container passed: SQLite migration, nonroot, CA, no local config, SIGTERM'
