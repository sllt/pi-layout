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

func runWithContext(ctx context.Context) error {
	return runWithOptions(ctx)
}

func runWithOptions(ctx context.Context, extraOpts ...fx.Option) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var piApp *pi.App

	opts := []fx.Option{
		bootstrap.CoreModule,
		bootstrap.InfraModule,
		bootstrap.RepositoryModule,
		bootstrap.ServiceModule,
		bootstrap.HandlerModule,
		fx.Invoke(server.NewHTTPServer),
		fx.Populate(&piApp),
	}
	opts = append(opts, extraOpts...)

	fxApp := fx.New(opts...)

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

	select {
	case <-ctx.Done():
		return nil
	case sig := <-fxApp.Wait():
		if sig.ExitCode != 0 {
			return piApp.Wait(context.Background())
		}
		return nil
	}
}
