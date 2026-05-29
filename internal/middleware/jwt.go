package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"kms-server/internal/auth"
)

// JWTAuth returns middleware that validates JWT tokens.
func JWTAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for public paths
		path := c.Request.URL.Path
		if isPublicPath(path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func isPublicPath(path string) bool {
	publicPaths := []string{
		"/health",
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/refresh",
		"/webhooks",
		"/api/profiles",
		"/api/published",
	}
	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
