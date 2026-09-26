package errcode

import "github.com/sllt/pi/pkg/pi/apperror"

var (
	// common errors
	ErrSuccess             = New(0, "ok")
	ErrBadRequest          = New(400, "Bad Request")
	ErrUnauthorized        = New(401, "Unauthorized")
	ErrForbidden           = New(403, "Forbidden")
	ErrNotFound            = New(404, "Not Found")
	ErrInternalServerError = New(500, "Internal Server Error")

	// biz errors
	ErrEmailAlreadyUse  = apperror.New(apperror.Conflict, 1001, "The email is already in use.")
	ErrInvalidSignature = New(1002, "Invalid signature")
)
