package bootstrap

import (
	"context"
	"os"

	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/config"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
	"go.uber.org/fx"
)

// NewPiApp creates a new pi.App.
func NewPiApp() (*pi.App, error) {
	cfg, err := config.LoadSnapshot("configs", os.Environ())
	if err != nil {
		return nil, err
	}
	values := cfg.Values()
	if cfg.Get("DB_DIALECT") == "sqlite" && cfg.Get("DB_MAX_OPEN_CONNECTION") == "" {
		values["DB_MAX_OPEN_CONNECTION"] = "1"
	}
	return pi.Build(
		pi.WithConfig(values),
		pi.WithManagedSQL(),
		pi.WithExplicitHTTPStatus(),
		pi.WithOpenAPI(DefaultOpenAPIConfig(values)),
	)
}

// RegisterRuntime makes Fx the lifecycle host. The startup budget does not own
// runtime workers; Pi's Wait reports fatal errors after resource cleanup.
func RegisterRuntime(lc fx.Lifecycle, app *pi.App, shutdowner fx.Shutdowner) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		if err := app.Start(ctx); err != nil {
			return err
		}
		go func() {
			if err := app.Wait(context.Background()); err != nil {
				app.Logger().Errorf("runtime failed: %v", err)
				_ = shutdowner.Shutdown(fx.ExitCode(1))
			}
		}()
		return nil
	}, OnStop: app.Stop})
}

// NewLogger extracts pi's logger from the app runtime and wraps it.
func NewLogger(app *pi.App) *log.Logger {
	return log.NewLogger(app.Logger())
}

// NewDB extracts sql.Client from the pi app runtime.
func NewDB(app *pi.App) piSQL.Client {
	return app.SQL()
}
