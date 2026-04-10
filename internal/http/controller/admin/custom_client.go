package admin

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type CustomClient struct {
	HD *deps.HandlerDeps
}

// Detail
// @Tags CustomClient
// @Summary Custom client detail
// @Description Get custom client config by ID
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.CustomClient}
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/detail/{id} [get]
// @Security token
func (ct *CustomClient) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	cc := ct.HD.Services.CustomClientService.InfoById(uint(iid))
	if cc.Id > 0 {
		response.Success(c, cc)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Create
// @Tags CustomClient
// @Summary Create custom client config
// @Description Create a new custom client configuration
// @Accept  json
// @Produce  json
// @Param body body admin.CustomClientForm true "Custom client info"
// @Success 200 {object} response.Response{data=model.CustomClient}
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/create [post]
// @Security token
func (ct *CustomClient) Create(c *gin.Context) {
	f := &admin.CustomClientForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	cc := f.ToCustomClient()
	err := ct.HD.Services.CustomClientService.Create(cc)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, cc)
}

// List
// @Tags CustomClient
// @Summary Custom client list
// @Description Get paginated list of custom client configs
// @Accept  json
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Response{data=model.CustomClientList}
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/list [get]
// @Security token
func (ct *CustomClient) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := ct.HD.Services.CustomClientService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Update
// @Tags CustomClient
// @Summary Update custom client config
// @Description Update an existing custom client configuration
// @Accept  json
// @Produce  json
// @Param body body admin.CustomClientForm true "Custom client info"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/update [post]
// @Security token
func (ct *CustomClient) Update(c *gin.Context) {
	f := &admin.CustomClientForm{}
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
	cc := f.ToCustomClient()
	err := ct.HD.Services.CustomClientService.Update(cc)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete
// @Tags CustomClient
// @Summary Delete custom client config
// @Description Delete a custom client configuration
// @Accept  json
// @Produce  json
// @Param body body admin.CustomClientForm true "Custom client info"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/delete [post]
// @Security token
func (ct *CustomClient) Delete(c *gin.Context) {
	f := &admin.CustomClientForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := ct.HD.Validator.ValidVar(c, f.Id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	cc := ct.HD.Services.CustomClientService.InfoById(f.Id)
	if cc.Id > 0 {
		err := ct.HD.Services.CustomClientService.Delete(cc)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Preview returns the signed custom.txt content (base64 string) without packaging.
// @Tags CustomClient
// @Summary Preview custom.txt content
// @Description Generate and return the signed custom.txt content for testing
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=string}
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/preview/{id} [get]
// @Security token
func (ct *CustomClient) Preview(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	cc := ct.HD.Services.CustomClientService.InfoById(uint(iid))
	if cc.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	txt, err := ct.HD.Services.CustomClientService.GenerateCustomTxt(cc)
	if err != nil {
		response.Fail(c, 101, fmt.Sprintf("Failed to generate custom.txt: %v", err))
		return
	}
	response.Success(c, gin.H{
		"custom_txt": txt,
	})
}
