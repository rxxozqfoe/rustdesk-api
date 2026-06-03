package admin

import (
	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type Rustdesk struct {
	HD *deps.HandlerDeps
}

type RustdeskCmd struct {
	Cmd    string `json:"cmd"`
	Option string `json:"option"`
	Target string `json:"target"`
}

func (r *Rustdesk) CmdList(c *gin.Context) {
	q := &admin.PageQuery{}
	if err := c.ShouldBindQuery(q); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := r.HD.Services.ServerCmdService.List(q.Page, 9999)
	//在列表前添加系统命令
	list := make([]*model.ServerCmd, 0)
	list = append(list, model.SysIdServerCmds...)
	list = append(list, model.SysRelayServerCmds...)
	list = append(list, res.ServerCmds...)
	res.ServerCmds = list
	response.Success(c, res)
}

func (r *Rustdesk) CmdDelete(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}

	ex := r.HD.Services.ServerCmdService.Info(f.Id)
	if ex.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}

	err := r.HD.Services.ServerCmdService.Delete(ex)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}
func (r *Rustdesk) CmdCreate(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := r.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	err := r.HD.Services.ServerCmdService.Create(f)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Rustdesk) CmdUpdate(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := r.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	ex := r.HD.Services.ServerCmdService.Info(f.Id)
	if ex.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := r.HD.Services.ServerCmdService.Update(f)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Rustdesk) SendCmd(c *gin.Context) {
	rc := &RustdeskCmd{}
	if err := c.ShouldBindJSON(rc); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if rc.Cmd == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if rc.Target == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if rc.Target != model.ServerCmdTargetIdServer && rc.Target != model.ServerCmdTargetRelayServer {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}

	port := 0
	switch rc.Target {
	case model.ServerCmdTargetIdServer:
		port = r.HD.Config.Admin.IdServerPort - 1
	case model.ServerCmdTargetRelayServer:
		port = r.HD.Config.Admin.RelayServerPort
	}

	res, err := r.HD.Services.SendCmd(port, rc.Cmd, rc.Option)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, res)
}
