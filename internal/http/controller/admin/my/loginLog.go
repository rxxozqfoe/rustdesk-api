package my

import (
	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/helper"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type LoginLog struct {
	HD *deps.HandlerDeps
}

// List 列表
// @Tags 我的登录日志
// @Summary 登录日志列表
// @Description 登录日志列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Param user_id query int false "用户ID"
// @Success 200 {object} response.Response{data=model.LoginLogList}
// @Failure 500 {object} response.Response
// @Router /admin/my/login_log/list [get]
// @Security token
func (ct *LoginLog) List(c *gin.Context) {
	query := &admin.LoginLogQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := helper.CurUser(c)
	res := ct.HD.Services.LoginLogService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ? and is_deleted = ?", u.Id, model.IsDeletedNo)
		tx.Order("id desc")
	})
	response.Success(c, res)
}

// Delete 删除
// @Tags 我的登录日志
// @Summary 登录日志删除
// @Description 登录日志删除
// @Accept  json
// @Produce  json
// @Param body body model.LoginLog true "登录日志信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/my/login_log/delete [post]
// @Security token
func (ct *LoginLog) Delete(c *gin.Context) {
	f := &model.LoginLog{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := ct.HD.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	l := ct.HD.Services.LoginLogService.InfoById(f.Id)
	if l.Id == 0 || l.IsDeleted == model.IsDeletedYes {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	u := helper.CurUser(c)
	if l.UserId != u.Id {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := ct.HD.Services.SoftDelete(l)
	if err == nil {
		response.Success(c, nil)
		return
	}
	response.Fail(c, 101, err.Error())
}

// BatchDelete 删除
// @Tags 我的登录日志
// @Summary 登录日志批量删除
// @Description 登录日志批量删除
// @Accept  json
// @Produce  json
// @Param body body admin.LoginLogIds true "登录日志"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/my/login_log/batchDelete [post]
// @Security token
func (ct *LoginLog) BatchDelete(c *gin.Context) {
	f := &admin.LoginLogIds{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.Ids) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	u := helper.CurUser(c)
	err := ct.HD.Services.BatchSoftDelete(u.Id, f.Ids)
	if err == nil {
		response.Success(c, nil)
		return
	}
	response.Fail(c, 101, err.Error())
}
