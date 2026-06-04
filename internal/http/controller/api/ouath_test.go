package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOidcAuth_UnknownProvider(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/oidc/auth", o.OidcAuth)

	rec := httptest.NewRecorder()
	// No oauth provider configured => BeginAuth returns ConfigNotFound.
	body := `{"op":"github","id":"d1","uuid":"u1"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/oidc/auth", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "ConfigNotFound")
}

func TestOidcAuth_WebauthProviderReturnsURL(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/oidc/auth", o.OidcAuth)

	rec := httptest.NewRecorder()
	// The "webauth" pseudo-provider needs no DB config; BeginAuth builds a URL.
	body := `{"op":"` + model.OauthTypeWebauth + `","id":"d1","uuid":"u1"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/oidc/auth", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.NotEmpty(t, res.Code)
	assert.Contains(t, res.URL, "/_admin/#/oauth/")

	// A cache item should have been stored under the returned state code.
	item := kit.Services.OauthService.GetOauthCache(res.Code)
	require.NotNil(t, item)
	assert.Equal(t, service.OauthActionTypeLogin, item.Action)
}

func TestOidcAuthQuery_ExpiredCode(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/oidc/auth-query", o.OidcAuthQuery)

	rec := httptest.NewRecorder()
	// Unknown code => cache miss => OauthExpired.
	req := testutil.JSONRequest(t, http.MethodGet, "/api/oidc/auth-query?code=nope&id=d1&uuid=u1", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "OauthExpired")
}

func TestOidcAuthQuery_AuthInProgress(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/oidc/auth-query", o.OidcAuthQuery)

	// Seed a cache item with no UserId yet => AuthInPrg branch.
	code := "pending-code"
	kit.Services.OauthService.SetOauthCache(code, &service.OauthCacheItem{
		Action: service.OauthActionTypeLogin,
	}, 0)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/oidc/auth-query?code="+code+"&id=d1&uuid=u1", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authorization in progress")
}

func TestOidcAuthQuery_Success(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "oauthuser" })

	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/oidc/auth-query", o.OidcAuthQuery)

	// Seed a cache item already bound to the user => login completes.
	code := "ready-code"
	kit.Services.OauthService.SetOauthCache(code, &service.OauthCacheItem{
		Action: service.OauthActionTypeLogin,
		UserId: u.Id,
	}, 0)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/oidc/auth-query?code="+code+"&id=d1&uuid=u1", "")
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
	assert.Equal(t, "oauthuser", res.User.Name)
}

func TestOauthMessage_RendersJS(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	o := &api.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/oauth/msg", o.Message)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/oauth/msg?title=Hello&msg=World", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/javascript", rec.Header().Get("Content-Type"))
	// Empty bundle => lookups fail, body stays empty; the handler must not panic.
	assert.Equal(t, http.StatusOK, rec.Code)
}
