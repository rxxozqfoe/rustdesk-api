package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"github.com/rxxozqfoe/rustdesk-api/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	// "golang.org/x/oauth2/google"
	"gorm.io/gorm"
	// "io"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OauthService struct {
	ctx *ServiceContext
}

// Define a struct to parse the .well-known/openid-configuration response
type OidcEndpoint struct {
	Issuer   string `json:"issuer"`
	AuthURL  string `json:"authorization_endpoint"`
	TokenURL string `json:"token_endpoint"`
	UserInfo string `json:"userinfo_endpoint"`
}

type OauthCacheItem struct {
	UserId     uint   `json:"user_id"`
	Id         string `json:"id"` //rustdesk的设备ID
	Op         string `json:"op"`
	Action     string `json:"action"`
	Uuid       string `json:"uuid"`
	DeviceName string `json:"device_name"`
	DeviceOs   string `json:"device_os"`
	DeviceType string `json:"device_type"`
	OpenId     string `json:"open_id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Verifier   string `json:"verifier"` // used for oauth pkce
	Nonce      string `json:"nonce"`
}

func (oci *OauthCacheItem) ToOauthUser() *model.OauthUser {
	return &model.OauthUser{
		OpenId:   oci.OpenId,
		Username: oci.Username,
		Name:     oci.Name,
		Email:    oci.Email,
	}
}

var OauthCache = &sync.Map{}

const (
	OauthActionTypeLogin = "login"
	OauthActionTypeBind  = "bind"
)

func (oci *OauthCacheItem) UpdateFromOauthUser(oauthUser *model.OauthUser) {
	oci.OpenId = oauthUser.OpenId
	oci.Username = oauthUser.Username
	oci.Name = oauthUser.Name
	oci.Email = oauthUser.Email
}

func (os *OauthService) GetOauthCache(key string) *OauthCacheItem {
	v, ok := OauthCache.Load(key)
	if !ok {
		return nil
	}
	return v.(*OauthCacheItem)
}

func (os *OauthService) SetOauthCache(key string, item *OauthCacheItem, expire uint) {
	OauthCache.Store(key, item)
	if expire > 0 {
		time.AfterFunc(time.Duration(expire)*time.Second, func() {
			os.DeleteOauthCache(key)
		})
	}
}

func (os *OauthService) DeleteOauthCache(key string) {
	OauthCache.Delete(key)
}

func (os *OauthService) BeginAuth(op string) (state, verifier, nonce, url string, err error) {
	state = utils.RandomString(10) + strconv.FormatInt(time.Now().Unix(), 10)
	verifier = ""
	nonce = ""
	if op == model.OauthTypeWebauth {
		url = os.ctx.Config.Rustdesk.ApiServer + "/_admin/#/oauth/" + state
		//url = "http://localhost:8888/_admin/#/oauth/" + code
		return state, verifier, nonce, url, nil
	}
	oauthInfo, oauthConfig, _, err := os.GetOauthConfig(op)
	if err == nil {
		extras := make([]oauth2.AuthCodeOption, 0, 3)

		nonce = utils.RandomString(10)
		extras = append(extras, oauth2.SetAuthURLParam("nonce", nonce))

		if oauthInfo.PkceEnable != nil && *oauthInfo.PkceEnable {
			extras = append(extras, oauth2.AccessTypeOffline)
			verifier = oauth2.GenerateVerifier()
			switch oauthInfo.PkceMethod {
			case model.PKCEMethodS256:
				extras = append(extras, oauth2.S256ChallengeOption(verifier))
			case model.PKCEMethodPlain:
				// oauth2 does not have a plain challenge option, so we add it manually
				extras = append(extras, oauth2.SetAuthURLParam("code_challenge_method", "plain"), oauth2.SetAuthURLParam("code_challenge", verifier))
			}
		}

		return state, verifier, nonce, oauthConfig.AuthCodeURL(state, extras...), err
	}

	return state, verifier, nonce, "", err
}

func (os *OauthService) FetchOidcProvider(issuer string) (*oidc.Provider, error) {

	// Get the HTTP client (with or without proxy based on configuration)
	client := getHTTPClientWithProxy(os.ctx)

	ctx := oidc.ClientContext(context.Background(), client)

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (os *OauthService) GithubProvider() *oidc.Provider {
	return (&oidc.ProviderConfig{
		IssuerURL:     "",
		AuthURL:       github.Endpoint.AuthURL,
		TokenURL:      github.Endpoint.TokenURL,
		DeviceAuthURL: github.Endpoint.DeviceAuthURL,
		UserInfoURL:   model.UserEndpointGithub,
		JWKSURL:       "",
		Algorithms:    nil,
	}).NewProvider(context.Background())
}

func (os *OauthService) LinuxdoProvider() *oidc.Provider {
	return (&oidc.ProviderConfig{
		IssuerURL:     "",
		AuthURL:       "https://connect.linux.do/oauth2/authorize",
		TokenURL:      "https://connect.linux.do/oauth2/token",
		DeviceAuthURL: "",
		UserInfoURL:   model.UserEndpointLinuxdo,
		JWKSURL:       "",
		Algorithms:    nil,
	}).NewProvider(context.Background())
}

// GetOauthConfig retrieves the OAuth2 configuration based on the provider name
func (os *OauthService) GetOauthConfig(op string) (oauthInfo *model.Oauth, oauthConfig *oauth2.Config, provider *oidc.Provider, err error) {
	//oauthInfo, oauthConfig, err = os.getOauthConfigGeneral(op)
	oauthInfo = os.InfoByOp(op)
	if oauthInfo.Id == 0 || oauthInfo.ClientId == "" || oauthInfo.ClientSecret == "" {
		return nil, nil, nil, errors.New("ConfigNotFound")
	}
	oauthConfig = &oauth2.Config{
		ClientID:     oauthInfo.ClientId,
		ClientSecret: oauthInfo.ClientSecret,
		RedirectURL:  os.ctx.Config.Rustdesk.ApiServer + "/api/oidc/callback",
	}

	// Maybe should validate the oauthConfig here
	oauthType := oauthInfo.OauthType
	err = model.ValidateOauthType(oauthType)
	if err != nil {
		return nil, nil, nil, err
	}
	switch oauthType {
	case model.OauthTypeGithub:
		oauthConfig.Endpoint = github.Endpoint
		oauthConfig.Scopes = []string{"read:user", "user:email"}
		provider = os.GithubProvider()
	case model.OauthTypeLinuxdo:
		provider = os.LinuxdoProvider()
		oauthConfig.Endpoint = provider.Endpoint()
		oauthConfig.Scopes = []string{"profile"}
	//case model.OauthTypeGoogle: //google单独出来，可以少一次FetchOidcEndpoint请求
	//	oauthConfig.Endpoint = google.Endpoint
	//	oauthConfig.Scopes = os.constructScopes(oauthInfo.Scopes)
	case model.OauthTypeOidc, model.OauthTypeGoogle:
		provider, err = os.FetchOidcProvider(oauthInfo.Issuer)
		if err != nil {
			return nil, nil, nil, err
		}
		oauthConfig.Endpoint = provider.Endpoint()
		oauthConfig.Scopes = os.constructScopes(oauthInfo.Scopes)
	default:
		return nil, nil, nil, errors.New("unsupported OAuth type")
	}
	return oauthInfo, oauthConfig, provider, nil
}

func getHTTPClientWithProxy(ctx *ServiceContext) *http.Client {
	//add timeout 30s
	timeout := time.Duration(60) * time.Second
	if ctx.Config.Proxy.Enable {
		if ctx.Config.Proxy.Host == "" {
			ctx.Logger.Warn("Proxy is enabled but proxy host is empty.")
			return http.DefaultClient
		}
		proxyURL, err := url.Parse(ctx.Config.Proxy.Host)
		if err != nil {
			ctx.Logger.Warn("Invalid proxy URL: ", err)
			return http.DefaultClient
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		return &http.Client{Transport: transport, Timeout: timeout}
	}
	return http.DefaultClient
}
func (os *OauthService) callbackBase(oauthConfig *oauth2.Config, provider *oidc.Provider, code string, verifier string, nonce string, userData interface{}) (client *http.Client, err error) {

	// 设置代理客户端
	httpClient := getHTTPClientWithProxy(os.ctx)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	exchangeOpts := make([]oauth2.AuthCodeOption, 0, 1)
	if verifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(verifier))
	}

	token, err := oauthConfig.Exchange(ctx, code, exchangeOpts...)

	if err != nil {
		os.ctx.Logger.Warn("oauthConfig.Exchange() failed: ", err)
		return nil, errors.New("GetOauthTokenError")
	}

	// 获取 ID Token， github没有id_token
	rawIDToken, ok := token.Extra("id_token").(string)
	if ok && rawIDToken != "" {
		// 验证 ID Token
		v := provider.Verifier(&oidc.Config{ClientID: oauthConfig.ClientID})
		idToken, err2 := v.Verify(ctx, rawIDToken)
		if err2 != nil {
			os.ctx.Logger.Warn("IdTokenVerifyError: ", err2)
			return nil, errors.New("IdTokenVerifyError")
		}
		if nonce != "" {
			// 验证 nonce
			var claims struct {
				Nonce string `json:"nonce"`
			}
			if err2 = idToken.Claims(&claims); err2 != nil {
				os.ctx.Logger.Warn("Failed to parse ID Token claims: ", err)
				return nil, errors.New("IDTokenClaimsError")
			}

			if claims.Nonce != nonce {
				os.ctx.Logger.Warn("Nonce does not match")
				return nil, errors.New("NonceDoesNotMatch")
			}
		}
	}

	// 获取用户信息
	client = oauthConfig.Client(ctx, token)
	resp, err := client.Get(provider.UserInfoEndpoint())
	if err != nil {
		os.ctx.Logger.Warn("failed getting user info: ", err)
		return nil, errors.New("GetOauthUserInfoError")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			os.ctx.Logger.Warn("failed closing response body: ", closeErr)
		}
	}()

	// 解析用户信息
	if err = json.NewDecoder(resp.Body).Decode(userData); err != nil {
		os.ctx.Logger.Warn("failed decoding user info: ", err)
		return nil, errors.New("DecodeOauthUserInfoError")
	}

	return client, nil
}

// githubCallback github回调
func (os *OauthService) githubCallback(oauthConfig *oauth2.Config, provider *oidc.Provider, code, verifier, nonce string) (*model.OauthUser, error) {
	var user = &model.GithubUser{}
	client, err := os.callbackBase(oauthConfig, provider, code, verifier, nonce, user)
	if err != nil {
		return nil, err
	}
	err = os.getGithubPrimaryEmail(client, user)
	if err != nil {
		return nil, err
	}
	return user.ToOauthUser(), nil
}

// linuxdoCallback linux.do回调
func (os *OauthService) linuxdoCallback(oauthConfig *oauth2.Config, provider *oidc.Provider, code, verifier, nonce string) (*model.OauthUser, error) {
	var user = &model.LinuxdoUser{}
	_, err := os.callbackBase(oauthConfig, provider, code, verifier, nonce, user)
	if err != nil {
		return nil, err
	}
	return user.ToOauthUser(), nil
}

// oidcCallback oidc回调, 通过code获取用户信息
func (os *OauthService) oidcCallback(oauthConfig *oauth2.Config, provider *oidc.Provider, code, verifier, nonce string) (*model.OauthUser, error) {
	var user = &model.OidcUser{}
	if _, err := os.callbackBase(oauthConfig, provider, code, verifier, nonce, user); err != nil {
		return nil, err
	}
	return user.ToOauthUser(), nil
}

// Callback: Get user information by code and op(Oauth provider)
func (os *OauthService) Callback(code, verifier, op, nonce string) (oauthUser *model.OauthUser, err error) {
	oauthInfo, oauthConfig, provider, err := os.GetOauthConfig(op)
	// oauthType is already validated in GetOauthConfig
	if err != nil {
		return nil, err
	}
	oauthType := oauthInfo.OauthType
	switch oauthType {
	case model.OauthTypeGithub:
		oauthUser, err = os.githubCallback(oauthConfig, provider, code, verifier, nonce)
	case model.OauthTypeLinuxdo:
		oauthUser, err = os.linuxdoCallback(oauthConfig, provider, code, verifier, nonce)
	case model.OauthTypeOidc, model.OauthTypeGoogle:
		oauthUser, err = os.oidcCallback(oauthConfig, provider, code, verifier, nonce)
	default:
		return nil, errors.New("unsupported OAuth type")
	}
	return oauthUser, err
}

func (os *OauthService) UserThirdInfo(op string, openId string) *model.UserThird {
	ut := &model.UserThird{}
	os.ctx.DB.Where("open_id = ? and op = ?", openId, op).First(ut)
	return ut
}

// BindOauthUser: Bind third party account
func (os *OauthService) BindOauthUser(userId uint, oauthUser *model.OauthUser, op string) error {
	utr := &model.UserThird{}
	oauthType, err := os.GetTypeByOp(op)
	if err != nil {
		return err
	}
	utr.FromOauthUser(userId, oauthUser, oauthType, op)
	return os.ctx.DB.Create(utr).Error
}

// UnBindOauthUser: Unbind third party account
func (os *OauthService) UnBindOauthUser(userId uint, op string) error {
	return os.UnBindThird(op, userId)
}

// UnBindThird: Unbind third party account
func (os *OauthService) UnBindThird(op string, userId uint) error {
	return os.ctx.DB.Where("user_id = ? and op = ?", userId, op).Delete(&model.UserThird{}).Error
}

// DeleteUserByUserId: When user is deleted, delete all third party bindings
func (os *OauthService) DeleteUserByUserId(userId uint) error {
	return os.ctx.DB.Where("user_id = ?", userId).Delete(&model.UserThird{}).Error
}

// InfoById 根据id获取Oauth信息
func (os *OauthService) InfoById(id uint) *model.Oauth {
	oauthInfo := &model.Oauth{}
	os.ctx.DB.Where("id = ?", id).First(oauthInfo)
	return oauthInfo
}

// InfoByOp 根据op获取Oauth信息
func (os *OauthService) InfoByOp(op string) *model.Oauth {
	oauthInfo := &model.Oauth{}
	os.ctx.DB.Where("op = ?", op).First(oauthInfo)
	return oauthInfo
}

// Helper function to construct scopes
func (os *OauthService) constructScopes(scopes string) []string {
	scopes = strings.TrimSpace(scopes)
	if scopes == "" {
		scopes = model.OIDC_DEFAULT_SCOPES
	}
	return strings.Split(scopes, ",")
}

func (os *OauthService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.OauthList {
	res := &model.OauthList{}
	queryList[model.Oauth](os.ctx.DB, page, pageSize, res, &res.Oauths, where)
	return res
}

// GetTypeByOp 根据op获取OauthType
func (os *OauthService) GetTypeByOp(op string) (string, error) {
	oauthInfo := &model.Oauth{}
	if os.ctx.DB.Where("op = ?", op).First(oauthInfo).Error != nil {
		return "", fmt.Errorf("OAuth provider with op '%s' not found", op)
	}
	return oauthInfo.OauthType, nil
}

// ValidateOauthProvider 验证Oauth提供者是否正确
func (os *OauthService) ValidateOauthProvider(op string) error {
	if !os.IsOauthProviderExist(op) {
		return fmt.Errorf("OAuth provider with op '%s' not found", op)
	}
	return nil
}

// IsOauthProviderExist 验证Oauth提供者是否存在
func (os *OauthService) IsOauthProviderExist(op string) bool {
	oauthInfo := &model.Oauth{}
	// 使用 Gorm 的 Take 方法查找符合条件的记录
	if err := os.ctx.DB.Where("op = ?", op).Take(oauthInfo).Error; err != nil {
		return false
	}
	return true
}

// FormatOauthInfo validates and sets defaults on an Oauth config before persistence.
func (os *OauthService) FormatOauthInfo(oa *model.Oauth) error {
	oauthType := strings.TrimSpace(oa.OauthType)
	if err := model.ValidateOauthType(oa.OauthType); err != nil {
		return err
	}
	switch oauthType {
	case model.OauthTypeGithub:
		oa.Op = model.OauthTypeGithub
	case model.OauthTypeGoogle:
		oa.Op = model.OauthTypeGoogle
	case model.OauthTypeLinuxdo:
		oa.Op = model.OauthTypeLinuxdo
	}
	if strings.TrimSpace(oa.Op) == "" && oauthType == model.OauthTypeOidc {
		oa.Op = model.OauthTypeOidc
	}
	if oauthType == model.OauthTypeGoogle && strings.TrimSpace(oa.Issuer) == "" {
		oa.Issuer = model.IssuerGoogle
	}
	if oa.PkceEnable == nil {
		oa.PkceEnable = new(bool)
		*oa.PkceEnable = false
	}
	if oa.PkceMethod == "" {
		oa.PkceMethod = model.PKCEMethodS256
	}
	return nil
}

// Create 创建
func (os *OauthService) Create(oauthInfo *model.Oauth) error {
	if err := os.FormatOauthInfo(oauthInfo); err != nil {
		return err
	}
	return os.ctx.DB.Create(oauthInfo).Error
}
func (os *OauthService) Delete(oauthInfo *model.Oauth) error {
	return os.ctx.DB.Delete(oauthInfo).Error
}

// Update 更新
func (os *OauthService) Update(oauthInfo *model.Oauth) error {
	if err := os.FormatOauthInfo(oauthInfo); err != nil {
		return err
	}
	return os.ctx.DB.Model(oauthInfo).Updates(oauthInfo).Error
}

// GetOauthProviders 获取所有的provider
func (os *OauthService) GetOauthProviders() []string {
	var res []string
	os.ctx.DB.Model(&model.Oauth{}).Pluck("op", &res)
	return res
}

// getGithubPrimaryEmail: Get the primary email of the user from Github
func (os *OauthService) getGithubPrimaryEmail(client *http.Client, githubUser *model.GithubUser) error {
	// the client is already set with the token
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch emails: %s", resp.Status)
	}

	// decode the response
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// find the primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			githubUser.Email = e.Email
			githubUser.VerifiedEmail = e.Verified
			return nil
		}
	}

	return fmt.Errorf("no primary verified email found")
}

// OauthCallbackResult holds the result of an OAuth callback for the controller to render.
type OauthCallbackResult struct {
	Success      bool
	Action       string // "bind" or "login"
	ErrorMsg     string // i18n message key
	SubError     string
	User         *model.User
	CacheKey     string
	RedirectURL  string // non-empty means redirect
	HTMLTemplate string // "oauth_success.html" or "oauth_fail.html"
	HTMLData     map[string]interface{}
}

func failResult(msg string, sub string) *OauthCallbackResult {
	return &OauthCallbackResult{
		HTMLTemplate: "oauth_fail.html",
		HTMLData:     map[string]interface{}{"message": msg, "sub_message": sub},
	}
}

func successResult(msg string) *OauthCallbackResult {
	return &OauthCallbackResult{
		Success:      true,
		HTMLTemplate: "oauth_success.html",
		HTMLData:     map[string]interface{}{"message": msg},
	}
}

// HandleCallback processes the OAuth callback logic (bind or login flow).
func (os *OauthService) HandleCallback(code, state string) *OauthCallbackResult {
	if state == "" {
		return failResult("ParamIsEmpty", "state")
	}
	cacheKey := state
	oauthCache := os.GetOauthCache(cacheKey)
	if oauthCache == nil {
		return failResult("OauthExpired", "")
	}

	oauthUser, err := os.Callback(code, oauthCache.Verifier, oauthCache.Op, oauthCache.Nonce)
	if err != nil {
		return failResult("OauthFailed", err.Error())
	}

	op := oauthCache.Op
	action := oauthCache.Action
	openid := oauthUser.OpenId

	if action == OauthActionTypeBind {
		utr := os.UserThirdInfo(op, openid)
		if utr.UserId > 0 {
			return failResult("OauthHasBindOtherUser", "")
		}
		user := os.ctx.Services.UserService.InfoById(oauthCache.UserId)
		if user == nil {
			return failResult("ItemNotFound", "")
		}
		if err := os.BindOauthUser(oauthCache.UserId, oauthUser, op); err != nil {
			return failResult("BindFail", "")
		}
		return successResult("BindSuccess")
	}

	if action == OauthActionTypeLogin {
		if oauthCache.UserId != 0 {
			return failResult("OauthHasBeenSuccess", "")
		}
		user := os.ctx.Services.InfoByOauthId(op, openid)
		if user == nil {
			oauthConfig := os.InfoByOp(op)
			if !*oauthConfig.AutoRegister {
				// Auto-register is off and this OAuth identity is not linked to
				// any user. The bind page lived in the old web admin, which is
				// removed here; until the front-end ships an inline bind flow,
				// fail clearly instead of leaving the poller to time out.
				return failResult("OauthFailed", "OAuth identity is not linked to a user")
			}
			user, err = os.ctx.Services.RegisterByOauth(oauthUser, op)
			if err != nil {
				return failResult(err.Error(), "")
			}
		}
		oauthCache.UserId = user.Id
		os.SetOauthCache(cacheKey, oauthCache, 0)
		// Login completes via the poll endpoint (which reads UserId from the
		// cache set above), so the callback window just renders a self-closing
		// success page for every client type instead of redirecting to a UI.
		return successResult("OauthSuccess")
	}

	return failResult("ParamsError", "")
}

// OidcAuthQueryResult holds the result of an OIDC auth query.
type OidcAuthQueryResult struct {
	User      *model.User
	Token     *model.UserToken
	ErrorMsg  string // empty on success
	AuthInPrg bool   // true when user hasn't completed auth yet
}

// HandleOidcAuthQuery processes the OIDC auth query polling request.
func (os *OauthService) HandleOidcAuthQuery(code, id, uuid, clientIP string) *OidcAuthQueryResult {
	v := os.GetOauthCache(code)
	if v == nil {
		return &OidcAuthQueryResult{ErrorMsg: "OauthExpired"}
	}
	if v.UserId == 0 {
		return &OidcAuthQueryResult{AuthInPrg: true}
	}
	u := os.ctx.Services.UserService.InfoById(v.UserId)
	if u == nil {
		return &OidcAuthQueryResult{ErrorMsg: "UserNotFound"}
	}
	os.DeleteOauthCache(code)
	ut := os.ctx.Services.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   v.DeviceType,
		DeviceId: v.Id,
		Uuid:     v.Uuid,
		Ip:       clientIP,
		Type:     model.LoginLogTypeOauth,
		Platform: v.DeviceOs,
	})
	if ut == nil {
		return &OidcAuthQueryResult{ErrorMsg: "LoginFailed"}
	}
	return &OidcAuthQueryResult{User: u, Token: ut}
}
