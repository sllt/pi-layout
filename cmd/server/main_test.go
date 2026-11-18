package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sllt/pi/pkg/pi"
	"go.uber.org/fx"
)

type stubFXApp struct {
	stopErr error
	stopCtx context.Context
}

func (s *stubFXApp) Stop(ctx context.Context) error {
	s.stopCtx = ctx
	return s.stopErr
}

func TestStopFXAppReturnsStopError(t *testing.T) {
	expectedErr := errors.New("stop failed")
	app := &stubFXApp{stopErr: expectedErr}

	err := stopFXApp(app, time.Millisecond)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected stop error %v, got %v", expectedErr, err)
	}
}

func TestStopFXAppUsesTimeoutContext(t *testing.T) {
	app := &stubFXApp{}

	if err := stopFXApp(app, time.Second); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if app.stopCtx == nil {
		t.Fatal("expected stop context to be passed")
	}
	if _, ok := app.stopCtx.Deadline(); !ok {
		t.Fatal("expected stop context to have a deadline")
	}
}

func setupServerTestEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir("configs", 0o700); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	dbPath := filepath.Join(t.TempDir(), "server.db")
	for k, v := range map[string]string{
		"PI_TELEMETRY":           "false",
		"METRICS_ENABLED":        "false",
		"METRICS_PORT":           "0",
		"JWT_SECRET":             "test-server-secret-at-least-32-bytes",
		"DB_DIALECT":             "sqlite",
		"DB_NAME":                dbPath,
		"REDIS_HOST":             "",
		"PUBSUB_BACKEND":         "",
		"TRACE_EXPORTER":         "",
		"TRACER_URL":             "",
		"TRACER_HOST":            "",
		"REMOTE_LOG_URL":         "",
		"CERT_FILE":              "",
		"KEY_FILE":               "",
		"LOG_LEVEL":              "ERROR",
		"HTTP_ENABLED":           "true",
		"HTTP_ADDR":              "127.0.0.1:0",
		"GRPC_ENABLED":           "false",
		"USER_GRPC_ENABLED":      "false",
		"CORS_ALLOWED_ORIGINS":   "https://web.example",
		"CORS_ALLOW_CREDENTIALS": "false",
	} {
		t.Setenv(k, v)
	}
}

func TestRunWithContext_CancelsCleanly(t *testing.T) {
	setupServerTestEnv(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := runWithContext(ctx); err != nil {
		t.Fatalf("expected nil error on context cancellation, got %v", err)
	}
}

func TestRunWithOptions_DownstreamOnStopRunsBeforeAppStop(t *testing.T) {
	setupServerTestEnv(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	started := make(chan struct{})
	var capturedApp *pi.App
	var downstreamRan bool
	var pingErrInDownstreamStop error

	done := make(chan error, 1)
	go func() {
		done <- runWithOptions(ctx, fx.NopLogger, fx.Invoke(func(lc fx.Lifecycle, app *pi.App) {
			capturedApp = app
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					close(started)
					return nil
				},
				OnStop: func(stopCtx context.Context) error {
					downstreamRan = true
					_, pingErrInDownstreamStop = app.SQL().ExecContext(stopCtx, "SELECT 1")
					return nil
				},
			})
		}))
	}()

	<-started
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("expected nil error from runWithOptions, got %v", err)
	}
	if !downstreamRan {
		t.Fatal("expected downstream OnStop hook to run")
	}
	if pingErrInDownstreamStop != nil {
		t.Fatalf("expected SQL to still be open during downstream OnStop, got %v", pingErrInDownstreamStop)
	}
	if capturedApp == nil {
		t.Fatal("expected pi.App to be captured")
	}
	if _, err := capturedApp.SQL().ExecContext(t.Context(), "SELECT 1"); err == nil {
		t.Fatal("expected SQL to be closed after Fx Stop completed")
	}
}
