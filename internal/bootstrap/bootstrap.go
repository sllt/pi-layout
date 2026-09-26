package bootstrap

import (
	"context"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/config"
	"github.com/sllt/pi/pkg/pi/infra"
	"go.uber.org/fx"
	"os"
)

// NewPiApp creates a new pi.App.
func NewPiApp() (*pi.App, error) {
	cfg, err := config.LoadSnapshot("configs", os.Environ())
	if err != nil {
		return nil, err
	}
	return pi.Build(pi.WithConfig(cfg.Values()), pi.WithManagedSQL())
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

// NewLogger extracts pi's logger from the container and wraps it.
func NewLogger(app *pi.App) *log.Logger {
	return log.NewLogger(app.Container().Logger)
}

// NewDB extracts infra.DB from the pi app's container.
func NewDB(app *pi.App) infra.DB {
	return app.Container().SQL
}
