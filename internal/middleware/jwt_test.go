package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/logging"
)

func testLogger() *log.Logger {
	return log.NewLogger(logging.NewLogger(logging.INFO))
}

func TestStrictAuth_MissingTokenUsesUnifiedErrorResponse(t *testing.T) {
	mw := StrictAuth(nil, testLogger())

	nextCalled := false
	handler := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", http.NoBody)
	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var resp errcode.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != errcode.ErrUnauthorized.Code() || resp.Message != errcode.ErrUnauthorized.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestNoStrictAuth_MissingTokenPassesThrough(t *testing.T) {
	mw := NoStrictAuth(nil, testLogger())

	nextCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/register", http.NoBody)
	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}
