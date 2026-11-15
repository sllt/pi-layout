# Pi Layout

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev/)
[![Pi Framework](https://img.shields.io/badge/Pi-v0.5.2-blue)](https://github.com/sllt/pi)

English | [简体中文](./README.zh-CN.md)

**Pi Layout** is the official production-grade Go application scaffold for the [Pi](https://github.com/sllt/pi) framework. It combines Uber [Fx](https://github.com/uber-go/fx) dependency injection with Pi's explicit `pi.Build` runtime, strongly-typed HTTP contracts, OpenAPI 3.0.3 generation, gRPC adapters, lock-protected migrations, and actor-supervised Cron tasks.

---

## Features

- **Clean Layered Architecture** — Strict separation across `API DTO / Proto → Handler → Service → Repository → Model`, sharing domain validation and authorization rules across HTTP, gRPC, and background tasks.
- **Explicit Fx + `pi.Build` Microkernel Lifecycle** — Fx constructors extract side-effect-free `sql.Client` (`app.SQL()`) and `app.Logger()` handles; `OnStart` activates managed SQL and any mounted adapters (`pi.WithAdapter` / `pi.Use[T]`), while `OnStop` drains in-flight work before releasing resources in reverse order.
- **Typed HTTP Contracts & OpenAPI 3.0.3** — Routes use generic `pi.Register` and `pi.Meta` DTOs with pre-flight validation (`app.RouteError()`). `cmd/openapi` exports deterministic `docs/openapi.json` and `docs/openapi.yaml` without `swag` comments, and `OPENAPI_SERVE_SPEC=true` / `OPENAPI_SERVE_UI=true` serve live specs plus the built-in offline Scalar UI (`/.well-known/swagger`).
- **Unified Identity & Authorization** — Verified `Bearer` JWT tokens (`HS256` with `iss`, `aud`, `exp`, and matching `sub`) establish a `pi/auth.Principal` at the transport edge; services enforce self-or-permission checks (`users:read:any`, `users:write:any`).
- **Safe SQL Transactions & One-Shot Migrations** — Repositories compose transactions via `sql.Client.BeginTxContext` and verify ownership via `sql.Client.Owns` with nested rollback-only safety. `cmd/migration` runs lock-protected schema plans and upgrades (`app.MigrateContext` / `app.MigrationPlanContext` / `app.MigrationStatusContext`) without starting HTTP/gRPC listeners.
- **Supervised Cron Tasks** — `cmd/task` schedules jobs on Pi's built-in actor-supervised Cron runtime (`CronSkipIfRunning` overlap protection in `time.UTC`) with graceful shutdown draining.
- **Reproducible Codegen & Containerization** — `make generator` compiles staged protobuf, gRPC wrapper, mock, and OpenAPI outputs before publishing; the root `Dockerfile` builds a minimal non-root (`UID 10001`) multi-binary image (`server`, `task`, `migration`).

---

## Quick Start

### Option A: Scaffold a New Project via `pi` CLI (Recommended)

```bash
go install github.com/sllt/pi/cmd/pi@v0.5.2
pi init --module example.com/acme/orders ./orders
cd ./orders
go run ./cmd/migration up
go run ./cmd/server
```

`pi init` automatically creates `storage/` and generates a local `configs/.env` (`0600`) with a cryptographically random 64-char hex `JWT_SECRET`.

### Option B: Run Directly from This Repository

#### 1. Prepare Configuration

```bash
make env
```

`make env` copies `configs/.env.example` to `configs/.env` and populates a random 64-char hex `JWT_SECRET` (you can also generate one manually via `openssl rand -hex 32`; placeholder values are rejected at startup).

#### 2. Run Database Migrations

```bash
go run ./cmd/migration plan
go run ./cmd/migration up --timeout=10m --lock-ttl=15m
go run ./cmd/migration status
```

> The bundled `users` and `user_profiles` DDL targets SQLite (`storage/app.db`) out of the box. When switching business SQL to PostgreSQL or MySQL, update the SQL migrations and dialect guard accordingly.

#### 3. Start the HTTP Server

```bash
go run ./cmd/server
```

Local `configs/.env.example` enables live OpenAPI 3.0.3 endpoints (`/.well-known/openapi.json`) and the built-in offline Scalar UI (`/.well-known/swagger`) out of the box, printing the clickable Swagger UI URL on startup (when unset in production environments, both default to `false`).

#### 4. Start the Scheduled Task Runner (Optional)

```bash
go run ./cmd/task
```

---

## Project Structure

```text
pi-layout/
├── api/
│   ├── proto/user/          # Protobuf definitions & generated gRPC stubs
│   └── v1/                  # Typed HTTP request/response DTOs (pi.Meta)
├── cmd/
│   ├── migration/           # One-shot database migration CLI entrypoint
│   ├── openapi/             # Offline OpenAPI 3.0.3 exporter & drift checker
│   ├── server/              # HTTP (and optional gRPC) service entrypoint
│   └── task/                # Supervised Cron worker entrypoint
├── configs/                 # .env.example template (configs/.env is gitignored)
├── deploy/docker-compose/   # Optional local MySQL / Redis compose profiles
├── docs/                    # Generated openapi.json & openapi.yaml artifacts
├── internal/
│   ├── bootstrap/           # Fx modules, Pi Build wiring & OpenAPI config
│   ├── grpc/user/           # Generated Pi gRPC wrappers + handwritten server adapter
│   ├── handler/             # HTTP transport adapters
│   ├── middleware/          # JWT Principal extraction middleware
│   ├── migrationcmd/        # Lock-protected migration command runner
│   ├── model/               # Persistence models
│   ├── repository/          # SQL data access & transaction management
│   ├── router/              # Route groups, auth middleware & OpenAPI doc tags
│   ├── server/              # HTTP, gRPC & Cron server registration
│   ├── service/             # Domain logic, validation & Principal authorization
│   ├── task/                # Scheduled background job implementations
│   └── types/               # Transport-neutral domain DTOs
├── migrations/              # Versioned SQL migrations
├── pkg/                     # Shared packages (config, errcode, jwt, log, sid)
├── scripts/                 # Reproducible codegen, smoke test & container checks
└── test/                    # Integration tests & generated mocks
```

---

## Configuration & Security Highlights

- **JWT Authentication** — Credentials are accepted strictly from `Authorization: Bearer <token>`. Configure `JWT_SECRET` (≥ 32 bytes), `JWT_TTL` (default `1h`), `JWT_ISSUER` (default `pi-layout`), and `JWT_AUDIENCE` (default `pi-api`).
- **CORS & TLS** — Cross-origin access is disabled by default; set comma-separated `CORS_ALLOWED_ORIGINS` when needed. Configure `CERT_FILE` / `KEY_FILE` (HTTP) and `GRPC_CERT_FILE` / `GRPC_KEY_FILE` (gRPC) for TLS; invalid key pairs fail fast instead of falling back to plaintext.
- **gRPC Service Gate** — The sample gRPC `UserService` is disabled by default and only registers when both `GRPC_ENABLED=true` and `USER_GRPC_ENABLED=true` are set.
- **Error Envelopes** — `pkg/errcode.Error` aliases Pi's `apperror.Error`, mapping domain error kinds consistently across HTTP JSON responses, gRPC status codes, and CLI exit codes.

---

## Common Commands

```bash
make init            # Install pinned pi, mockgen, air, and protoc plugins into .tools/bin
make env             # Generate local configs/.env with a random 64-char hex JWT_SECRET
make build           # Build server, task, migration, and openapi binaries into ./bin
make test            # Run all unit and integration tests
make race            # Run all tests with -race enabled
make openapi         # Regenerate docs/openapi.json and docs/openapi.yaml
make generator       # Regenerate protobufs, gRPC wrappers, mocks, and OpenAPI specs safely
make check-generated # Verify zero drift in generated files
make smoke           # End-to-end binary smoke test (migration, auth, status codes, SIGTERM)
make container       # Build and verify the multi-entrypoint container image
```
