package errcode

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) Response {
	t.Helper()

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}

	return resp
}

func TestWriteHTTPError_HTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)

	WriteHTTPError(rec, req, ErrUnauthorized)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	resp := decodeResponse(t, rec)
	if resp.Code != ErrUnauthorized.Code() || resp.Message != ErrUnauthorized.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteHTTPError_BusinessError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", http.NoBody)

	WriteHTTPError(rec, req, ErrEmailAlreadyUse)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	resp := decodeResponse(t, rec)
	if resp.Code != ErrEmailAlreadyUse.Code() || resp.Message != ErrEmailAlreadyUse.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteHTTPError_UnknownErrorIsSanitized(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", http.NoBody)

	WriteHTTPError(rec, req, errors.New("database password leaked"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	resp := decodeResponse(t, rec)
	if resp.Code != ErrInternalServerError.Code() || resp.Message != ErrInternalServerError.Error() || resp.Data != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
