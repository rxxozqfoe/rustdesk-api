package admin_test

import (
	"net/http"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Success(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "admin"
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"admin","password":"password"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	// Login success returns a LoginPayload carrying a token.
	assert.Contains(t, rec.Body.String(), `"token"`)

	// A login log + user token row should have been created.
	var tokenCount int64
	require.NoError(t, kit.DB.Model(&model.UserToken{}).Count(&tokenCount).Error)
	assert.Equal(t, int64(1), tokenCount)
}

func TestLogin_WrongPassword(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "admin" })

	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"admin","password":"wrong"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "UsernameOrPasswordError")
}

func TestLogin_UnknownUser(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"nobody","password":"password"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "UsernameOrPasswordError")
}

func TestLogin_DisabledUser(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "admin"
		u.Status = model.COMMON_STATUS_DISABLED
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"admin","password":"password"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "UserDisabled")
}

func TestLogin_MissingFieldsValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	// No password -> validator rejects with code 101.
	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"admin"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestLogin_PwdLoginDisabled(t *testing.T) {
	hd, _ := newDeps(t)
	hd.Config.App.DisablePwdLogin = true
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/login", cont.Login)

	rec, env := doJSON(t, engine, http.MethodPost, "/login",
		`{"username":"admin","password":"password"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "PwdLoginDisabled")
}

func TestLogout_RevokesToken(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)

	u, token := seedTokenUser(t, kit)
	engine.POST("/logout", withCurUser(u, token), cont.Logout)

	rec, env := doJSON(t, engine, http.MethodPost, "/logout", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	// Token row should be gone after logout.
	var cnt int64
	require.NoError(t, kit.DB.Model(&model.UserToken{}).
		Where("token = ?", token).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestLoginOptions_ReturnsFlags(t *testing.T) {
	hd, _ := newDeps(t)
	hd.Config.App.Register = true
	cont := &admin.Login{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/login-options", cont.LoginOptions)

	rec, env := doJSON(t, engine, http.MethodGet, "/login-options", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), `"register":true`)
	assert.Contains(t, rec.Body.String(), `"need_captcha"`)
}
