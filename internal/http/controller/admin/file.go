package admin

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	deps "github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/upload"
)

type File struct {
	HD *deps.HandlerDeps
}

// OssToken 文件
// @Tags 文件
// @Summary 获取ossToken
// @Description 获取ossToken
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/file/oss_token [get]
// @Security token
func (f *File) OssToken(c *gin.Context) {
	token := f.HD.Oss.GetPolicyToken("")
	response.Success(c, token)
}

type FileBack struct {
	upload.CallbackBaseForm
	Url string `json:"url"`
}

// Notify 上传成功后回调
func (f *File) Notify(c *gin.Context) {

	res := f.HD.Oss.Verify(c.Request)
	if !res {
		response.Fail(c, 101, response.TranslateMsg(c, "NoAccess"))
		return
	}
	fm := &FileBack{}
	if err := c.ShouldBind(fm); err != nil {
		fmt.Println(err)
	}
	fm.Url = f.HD.Config.Oss.Host + "/" + fm.Filename
	response.Success(c, fm)

}

// Upload 上传文件到本地
// @Tags 文件
// @Summary 上传文件到本地
// @Description 上传文件到本地
// @Accept  multipart/form-data
// @Produce  json
// @Param file formData file true "上传文件示例"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/file/upload [post]
// @Security token
func (f *File) Upload(c *gin.Context) {
	file, _ := c.FormFile("file")
	timePath := time.Now().Format("20060102") + "/"
	webPath := "/upload/" + timePath
	path := f.HD.Config.Gin.ResourcesPath + webPath
	dst := path + file.Filename
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return
	}
	// 上传文件至指定目录
	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		return
	}
	// 返回文件web地址
	response.Success(c, gin.H{
		"url": webPath + file.Filename,
	})
}
