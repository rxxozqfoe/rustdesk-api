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

// rustAuthEngine registers RustAuth plus a handler that echoes the user id set
// in context, so tests can confirm the passthrough populated curUser.
func rustAuthEngine(t *testing.T, kit *servicekit.Kit) *gin.Engine {
	engine := testutil.NewEngine(t)
	engine.GET("/api/ping",
		middleware.RustAuth(kit.Config, kit.Services.UserService),
		func(c *gin.Context) {
			u, _ := c.Get("curUser")
			usr, _ := u.(*model.User)
			id := uint(0)
			if usr != nil {
				id = usr.Id
			}
			c.JSON(http.StatusOK, gin.H{"uid": id})
		},
	)
	return engine
}

func TestRustAuth_ValidTokenPasses(t *testing.T) {
	kit := servicekit.New(t)
	u, token := seedTokenUser(t, kit)
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"uid":`)
	assert.NotZero(t, u.Id)
	assert.Contains(t, rec.Body.String(), `"uid":1`)
}

func TestRustAuth_MissingHeaderAborts(t *testing.T) {
	kit := servicekit.New(t)
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRustAuth_ShortHeaderAborts(t *testing.T) {
	kit := servicekit.New(t)
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	// 7 bytes or fewer is rejected before the prefix is stripped.
	req.Header.Set("Authorization", "Bearer ")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRustAuth_InvalidJWTAborts(t *testing.T) {
	kit := servicekit.New(t)
	// kit.Config.Jwt.Key is non-empty, so VerifyJWT runs and rejects garbage.
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt-token-value")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRustAuth_UnknownButValidJWTAborts(t *testing.T) {
	kit := servicekit.New(t)
	// Mint a structurally valid JWT for a user that has no UserToken row, so
	// VerifyJWT passes but InfoByAccessToken returns id 0.
	token := kit.Services.UserService.GenerateToken(&model.User{IdModel: model.IdModel{Id: 999}})
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRustAuth_DisabledUserAborts(t *testing.T) {
	kit := servicekit.New(t)
	_, token := seedTokenUser(t, kit, func(u *model.User) {
		u.Status = model.COMMON_STATUS_DISABLED
	})
	engine := rustAuthEngine(t, kit)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ping", "")
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
