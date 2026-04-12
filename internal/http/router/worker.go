package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/worker"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
)

// WorkerInit registers the /api/worker/* routes used by the build-worker service.
// These are only active when worker.token is configured.
func WorkerInit(g *gin.Engine, hd *deps.HandlerDeps) {
	token := hd.Config.Worker.Token
	if token == "" {
		return // worker API disabled
	}

	wg := g.Group("/api/worker")
	wg.Use(middleware.WorkerAuth(token))

	jobs := &worker.Jobs{HD: hd}
	wg.POST("/register", jobs.Register)
	wg.POST("/heartbeat", jobs.Heartbeat)
	wg.POST("/versions", jobs.PushVersions)
	wg.POST("/jobs/pending", jobs.FetchPending)
	wg.GET("/jobs/:id/status", jobs.JobStatus)
	wg.POST("/jobs/:id/start", jobs.Start)
	wg.POST("/jobs/:id/log", jobs.AppendLog)
	wg.POST("/jobs/:id/complete", jobs.Complete)
	wg.POST("/jobs/:id/fail", jobs.Fail)
}
