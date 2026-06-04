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

// backendAuthEngine registers BackendUserAuth plus a handler that reports the
// curUser id and the token, both set by the middleware on success.
func backendAuthEngine(t *testing.T, kit *servicekit.Kit) *gin.Engine {
	engine := testutil.NewEngine(t)
	engine.GET("/admin/ping",
		middleware.BackendUserAuth(kit.Services.UserService),
		func(c *gin.Context) {
			u, _ := c.Get("curUser")
			tok, _ := c.Get("token")
			usr, _ := u.(*model.User)
			id := uint(0)
			if usr != nil {
				id = usr.Id
			}
			c.JSON(http.StatusOK, gin.H{"uid": id, "token": tok})
		},
	)
	return engine
}

func TestBackendUserAuth_ValidTokenPassesAndSetsUser(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	_, token := seedTokenUser(t, kit)
	engine := backendAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/ping", "")
	req.Header.Set("api-token", token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"uid":1`)
	assert.Contains(t, rec.Body.String(), token)
}

func TestBackendUserAuth_MissingTokenRejected(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	engine := backendAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/ping", "")
	engine.ServeHTTP(rec, req)

	// response.Fail wraps the business code in a 200 envelope; the business
	// code carries the failure.
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":403`)
	assert.Contains(t, rec.Body.String(), "NeedLogin")
}

func TestBackendUserAuth_UnknownTokenRejected(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	engine := backendAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/ping", "")
	req.Header.Set("api-token", "totally-bogus-token")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":403`)
	assert.Contains(t, rec.Body.String(), "NeedLogin")
}

func TestBackendUserAuth_ExpiredTokenRejected(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	_, token := seedExpiredToken(t, kit)
	engine := backendAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/ping", "")
	req.Header.Set("api-token", token)
	engine.ServeHTTP(rec, req)

	// Expired token -> InfoByAccessToken returns user id 0 -> NeedLogin.
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":403`)
	assert.Contains(t, rec.Body.String(), "NeedLogin")
}

func TestBackendUserAuth_DisabledUserRejected(t *testing.T) {
	wireResponse(t)
	kit := servicekit.New(t)
	_, token := seedTokenUser(t, kit, func(u *model.User) {
		u.Status = model.COMMON_STATUS_DISABLED
	})
	engine := backendAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/admin/ping", "")
	req.Header.Set("api-token", token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":401`)
	assert.Contains(t, rec.Body.String(), "UserDisabled")
}
