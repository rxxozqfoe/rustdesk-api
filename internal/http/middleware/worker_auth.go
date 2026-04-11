package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

// WorkerAuth validates the Bearer token from build-worker requests against
// the configured worker.token shared secret.
func WorkerAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			response.Fail(c, 403, "worker API is not configured")
			c.Abort()
			return
		}

		auth := c.GetHeader("Authorization")
		expected := "Bearer " + token
		if auth != expected {
			response.Fail(c, 401, "invalid worker token")
			c.Abort()
			return
		}

		c.Next()
	}
}
