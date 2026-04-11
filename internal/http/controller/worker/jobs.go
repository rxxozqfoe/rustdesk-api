package worker

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

type Jobs struct {
	HD *deps.HandlerDeps
}

// FetchPending returns one pending job for the worker.
// GET /api/worker/jobs/pending
func (j *Jobs) FetchPending(c *gin.Context) {
	job, err := j.HD.Services.WorkerService.FetchPendingJob()
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if job == nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}
	response.Success(c, job)
}

// Start marks a job as started by the worker.
// POST /api/worker/jobs/:id/start
func (j *Jobs) Start(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Type string `json:"type"` // "pre-build" or "bundle"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.WorkerService.StartJob(uint(id), req.Type); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// AppendLog appends log content to a pre-build job.
// POST /api/worker/jobs/:id/log
func (j *Jobs) AppendLog(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.WorkerService.AppendLog(uint(id), req.Content); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// Complete marks a job as completed.
// POST /api/worker/jobs/:id/complete
func (j *Jobs) Complete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Type     string `json:"type"`
		S3Key    string `json:"s3_key"`
		FileSize int64  `json:"file_size,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}

	var err error
	switch req.Type {
	case "pre-build":
		err = j.HD.Services.WorkerService.CompletePreBuild(uint(id), req.S3Key)
	case "bundle":
		err = j.HD.Services.WorkerService.CompleteBundle(uint(id), req.S3Key, req.FileSize)
	default:
		response.Fail(c, 101, "unknown job type: "+req.Type)
		return
	}
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// Fail marks a job as failed.
// POST /api/worker/jobs/:id/fail
func (j *Jobs) Fail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.WorkerService.FailJob(uint(id), req.Type, req.Error); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

