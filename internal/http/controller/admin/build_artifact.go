package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type BuildArtifact struct {
	HD *deps.HandlerDeps
}

// List
// @Tags BuildArtifact
// @Summary List build artifacts
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Response{data=model.BuildArtifactList}
// @Router /admin/build-artifact/list [get]
// @Security token
func (ct *BuildArtifact) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := ct.HD.Services.BuildArtifactService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Delete
// @Tags BuildArtifact
// @Summary Delete build artifact
// @Accept  json
// @Produce  json
// @Param body body object true "id"
// @Success 200 {object} response.Response
// @Router /admin/build-artifact/delete [post]
// @Security token
func (ct *BuildArtifact) Delete(c *gin.Context) {
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
	ba := ct.HD.Services.BuildArtifactService.InfoById(req.Id)
	if ba.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if err := ct.HD.Services.BuildArtifactService.Delete(ba); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Detail returns a single build artifact by ID.
func (ct *BuildArtifact) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	ba := ct.HD.Services.BuildArtifactService.InfoById(uint(iid))
	if ba.Id > 0 {
		response.Success(c, ba)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}
