package response

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

// Envelope is the fixed API response shape (docs/04).
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
}

type ErrorBody struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Details []apperr.Detail    `json:"details,omitempty"`
}

func OK(c *gin.Context, data interface{}, message string) {
	c.JSON(200, Envelope{Success: true, Data: data, Message: message})
}

func Created(c *gin.Context, data interface{}, message string) {
	c.JSON(201, Envelope{Success: true, Data: data, Message: message})
}

func NoContent(c *gin.Context) {
	c.Status(204)
}

// Err maps any error to the fixed error envelope. Application errors pass
// through; everything else becomes a generic 500 — internals never leak.
func Err(c *gin.Context, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		ae = apperr.Internal(err)
	}
	if ae.Status >= 500 {
		slog.ErrorContext(c.Request.Context(), "internal error",
			"request_id", c.GetString("request_id"),
			"code", ae.Code,
			"cause", ae.Cause())
	}
	c.JSON(ae.Status, Envelope{
		Success:   false,
		Error:     &ErrorBody{Code: ae.Code, Message: ae.Message, Details: ae.Details},
		RequestID: c.GetString("request_id"),
	})
}
