package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Success(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// default fixture password is "password"; username must be >=2 chars.
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "alice" })

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	body := `{"username":"alice","password":"password","id":"dev1","uuid":"uuid-x"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var res struct {
		AccessToken string `json:"access_token"`
		Type        string `json:"type"`
		User        struct {
			Name string `json:"name"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, "access_token", res.Type)
	assert.Equal(t, "alice", res.User.Name)

	// A UserToken row was persisted for the user.
	var count int64
	kit.DB.Model(&model.UserToken{}).Where("user_id = ? AND token = ?", u.Id, res.AccessToken).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestLogin_WrongPassword(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "alice" })

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	body := `{"username":"alice","password":"wrongpass"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	// response.Error sends 400 with {"error": ...}.
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "UsernameOrPasswordError")
}

func TestLogin_UnknownUser(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	body := `{"username":"nobody","password":"password"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "UsernameOrPasswordError")
}

func TestLogin_ValidationError(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	// username too short (< 2) triggers the validator before any DB lookup.
	body := `{"username":"a","password":"password"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_DisabledUser(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "alice"
		u.Status = model.COMMON_STATUS_DISABLED
	})

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	body := `{"username":"alice","password":"password"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "UserDisabled")
}

func TestLogin_PwdLoginDisabled(t *testing.T) {
	kit := servicekit.New(t)
	kit.Config.App.DisablePwdLogin = true
	hd := newDeps(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/login", l.Login)

	rec := httptest.NewRecorder()
	body := `{"username":"alice","password":"password"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/login", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "PwdLoginDisabled")
}

func TestLoginOptions_Empty(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/login-options", l.LoginOptions)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/login-options", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res []string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	// With no oauth providers configured, only the common-oidc descriptor is set.
	require.Len(t, res, 1)
	assert.Contains(t, res[0], "common-oidc/")
}

func TestLoginOptions_WithProvider(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// Seed an oauth provider row; GetOauthProviders plucks the op column.
	require.NoError(t, kit.DB.Create(&model.Oauth{Op: "github", OauthType: "github"}).Error)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/login-options", l.LoginOptions)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/login-options", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res []string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Len(t, res, 2)
	assert.Contains(t, res[0], "github")
	assert.Equal(t, "oidc/github", res[1])
}

func TestLogout_DeletesToken(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u, token := seedTokenUser(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/logout", realAuth(kit), l.Logout)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/logout", "")
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// The UserToken should be gone after logout.
	var count int64
	kit.DB.Model(&model.UserToken{}).Where("user_id = ? AND token = ?", u.Id, token).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestLogout_Unauthorized(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)

	l := &api.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/logout", realAuth(kit), l.Logout)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/logout", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
