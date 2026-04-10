package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"gorm.io/gorm"
)

type BuildJob struct {
	HD *deps.HandlerDeps
}

// Versions returns available git tags from the rustdesk source tree.
// @Tags BuildJob
// @Summary List available versions
// @Description List git tags (semver) from rustdesk source for building
// @Produce  json
// @Success 200 {object} response.Response{data=[]string}
// @Failure 500 {object} response.Response
// @Router /admin/build-job/versions [get]
// @Security token
func (ct *BuildJob) Versions(c *gin.Context) {
	versions, err := ct.HD.Services.BuildJobService.ListVersions()
	if err != nil {
		response.Fail(c, 101, "failed to list versions: "+err.Error())
		return
	}
	response.Success(c, versions)
}

// Trigger starts a new build job.
// @Tags BuildJob
// @Summary Trigger a build
// @Description Start building a rustdesk client for the specified version and platform
// @Accept  json
// @Produce  json
// @Param body body admin.BuildJobTriggerForm true "Build parameters"
// @Success 200 {object} response.Response{data=model.BuildJob}
// @Failure 500 {object} response.Response
// @Router /admin/build-job/trigger [post]
// @Security token
func (ct *BuildJob) Trigger(c *gin.Context) {
	f := &admin.BuildJobTriggerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	job, err := ct.HD.Services.BuildJobService.Trigger(f.Version, f.Platform, f.Arch, f.Format)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, job)
}

// List returns paginated build jobs.
// @Tags BuildJob
// @Summary List build jobs
// @Description Get paginated list of build jobs, optionally filtered by status
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Param status query string false "Filter by status (pending, building, completed, failed)"
// @Success 200 {object} response.Response{data=model.BuildJobList}
// @Failure 500 {object} response.Response
// @Router /admin/build-job/list [get]
// @Security token
func (ct *BuildJob) List(c *gin.Context) {
	query := &admin.BuildJobListQuery{}
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
	res := ct.HD.Services.BuildJobService.List(query.Page, query.PageSize, where)
	response.Success(c, res)
}

// Detail returns a single build job.
// @Tags BuildJob
// @Summary Build job detail
// @Description Get build job details by ID
// @Produce  json
// @Param id path int true "Build Job ID"
// @Success 200 {object} response.Response{data=model.BuildJob}
// @Failure 500 {object} response.Response
// @Router /admin/build-job/detail/{id} [get]
// @Security token
func (ct *BuildJob) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	job := ct.HD.Services.BuildJobService.InfoById(uint(iid))
	if job.Id > 0 {
		response.Success(c, job)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Log returns build log content from a given offset.
// @Tags BuildJob
// @Summary Get build log
// @Description Read build log incrementally (for live tailing)
// @Produce  json
// @Param id path int true "Build Job ID"
// @Param offset query int false "Byte offset to read from"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/build-job/log/{id} [get]
// @Security token
func (ct *BuildJob) Log(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	content, newOffset, err := ct.HD.Services.BuildJobService.GetLog(uint(iid), offset)
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
// @Tags BuildJob
// @Summary Cancel build job
// @Description Cancel a pending or running build
// @Produce  json
// @Param id path int true "Build Job ID"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/build-job/cancel/{id} [post]
// @Security token
func (ct *BuildJob) Cancel(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	if err := ct.HD.Services.BuildJobService.Cancel(uint(iid)); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete removes a build job record and its log file.
// @Tags BuildJob
// @Summary Delete build job
// @Description Delete a build job record
// @Accept  json
// @Produce  json
// @Param body body object true "id"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/build-job/delete [post]
// @Security token
func (ct *BuildJob) Delete(c *gin.Context) {
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
	job := ct.HD.Services.BuildJobService.InfoById(req.Id)
	if job.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if err := ct.HD.Services.BuildJobService.Delete(job); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}
