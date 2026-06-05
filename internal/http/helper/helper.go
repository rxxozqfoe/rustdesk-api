package helper

import (
	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
)

// UserRouteNames defines accessible route names for regular users.
var UserRouteNames = []string{
	"MyTagList", "MyAddressBookList", "MyInfo", "MyAddressBookCollection", "MyPeer", "MyShareRecordList", "MyLoginLog",
}

// AdminRouteNames defines accessible route names for admin users.
var AdminRouteNames = []string{"*"}

// CurUser extracts the current user from the Gin context.
// The user is set by auth middleware (BackendUserAuth / RustAuth).
func CurUser(c *gin.Context) *model.User {
	user, _ := c.Get("curUser")
	u, ok := user.(*model.User)
	if !ok {
		return nil
	}
	return u
}
