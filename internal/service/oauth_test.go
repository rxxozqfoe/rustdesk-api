package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newServiceAggregate builds the full *Service aggregate backed by a fresh
// in-memory DB, so services that reach siblings via ctx.Services work. The
// returned *gorm.DB is the same one the services use.
func newServiceAggregate(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := testutil.NewMemDB(t)
	cfg := testutil.NewConfig()
	svc := New(cfg, db, testutil.NewLogger(t), testutil.NewJwt(t), testutil.NewLock(), nil)
	return svc, db
}

// newOauthService builds just the OauthService against a ctx with a back-pointer
// to the aggregate, for tests that don't need siblings.
func newOauthService(t *testing.T) (*OauthService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.OauthService, db
}

func boolPtr(b bool) *bool { return &b }

// --- OauthCache get/set/delete + expiry ---

func TestOauthCache_SetGetDelete(t *testing.T) {
	os, _ := newOauthService(t)
	key := "cache-key-1"
	item := &OauthCacheItem{Op: "github", Action: OauthActionTypeLogin, UserId: 0}

	require.Nil(t, os.GetOauthCache(key), "cache should start empty")

	os.SetOauthCache(key, item, 0) // 0 = no expiry
	got := os.GetOauthCache(key)
	require.NotNil(t, got)
	assert.Equal(t, "github", got.Op)

	os.DeleteOauthCache(key)
	assert.Nil(t, os.GetOauthCache(key), "cache should be empty after delete")
}

func TestOauthCacheItem_ToOauthUser(t *testing.T) {
	oci := &OauthCacheItem{
		OpenId:   "open-1",
		Username: "alice",
		Name:     "Alice",
		Email:    "alice@example.com",
	}
	ou := oci.ToOauthUser()
	assert.Equal(t, "open-1", ou.OpenId)
	assert.Equal(t, "alice", ou.Username)
	assert.Equal(t, "Alice", ou.Name)
	assert.Equal(t, "alice@example.com", ou.Email)
}

func TestOauthCacheItem_UpdateFromOauthUser(t *testing.T) {
	oci := &OauthCacheItem{}
	oci.UpdateFromOauthUser(&model.OauthUser{
		OpenId:   "o",
		Username: "u",
		Name:     "n",
		Email:    "e@x.com",
	})
	assert.Equal(t, "o", oci.OpenId)
	assert.Equal(t, "u", oci.Username)
	assert.Equal(t, "n", oci.Name)
	assert.Equal(t, "e@x.com", oci.Email)
}

// --- BeginAuth ---

func TestBeginAuth_Webauth(t *testing.T) {
	os, _ := newOauthService(t)
	state, verifier, nonce, url, err := os.BeginAuth(model.OauthTypeWebauth)
	require.NoError(t, err)
	assert.Empty(t, verifier)
	assert.Empty(t, nonce)
	assert.NotEmpty(t, state)
	// state must be appended to the admin oauth url
	assert.Contains(t, url, "/_admin/#/oauth/")
	assert.Contains(t, url, state)
}

func TestBeginAuth_StateIsUnique(t *testing.T) {
	os, _ := newOauthService(t)
	s1, _, _, _, _ := os.BeginAuth(model.OauthTypeWebauth)
	s2, _, _, _, _ := os.BeginAuth(model.OauthTypeWebauth)
	assert.NotEqual(t, s1, s2, "state should be random per call")
	assert.Greater(t, len(s1), 10, "state = random(10) + unix timestamp")
}

func TestBeginAuth_ConfigNotFound(t *testing.T) {
	os, _ := newOauthService(t)
	// No oauth rows: GetOauthConfig returns ConfigNotFound, BeginAuth propagates.
	state, verifier, nonce, url, err := os.BeginAuth("github")
	require.Error(t, err)
	assert.Empty(t, url)
	assert.NotEmpty(t, state) // state is generated before config lookup
	assert.Empty(t, verifier)
	assert.Empty(t, nonce)
}

