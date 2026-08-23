package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	jwtpkg "github.com/mumtaz/reimbursement-system/backend/internal/pkg/jwt"
)

const (
	CtxUserID   = "auth_user_id"
	CtxRole     = "auth_role"
	CtxDeptID   = "auth_department_id"
	CtxUserName = "auth_name"
)

// AuthN verifies the access-token cookie (stateless JWT) and injects identity
// into the request context. Tokens never touch JS-readable storage.
func AuthN(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil || strings.TrimSpace(token) == "" {
			c.Error(apperr.Unauthorized("Authentication required"))
			c.Abort()
			return
		}
		claims, err := jwtpkg.Verify(secret, token)
		if err != nil {
			c.Error(apperr.Unauthorized("Session expired"))
			c.Abort()
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxDeptID, claims.DepartmentID)
		c.Set(CtxUserName, claims.Name)
		c.Next()
	}
}

// RequireRole is the coarse route-level RBAC gate; object-level checks stay in
// services (docs/06 authorization layers).
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(CtxRole)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.Error(apperr.Forbidden("You do not have permission to perform this action"))
		c.Abort()
	}
}