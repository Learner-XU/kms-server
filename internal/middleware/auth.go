package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth returns a middleware that validates the Authorization: Bearer <key> header.
// If apiKey is empty, all requests are allowed (development mode).
// The middleware skips /health and /webhooks/gitea paths.
func APIKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no API key configured, allow all (development mode)
		if apiKey == "" {
			c.Next()
			return
		}

		// Skip auth for health check and webhook endpoints
		path := c.Request.URL.Path
		if path == "/health" || path == "/webhooks/gitea" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		// Expect "Bearer <key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		if parts[1] != apiKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid API key"})
			return
		}

		c.Next()
	}
}
