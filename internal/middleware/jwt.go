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
	// Exact-match public endpoints (no prefix matching to avoid auth bypass)
	exactPaths := map[string]bool{
		"/health":             true,
		"/api/auth/login":     true,
		"/api/auth/register":  true,
		"/api/auth/refresh":   true,
		"/api/profiles":       true, // GET list
	}
	// Prefix-match with trailing slash — must match a path segment boundary
	prefixPaths := []string{
		"/webhooks/",
		"/api/profiles/",
		"/api/published/",
		"/p/",
	}

	if exactPaths[path] {
		return true
	}
	for _, p := range prefixPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
