package server

import (
	"context"

	"github.com/sllt/pi-layout/migrations"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"go.uber.org/fx"
)

func RegisterMigrateServer(lc fx.Lifecycle, shutdowner fx.Shutdowner, app *pi.App, log *log.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			app.Migrate(migrations.All())
			log.Info("Migration success")

			go func() {
				if err := shutdowner.Shutdown(); err != nil {
					log.Errorf("Migration shutdown error: %v", err)
				}
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			log.Info("Migration stop")
			return nil
		},
	})
}
