package log

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/sllt/pi/pkg/pi/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerWithContext_PropagatesTraceID(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(logging.NewWriterLogger(logging.DEBUG, &buf, &buf))

	tid, err := trace.TraceIDFromHex("11111111111111111111111111111111")
	require.NoError(t, err)
	sid, err := trace.SpanIDFromHex("2222222222222222")
	require.NoError(t, err)
	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	}))

	l.WithContext(spanCtx).Errorf("operation failed: %s", "boom")

	var entry struct {
		Level   string `json:"level"`
		Message string `json:"message"`
		TraceID string `json:"trace_id"`
	}
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "operation failed: boom", entry.Message)
	assert.Equal(t, tid.String(), entry.TraceID)
}

func TestLoggerWithContext_NilAndZeroSafety(t *testing.T) {
	tid, err := trace.TraceIDFromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	sid, err := trace.SpanIDFromHex("bbbbbbbbbbbbbbbb")
	require.NoError(t, err)
	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	}))

	var nilLogger *Logger
	scoped := nilLogger.WithContext(spanCtx)
	require.NotNil(t, scoped)
	require.NotPanics(t, func() {
		scoped.Errorf("nil logger safe: %d", 1)
	})

	zeroLogger := &Logger{}
	require.NotPanics(t, func() {
		zeroLogger.WithContext(nil).Errorf("zero logger safe: %d", 2)
	})

	noopWrapped := NewLogger(nil)
	require.NotNil(t, noopWrapped)
	require.NotPanics(t, func() {
		noopWrapped.Info("nil base logger safe")
		noopWrapped.WithContext(spanCtx).Infof("nil base logger with context safe: %d", 3)
	})
}

func TestLogger_StructuredMethodsAndWith(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(logging.NewWriterLogger(logging.DEBUG, &buf, &buf))

	l.Debugw("dbg", "k1", "v1")
	l.Infow("info", "k2", 2)
	l.Warnw("warn", "k3", true)
	l.Errorw("err", "k4", "boom")
	l.With("component", "auth").Infow("login", "user_id", 42)

	out := buf.String()
	assert.Contains(t, out, `"message":"dbg"`)
	assert.Contains(t, out, `"k1":"v1"`)
	assert.Contains(t, out, `"message":"info"`)
	assert.Contains(t, out, `"k2":2`)
	assert.Contains(t, out, `"message":"warn"`)
	assert.Contains(t, out, `"k3":true`)
	assert.Contains(t, out, `"message":"err"`)
	assert.Contains(t, out, `"k4":"boom"`)
	assert.Contains(t, out, `"component":"auth"`)
	assert.Contains(t, out, `"user_id":42`)

	var nilLogger *Logger
	require.NotPanics(t, func() {
		nilLogger.Debugw("d", "a", 1)
		nilLogger.Infow("i", "a", 1)
		nilLogger.Warnw("w", "a", 1)
		nilLogger.Errorw("e", "a", 1)
		nilLogger.With("a", 1).Infow("x")
	})

	t.Run("SetShowCaller attributes Infow to caller instead of log.go", func(t *testing.T) {
		var callerBuf bytes.Buffer
		base := logging.NewWriterLogger(logging.DEBUG, &callerBuf, &callerBuf)
		if cc, ok := base.(logging.CallerConfigurer); ok {
			cc.SetShowCaller(true)
		}
		wrapped := NewLogger(base)
		wrapped.Infow("caller check", "k", "v")

		var entry struct {
			Caller  string         `json:"caller"`
			Message string         `json:"message"`
			Fields  map[string]any `json:"fields"`
		}
		require.NoError(t, json.Unmarshal(bytes.TrimSpace(callerBuf.Bytes()), &entry))
		assert.Equal(t, "caller check", entry.Message)
		assert.Equal(t, "v", entry.Fields["k"])
		assert.Contains(t, entry.Caller, "log_test.go:")
		assert.NotContains(t, entry.Caller, "pkg/log/log.go:")
	})
}
