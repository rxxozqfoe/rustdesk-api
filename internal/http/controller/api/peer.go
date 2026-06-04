package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	requstform "github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type Peer struct {
	HD *deps.HandlerDeps
}

// SysInfo
// @Tags System
// @Summary 提交系统信息
// @Description 提交系统信息
// @Accept  json
// @Produce  json
// @Param body body requstform.PeerForm true "系统信息表单"
// @Success 200 {string} string "SYSINFO_UPDATED,ID_NOT_FOUND"
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo [post]
func (p *Peer) SysInfo(c *gin.Context) {
	f := &requstform.PeerForm{}
	err := c.ShouldBindBodyWith(f, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	fpe := f.ToPeer()
	pe := p.HD.Services.FindById(f.Id)
	if pe.RowId == 0 {
		pe = f.ToPeer()
		pe.UserId = p.HD.Services.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		err = p.HD.Services.PeerService.Create(pe)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	} else {
		if pe.UserId == 0 {
			pe.UserId = p.HD.Services.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		}
		fpe.RowId = pe.RowId
		fpe.UserId = pe.UserId
		err = p.HD.Services.PeerService.Update(fpe)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}
	// Handle preset-strategy-name: auto-assign strategy to this peer
	if f.PresetStrategyName != "" {
		strategy := p.HD.Services.InfoByName(f.PresetStrategyName)
		if strategy != nil && strategy.Id > 0 {
			if err := p.HD.Services.AssignToPeer(strategy.Id, pe.RowId); err != nil {
				p.HD.Logger.Warnf("AssignToPeer fail: strategy=%d peer=%d %v", strategy.Id, pe.RowId, err)
			}
		}
	}

	// Handle preset-device-group-name: auto-assign device group
	if f.PresetDeviceGroupName != "" && pe.GroupId == 0 {
		allGroups := p.HD.Services.DeviceGroupList(1, 999, nil)
		for _, g := range allGroups.DeviceGroups {
			if g.Name == f.PresetDeviceGroupName {
				pe.GroupId = g.Id
				if err := p.HD.Services.PeerService.Update(&model.Peer{RowId: pe.RowId, GroupId: g.Id}); err != nil {
					p.HD.Logger.Warnf("assign device group fail: peer=%d group=%d %v", pe.RowId, g.Id, err)
				}
				break
			}
		}
	}

	// Handle preset-username: auto-assign user
	if f.PresetUsername != "" && pe.UserId == 0 {
		u := p.HD.Services.InfoByUsername(f.PresetUsername)
		if u != nil && u.Id != 0 {
			if err := p.HD.Services.PeerService.Update(&model.Peer{RowId: pe.RowId, UserId: u.Id}); err != nil {
				p.HD.Logger.Warnf("assign peer user fail: peer=%d user=%d %v", pe.RowId, u.Id, err)
			}
		}
	}

	//SYSINFO_UPDATED 上传成功
	//ID_NOT_FOUND 下次心跳会上传
	//直接响应文本
	c.String(http.StatusOK, "SYSINFO_UPDATED")
}

// SysInfoVer
// @Tags System
// @Summary 获取系统版本信息
// @Description 获取系统版本信息
// @Accept  json
// @Produce  json
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo_ver [post]
func (p *Peer) SysInfoVer(c *gin.Context) {
	//读取resources/version文件
	v := p.HD.Services.GetAppVersion()
	// 加上启动时间，方便client上传信息
	v = fmt.Sprintf("%s\n%s", v, p.HD.Services.GetStartTime())
	c.String(http.StatusOK, v)
}
