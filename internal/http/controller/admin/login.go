package admin

import (
	"fmt"

	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/helper"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/request/admin"
	apiReq "github.com/rxxozqfoe/rustdesk-api/internal/http/request/api"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	adResp "github.com/rxxozqfoe/rustdesk-api/internal/http/response/admin"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"github.com/rxxozqfoe/rustdesk-api/internal/service"
)

type Login struct {
	HD *deps.HandlerDeps
}

// Login 登录
// @Tags 登录
// @Summary 登录
// @Description 登录
// @Accept  json
// @Produce  json
// @Param body body admin.Login true "登录信息"
// @Success 200 {object} response.Response{data=adResp.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/login [post]
// @Security token
func (ct *Login) Login(c *gin.Context) {
	if ct.HD.Config.App.DisablePwdLogin {
		response.Fail(c, 101, response.TranslateMsg(c, "PwdLoginDisabled"))
		return
	}

	// 检查登录限制
	loginLimiter := ct.HD.LoginLimiter
	clientIp := c.ClientIP()
	_, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)

	f := &admin.Login{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		loginLimiter.RecordFailedAttempt(clientIp)
		ct.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		ct.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Fail(c, 101, errList[0])
		return
	}

	// 检查是否需要验证码
	if needCaptcha {
		if f.CaptchaId == "" || f.Captcha == "" || !loginLimiter.VerifyCaptcha(f.CaptchaId, f.Captcha) {
			response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError"))
			return
		}
	}

	u := ct.HD.Services.InfoByUsernamePassword(f.Username, f.Password)

	if u.Id == 0 {
		ct.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "UsernameOrPasswordError", c.RemoteIP(), clientIp))
		loginLimiter.RecordFailedAttempt(clientIp)
		if _, needCaptcha = loginLimiter.CheckSecurityStatus(clientIp); needCaptcha {
			response.Fail(c, 110, response.TranslateMsg(c, "UsernameOrPasswordError"))
		} else {
			response.Fail(c, 101, response.TranslateMsg(c, "UsernameOrPasswordError"))
		}
		return
	}

	if !ct.HD.Services.CheckUserEnable(u) {
		if needCaptcha {
			response.Fail(c, 110, response.TranslateMsg(c, "UserDisabled"))
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "UserDisabled"))
		return
	}

	ut := ct.HD.Services.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   model.LoginLogClientWebAdmin,
		Uuid:     "", //must be empty
		Ip:       clientIp,
		Type:     model.LoginLogTypeAccount,
		Platform: f.Platform,
	})

	// 登录成功，清除登录限制
	loginLimiter.RemoveAttempts(clientIp)
	responseLoginSuccess(c, ct.HD.Services.UserService, u, ut.Token)
}
func (ct *Login) Captcha(c *gin.Context) {
	loginLimiter := ct.HD.LoginLimiter
	clientIp := c.ClientIP()
	banned, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Fail(c, 101, response.TranslateMsg(c, "LoginBanned"))
		return
	}
	if !needCaptcha {
		response.Fail(c, 101, response.TranslateMsg(c, "NoCaptchaRequired"))
		return
	}
	captcha, err := loginLimiter.RequireCaptcha()
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError")+err.Error())
		return
	}
	b64, err := loginLimiter.DrawCaptcha(captcha.Content)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError")+err.Error())
		return
	}
	response.Success(c, gin.H{
		"captcha": gin.H{
			"id":  captcha.Id,
			"b64": b64,
		},
	})
}

// Logout 登出
// @Tags 登录
// @Summary 登出
// @Description 登出
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/logout [post]
func (ct *Login) Logout(c *gin.Context) {
	u := helper.CurUser(c)
	token, ok := c.Get("token")
	if ok {
		if err := ct.HD.Services.Logout(u, token.(string)); err != nil {
			ct.HD.Logger.Warn(fmt.Sprintf("Logout Fail: %s %v", u.Username, err))
		}
	}
	response.Success(c, nil)
}

// LoginOptions
// @Tags 登录
// @Summary 登录选项
// @Description 登录选项
// @Accept  json
// @Produce  json
// @Success 200 {object} []string
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/login-options [post]
func (ct *Login) LoginOptions(c *gin.Context) {
	loginLimiter := ct.HD.LoginLimiter
	clientIp := c.ClientIP()
	banned, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Fail(c, 101, response.TranslateMsg(c, "LoginBanned"))
		return
	}
	ops := ct.HD.Services.GetOauthProviders()
	response.Success(c, gin.H{
		"ops":          ops,
		"register":     ct.HD.Config.App.Register,
		"need_captcha": needCaptcha,
		"disable_pwd":  ct.HD.Config.App.DisablePwdLogin,
		"auto_oidc":    ct.HD.Config.App.DisablePwdLogin && len(ops) == 1,
	})
}

// OidcAuth
// @Tags Oauth
// @Summary OidcAuth
// @Description OidcAuth
// @Accept  json
// @Produce  json
// @Router /admin/oidc/auth [post]
func (ct *Login) OidcAuth(c *gin.Context) {
	// o := &api.Oauth{}
	// o.OidcAuth(c)
	f := &apiReq.OidcAuthRequest{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	state, verifier, nonce, url, err := ct.HD.Services.BeginAuth(f.Op)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
		return
	}

	ct.HD.Services.SetOauthCache(state, &service.OauthCacheItem{
		Action:     service.OauthActionTypeLogin,
		Op:         f.Op,
		Id:         f.Id,
		DeviceType: "webadmin",
		// DeviceOs: ct.Platform(c),
		DeviceOs: f.DeviceInfo.Os,
		Uuid:     f.Uuid,
		Verifier: verifier,
		Nonce:    nonce,
	}, 5*60)

	response.Success(c, gin.H{
		"code": state,
		"url":  url,
	})
}

// OidcAuthQuery
// @Tags Oauth
// @Summary OidcAuthQuery
// @Description OidcAuthQuery
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response{data=adResp.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/oidc/auth-query [get]
func (ct *Login) OidcAuthQuery(c *gin.Context) {
	code := c.Query("code")
	id := c.Query("id")
	uuid := c.Query("uuid")
	result := ct.HD.Services.HandleOidcAuthQuery(code, id, uuid, c.ClientIP())
	if result.AuthInPrg {
		response.Fail(c, 101, response.TranslateMsg(c, "OauthInProgress"))
		return
	}
	if result.ErrorMsg != "" {
		response.Fail(c, 101, response.TranslateMsg(c, result.ErrorMsg))
		return
	}
	responseLoginSuccess(c, ct.HD.Services.UserService, result.User, result.Token.Token)
}

// responseLoginSuccess is a package-level helper shared by Login and User
// controllers. It takes an explicit *service.UserService so callers inject
// their own dependency rather than pulling from a global.
func responseLoginSuccess(c *gin.Context, users *service.UserService, u *model.User, token string) {
	lp := &adResp.LoginPayload{}
	lp.FromUser(u)
	lp.Token = token
	if users.IsAdmin(u) {
		lp.RouteNames = helper.AdminRouteNames
	} else {
		lp.RouteNames = helper.UserRouteNames
	}
	response.Success(c, lp)
}
