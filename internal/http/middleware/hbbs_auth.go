package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

// HbbsAuth validates the Bearer token from rendezvous-server (hbbs) requests
// against the configured hbbs.token shared secret.
func HbbsAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			response.Fail(c, 403, "hbbs API is not configured")
			c.Abort()
			return
		}

		auth := c.GetHeader("Authorization")
		expected := "Bearer " + token
		if auth != expected {
			response.Fail(c, 401, "invalid hbbs token")
			c.Abort()
			return
		}

		c.Next()
	}
}
