package log

import (
	"context"

	"github.com/sllt/pi/pkg/pi/logging"
)

// Logger wraps pi's logging.Logger so existing code can keep using *log.Logger
// without changing function signatures.
type Logger struct {
	logging.Logger
}

// NewLogger creates a Logger backed by the given pi logging.Logger.
func NewLogger(l logging.Logger) *Logger {
	if l == nil {
		l = logging.Noop()
	}
	return &Logger{Logger: l}
}

// WithContext returns a request-scoped ContextLogger that automatically attaches
// the OpenTelemetry TraceID from ctx (if present) to structured log entries.
func (l *Logger) WithContext(ctx context.Context) *logging.ContextLogger {
	if l == nil || l.Logger == nil {
		return logging.WithContext(ctx, logging.Noop())
	}
	return logging.WithContext(ctx, l.Logger)
}

// Debugw logs a debug message with structured key-value fields.
func (l *Logger) Debugw(msg string, kv ...any) {
	l.WithContext(context.Background()).WithCallerSkip(1).Debugw(msg, kv...)
}

// Infow logs an info message with structured key-value fields.
func (l *Logger) Infow(msg string, kv ...any) {
	l.WithContext(context.Background()).WithCallerSkip(1).Infow(msg, kv...)
}

// Warnw logs a warning message with structured key-value fields.
func (l *Logger) Warnw(msg string, kv ...any) {
	l.WithContext(context.Background()).WithCallerSkip(1).Warnw(msg, kv...)
}

// Errorw logs an error message with structured key-value fields.
func (l *Logger) Errorw(msg string, kv ...any) {
	l.WithContext(context.Background()).WithCallerSkip(1).Errorw(msg, kv...)
}

// With returns an immutable ContextLogger copy with pre-bound key-value fields.
func (l *Logger) With(kv ...any) *logging.ContextLogger {
	return l.WithContext(context.Background()).With(kv...)
}
