package apperr

// Error is the single application error type. Services return it; the central
// HTTP error mapper converts anything into the fixed envelope (docs/04).
// Cause is logged with request_id, never serialized to clients.
type Error struct {
	Status  int      `json:"-"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []Detail `json:"details,omitempty"`
	cause   error
}

type Detail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.cause }
func (e *Error) Cause() error  { return e.cause }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Wrap(err error, status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, cause: err}
}

func WithDetails(e *Error, details ...Detail) *Error {
	e.Details = append(e.Details, details...)
	return e
}

// Common constructors keep call sites terse and codes consistent.
func BadRequest(msg string) *Error        { return New(400, "BAD_REQUEST", msg) }
func Unauthorized(msg string) *Error      { return New(401, "UNAUTHORIZED", msg) }
func Forbidden(msg string) *Error         { return New(403, "FORBIDDEN", msg) }
func NotFound(msg string) *Error          { return New(404, "NOT_FOUND", msg) }
func Conflict(msg string) *Error          { return New(409, "CONFLICT", msg) }
func Validation(msg string) *Error        { return New(422, "VALIDATION_ERROR", msg) }
func BusinessRule(msg string) *Error      { return New(422, "BUSINESS_RULE_VIOLATED", msg) }
func RateLimited(msg string) *Error       { return New(429, "RATE_LIMITED", msg) }
func Internal(err error) *Error           { return Wrap(err, 500, "INTERNAL_ERROR", "Something went wrong") }
