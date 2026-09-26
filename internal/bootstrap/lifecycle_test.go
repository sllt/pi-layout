package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/infra"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestFXStableDatabaseAndStartupContext(t *testing.T) {
	t.Chdir(t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "fx.db")
	for k, v := range map[string]string{"DB_DIALECT": "sqlite", "DB_NAME": dbPath, "HTTP_ENABLED": "false", "GRPC_ENABLED": "false", "METRICS_ENABLED": "false"} {
		t.Setenv(k, v)
	}
	var app *pi.App
	var db infra.DB
	f := fx.New(CoreModule, fx.Populate(&app, &db), fx.NopLogger)
	require.NoError(t, f.Err())
	require.NotNil(t, db)
	_, err := os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
	workerContext := make(chan context.Context, 1)
	app.Go("fixture", func(c *pi.Context) error { workerContext <- c.Context; <-c.Done(); return c.Err() })
	startCtx, cancel := context.WithTimeout(t.Context(), time.Second)
	require.NoError(t, f.Start(startCtx))
	cancel()
	runtimeCtx := <-workerContext
	require.NoError(t, runtimeCtx.Err())
	require.Same(t, db, app.Container().SQL)
	_, err = db.ExecContext(t.Context(), "CREATE TABLE fixture (id INTEGER)")
	require.NoError(t, err)
	require.NoError(t, f.Stop(t.Context()))
	_, err = db.ExecContext(t.Context(), "SELECT 1")
	require.Error(t, err)
}
