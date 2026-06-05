package worker

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
)

type Jobs struct {
	HD *deps.HandlerDeps
}

// Register handles worker registration.
// POST /api/worker/register
func (j *Jobs) Register(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name"`
		Platforms []model.WorkerPlatform `json:"platforms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if req.Name == "" {
		response.Fail(c, 101, "name is required")
		return
	}
	w, err := j.HD.Services.WorkerRegistryService.Register(req.Name, req.Platforms)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, w)
}

// Heartbeat handles worker heartbeat.
// POST /api/worker/heartbeat
func (j *Jobs) Heartbeat(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.Heartbeat(req.Name); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// PushVersions handles worker pushing available versions.
// POST /api/worker/versions
func (j *Jobs) PushVersions(c *gin.Context) {
	var req struct {
		Name     string   `json:"name"`
		Versions []string `json:"versions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.PushVersions(req.Name, req.Versions); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

// FetchPending returns one pending job for the worker.
// POST /api/worker/jobs/pending
func (j *Jobs) FetchPending(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name"`
		Platforms []model.WorkerPlatform `json:"platforms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	job, err := j.HD.Services.FetchPendingJob(req.Name, req.Platforms)
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

// JobStatus returns the current status of a job (for worker to check cancellation).
// GET /api/worker/jobs/:id/status
func (j *Jobs) JobStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	jobType := c.DefaultQuery("type", "pre-build")
	var status string
	switch jobType {
	case "pre-build":
		pb := j.HD.Services.PreBuildService.InfoById(uint(id))
		if pb.Id == 0 {
			response.Fail(c, 101, "job not found")
			return
		}
		status = pb.Status
	case "bundle":
		cc := j.HD.Services.CustomClientService.InfoById(uint(id))
		if cc.Id == 0 {
			response.Fail(c, 101, "job not found")
			return
		}
		status = cc.Status
	default:
		response.Fail(c, 101, "unknown job type")
		return
	}
	response.Success(c, gin.H{"status": status})
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
	if err := j.HD.Services.StartJob(uint(id), req.Type); err != nil {
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
	if err := j.HD.Services.AppendLog(uint(id), req.Content); err != nil {
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
		LogS3Key string `json:"log_s3_key,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}

	var err error
	switch req.Type {
	case "pre-build":
		err = j.HD.Services.CompletePreBuild(uint(id), req.S3Key, req.LogS3Key)
	case "bundle":
		err = j.HD.Services.CompleteBundle(uint(id), req.S3Key, req.FileSize)
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
		Type     string `json:"type"`
		Error    string `json:"error"`
		LogS3Key string `json:"log_s3_key,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "invalid request: "+err.Error())
		return
	}
	if err := j.HD.Services.FailJob(uint(id), req.Type, req.Error, req.LogS3Key); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}
