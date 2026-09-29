package my

import (
	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/helper"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/request/admin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"gorm.io/gorm"
)

type ShareRecord struct {
	HD *deps.HandlerDeps
}

// List 分享记录列表
// @Tags 我的分享记录
// @Summary 分享记录列表
// @Description 分享记录列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/my/share_record/list [get]
// @Security token
func (sr *ShareRecord) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := helper.CurUser(c)
	res := sr.HD.Services.ShareRecordService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ?", u.Id)
	})
	response.Success(c, res)
}

// Delete 分享记录删除
// @Tags 我的分享记录
// @Summary 分享记录删除
// @Description 分享记录删除
// @Accept  json
// @Produce  json
// @Param body body admin.ShareRecordForm true "分享记录信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/my/share_record/delete [post]
// @Security token
func (sr *ShareRecord) Delete(c *gin.Context) {
	f := &admin.ShareRecordForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := sr.HD.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := helper.CurUser(c)
	i := sr.HD.Services.ShareRecordService.InfoById(f.Id)
	if i.UserId != u.Id {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if i.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := sr.HD.Services.ShareRecordService.Delete(i)
	if err == nil {
		response.Success(c, nil)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
}

// BatchDelete 批量删除我的分享记录
// @Tags 我的
// @Summary 批量删除我的分享记录
// @Description 批量删除我的分享记录
// @Accept  json
// @Produce  json
// @Param body body admin.PeerShareRecordBatchDeleteForm true "id"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/my/share_record/batchDelete [post]
// @Security token
func (sr *ShareRecord) BatchDelete(c *gin.Context) {
	f := &admin.PeerShareRecordBatchDeleteForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.Ids) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	u := helper.CurUser(c)
	l := int64(len(f.Ids))
	res := sr.HD.Services.ShareRecordService.List(1, uint(l), func(tx *gorm.DB) {
		tx.Where("user_id = ?", u.Id)
		tx.Where("id in ?", f.Ids)
	})
	if res.Total != l {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := sr.HD.Services.ShareRecordService.BatchDelete(f.Ids)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}
