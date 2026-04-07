package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
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
