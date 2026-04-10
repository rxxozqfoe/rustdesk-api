package admin

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type BuildArtifact struct {
	HD *deps.HandlerDeps
}

// List
// @Tags BuildArtifact
// @Summary List build artifacts
// @Description Get paginated list of pre-built base binaries
// @Accept  json
// @Produce  json
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Response{data=model.BuildArtifactList}
// @Failure 500 {object} response.Response
// @Router /admin/build-artifact/list [get]
// @Security token
func (ct *BuildArtifact) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := ct.HD.Services.BuildArtifactService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Upload handles multipart file upload of a base binary.
// @Tags BuildArtifact
// @Summary Upload base binary
// @Description Upload a pre-built binary package for repackaging
// @Accept  multipart/form-data
// @Produce  json
// @Param file formance file true "Binary file"
// @Param platform formData string true "Platform (linux, windows, macos, android)"
// @Param arch formData string true "Architecture (x86_64, aarch64)"
// @Param format formData string true "Format (deb, exe, dmg, apk)"
// @Param version formData string false "Version"
// @Success 200 {object} response.Response{data=model.BuildArtifact}
// @Failure 500 {object} response.Response
// @Router /admin/build-artifact/upload [post]
// @Security token
func (ct *BuildArtifact) Upload(c *gin.Context) {
	platform := c.PostForm("platform")
	arch := c.PostForm("arch")
	format := c.PostForm("format")
	version := c.PostForm("version")

	if platform == "" || arch == "" || format == "" {
		response.Fail(c, 101, "platform, arch, and format are required")
		return
	}

	validPlatforms := map[string]bool{"linux": true, "windows": true, "macos": true, "android": true}
	if !validPlatforms[platform] {
		response.Fail(c, 101, "invalid platform: must be linux, windows, macos, or android")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 101, "file is required: "+err.Error())
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Fail(c, 101, "failed to open uploaded file: "+err.Error())
		return
	}
	defer src.Close()

	ba, err := ct.HD.Services.BuildArtifactService.SaveUploadedFile(src, file.Filename, platform, arch, format, version)
	if err != nil {
		response.Fail(c, 101, "failed to save file: "+err.Error())
		return
	}
	response.Success(c, ba)
}

// Delete
// @Tags BuildArtifact
// @Summary Delete build artifact
// @Description Delete a build artifact and its file from disk
// @Accept  json
// @Produce  json
// @Param body body object true "id"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/build-artifact/delete [post]
// @Security token
func (ct *BuildArtifact) Delete(c *gin.Context) {
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
	ba := ct.HD.Services.BuildArtifactService.InfoById(req.Id)
	if ba.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := ct.HD.Services.BuildArtifactService.Delete(ba)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Download repackages a base binary with the custom.txt from a specific config and streams it.
// @Tags CustomClient
// @Summary Download repackaged binary
// @Description Download a repackaged binary with custom client config injected
// @Produce  octet-stream
// @Param id path int true "Custom Client Config ID"
// @Param platform query string true "Platform (linux, windows, macos, android)"
// @Param arch query string true "Architecture (x86_64, aarch64)"
// @Param format query string true "Format (deb, exe, dmg, apk)"
// @Success 200 {file} binary
// @Failure 500 {object} response.Response
// @Router /admin/custom-client/download/{id} [get]
// @Security token
func (ct *BuildArtifact) Download(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	platform := c.Query("platform")
	arch := c.Query("arch")
	format := c.Query("format")

	if platform == "" || arch == "" || format == "" {
		response.Fail(c, 101, "platform, arch, and format query params are required")
		return
	}

	// Look up the custom client config
	cc := ct.HD.Services.CustomClientService.InfoById(uint(iid))
	if cc.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}

	// Generate custom.txt content
	customTxt, err := ct.HD.Services.CustomClientService.GenerateCustomTxt(cc)
	if err != nil {
		response.Fail(c, 101, "failed to generate custom.txt: "+err.Error())
		return
	}

	// Look up the matching base binary
	ba := ct.HD.Services.BuildArtifactService.FindByPlatformArchFormat(platform, arch, format)
	if ba.Id == 0 {
		response.Fail(c, 101, "no base binary found for "+platform+"/"+arch+"/"+format)
		return
	}

	// Repackage the base binary with custom.txt injected
	result, err := ct.HD.Services.RepackagerService.Repackage(ba.FilePath, format, customTxt)
	if err != nil {
		response.Fail(c, 101, "repackaging failed: "+err.Error())
		return
	}
	defer result.Cleanup()

	// Determine download filename
	appName := cc.AppName
	if appName == "" {
		appName = "rustdesk"
	}
	ext := format
	if format == "exe" {
		ext = "zip" // Windows is delivered as zip containing exe + custom.txt
	}
	downloadName := fmt.Sprintf("%s-%s-%s.%s", appName, platform, arch, ext)

	c.FileAttachment(result.FilePath, filepath.Base(downloadName))
}
