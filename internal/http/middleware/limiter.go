package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/utils"
	"net/http"
)

// Limiter builds a login/abuse rate-limiting middleware backed by the injected
// LoginLimiter.
func Limiter(loginLimiter *utils.LoginLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIp := c.ClientIP()
		banned, _ := loginLimiter.CheckSecurityStatus(clientIp)
		if banned {
			response.Fail(c, http.StatusLocked, response.TranslateMsg(c, "Banned"))
			c.Abort()
			return
		}
		c.Next()
	}
}
