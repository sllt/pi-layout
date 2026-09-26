#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="$ROOT/.tools/bin:$PATH"
PI="${PI:-pi}"
[[ "$("$PI" --version)" == "pi version ${PI_VERSION:-v0.4.1}" ]] || { echo 'Pi generator version mismatch' >&2; exit 1; }
[[ "$(protoc --version)" == 'libprotoc 33.1' ]] || { echo 'protoc 33.1 required' >&2; exit 1; }
[[ "$(protoc-gen-go --version)" == 'protoc-gen-go v1.28.0' ]] || exit 1
[[ "$(protoc-gen-go-grpc --version)" == 'protoc-gen-go-grpc 1.2.0' ]] || exit 1
[[ "$(mockgen -version)" == 'v1.6.0' ]] || exit 1
[[ "$(swag --version)" == 'swag version v1.16.4' ]] || exit 1
STAGE="$(mktemp -d "${TMPDIR:-/tmp}/pi-generate.XXXXXX")"
VERIFY="$(mktemp -d "${TMPDIR:-/tmp}/pi-generate-check.XXXXXX")"
trap 'rm -rf "$STAGE" "$VERIFY"' EXIT
cd "$ROOT"
mkdir -p "$STAGE/internal/grpc/user" "$STAGE/test/mocks/service" "$STAGE/test/mocks/repository"
MODULE="$(GOWORK=off go list -m)"
protoc --go_out="$STAGE" --go_opt="module=$MODULE" --go-grpc_out="$STAGE" --go-grpc_opt="module=$MODULE" api/proto/user/user.proto
"$PI" wrap grpc server --proto api/proto/user/user.proto --out "$STAGE/internal/grpc/user"
# Only generated wrappers are delivered. The handwritten server is never replaced.
rm "$STAGE/internal/grpc/user/userservice_server.go"
mockgen -source=internal/service/user.go -destination "$STAGE/test/mocks/service/user.go"
mockgen -source=internal/repository/user.go -destination "$STAGE/test/mocks/repository/user.go"
mockgen -source=internal/repository/repository.go -destination "$STAGE/test/mocks/repository/repository.go"
swag init -g cmd/server/main.go -o "$STAGE/docs" --parseInternal --outputTypes json,yaml
gofmt -w "$STAGE/internal" "$STAGE/test"
# Compile the rendered result before publishing any files to the project.
tar --exclude=.git --exclude=.tools --exclude=storage --exclude=tmp --exclude=bin --exclude=configs/.env --exclude=go.work --exclude=go.work.sum -cf - . | tar -xf - -C "$VERIFY"
cp -R "$STAGE/." "$VERIFY/"
(cd "$VERIFY" && GOWORK=off go build -mod=readonly ./...)
while IFS= read -r file; do
  relative="${file#"$STAGE/"}"
  if [[ "${1:-}" == '--check' ]]; then
    diff -u "$ROOT/$relative" "$file"
  else
    mkdir -p "$(dirname "$ROOT/$relative")"
    cp "$file" "$ROOT/$relative.pi-generated"
    mv "$ROOT/$relative.pi-generated" "$ROOT/$relative"
  fi
done < <(find "$STAGE" -type f | sort)
