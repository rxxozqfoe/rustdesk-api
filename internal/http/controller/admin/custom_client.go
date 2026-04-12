package admin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type CustomClient struct {
	HD *deps.HandlerDeps
}

// Detail
// @Tags CustomClient
// @Summary Custom client detail
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.CustomClient}
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

// Create creates a custom client and immediately starts bundling.
// @Tags CustomClient
// @Summary Create and bundle custom client
// @Accept  json
// @Produce  json
// @Param body body admin.CustomClientForm true "Custom client info"
// @Success 200 {object} response.Response{data=model.CustomClient}
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
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, cc)
}

// List
// @Tags CustomClient
// @Summary List custom clients
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Response{data=model.CustomClientList}
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

// Delete
// @Tags CustomClient
// @Summary Delete custom client
// @Accept  json
// @Produce  json
// @Param body body object true "id"
// @Success 200 {object} response.Response
// @Router /admin/custom-client/delete [post]
// @Security token
func (ct *CustomClient) Delete(c *gin.Context) {
	type deleteReq struct {
		Id uint `json:"id"`
	}
	req := &deleteReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if req.Id == 0 {
		response.Fail(c, 101, "id is required")
		return
	}
	cc := ct.HD.Services.CustomClientService.InfoById(req.Id)
	if cc.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if err := ct.HD.Services.CustomClientService.Delete(cc); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Download serves the bundled installer file (public, no auth required).
// When S3 is configured, proxies the file from S3 through the API server.
// @Tags CustomClient
// @Summary Download bundled installer
// @Produce  octet-stream
// @Param id path int true "ID"
// @Success 200 {file} binary
// @Router /admin/custom-client/download/{id} [get]
func (ct *CustomClient) Download(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	cc := ct.HD.Services.CustomClientService.InfoById(uint(iid))
	if cc.Id == 0 {
		c.String(404, "not found")
		return
	}
	if cc.Status != model.BundleStatusCompleted {
		c.String(400, "bundle is not ready (status: %s)", cc.Status)
		return
	}

	appName := cc.AppName
	if appName == "" {
		appName = "rustdesk"
	}
	filename := fmt.Sprintf("%s-%s-%s-%s.%s", appName, cc.Version, cc.Platform, cc.Arch, cc.Format)

	// Proxy from S3
	if cc.S3Key != "" && ct.HD.S3 != nil {
		reader, err := ct.HD.S3.GetObject(context.Background(), cc.S3Key)
		if err == nil {
			defer reader.Close()

			contentType := "application/octet-stream"
			switch cc.Format {
			case "deb":
				contentType = "application/vnd.debian.binary-package"
			case "zip":
				contentType = "application/zip"
			}

			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			c.DataFromReader(200, -1, contentType, reader, nil)
			return
		}
		// Fall through to local file on S3 error
	}

	if cc.FilePath == "" {
		c.String(404, "bundled file not found")
		return
	}

	c.FileAttachment(cc.FilePath, filename)
}

// Preview returns the signed custom.txt content for testing.
// @Tags CustomClient
// @Summary Preview custom.txt
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response
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
	response.Success(c, gin.H{"custom_txt": txt})
}
