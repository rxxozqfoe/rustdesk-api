package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setCurUser returns a middleware that stashes u as curUser, emulating the
// upstream BackendUserAuth that AdminPrivilege depends on.
func setCurUser(u *model.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("curUser", u)
		c.Next()
	}
}

func privilegeEngine(t *testing.T, kit *servicekit.Kit, u *model.User) *gin.Engine {
	engine := testutil.NewEngine(t)
	engine.GET("/admin/secret",
		setCurUser(u),
		middleware.AdminPrivilege(kit.Services.UserService),
		okHandler,
	)
	return engine
}

func TestAdminPrivilege_AdminAllowed(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	admin := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		b := true
		u.IsAdmin = &b
	})
	engine := privilegeEngine(t, kit, admin)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/secret", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestAdminPrivilege_NonAdminRejected(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	// Default fixture user has IsAdmin=false.
	user := testutil.CreateUser(t, kit.DB)
	engine := privilegeEngine(t, kit, user)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/secret", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":403`)
	assert.Contains(t, rec.Body.String(), "NoAccess")
	assert.NotContains(t, rec.Body.String(), "pong")
}