func TestBeginAuth_GithubBuildsAuthURL(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "github",
		OauthType:    model.OauthTypeGithub,
		ClientId:     "cid",
		ClientSecret: "secret",
		AutoRegister: boolPtr(true),
	}).Error)

	state, verifier, nonce, url, err := os.BeginAuth("github")
	require.NoError(t, err)
	assert.NotEmpty(t, state)
	// nonce is always generated for non-webauth providers
	assert.NotEmpty(t, nonce)
	// PKCE disabled by default => no verifier
	assert.Empty(t, verifier)
	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "client_id=cid")
	assert.Contains(t, url, "state="+state)
	assert.Contains(t, url, "nonce="+nonce)
}

func TestBeginAuth_GithubWithPKCE_S256(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "github",
		OauthType:    model.OauthTypeGithub,
		ClientId:     "cid",
		ClientSecret: "secret",
		AutoRegister: boolPtr(true),
		PkceEnable:   boolPtr(true),
		PkceMethod:   model.PKCEMethodS256,
	}).Error)

	_, verifier, _, url, err := os.BeginAuth("github")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier, "S256 PKCE should generate a verifier")
	assert.Contains(t, url, "code_challenge=")
	assert.Contains(t, url, "code_challenge_method=S256")
}

func TestBeginAuth_GithubWithPKCE_Plain(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "github",
		OauthType:    model.OauthTypeGithub,
		ClientId:     "cid",
		ClientSecret: "secret",
		AutoRegister: boolPtr(true),
		PkceEnable:   boolPtr(true),
		PkceMethod:   model.PKCEMethodPlain,
	}).Error)

	_, verifier, _, url, err := os.BeginAuth("github")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, url, "code_challenge_method=plain")
	// for plain, the challenge equals the verifier
	assert.Contains(t, url, "code_challenge="+verifier)
}

// --- GetOauthConfig ---

func TestGetOauthConfig_NotFound(t *testing.T) {
	os, _ := newOauthService(t)
	_, _, _, err := os.GetOauthConfig("nope")
	require.Error(t, err)
	assert.Equal(t, "ConfigNotFound", err.Error())
}

func TestGetOauthConfig_MissingClientSecret(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:        "github",
		OauthType: model.OauthTypeGithub,
		ClientId:  "cid",
		// ClientSecret empty
	}).Error)
	_, _, _, err := os.GetOauthConfig("github")
	require.Error(t, err)
	assert.Equal(t, "ConfigNotFound", err.Error())
}

func TestGetOauthConfig_Github(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "github",
		OauthType:    model.OauthTypeGithub,
		ClientId:     "cid",
		ClientSecret: "secret",
	}).Error)

	info, cfg, provider, err := os.GetOauthConfig("github")
	require.NoError(t, err)
	assert.Equal(t, "cid", cfg.ClientID)
	assert.Equal(t, "secret", cfg.ClientSecret)
	assert.Contains(t, cfg.RedirectURL, "/api/oidc/callback")
	assert.ElementsMatch(t, []string{"read:user", "user:email"}, cfg.Scopes)
	assert.Equal(t, model.OauthTypeGithub, info.OauthType)
	require.NotNil(t, provider)
}

func TestGetOauthConfig_Linuxdo(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "linuxdo",
		OauthType:    model.OauthTypeLinuxdo,
		ClientId:     "cid",
		ClientSecret: "secret",
	}).Error)

	_, cfg, provider, err := os.GetOauthConfig("linuxdo")
	require.NoError(t, err)
	assert.Equal(t, []string{"profile"}, cfg.Scopes)
	assert.Equal(t, "https://connect.linux.do/oauth2/authorize", cfg.Endpoint.AuthURL)
	require.NotNil(t, provider)
}

// Webauth is not a valid OAuth2 provider (it has no endpoint), so GetOauthConfig
// should reject it via the default switch branch.
func TestGetOauthConfig_WebauthUnsupported(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{
		Op:           "webauth",
		OauthType:    model.OauthTypeWebauth,
		ClientId:     "cid",
		ClientSecret: "secret",
	}).Error)

	_, _, _, err := os.GetOauthConfig("webauth")
	require.Error(t, err)
	assert.Equal(t, "unsupported OAuth type", err.Error())
}

// Note: OauthTypeOidc/Google branches call FetchOidcProvider which performs a
// live OIDC discovery HTTP request, so they are not unit-tested here.

// --- GetTypeByOp ---

func TestGetTypeByOp(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)

	typ, err := os.GetTypeByOp("github")
	require.NoError(t, err)
	assert.Equal(t, model.OauthTypeGithub, typ)
}

