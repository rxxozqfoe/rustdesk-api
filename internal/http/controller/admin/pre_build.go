package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/request/admin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"gorm.io/gorm"
)

type PreBuild struct {
	HD *deps.HandlerDeps
}

// Versions returns available versions from registered workers.
// @Tags PreBuild
// @Summary List available versions
// @Produce  json
// @Success 200 {object} response.Response{data=[]string}
// @Router /admin/pre-build/versions [get]
// @Security token
func (ct *PreBuild) Versions(c *gin.Context) {
	versions := ct.HD.Services.GetAllVersions()
	response.Success(c, versions)
}

// Trigger starts a new build job (creates a pending record for worker to pick up).
// @Tags PreBuild
// @Summary Trigger a build
// @Accept  json
// @Produce  json
// @Param body body admin.PreBuildTriggerForm true "Build parameters"
// @Success 200 {object} response.Response{data=model.PreBuild}
// @Router /admin/pre-build/trigger [post]
// @Security token
func (ct *PreBuild) Trigger(c *gin.Context) {
	f := &admin.PreBuildTriggerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	job, err := ct.HD.Services.Trigger(f.Version, f.Platform, f.Arch)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, job)
}

// List returns paginated build jobs.
// @Tags PreBuild
// @Summary List build jobs
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Param status query string false "Filter by status"
// @Success 200 {object} response.Response{data=model.PreBuildList}
// @Router /admin/pre-build/list [get]
// @Security token
func (ct *PreBuild) List(c *gin.Context) {
	query := &admin.PreBuildListQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	var where func(tx *gorm.DB)
	if query.Status != "" {
		where = func(tx *gorm.DB) {
			tx.Where("status = ?", query.Status)
		}
	}
	res := ct.HD.Services.PreBuildService.List(query.Page, query.PageSize, where)
	response.Success(c, res)
}

// Detail returns a single build job.
// @Tags PreBuild
// @Summary Build job detail
// @Produce  json
// @Param id path int true "Build Job ID"
// @Success 200 {object} response.Response{data=model.PreBuild}
// @Router /admin/pre-build/detail/{id} [get]
// @Security token
func (ct *PreBuild) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	job := ct.HD.Services.PreBuildService.InfoById(uint(iid))
	if job.Id > 0 {
		response.Success(c, job)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Log returns build log content.
// @Tags PreBuild
// @Summary Get build log
// @Produce  json
// @Param id path int true "Build Job ID"
// @Param offset query int false "Byte offset to read from"
// @Success 200 {object} response.Response
// @Router /admin/pre-build/log/{id} [get]
// @Security token
func (ct *PreBuild) Log(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	content, newOffset, err := ct.HD.Services.GetLog(uint(iid), offset)
	if err != nil {
		response.Fail(c, 101, "failed to read log: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"log":    content,
		"offset": newOffset,
	})
}

// Cancel cancels a pending or running build job.
// @Tags PreBuild
// @Summary Cancel build job
// @Produce  json
// @Param id path int true "Build Job ID"
// @Success 200 {object} response.Response
// @Router /admin/pre-build/cancel/{id} [post]
// @Security token
func (ct *PreBuild) Cancel(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	if err := ct.HD.Services.Cancel(uint(iid)); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete removes a build job record.
// @Tags PreBuild
// @Summary Delete build job
// @Accept  json
// @Produce  json
// @Param body body object true "id"
// @Success 200 {object} response.Response
// @Router /admin/pre-build/delete [post]
// @Security token
func (ct *PreBuild) Delete(c *gin.Context) {
	type deleteReq struct {
		Id uint `json:"id"`
	}
	req := &deleteReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if req.Id == 0 {
		response.Fail(c, 101, "id is required")
		return
	}
	job := ct.HD.Services.PreBuildService.InfoById(req.Id)
	if job.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if err := ct.HD.Services.PreBuildService.Delete(job); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}
