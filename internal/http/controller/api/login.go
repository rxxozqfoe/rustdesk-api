package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/helper"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/internal/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
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
// @Param body body api.LoginForm true "登录表单"
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /login [post]
func (l *Login) Login(c *gin.Context) {
	if l.HD.Config.App.DisablePwdLogin {
		response.Error(c, response.TranslateMsg(c, "PwdLoginDisabled"))
		return
	}

	// 检查登录限制
	loginLimiter := l.HD.LoginLimiter
	clientIp := c.ClientIP()

	f := &api.LoginForm{}
	err := c.ShouldBindJSON(f)
	//fmt.Println(f)
	if err != nil {
		loginLimiter.RecordFailedAttempt(clientIp)
		l.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := l.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		l.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, errList[0])
		return
	}

	u := l.HD.Services.InfoByUsernamePassword(f.Username, f.Password)

	if u.Id == 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		l.HD.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "UsernameOrPasswordError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, response.TranslateMsg(c, "UsernameOrPasswordError"))
		return
	}

	if !l.HD.Services.CheckUserEnable(u) {
		response.Error(c, response.TranslateMsg(c, "UserDisabled"))
		return
	}

	//根据refer判断是webclient还是app
	ref := c.GetHeader("referer")
	if ref != "" {
		f.DeviceInfo.Type = model.LoginLogClientWeb
	}

	ut := l.HD.Services.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   f.DeviceInfo.Type,
		DeviceId: f.Id,
		Uuid:     f.Uuid,
		Ip:       c.ClientIP(),
		Type:     model.LoginLogTypeAccount,
		Platform: f.DeviceInfo.Os,
	})

	c.JSON(http.StatusOK, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}

// LoginOptions
// @Tags 登录
// @Summary 登录选项
// @Description 登录选项
// @Accept  json
// @Produce  json
// @Success 200 {object} []string
// @Failure 500 {object} response.ErrorResponse
// @Router /login-options [get]
func (l *Login) LoginOptions(c *gin.Context) {
	ops := l.HD.Services.GetOauthProviders()
	if l.HD.Config.App.WebSso {
		ops = append(ops, model.OauthTypeWebauth)
	}
	var oidcItems []map[string]string
	for _, v := range ops {
		oidcItems = append(oidcItems, map[string]string{"name": v})
	}
	common, err := json.Marshal(oidcItems)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "SystemError")+err.Error())
		return
	}
	var res []string
	res = append(res, "common-oidc/"+string(common))
	for _, v := range ops {
		res = append(res, "oidc/"+v)
	}
	c.JSON(http.StatusOK, res)
}

// Logout
// @Tags 登录
// @Summary 登出
// @Description 登出
// @Accept  json
// @Produce  json
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /logout [post]
func (l *Login) Logout(c *gin.Context) {
	u := helper.CurUser(c)
	token, ok := c.Get("token")
	if ok {
		if err := l.HD.Services.Logout(u, token.(string)); err != nil {
			l.HD.Logger.Warn(fmt.Sprintf("Logout Fail: %s %v", u.Username, err))
		}
	}
	c.JSON(http.StatusOK, nil)

}