func TestGetTypeByOp_NotFound(t *testing.T) {
	os, _ := newOauthService(t)
	_, err := os.GetTypeByOp("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

// --- provider existence helpers ---

func TestIsOauthProviderExist_AndValidate(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)

	assert.True(t, os.IsOauthProviderExist("github"))
	assert.False(t, os.IsOauthProviderExist("nope"))

	assert.NoError(t, os.ValidateOauthProvider("github"))
	assert.Error(t, os.ValidateOauthProvider("nope"))
}

func TestGetOauthProviders(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)
	require.NoError(t, db.Create(&model.Oauth{Op: "linuxdo", OauthType: model.OauthTypeLinuxdo}).Error)

	ops := os.GetOauthProviders()
	assert.ElementsMatch(t, []string{"github", "linuxdo"}, ops)
}

// --- constructScopes ---

func TestConstructScopes(t *testing.T) {
	os, _ := newOauthService(t)
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty defaults", "", []string{"openid", "profile", "email"}},
		{"whitespace defaults", "   ", []string{"openid", "profile", "email"}},
		{"custom comma split", "a,b,c", []string{"a", "b", "c"}},
		{"single", "only", []string{"only"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, os.constructScopes(tt.in))
		})
	}
}

// --- FormatOauthInfo (validation + defaults) ---

func TestFormatOauthInfo_InvalidType(t *testing.T) {
	os, _ := newOauthService(t)
	err := os.FormatOauthInfo(&model.Oauth{OauthType: "bogus"})
	require.Error(t, err)
}

func TestFormatOauthInfo_GithubSetsOpAndDefaults(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGithub}
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.Equal(t, model.OauthTypeGithub, oa.Op)
	require.NotNil(t, oa.PkceEnable)
	assert.False(t, *oa.PkceEnable, "PkceEnable defaults to false")
	assert.Equal(t, model.PKCEMethodS256, oa.PkceMethod, "PkceMethod defaults to S256")
}

func TestFormatOauthInfo_GoogleSetsIssuer(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGoogle}
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.Equal(t, model.OauthTypeGoogle, oa.Op)
	assert.Equal(t, model.IssuerGoogle, oa.Issuer, "Google issuer defaults when empty")
}

func TestFormatOauthInfo_GoogleKeepsCustomIssuer(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGoogle, Issuer: "https://custom"}
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.Equal(t, "https://custom", oa.Issuer)
}

func TestFormatOauthInfo_OidcDefaultsOp(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeOidc} // Op empty
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.Equal(t, model.OauthTypeOidc, oa.Op, "oidc with empty op defaults to 'oidc'")
}

func TestFormatOauthInfo_OidcKeepsCustomOp(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeOidc, Op: "keycloak"}
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.Equal(t, "keycloak", oa.Op, "custom oidc op preserved")
}

func TestFormatOauthInfo_PreservesExplicitPkce(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeOidc, PkceEnable: boolPtr(true), PkceMethod: model.PKCEMethodPlain}
	require.NoError(t, os.FormatOauthInfo(oa))
	assert.True(t, *oa.PkceEnable)
	assert.Equal(t, model.PKCEMethodPlain, oa.PkceMethod)
}

// --- Create / Update / Delete / Info ---

func TestOauthCreate_AppliesDefaults(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGithub, ClientId: "c", ClientSecret: "s"}
	require.NoError(t, os.Create(oa))
	assert.NotZero(t, oa.Id)
	assert.Equal(t, model.OauthTypeGithub, oa.Op)

	loaded := os.InfoById(oa.Id)
	assert.Equal(t, "github", loaded.Op)
	require.NotNil(t, loaded.PkceEnable)
	assert.False(t, *loaded.PkceEnable)
}

func TestOauthCreate_InvalidTypeRejected(t *testing.T) {
	os, _ := newOauthService(t)
	err := os.Create(&model.Oauth{OauthType: "garbage"})
	require.Error(t, err)
}

func TestOauthInfoByOp_AndInfoById(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGithub, ClientId: "c", ClientSecret: "s"}
	require.NoError(t, os.Create(oa))

	byOp := os.InfoByOp("github")
	assert.Equal(t, oa.Id, byOp.Id)

	byId := os.InfoById(oa.Id)
	assert.Equal(t, "github", byId.Op)

	// not found returns zero-value struct, not nil
	none := os.InfoByOp("missing")
	assert.Zero(t, none.Id)
}

