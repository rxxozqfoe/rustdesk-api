package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	request "github.com/rxxozqfoe/rustdesk-api/internal/http/request/api"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
)

type Audit struct {
	HD *deps.HandlerDeps
}

// AuditConn
// @Tags 审计
// @Summary 审计连接
// @Description 审计连接
// @Accept  json
// @Produce  json
// @Param body body request.AuditConnForm true "审计连接"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/conn [post]
func (a *Audit) AuditConn(c *gin.Context) {
	af := &request.AuditConnForm{}
	err := c.ShouldBindBodyWith(af, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	/*ttt := &gin.H{}
	c.ShouldBindBodyWith(ttt, binding.JSON)
	fmt.Println(ttt)*/
	ac := af.ToAuditConn()
	switch af.Action {
	case model.AuditActionNew:
		if err := a.HD.Services.CreateAuditConn(ac); err != nil {
			a.HD.Logger.Warnf("CreateAuditConn fail: %v", err)
		}
	case model.AuditActionClose:
		ex := a.HD.Services.InfoByPeerIdAndConnId(af.Id, af.ConnId)
		if ex.Id != 0 {
			ex.CloseTime = time.Now().Unix()
			if err := a.HD.Services.UpdateAuditConn(ex); err != nil {
				a.HD.Logger.Warnf("UpdateAuditConn fail: %v", err)
			}
		}
	case "":
		ex := a.HD.Services.InfoByPeerIdAndConnId(af.Id, af.ConnId)
		if ex.Id != 0 {
			up := &model.AuditConn{
				IdModel:   model.IdModel{Id: ex.Id},
				FromPeer:  ac.FromPeer,
				FromName:  ac.FromName,
				SessionId: ac.SessionId,
				Type:      ac.Type,
			}
			if err := a.HD.Services.UpdateAuditConn(up); err != nil {
				a.HD.Logger.Warnf("UpdateAuditConn fail: %v", err)
			}
		}
	}
	response.Success(c, "")
}

// AuditFile
// @Tags 审计
// @Summary 审计文件
// @Description 审计文件
// @Accept  json
// @Produce  json
// @Param body body request.AuditFileForm true "审计文件"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/file [post]
func (a *Audit) AuditFile(c *gin.Context) {
	aff := &request.AuditFileForm{}
	err := c.ShouldBindBodyWith(aff, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	//ttt := &gin.H{}
	//c.ShouldBindBodyWith(ttt, binding.JSON)
	//fmt.Println(ttt)
	af := aff.ToAuditFile()
	if err := a.HD.Services.CreateAuditFile(af); err != nil {
		a.HD.Logger.Warnf("CreateAuditFile fail: %v", err)
	}
	response.Success(c, "")
}
