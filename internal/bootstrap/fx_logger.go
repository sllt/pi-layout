package bootstrap

import (
	"strings"
	"time"

	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/logging"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// FxLoggerDeps injects the application logger when available, falling back to
// a standard Pi logger so early DI failures still render in Pi's unified format.
type FxLoggerDeps struct {
	fx.In
	Logger *log.Logger `optional:"true"`
}

// NewFxLogger adapts Uber Fx lifecycle and error events onto Pi's logger.
func NewFxLogger(deps FxLoggerDeps) fxevent.Logger {
	if deps.Logger != nil && deps.Logger.Logger != nil {
		return &fxLogger{logger: deps.Logger.Logger}
	}
	return &fxLogger{logger: logging.NewLogger(logging.INFO)}
}

type fxLogger struct {
	logger logging.Logger
}

func (l *fxLogger) LogEvent(event fxevent.Event) {
	if l == nil || l.logger == nil || event == nil {
		return
	}

	switch e := event.(type) {
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.logger.Errorf("fx logger initialization failed: %v", e.Err)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.logger.Errorf("fx supply %s failed: %v", e.TypeName, e.Err)
		}
	case *fxevent.Provided:
		if e.Err != nil {
			l.logger.Errorf("fx provide %s failed: %v", shortFuncName(e.ConstructorName), e.Err)
		}
	case *fxevent.Replaced:
		if e.Err != nil {
			l.logger.Errorf("fx replace failed: %v", e.Err)
		}
	case *fxevent.Decorated:
		if e.Err != nil {
			l.logger.Errorf("fx decorate %s failed: %v", shortFuncName(e.DecoratorName), e.Err)
		}
	case *fxevent.Run:
		if e.Err != nil {
			l.logger.Errorf("fx %s %s failed: %v", e.Kind, shortFuncName(e.Name), e.Err)
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			l.logger.Errorf("fx invoke %s failed: %v", shortFuncName(e.FunctionName), e.Err)
		}
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logger.Errorf("fx OnStart %s failed (%s): %v", shortFuncName(e.CallerName), e.Runtime.Round(time.Microsecond), e.Err)
			return
		}
		l.logger.Debugf("fx OnStart %s (%s)", shortFuncName(e.CallerName), e.Runtime.Round(time.Microsecond))
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logger.Errorf("fx OnStop %s failed (%s): %v", shortFuncName(e.CallerName), e.Runtime.Round(time.Microsecond), e.Err)
			return
		}
		l.logger.Debugf("fx OnStop %s (%s)", shortFuncName(e.CallerName), e.Runtime.Round(time.Microsecond))
	case *fxevent.RollingBack:
		l.logger.Errorf("fx startup failed, rolling back: %v", e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.logger.Errorf("fx rollback failed: %v", e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.logger.Errorf("fx failed to start: %v", e.Err)
			return
		}
		l.logger.Infof("Application started")
	case *fxevent.Stopping:
		l.logger.Infof("Stopping application (%s)", strings.ToLower(e.Signal.String()))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.logger.Errorf("fx stopped with error: %v", e.Err)
			return
		}
		l.logger.Infof("Application stopped")
	}
}

func shortFuncName(name string) string {
	trimmed := strings.TrimSpace(name)
	if idx := strings.LastIndexByte(trimmed, '/'); idx >= 0 && idx+1 < len(trimmed) {
		return trimmed[idx+1:]
	}
	return trimmed
}
