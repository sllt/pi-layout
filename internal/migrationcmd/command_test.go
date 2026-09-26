package migrationcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCommandSQLiteLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DB_DIALECT", "sqlite")
	t.Setenv("DB_NAME", filepath.Join(dir, "app.db"))
	call := func(args ...string) summary {
		t.Helper()
		var out, logs bytes.Buffer
		args = append(args, "--config-dir", dir)
		if err := Run(t.Context(), args, &out, &logs); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, logs.String())
		}
		var s summary
		if err := json.Unmarshal(out.Bytes(), &s); err != nil {
			t.Fatalf("invalid summary: %v: %s", err, out.String())
		}
		if !s.OK {
			t.Fatal(s.Error)
		}
		return s
	}
	if s := call("plan"); len(s.Plan) != 2 || s.LockEnabled {
		t.Fatalf("unexpected plan: %+v", s)
	}
	if s := call("up", "--dry-run"); len(s.Skipped) != 2 || len(s.Applied) != 0 || !s.LockEnabled {
		t.Fatalf("unexpected dry-run: %+v", s)
	}
	if s := call("status"); len(s.Pending) != 2 || len(s.Applied) != 0 {
		t.Fatalf("dry-run changed state: %+v", s)
	}
	if s := call("up", "--target=20260206104000"); len(s.Applied) != 1 || len(s.Skipped) != 1 {
		t.Fatalf("unexpected target: %+v", s)
	}
	if s := call("up"); len(s.Applied) != 1 {
		t.Fatalf("unexpected apply: %+v", s)
	}
	if s := call("up"); len(s.Applied) != 0 || len(s.Skipped) != 2 {
		t.Fatalf("repeat ran migrations: %+v", s)
	}
	if s := call("status"); len(s.Applied) != 2 || len(s.Pending) != 0 || !s.StatePrecise {
		t.Fatalf("unexpected status: %+v", s)
	}
	if s := call("up", "--lock=false"); s.LockEnabled {
		t.Fatal("lock opt-out ignored")
	}
	// Each up reacquires the lease: the previous command released it before returning.
}

func TestCommandErrors(t *testing.T) {
	t.Setenv("DB_DIALECT", "sqlite")
	dir := t.TempDir()
	t.Setenv("DB_NAME", filepath.Join(dir, "missing", "app.db"))
	var out, logs bytes.Buffer
	if err := Run(t.Context(), []string{"up", "--config-dir", dir}, &out, &logs); err == nil {
		t.Fatal("expected open error")
	}
	var s summary
	if err := json.Unmarshal(out.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.OK || s.Error == "" {
		t.Fatalf("failure reported success: %+v", s)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	out.Reset()
	if err := Run(ctx, []string{"up", "--config-dir", dir}, &out, &logs); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	for _, args := range [][]string{{"down"}, {"up", "--timeout=0"}, {"up", "--lock-ttl=1s"}, {"up", "--target=-1"}, {"plan", "--lock=false"}, {"status", "--dry-run"}, {"up", "extra"}} {
		out.Reset()
		if err := Run(t.Context(), args, &out, &logs); err == nil {
			t.Fatalf("expected argument error: %v", args)
		}
		if out.Len() != 0 {
			t.Fatal("argument error opened execution path")
		}
	}
	if err := Run(t.Context(), []string{"--help"}, &out, &logs); err != nil {
		t.Fatal(err)
	}
}

func TestConfigPrecedenceAndParseErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_ENV", "test")
	t.Setenv("PI_MIGRATION_CONFIG_TEST", "os")
	for name, data := range map[string]string{
		".env":      "PI_MIGRATION_CONFIG_TEST=base\nPI_MIGRATION_FILE_ONLY=base\n",
		".test.env": "PI_MIGRATION_CONFIG_TEST=override\nPI_MIGRATION_FILE_ONLY=override\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Get("PI_MIGRATION_CONFIG_TEST") != "os" || cfg.Get("PI_MIGRATION_FILE_ONLY") != "override" {
		t.Fatal("wrong precedence")
	}
	if os.Getenv("PI_MIGRATION_FILE_ONLY") != "" {
		t.Fatal("loader changed process environment")
	}
	if err := os.WriteFile(filepath.Join(dir, ".test.env"), []byte("A='unterminated"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(dir); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestCommandDeadline(t *testing.T) {
	t.Setenv("DB_DIALECT", "sqlite")
	t.Setenv("DB_NAME", filepath.Join(t.TempDir(), "db"))
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	var out, logs bytes.Buffer
	err := Run(ctx, []string{"up", "--config-dir", t.TempDir()}, &out, &logs)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
