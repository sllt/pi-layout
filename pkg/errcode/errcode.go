package errcode

import "github.com/sllt/pi/pkg/pi/apperror"

// Error is transport independent. Adapters map Kind to HTTP/gRPC/CLI semantics.
type Error = apperror.Error

// New preserves the existing catalog helper; new business errors should choose
// their Kind explicitly with apperror.New.
func New(code int, msg string) *Error {
	kind := apperror.InvalidArgument
	switch code {
	case 401:
		kind = apperror.Unauthenticated
	case 403:
		kind = apperror.Forbidden
	case 404:
		kind = apperror.NotFound
	case 409:
		kind = apperror.Conflict
	case 500:
		kind = apperror.Internal
	case 503:
		kind = apperror.Unavailable
	}
	return apperror.New(kind, code, msg)
}
