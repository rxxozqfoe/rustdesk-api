package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	requstform "github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type Index struct {
	HD *deps.HandlerDeps
}

// Index 首页
// @Tags 首页
// @Summary 首页
// @Description 首页
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router / [get]
func (i *Index) Index(c *gin.Context) {
	response.Success(
		c,
		"Hello Gwen",
	)
}

// Heartbeat 心跳
// @Tags 首页
// @Summary 心跳
// @Description 心跳
// @Accept  json
// @Produce  json
// @Success 200 {object} nil
// @Failure 500 {object} response.Response
// @Router /heartbeat [post]
func (i *Index) Heartbeat(c *gin.Context) {
	info := &requstform.PeerInfoInHeartbeat{}
	err := c.ShouldBindJSON(info)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if info.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	peer := i.HD.Services.PeerService.FindById(info.Id)
	if peer == nil || peer.RowId == 0 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	//如果在40s以内则不更新
	if time.Now().Unix()-peer.LastOnlineTime >= 30 {
		upp := &model.Peer{RowId: peer.RowId, LastOnlineTime: time.Now().Unix(), LastOnlineIp: c.ClientIP()}
		i.HD.Services.PeerService.Update(upp)
	}

	resp := gin.H{}

	// Resolve strategy for this peer
	strategy := i.HD.Services.StrategyService.ResolveForPeer(peer)
	if strategy != nil {
		serverModifiedAt := time.Time(strategy.UpdatedAt).Unix()
		resp["modified_at"] = serverModifiedAt
		// Only include strategy payload when client's timestamp differs
		if info.ModifiedAt != serverModifiedAt {
			resp["strategy"] = gin.H{
				"config_options": i.HD.Services.StrategyService.ConfigOptionsMap(strategy),
				"extra":          i.HD.Services.StrategyService.ExtraMap(strategy),
			}
		}
	} else {
		resp["modified_at"] = 0
	}

	// Signal client to re-upload sysinfo when peer record is incomplete
	if peer.Version == "" || peer.Os == "" {
		resp["sysinfo"] = true
	}

	// Collect pending disconnect commands for this peer
	cmds := i.HD.Services.PeerCommandService.PendingByPeerId(peer.Id)
	if len(cmds) > 0 {
		var allConnIds []int
		var processedIds []uint
		hasDisconnect := false
		for _, cmd := range cmds {
			if cmd.Command == model.PeerCommandDisconnect {
				hasDisconnect = true
				var connIds []int
				if err := json.Unmarshal([]byte(cmd.Payload), &connIds); err == nil {
					allConnIds = append(allConnIds, connIds...)
				}
			}
			processedIds = append(processedIds, cmd.Id)
		}
		if hasDisconnect {
			if allConnIds == nil {
				allConnIds = []int{}
			}
			resp["disconnect"] = allConnIds
		}
		i.HD.Services.PeerCommandService.DeleteByIds(processedIds)
	}

	c.JSON(http.StatusOK, resp)
}

// Version 版本
// @Tags 首页
// @Summary 版本
// @Description 版本
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /version [get]
func (i *Index) Version(c *gin.Context) {
	//读取resources/version文件
	v := i.HD.Services.AppService.GetAppVersion()
	response.Success(
		c,
		v,
	)
}
