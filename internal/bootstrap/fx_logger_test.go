package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func TestFxLoggerFormatsLifecycleAndErrors(t *testing.T) {
	var buf bytes.Buffer
	piLogger := logging.NewWriterLogger(logging.DEBUG, &buf, &buf)
	fxLog := NewFxLogger(FxLoggerDeps{Logger: log.NewLogger(piLogger)})

	// Construction-time DI noise is suppressed.
	fxLog.LogEvent(&fxevent.Provided{ConstructorName: "github.com/sllt/pi-layout/internal/bootstrap.NewPiApp"})
	fxLog.LogEvent(&fxevent.Invoking{FunctionName: "github.com/sllt/pi-layout/internal/server.NewHTTPServer"})
	require.Empty(t, buf.String())

	// Normal lifecycle hooks and start/stop milestones use Pi logger levels.
	fxLog.LogEvent(&fxevent.OnStartExecuted{
		CallerName: "github.com/sllt/pi-layout/internal/bootstrap.RegisterRuntime",
		Runtime:    250 * time.Microsecond,
	})
	fxLog.LogEvent(&fxevent.Started{})
	fxLog.LogEvent(&fxevent.Stopping{Signal: os.Interrupt})
	fxLog.LogEvent(&fxevent.OnStopExecuted{
		CallerName: "github.com/sllt/pi-layout/internal/bootstrap.RegisterRuntime",
		Runtime:    120 * time.Microsecond,
	})
	fxLog.LogEvent(&fxevent.Stopped{})

	out := buf.String()
	require.Contains(t, out, "fx OnStart bootstrap.RegisterRuntime")
	require.Contains(t, out, "Application started")
	require.Contains(t, out, "Stopping application (interrupt)")
	require.Contains(t, out, "fx OnStop bootstrap.RegisterRuntime")
	require.Contains(t, out, "Application stopped")

	// Errors are logged at ERROR level.
	buf.Reset()
	fxLog.LogEvent(&fxevent.OnStartExecuted{
		CallerName: "github.com/sllt/pi-layout/internal/bootstrap.RegisterRuntime",
		Runtime:    time.Millisecond,
		Err:        errors.New("db down"),
	})
	fxLog.LogEvent(&fxevent.RollingBack{StartErr: errors.New("db down")})
	fxLog.LogEvent(&fxevent.Started{Err: errors.New("db down")})
	errOut := buf.String()
	require.Contains(t, errOut, "fx OnStart bootstrap.RegisterRuntime failed")
	require.Contains(t, errOut, "fx startup failed, rolling back: db down")
	require.Contains(t, errOut, "fx failed to start: db down")
}

func TestFxLoggerFallbackAndIntegration(t *testing.T) {
	// Fallback when *log.Logger is nil does not panic.
	fallback := NewFxLogger(FxLoggerDeps{})
	require.NotNil(t, fallback)

	// End-to-end Fx app routes events through NewFxLogger instead of stderr ConsoleLogger.
	var buf bytes.Buffer
	piLogger := logging.NewWriterLogger(logging.INFO, &buf, &buf)
	app := fx.New(
		fx.Provide(func() *log.Logger { return log.NewLogger(piLogger) }),
		fx.WithLogger(NewFxLogger),
		fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error { return nil },
				OnStop:  func(context.Context) error { return nil },
			})
		}),
	)
	require.NoError(t, app.Start(t.Context()))
	require.NoError(t, app.Stop(t.Context()))
	require.Contains(t, buf.String(), "Application started")
	require.Contains(t, buf.String(), "Application stopped")
	require.NotContains(t, buf.String(), "[Fx]")
}
