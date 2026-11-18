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
