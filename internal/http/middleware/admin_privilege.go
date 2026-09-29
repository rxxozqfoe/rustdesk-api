package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/helper"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/service"
)

// AdminPrivilege builds a middleware that allows the request through only if
// the current user (set by BackendUserAuth) is an admin.
func AdminPrivilege(users *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := helper.CurUser(c)
		if !users.IsAdmin(u) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}
		c.Next()
	}
}
