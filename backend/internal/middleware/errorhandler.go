package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

// ErrorHandler is mounted right after RequestID so it wraps every later
// middleware + route handler. Handlers/middleware signal failures via
// c.Error(err) (+Abort); once the chain unwinds, the last error is mapped to
// the fixed envelope (docs/04). Anything non-apperr becomes generic 500 —
// internals never reach clients.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() {
			return
		}
		last := c.Errors.Last()
		if last == nil {
			return
		}

		var vErrs validator.ValidationErrors
		if errors.As(last.Err, &vErrs) && last.Err != nil {
			details := make([]apperr.Detail, 0, len(vErrs))
			for _, ve := range vErrs {
				details = append(details, apperr.Detail{
					Field:   ve.Field(),
					Message: validationMessage(ve),
				})
			}
			response.Err(c, apperr.WithDetails(
				apperr.New(422, "VALIDATION_ERROR", "Validation failed"),
				details...,
			))
			return
		}
		response.Err(c, last.Err)
	}
}

func validationMessage(ve validator.FieldError) string {
	switch ve.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "is too short"
	case "max":
		return "is too long"
	default:
		return "failed " + ve.Tag() + " validation"
	}
}
