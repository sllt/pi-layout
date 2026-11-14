GO ?= go
PI_VERSION ?= v0.5.0
TOOLS := $(CURDIR)/.tools/bin
export PATH := $(TOOLS):$(PATH)

.PHONY: init bootstrap build unit test race integration generator check-generated smoke container mock openapi swag
init:
	mkdir -p $(TOOLS)
	GOBIN=$(TOOLS) $(GO) install github.com/golang/mock/mockgen@v1.6.0
	GOBIN=$(TOOLS) $(GO) install github.com/air-verse/air@v1.61.7
	GOBIN=$(TOOLS) $(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
	GOBIN=$(TOOLS) $(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
	GOBIN=$(TOOLS) $(GO) install github.com/sllt/pi/cmd/pi@$(PI_VERSION)

bootstrap:
	mkdir -p storage
	$(GO) run ./cmd/migration up
	air

build:
	$(GO) build -mod=readonly -o ./bin/server ./cmd/server
	$(GO) build -mod=readonly -o ./bin/task ./cmd/task
	$(GO) build -mod=readonly -o ./bin/migration ./cmd/migration
	$(GO) build -mod=readonly -o ./bin/openapi ./cmd/openapi

unit test:
	$(GO) test ./...
race:
	$(GO) test -race ./...
integration:
	$(GO) test ./internal/migrationcmd ./internal/server ./test/server/repository -count=1
generator:
	./scripts/generate.sh
check-generated:
	./scripts/generate.sh --check
mock:
	./scripts/generate.sh
openapi:
	./scripts/generate.sh
swag: openapi
smoke:
	./scripts/smoke.sh
container:
	./scripts/container.sh
