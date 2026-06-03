package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/internal/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Oauth struct {
	HD *deps.HandlerDeps
}

// OidcAuth
// @Tags Oauth
// @Summary OidcAuth
// @Description OidcAuth
// @Accept  json
// @Produce  json
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /oidc/auth [post]
func (o *Oauth) OidcAuth(c *gin.Context) {
	f := &api.OidcAuthRequest{}
	err := c.ShouldBindJSON(&f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	oauthService := o.HD.Services.OauthService

	state, verifier, nonce, url, err := oauthService.BeginAuth(f.Op)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	o.HD.Services.SetOauthCache(state, &service.OauthCacheItem{
		Action:     service.OauthActionTypeLogin,
		Id:         f.Id,
		Op:         f.Op,
		Uuid:       f.Uuid,
		DeviceName: f.DeviceInfo.Name,
		DeviceOs:   f.DeviceInfo.Os,
		DeviceType: f.DeviceInfo.Type,
		Verifier:   verifier,
		Nonce:      nonce,
	}, 5*60)
	//fmt.Println("code url", code, url)
	c.JSON(http.StatusOK, gin.H{
		"code": state,
		"url":  url,
	})
}

func (o *Oauth) OidcAuthQueryPre(c *gin.Context) (*model.User, *model.UserToken) {
	q := &api.OidcAuthQuery{}
	if err := c.ShouldBindQuery(q); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+": "+err.Error())
		return nil, nil
	}
	result := o.HD.Services.HandleOidcAuthQuery(q.Code, q.Id, q.Uuid, c.ClientIP())
	if result.AuthInPrg {
		c.JSON(http.StatusOK, gin.H{"message": "Authorization in progress, please login and bind", "error": "No authed oidc is found"})
		return nil, nil
	}
	if result.ErrorMsg != "" {
		response.Error(c, response.TranslateMsg(c, result.ErrorMsg))
		return nil, nil
	}
	return result.User, result.Token
}

// OidcAuthQuery
// @Tags Oauth
// @Summary OidcAuthQuery
// @Description OidcAuthQuery
// @Accept  json
// @Produce  json
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /oidc/auth-query [get]
func (o *Oauth) OidcAuthQuery(c *gin.Context) {
	u, ut := o.OidcAuthQueryPre(c)
	if u == nil || ut == nil {
		return
	}
	c.JSON(http.StatusOK, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}

// OauthCallback 回调
// @Tags Oauth
// @Summary OauthCallback
// @Description OauthCallback
// @Accept  json
// @Produce  json
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /oidc/callback [get]
func (o *Oauth) OauthCallback(c *gin.Context) {
	result := o.HD.Services.HandleCallback(c.Query("code"), c.Query("state"))
	if result.RedirectURL != "" {
		c.Redirect(http.StatusFound, result.RedirectURL)
		return
	}
	c.HTML(http.StatusOK, result.HTMLTemplate, result.HTMLData)
}

type MessageParams struct {
	Lang  string `json:"lang" form:"lang"`
	Title string `json:"title" form:"title"`
	Msg   string `json:"msg" form:"msg"`
}

func (o *Oauth) Message(c *gin.Context) {
	mp := &MessageParams{}
	if err := c.ShouldBindQuery(mp); err != nil {
		return
	}
	localizer := o.HD.Localizer(mp.Lang)
	res := ""
	if mp.Title != "" {
		title, err := localizer.LocalizeMessage(&i18n.Message{
			ID: mp.Title,
		})
		if err == nil {
			res = utils.StringConcat(";title='", title, "';")
		}

	}
	if mp.Msg != "" {
		msg, err := localizer.LocalizeMessage(&i18n.Message{
			ID: mp.Msg,
		})
		if err == nil {
			res = utils.StringConcat(res, "msg = '", msg, "';")
		}
	}

	//返回js内容
	c.Header("Content-Type", "application/javascript")
	c.String(http.StatusOK, res)
}
