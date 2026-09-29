package admin

import (
	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/request/admin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
)

type WorkerStatus struct {
	HD *deps.HandlerDeps
}

func (ct *WorkerStatus) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := ct.HD.Services.WorkerRegistryService.List(query.Page, query.PageSize)
	response.Success(c, res)
}
