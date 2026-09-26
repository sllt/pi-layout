package errcode

import (
	"errors"
	"net/http"

	piHTTP "github.com/sllt/pi/pkg/pi/http"
)

// AsError normalizes arbitrary errors into layout's public error contract.
//
// Handlers and services should return *Error values directly when the error is
// safe to expose. Unknown errors are intentionally converted to the generic
// internal-server error so HTTP middleware does not leak implementation details.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	return ErrInternalServerError.WithCause(err)
}

// WriteHTTPError renders the same response envelope that Pi handlers use:
//
//	{"code": <business-code>, "data": null, "message": "..."}
//
// Use this helper in net/http middleware, where errors cannot be returned to
// Pi's normal handler responder.
func WriteHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	method := http.MethodGet
	if r != nil {
		method = r.Method
	}

	if err == nil {
		piHTTP.NewResponder(w, method).Respond(nil, nil)
		return
	}
	piHTTP.NewResponder(w, method).Respond(nil, AsError(err))
}
