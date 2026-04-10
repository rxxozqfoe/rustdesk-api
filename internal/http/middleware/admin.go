package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
)

// BackendUserAuth builds the admin-panel auth middleware. It validates the
// api-token header against the injected UserService, stashes the user and
// token on the gin.Context, and auto-refreshes the token when it's close to
// expiry.
func BackendUserAuth(users *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("api-token")
		if token == "" {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}
		user, ut := users.InfoByAccessToken(token)
		if user.Id == 0 {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}

		if !users.CheckUserEnable(user) {
			response.Fail(c, 401, response.TranslateMsg(c, "UserDisabled"))
			c.Abort()
			return
		}

		c.Set("curUser", user)
		c.Set("token", token)
		users.AutoRefreshAccessToken(ut)

		c.Next()
	}
}
