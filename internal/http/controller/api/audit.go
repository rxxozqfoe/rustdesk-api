package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	request "github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
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
	// Resolve the controller-user attribution token (RustDesk 1.4.9+) into a
	// concrete user, if the rendezvous server recorded a snapshot for it.
	a.resolveController(ac)
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
				// Auth details / controller attribution may arrive with the on_open
				// update rather than the initial record. GORM Updates skips zero
				// values, so these only overwrite when actually provided.
				PrimaryAuth:        ac.PrimaryAuth,
				TwoFactor:          ac.TwoFactor,
				ConnAuditRef:       ac.ConnAuditRef,
				ControllerUserId:   ac.ControllerUserId,
				ControllerUsername: ac.ControllerUsername,
			}
			if err := a.HD.Services.UpdateAuditConn(up); err != nil {
				a.HD.Logger.Warnf("UpdateAuditConn fail: %v", err)
			}
		}
	}
	response.Success(c, "")
}

// resolveController fills ControllerUserId/ControllerUsername from the
// conn_audit_ref snapshot written by the rendezvous server, when present.
func (a *Audit) resolveController(ac *model.AuditConn) {
	if ac.ConnAuditRef == "" {
		return
	}
	snap := a.HD.Services.ConnAuditRefService.Lookup(ac.ConnAuditRef)
	if snap == nil {
		return
	}
	ac.ControllerUserId = snap.UserId
	ac.ControllerUsername = snap.Username
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
