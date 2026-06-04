package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type Record struct {
	HD *deps.HandlerDeps
}

func (r *Record) recordDir() string {
	return filepath.Join(r.HD.Config.Gin.ResourcesPath, "public", "records")
}

// Upload handles recording upload operations (new/part/tail/remove).
// @Tags Record
// @Summary Upload recording
// @Description Upload connection recording file
// @Accept  octet-stream
// @Produce  json
// @Param type query string true "Operation type: new, part, tail, remove"
// @Param file query string true "File name"
// @Param offset query string false "Byte offset (for part/tail)"
// @Param length query string false "Segment length (for part/tail)"
// @Success 200 {object} nil
// @Failure 500 {object} response.ErrorResponse
// @Router /record [post]
func (r *Record) Upload(c *gin.Context) {
	opType := c.Query("type")
	fileName := c.Query("file")

	if fileName == "" {
		response.Error(c, "file parameter is required")
		return
	}

	// Sanitize filename to prevent path traversal
	fileName = filepath.Base(fileName)

	dir := r.recordDir()

	switch opType {
	case "new":
		// Create directory if not exists
		if err := os.MkdirAll(dir, 0755); err != nil {
			response.Error(c, "failed to create record directory: "+err.Error())
			return
		}
		// Create empty file
		fp := filepath.Join(dir, fileName)
		f, err := os.Create(fp)
		if err != nil {
			response.Error(c, "failed to create file: "+err.Error())
			return
		}
		if err := f.Close(); err != nil {
			response.Error(c, "failed to close file: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{})

	case "part":
		offsetStr := c.Query("offset")
		offset, _ := strconv.ParseInt(offsetStr, 10, 64)

		fp := filepath.Join(dir, fileName)
		f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			response.Error(c, "failed to open file: "+err.Error())
			return
		}

		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			response.Error(c, "failed to seek: "+err.Error())
			return
		}

		if _, err := io.Copy(f, c.Request.Body); err != nil {
			_ = f.Close()
			response.Error(c, "failed to write data: "+err.Error())
			return
		}
		if err := f.Close(); err != nil {
			response.Error(c, "failed to close file: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{})

	case "tail":
		offsetStr := c.Query("offset")
		offset, _ := strconv.ParseInt(offsetStr, 10, 64)

		fp := filepath.Join(dir, fileName)
		f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			response.Error(c, "failed to open file: "+err.Error())
			return
		}

		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			response.Error(c, "failed to seek: "+err.Error())
			return
		}

		if _, err := io.Copy(f, c.Request.Body); err != nil {
			_ = f.Close()
			response.Error(c, "failed to write data: "+err.Error())
			return
		}
		if err := f.Close(); err != nil {
			response.Error(c, "failed to close file: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{})

	case "remove":
		fp := filepath.Join(dir, fileName)
		// best-effort removal; report failure to caller
		if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
			response.Error(c, "failed to remove file: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{})

	default:
		response.Error(c, "invalid type parameter")
	}
}
