# syntax=docker/dockerfile:1
FROM golang:1.25.0-bookworm AS build
WORKDIR /src
ENV GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -mod=readonly -trimpath -o /out/server ./cmd/server && \
    go build -mod=readonly -trimpath -o /out/task ./cmd/task && \
    go build -mod=readonly -trimpath -o /out/migration ./cmd/migration

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /app/storage /app/configs && chown -R 10001:10001 /app
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
USER 10001:10001
ENV HTTP_HOST=0.0.0.0 HTTP_PORT=8000 GRPC_ENABLED=false METRICS_ENABLED=false \
    DB_DIALECT=sqlite DB_NAME=storage/app.db PI_TELEMETRY=false
EXPOSE 8000
STOPSIGNAL SIGTERM
CMD ["server"]
