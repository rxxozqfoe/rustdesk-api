package admin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ServerConfig(t *testing.T) {
	hd, _ := newDeps(t)
	hd.Config.Rustdesk.IdServer = "id.example.com"
	hd.Config.Rustdesk.ApiServer = "http://api.example.com"
	cont := &admin.Config{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/config/server", cont.ServerConfig)

	rec, env := doJSON(t, engine, http.MethodGet, "/config/server", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "id.example.com")
	assert.Contains(t, rec.Body.String(), "api.example.com")
}

func TestConfig_AppConfig(t *testing.T) {
	hd, _ := newDeps(t)
	hd.Config.App.WebClient = 1
	cont := &admin.Config{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/config/app", cont.AppConfig)

	rec, env := doJSON(t, engine, http.MethodGet, "/config/app", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "web_client")
}

func TestConfig_AdminConfig_Anonymous(t *testing.T) {
	hd, _ := newDeps(t)
	hd.Config.Admin.Title = "My Console"
	cont := &admin.Config{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/config/admin", cont.AdminConfig)

	// No api-token header -> only the title is returned, no hello.
	rec, env := doJSON(t, engine, http.MethodGet, "/config/admin", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "My Console")
	assert.NotContains(t, rec.Body.String(), "hello")
}

func TestConfig_AdminConfig_Authenticated(t *testing.T) {
	hd, kit := newDeps(t)
	hd.Config.Admin.Title = "My Console"
	hd.Config.Admin.Hello = "Hi {{username}}"
	cont := &admin.Config{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/config/admin", cont.AdminConfig)

	u, token := seedTokenUser(t, kit, func(u *model.User) { u.Username = "consoleuser" })

	w := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/config/admin", "")
	req.Header.Set("api-token", token)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// hello template is rendered with the username substituted.
	assert.Contains(t, w.Body.String(), "Hi "+u.Username)
}