func TestOauthUpdate(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGithub, ClientId: "c", ClientSecret: "s"}
	require.NoError(t, os.Create(oa))

	oa.ClientId = "new-cid"
	require.NoError(t, os.Update(oa))
	assert.Equal(t, "new-cid", os.InfoById(oa.Id).ClientId)
}

func TestOauthDelete(t *testing.T) {
	os, _ := newOauthService(t)
	oa := &model.Oauth{OauthType: model.OauthTypeGithub, ClientId: "c", ClientSecret: "s"}
	require.NoError(t, os.Create(oa))
	require.NoError(t, os.Delete(oa))
	assert.Zero(t, os.InfoById(oa.Id).Id)
}

func TestOauthList_Pagination(t *testing.T) {
	os, _ := newOauthService(t)
	for _, op := range []string{"a", "b", "c"} {
		require.NoError(t, os.ctx.DB.Create(&model.Oauth{Op: op, OauthType: model.OauthTypeGithub}).Error)
	}
	res := os.List(1, 2, nil)
	assert.EqualValues(t, 3, res.Total)
	assert.Len(t, res.Oauths, 2)
}

// --- UserThird binding/unbinding ---

func TestBindAndUnbindOauthUser(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)

	ou := &model.OauthUser{OpenId: "open-1", Username: "u", Email: "U@X.com"}
	require.NoError(t, os.BindOauthUser(42, ou, "github"))

	ut := os.UserThirdInfo("github", "open-1")
	require.NotZero(t, ut.Id)
	assert.EqualValues(t, 42, ut.UserId)
	assert.Equal(t, model.OauthTypeGithub, ut.OauthType)
	assert.Equal(t, "u@x.com", ut.Email, "FromOauthUser lowercases email")

	require.NoError(t, os.UnBindOauthUser(42, "github"))
	assert.Zero(t, os.UserThirdInfo("github", "open-1").Id)
}

func TestBindOauthUser_OpNotFound(t *testing.T) {
	os, _ := newOauthService(t)
	// GetTypeByOp fails when op missing => bind returns error.
	err := os.BindOauthUser(1, &model.OauthUser{OpenId: "x"}, "ghost")
	require.Error(t, err)
}

func TestDeleteUserByUserId_RemovesAllBindings(t *testing.T) {
	os, db := newOauthService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)
	require.NoError(t, db.Create(&model.Oauth{Op: "linuxdo", OauthType: model.OauthTypeLinuxdo}).Error)
	require.NoError(t, os.BindOauthUser(7, &model.OauthUser{OpenId: "a"}, "github"))
	require.NoError(t, os.BindOauthUser(7, &model.OauthUser{OpenId: "b"}, "linuxdo"))

	require.NoError(t, os.DeleteUserByUserId(7))
	assert.Zero(t, os.UserThirdInfo("github", "a").Id)
	assert.Zero(t, os.UserThirdInfo("linuxdo", "b").Id)
}

// --- failResult / successResult helpers ---

func TestFailResult(t *testing.T) {
	r := failResult("MsgKey", "sub")
	assert.False(t, r.Success)
	assert.Equal(t, "oauth_fail.html", r.HTMLTemplate)
	assert.Equal(t, "MsgKey", r.HTMLData["message"])
	assert.Equal(t, "sub", r.HTMLData["sub_message"])
}

func TestSuccessResult(t *testing.T) {
	r := successResult("OK")
	assert.True(t, r.Success)
	assert.Equal(t, "oauth_success.html", r.HTMLTemplate)
	assert.Equal(t, "OK", r.HTMLData["message"])
}

// --- HandleCallback (branches not needing a live provider) ---

func TestHandleCallback_EmptyState(t *testing.T) {
	os, _ := newOauthService(t)
	r := os.HandleCallback("code", "")
	assert.False(t, r.Success)
	assert.Equal(t, "ParamIsEmpty", r.HTMLData["message"])
}

func TestHandleCallback_ExpiredCache(t *testing.T) {
	os, _ := newOauthService(t)
	r := os.HandleCallback("code", "unknown-state")
	assert.False(t, r.Success)
	assert.Equal(t, "OauthExpired", r.HTMLData["message"])
}

