package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/helper"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	adminReq "github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
)

type Oauth struct {
	HD *deps.HandlerDeps
}

// Info
func (o *Oauth) Info(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	v := o.HD.Services.OauthService.GetOauthCache(code)
	if v == nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	response.Success(c, v)
}

func (o *Oauth) ToBind(c *gin.Context) {
	f := &adminReq.BindOauthForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := helper.CurUser(c)

	utr := o.HD.Services.UserService.UserThirdInfo(u.Id, f.Op)
	if utr.Id > 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "OauthHasBindOtherUser"))
		return
	}

	err, state, verifier, nonce, url := o.HD.Services.OauthService.BeginAuth(f.Op)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
		return
	}

	o.HD.Services.OauthService.SetOauthCache(state, &service.OauthCacheItem{
		Action:   service.OauthActionTypeBind,
		Op:       f.Op,
		UserId:   u.Id,
		Verifier: verifier,
		Nonce:    nonce,
	}, 5*60)

	response.Success(c, gin.H{
		"code": state,
		"url":  url,
	})
}

// Confirm 确认授权登录
func (o *Oauth) Confirm(c *gin.Context) {
	j := &adminReq.OauthConfirmForm{}
	err := c.ShouldBindJSON(j)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if j.Code == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	v := o.HD.Services.OauthService.GetOauthCache(j.Code)
	if v == nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OauthExpired"))
		return
	}
	u := helper.CurUser(c)
	v.UserId = u.Id
	o.HD.Services.OauthService.SetOauthCache(j.Code, v, 0)
	response.Success(c, v)
}

func (o *Oauth) BindConfirm(c *gin.Context) {
	j := &adminReq.OauthConfirmForm{}
	err := c.ShouldBindJSON(j)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if j.Code == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	oauthService := o.HD.Services.OauthService
	oauthCache := oauthService.GetOauthCache(j.Code)
	if oauthCache == nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OauthExpired"))
		return
	}
	oauthUser := oauthCache.ToOauthUser()
	user := helper.CurUser(c)
	err = oauthService.BindOauthUser(user.Id, oauthUser, oauthCache.Op)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "BindFail"))
		return
	}

	oauthCache.UserId = user.Id
	oauthService.SetOauthCache(j.Code, oauthCache, 0)
	response.Success(c, oauthCache)
}

func (o *Oauth) Unbind(c *gin.Context) {
	f := &adminReq.UnBindOauthForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := helper.CurUser(c)
	utr := o.HD.Services.UserService.UserThirdInfo(u.Id, f.Op)
	if utr.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err = o.HD.Services.OauthService.UnBindOauthUser(u.Id, f.Op)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Detail Oauth
// @Tags Oauth
// @Summary Oauth详情
// @Description Oauth详情
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Oauth}
// @Failure 500 {object} response.Response
// @Router /admin/oauth/detail/{id} [get]
// @Security token
func (o *Oauth) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := o.HD.Services.OauthService.InfoById(uint(iid))
	if u.Id > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
	return
}

// Create 创建Oauth
// @Tags Oauth
// @Summary 创建Oauth
// @Description 创建Oauth
// @Accept  json
// @Produce  json
// @Param body body admin.OauthForm true "Oauth信息"
// @Success 200 {object} response.Response{data=model.Oauth}
// @Failure 500 {object} response.Response
// @Router /admin/oauth/create [post]
// @Security token
func (o *Oauth) Create(c *gin.Context) {
	f := &admin.OauthForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := o.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToOauth()
	if err := o.HD.Services.OauthService.FormatOauthInfo(u); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	ex := o.HD.Services.OauthService.InfoByOp(u.Op)
	if ex.Id > 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemExists"))
		return
	}
	if err := o.HD.Services.OauthService.Create(u); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// List 列表
// @Tags Oauth
// @Summary Oauth列表
// @Description Oauth列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Success 200 {object} response.Response{data=model.OauthList}
// @Failure 500 {object} response.Response
// @Router /admin/oauth/list [get]
// @Security token
func (o *Oauth) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := o.HD.Services.OauthService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Update 编辑
// @Tags Oauth
// @Summary Oauth编辑
// @Description Oauth编辑
// @Accept  json
// @Produce  json
// @Param body body admin.OauthForm true "Oauth信息"
// @Success 200 {object} response.Response{data=model.OauthList}
// @Failure 500 {object} response.Response
// @Router /admin/oauth/update [post]
// @Security token
func (o *Oauth) Update(c *gin.Context) {
	f := &admin.OauthForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := o.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToOauth()
	err := o.HD.Services.OauthService.Update(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete 删除
// @Tags Oauth
// @Summary Oauth删除
// @Description Oauth删除
// @Accept  json
// @Produce  json
// @Param body body admin.OauthForm true "Oauth信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/oauth/delete [post]
// @Security token
func (o *Oauth) Delete(c *gin.Context) {
	f := &admin.OauthForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := o.HD.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := o.HD.Services.OauthService.InfoById(f.Id)
	if u.Id > 0 {
		err := o.HD.Services.OauthService.Delete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}
