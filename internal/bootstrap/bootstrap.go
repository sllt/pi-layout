package bootstrap

import (
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/infra"
)

// NewPiApp creates a new pi.App.
func NewPiApp() *pi.App {
	return pi.New()
}

// NewLogger extracts pi's logger from the container and wraps it.
func NewLogger(app *pi.App) *log.Logger {
	return log.NewLogger(app.Container().Logger)
}

// NewDB extracts infra.DB from the pi app's container.
func NewDB(app *pi.App) infra.DB {
	return app.Container().SQL
}
