package middleware

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	csrfpkg "github.com/mumtaz/reimbursement-system/backend/internal/pkg/csrftoken"
)

// CSRF enforces the signed double-submit check on every mutating request.
// Login/refresh are exempt: no authenticated session exists to abuse, and the
// refresh cookie is SameSite=Strict so cross-site requests cannot carry it.
func CSRF(secret string) gin.HandlerFunc {
	exempt := map[string]bool{
		"POST /api/v1/auth/login":   true,
		"POST /api/v1/auth/refresh": true,
	}
	return func(c *gin.Context) {
		switch c.Request.Method {
		case "POST", "PUT", "PATCH", "DELETE":
		default:
			c.Next()
			return
		}
		if exempt[c.Request.Method+" "+c.FullPath()] {
			c.Next()
			return
		}
		// Missing cookie == empty string → Verify fails below.
		cookieVal, _ := c.Cookie("csrf_token")
		headerVal := c.GetHeader("X-CSRF-Token")
		if err := csrfpkg.Verify(secret, cookieVal, headerVal); err != nil {
			c.Error(apperr.Forbidden("CSRF token missing or invalid"))
			c.Abort()
			return
		}
		c.Next()
	}
}
