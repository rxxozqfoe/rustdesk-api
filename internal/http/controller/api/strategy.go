package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	requstform "github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type StrategyController struct {
	HD *deps.HandlerDeps
}

// List returns all strategies as a JSON array (matching strategies.py contract).
// @Tags Strategy
// @Summary List strategies
// @Description List all strategies
// @Accept  json
// @Produce  json
// @Success 200 {array} model.Strategy
// @Router /strategies [get]
// @Security BearerAuth
func (sc *StrategyController) List(c *gin.Context) {
	strategies := sc.HD.Services.ListAll()
	c.JSON(http.StatusOK, strategies)
}

// Detail returns a single strategy by GUID.
// @Tags Strategy
// @Summary Get strategy by GUID
// @Description Get strategy detail by GUID
// @Accept  json
// @Produce  json
// @Param guid path string true "Strategy GUID"
// @Success 200 {object} model.Strategy
// @Failure 500 {object} response.ErrorResponse
// @Router /strategies/{guid} [get]
// @Security BearerAuth
func (sc *StrategyController) Detail(c *gin.Context) {
	guid := c.Param("guid")
	s := sc.HD.Services.InfoByGuid(guid)
	if s.Id > 0 {
		c.JSON(http.StatusOK, s)
		return
	}
	response.Error(c, "Strategy not found")
}

// UpdateStatus enables or disables a strategy.
// @Tags Strategy
// @Summary Enable/disable strategy
// @Description Update strategy enabled status
// @Accept  json
// @Produce  json
// @Param guid path string true "Strategy GUID"
// @Param body body bool true "Enabled status"
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /strategies/{guid}/status [put]
// @Security BearerAuth
func (sc *StrategyController) UpdateStatus(c *gin.Context) {
	guid := c.Param("guid")
	s := sc.HD.Services.InfoByGuid(guid)
	if s.Id == 0 {
		response.Error(c, "Strategy not found")
		return
	}

	var enabled bool
	if err := c.ShouldBindJSON(&enabled); err != nil {
		response.Error(c, "Invalid status value")
		return
	}

	if err := sc.HD.Services.SetEnabled(s.Id, enabled); err != nil {
		response.Error(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// Assign assigns or unassigns a strategy to peers, users, and device groups.
// @Tags Strategy
// @Summary Assign strategy
// @Description Assign strategy to peers/users/device groups, or unassign if strategy is empty
// @Accept  json
// @Produce  json
// @Param body body admin.StrategyAssignForm true "Assignment info"
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /strategies/assign [post]
// @Security BearerAuth
func (sc *StrategyController) Assign(c *gin.Context) {
	f := &requstform.StrategyAssignForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, "Invalid request: "+err.Error())
		return
	}

	var strategyId uint
	if f.Strategy != "" {
		s := sc.HD.Services.InfoByGuid(f.Strategy)
		if s.Id == 0 {
			response.Error(c, "Strategy not found")
			return
		}
		strategyId = s.Id
	}

	// Assign/unassign peers
	for _, peerId := range f.Peers {
		peer := sc.HD.Services.FindById(peerId)
		if peer == nil || peer.RowId == 0 {
			continue
		}
		if strategyId > 0 {
			if err := sc.HD.Services.AssignToPeer(strategyId, peer.RowId); err != nil {
				sc.HD.Logger.Warnf("AssignToPeer fail: strategy=%d peer=%d %v", strategyId, peer.RowId, err)
			}
		} else {
			if err := sc.HD.Services.UnassignPeer(peer.RowId); err != nil {
				sc.HD.Logger.Warnf("UnassignPeer fail: peer=%d %v", peer.RowId, err)
			}
		}
	}

	// Assign/unassign users
	for _, userId := range f.Users {
		if strategyId > 0 {
			if err := sc.HD.Services.AssignToUser(strategyId, userId); err != nil {
				sc.HD.Logger.Warnf("AssignToUser fail: strategy=%d user=%d %v", strategyId, userId, err)
			}
		} else {
			if err := sc.HD.Services.UnassignUser(userId); err != nil {
				sc.HD.Logger.Warnf("UnassignUser fail: user=%d %v", userId, err)
			}
		}
	}

	// Assign/unassign device groups
	for _, groupId := range f.Groups {
		if strategyId > 0 {
			if err := sc.HD.Services.AssignToDeviceGroup(strategyId, groupId); err != nil {
				sc.HD.Logger.Warnf("AssignToDeviceGroup fail: strategy=%d group=%d %v", strategyId, groupId, err)
			}
		} else {
			if err := sc.HD.Services.UnassignDeviceGroup(groupId); err != nil {
				sc.HD.Logger.Warnf("UnassignDeviceGroup fail: group=%d %v", groupId, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{})
}
