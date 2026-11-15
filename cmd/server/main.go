package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sllt/pi-layout/internal/bootstrap"
	"github.com/sllt/pi-layout/internal/server"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/logging"
	"go.uber.org/fx"
)

const fxLifecycleTimeout = 15 * time.Second

type fxStopper interface {
	Stop(context.Context) error
}

func stopFXApp(app fxStopper, timeout time.Duration) error {
	stopCtx, stopCancel := context.WithTimeout(context.Background(), timeout)
	defer stopCancel()

	return app.Stop(stopCtx)
}

func main() {
	if err := run(); err != nil {
		logging.NewLogger(logging.ERROR).Errorf("server error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) (err error) {
	var piApp *pi.App

	fxApp := fx.New(
		bootstrap.CoreModule,
		bootstrap.InfraModule,
		bootstrap.RepositoryModule,
		bootstrap.ServiceModule,
		bootstrap.HandlerModule,
		fx.Invoke(server.NewHTTPServer),
		fx.Populate(&piApp),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), fxLifecycleTimeout)
	defer cancel()

	if err := fxApp.Start(startCtx); err != nil {
		return err
	}
	defer func() {
		if stopErr := stopFXApp(fxApp, fxLifecycleTimeout); stopErr != nil {
			err = errors.Join(err, stopErr)
		}
	}()

	if piApp == nil {
		return errors.New("pi app was not populated")
	}

	return piApp.RunContext(ctx)
}
