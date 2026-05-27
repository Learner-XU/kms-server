package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole returns middleware that checks the JWT "role" claim against the
// allowed roles list. The role must already be set in the Gin context by the
// JWTAuth middleware (key: "role").
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found in token"})
			return
		}
		role, ok := roleVal.(string)
		if !ok || !allowed[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}
