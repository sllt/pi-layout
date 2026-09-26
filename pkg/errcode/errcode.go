package errcode

// Error represents a structured application error with business code and HTTP status.
// It implements pi's StatusCodeResponder and CodeResponder interfaces,
// so pi automatically renders the correct HTTP status and JSON response.
type Error struct {
	BizCode int
	Message string
}

func New(code int, msg string) *Error {
	return &Error{
		BizCode: code,
		Message: msg,
	}
}

func (e *Error) Error() string {
	return e.Message
}

// PublicMessage marks this catalogued application message safe for clients.
func (e *Error) PublicMessage() string { return e.Message }

// Code implements pi's http.CodeResponder interface.
// Returns the business error code for the JSON response "code" field.
func (e *Error) Code() int {
	return e.BizCode
}

// StatusCode returns the appropriate HTTP status code for the error.
// For common HTTP errors (400-599), it returns the BizCode directly.
// For business errors (>= 1000), it returns 400 (Bad Request).
func (e *Error) StatusCode() int {
	if e.BizCode >= 400 && e.BizCode < 600 {
		return e.BizCode
	}
	return 400
}
