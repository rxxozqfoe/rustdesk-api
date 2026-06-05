package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/request/admin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
)

type Strategy struct {
	HD *deps.HandlerDeps
}

// Detail
// @Tags Strategy
// @Summary Strategy detail
// @Description Get strategy detail by ID
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Strategy}
// @Failure 500 {object} response.Response
// @Router /admin/strategy/detail/{id} [get]
// @Security token
func (ct *Strategy) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	s := ct.HD.Services.StrategyService.InfoById(uint(iid))
	if s.Id > 0 {
		response.Success(c, s)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Create
// @Tags Strategy
// @Summary Create strategy
// @Description Create a new strategy
// @Accept  json
// @Produce  json
// @Param body body admin.StrategyForm true "Strategy info"
// @Success 200 {object} response.Response{data=model.Strategy}
// @Failure 500 {object} response.Response
// @Router /admin/strategy/create [post]
// @Security token
func (ct *Strategy) Create(c *gin.Context) {
	f := &admin.StrategyForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	s := f.ToStrategy()
	err := ct.HD.Services.StrategyService.Create(s)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, s)
}

// List
// @Tags Strategy
// @Summary Strategy list
// @Description Get paginated strategy list
// @Accept  json
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Response{data=model.StrategyList}
// @Failure 500 {object} response.Response
// @Router /admin/strategy/list [get]
// @Security token
func (ct *Strategy) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := ct.HD.Services.StrategyService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Update
// @Tags Strategy
// @Summary Update strategy
// @Description Update an existing strategy
// @Accept  json
// @Produce  json
// @Param body body admin.StrategyForm true "Strategy info"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/strategy/update [post]
// @Security token
func (ct *Strategy) Update(c *gin.Context) {
	f := &admin.StrategyForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	s := f.ToStrategy()
	err := ct.HD.Services.StrategyService.Update(s)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete
// @Tags Strategy
// @Summary Delete strategy
// @Description Delete a strategy and all its associations
// @Accept  json
// @Produce  json
// @Param body body admin.StrategyForm true "Strategy info"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/strategy/delete [post]
// @Security token
func (ct *Strategy) Delete(c *gin.Context) {
	f := &admin.StrategyForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidVar(c, f.Id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	s := ct.HD.Services.StrategyService.InfoById(f.Id)
	if s.Id > 0 {
		err := ct.HD.Services.StrategyService.Delete(s)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Assign assigns or unassigns a strategy to peers, users, and device groups.
// @Tags Strategy
// @Summary Assign strategy
// @Description Assign strategy to peers/users/device groups, or unassign if strategy GUID is empty
// @Accept  json
// @Produce  json
// @Param body body admin.StrategyAssignForm true "Assignment info"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/strategy/assign [post]
// @Security token
func (ct *Strategy) Assign(c *gin.Context) {
	f := &admin.StrategyAssignForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	var strategyId uint
	if f.Strategy != "" {
		s := ct.HD.Services.InfoByGuid(f.Strategy)
		if s.Id == 0 {
			response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
			return
		}
		strategyId = s.Id
	}

	for _, peerId := range f.Peers {
		peer := ct.HD.Services.FindById(peerId)
		if peer == nil || peer.RowId == 0 {
			continue
		}
		if strategyId > 0 {
			if err := ct.HD.Services.AssignToPeer(strategyId, peer.RowId); err != nil {
				ct.HD.Logger.Warnf("AssignToPeer fail: strategy=%d peer=%d %v", strategyId, peer.RowId, err)
			}
		} else {
			if err := ct.HD.Services.UnassignPeer(peer.RowId); err != nil {
				ct.HD.Logger.Warnf("UnassignPeer fail: peer=%d %v", peer.RowId, err)
			}
		}
	}

	for _, userId := range f.Users {
		if strategyId > 0 {
			if err := ct.HD.Services.AssignToUser(strategyId, userId); err != nil {
				ct.HD.Logger.Warnf("AssignToUser fail: strategy=%d user=%d %v", strategyId, userId, err)
			}
		} else {
			if err := ct.HD.Services.UnassignUser(userId); err != nil {
				ct.HD.Logger.Warnf("UnassignUser fail: user=%d %v", userId, err)
			}
		}
	}

	for _, groupId := range f.Groups {
		if strategyId > 0 {
			if err := ct.HD.Services.AssignToDeviceGroup(strategyId, groupId); err != nil {
				ct.HD.Logger.Warnf("AssignToDeviceGroup fail: strategy=%d group=%d %v", strategyId, groupId, err)
			}
		} else {
			if err := ct.HD.Services.UnassignDeviceGroup(groupId); err != nil {
				ct.HD.Logger.Warnf("UnassignDeviceGroup fail: group=%d %v", groupId, err)
			}
		}
	}

	response.Success(c, nil)
}

// StrategyAssignment is a single assignment record for the admin response.
type StrategyAssignment struct {
	Type string      `json:"type"` // "peer", "user", "device_group"
	Id   interface{} `json:"id"`   // peer device ID (string) or user/group numeric ID
	Name string      `json:"name"` // display name
}

// Assignments returns all current assignments for a given strategy.
// @Tags Strategy
// @Summary Strategy assignments
// @Description Get all peer/user/device_group assignments for a strategy
// @Accept  json
// @Produce  json
// @Param id path int true "Strategy ID"
// @Success 200 {object} response.Response{data=[]StrategyAssignment}
// @Failure 500 {object} response.Response
// @Router /admin/strategy/assignments/{id} [get]
// @Security token
func (ct *Strategy) Assignments(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	s := ct.HD.Services.StrategyService.InfoById(uint(iid))
	if s.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}

	var assignments []StrategyAssignment

	// Peer assignments
	sps := ct.HD.Services.PeerAssignments(s.Id)
	for _, sp := range sps {
		peer := ct.HD.Services.PeerService.InfoByRowId(sp.PeerRowId)
		if peer.RowId > 0 {
			name := peer.Hostname
			if peer.Alias != "" {
				name = peer.Alias
			}
			assignments = append(assignments, StrategyAssignment{Type: "peer", Id: peer.Id, Name: name})
		}
	}

	// User assignments
	sus := ct.HD.Services.UserAssignments(s.Id)
	for _, su := range sus {
		user := ct.HD.Services.UserService.InfoById(su.UserId)
		if user.Id > 0 {
			name := user.Nickname
			if name == "" {
				name = user.Username
			}
			assignments = append(assignments, StrategyAssignment{Type: "user", Id: user.Id, Name: name})
		}
	}

	// Device group assignments
	sdgs := ct.HD.Services.DeviceGroupAssignments(s.Id)
	for _, sdg := range sdgs {
		dg := ct.HD.Services.DeviceGroupInfoById(sdg.DeviceGroupId)
		if dg.Id > 0 {
			assignments = append(assignments, StrategyAssignment{Type: "device_group", Id: dg.Id, Name: dg.Name})
		}
	}

	if assignments == nil {
		assignments = []StrategyAssignment{}
	}
	response.Success(c, assignments)
}
