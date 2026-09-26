package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/sllt/pi-layout/pkg/errcode"
)

func TestSignMiddleware_MissingHeaderUsesUnifiedErrorResponse(t *testing.T) {
	t.Setenv("API_SIGN_KEY", "app-key")
	t.Setenv("API_SIGN_SECRET", "app-secret")

	handler := SignMiddleware(testLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp errcode.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != errcode.ErrBadRequest.Code() || resp.Message != errcode.ErrBadRequest.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSignMiddleware_InvalidSignatureUsesUnifiedErrorResponse(t *testing.T) {
	t.Setenv("API_SIGN_KEY", "app-key")
	t.Setenv("API_SIGN_SECRET", "app-secret")

	handler := SignMiddleware(testLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed", http.NoBody)
	req.Header.Set("Timestamp", "1700000000")
	req.Header.Set("Nonce", "nonce")
	req.Header.Set("App-Version", "1.0.0")
	req.Header.Set("Sign", "wrong")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp errcode.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != errcode.ErrInvalidSignature.Code() || resp.Message != errcode.ErrInvalidSignature.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSignMiddleware_ValidSignaturePassesThrough(t *testing.T) {
	t.Setenv("API_SIGN_KEY", "app-key")
	t.Setenv("API_SIGN_SECRET", "app-secret")

	nextCalled := false
	handler := SignMiddleware(testLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed", http.NoBody)
	req.Header.Set("Timestamp", "1700000000")
	req.Header.Set("Nonce", "nonce")
	req.Header.Set("App-Version", "1.0.0")
	req.Header.Set("Sign", signForTest("app-key", "1700000000", "nonce", "1.0.0", "app-secret"))
	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func signForTest(appKey, timestamp, nonce, appVersion, secret string) string {
	base := "AppKey" + appKey + "AppVersion" + appVersion + "Nonce" + nonce + "Timestamp" + timestamp + secret
	return strings.ToUpper(cryptor.Md5String(base))
}
