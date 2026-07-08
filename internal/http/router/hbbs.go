package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/hbbs"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
)

// HbbsInit registers the internal /api/hbbs/* routes used by the RustDesk
// rendezvous server. These are only active when hbbs.token is configured.
func HbbsInit(g *gin.Engine, hd *deps.HandlerDeps) {
	token := hd.Config.Hbbs.Token
	if token == "" {
		return // hbbs internal API disabled
	}

	hg := g.Group("/api/hbbs")
	hg.Use(middleware.HbbsAuth(token))

	h := &hbbs.Hbbs{HD: hd}
	hg.POST("/conn-audit-ref", h.ConnAuditRef)
	hg.GET("/device-deployed", h.DeviceDeployed)
}
