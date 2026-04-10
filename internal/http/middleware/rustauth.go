package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
)

// RustAuth builds the RustDesk PC-client auth middleware. It validates the
// Bearer token (optionally verifying as a JWT when a jwt.key is configured)
// against the injected UserService.
func RustAuth(cfg *config.Config, users *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		if len(token) <= 7 {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		// Strip "Bearer " prefix.
		token = token[7:]

		if len(cfg.Jwt.Key) > 0 {
			uid, _ := users.VerifyJWT(token)
			if uid == 0 {
				c.JSON(401, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
		}

		user, ut := users.InfoByAccessToken(token)
		if user.Id == 0 {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		if !users.CheckUserEnable(user) {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("curUser", user)
		c.Set("token", token)
		users.AutoRefreshAccessToken(ut)

		c.Next()
	}
}