// --- HandleOidcAuthQuery polling ---

func TestHandleOidcAuthQuery_Expired(t *testing.T) {
	os, _ := newOauthService(t)
	r := os.HandleOidcAuthQuery("missing-code", "id", "uuid", "1.2.3.4")
	assert.Equal(t, "OauthExpired", r.ErrorMsg)
}

func TestHandleOidcAuthQuery_InProgress(t *testing.T) {
	os, _ := newOauthService(t)
	os.SetOauthCache("code-1", &OauthCacheItem{UserId: 0}, 0)
	r := os.HandleOidcAuthQuery("code-1", "id", "uuid", "1.2.3.4")
	assert.True(t, r.AuthInPrg)
	assert.Empty(t, r.ErrorMsg)
}

func TestHandleOidcAuthQuery_UserNotFound(t *testing.T) {
	os, _ := newOauthService(t)
	// cache says a user completed auth, but that user id does not exist in DB.
	os.SetOauthCache("code-2", &OauthCacheItem{UserId: 9999}, 0)
	r := os.HandleOidcAuthQuery("code-2", "id", "uuid", "1.2.3.4")
	assert.Equal(t, "UserNotFound", r.ErrorMsg)
}

func TestHandleOidcAuthQuery_Success(t *testing.T) {
	svc, db := newServiceAggregate(t)
	os := svc.OauthService
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "oauthlogin" })

	os.SetOauthCache("code-ok", &OauthCacheItem{
		UserId:     u.Id,
		DeviceType: "desktop",
		DeviceOs:   "linux",
		Id:         "devid",
		Uuid:       "dev-uuid",
	}, 0)

	r := os.HandleOidcAuthQuery("code-ok", "devid", "dev-uuid", "5.6.7.8")
	require.Empty(t, r.ErrorMsg)
	require.NotNil(t, r.User)
	require.NotNil(t, r.Token)
	assert.Equal(t, u.Id, r.User.Id)
	assert.NotEmpty(t, r.Token.Token)

	// cache should be consumed after a successful login
	assert.Nil(t, os.GetOauthCache("code-ok"))

	// a login log of type oauth should have been written
	var cnt int64
	db.Model(&model.LoginLog{}).Where("user_id = ? and type = ?", u.Id, model.LoginLogTypeOauth).Count(&cnt)
	assert.EqualValues(t, 1, cnt)
}

// --- provider constructors ---

func TestGithubProvider_Endpoint(t *testing.T) {
	os, _ := newOauthService(t)
	p := os.GithubProvider()
	require.NotNil(t, p)
	assert.Equal(t, model.UserEndpointGithub, p.UserInfoEndpoint())
}

func TestLinuxdoProvider_Endpoint(t *testing.T) {
	os, _ := newOauthService(t)
	p := os.LinuxdoProvider()
	require.NotNil(t, p)
	assert.Equal(t, model.UserEndpointLinuxdo, p.UserInfoEndpoint())
	assert.Equal(t, "https://connect.linux.do/oauth2/token", p.Endpoint().TokenURL)
}

// --- getHTTPClientWithProxy (pure branches) ---

func TestGetHTTPClientWithProxy_Disabled(t *testing.T) {
	os, _ := newOauthService(t)
	c := getHTTPClientWithProxy(os.ctx)
	assert.Equal(t, http.DefaultClient, c)
}

func TestGetHTTPClientWithProxy_EnabledEmptyHost(t *testing.T) {
	os, _ := newOauthService(t)
	os.ctx.Config.Proxy.Enable = true
	os.ctx.Config.Proxy.Host = ""
	c := getHTTPClientWithProxy(os.ctx)
	assert.Equal(t, http.DefaultClient, c, "empty proxy host falls back to default client")
}

func TestGetHTTPClientWithProxy_EnabledValidHost(t *testing.T) {
	os, _ := newOauthService(t)
	os.ctx.Config.Proxy.Enable = true
	os.ctx.Config.Proxy.Host = "http://127.0.0.1:8888"
	c := getHTTPClientWithProxy(os.ctx)
	require.NotEqual(t, http.DefaultClient, c)
	assert.Equal(t, 60*time.Second, c.Timeout)
}
