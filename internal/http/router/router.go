package router

import (
	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/controller/web"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
)

// WebInit registers web-UI static routes. It takes the shared HandlerDeps so
// it can read Config and inject it into the web.Index controller.
func WebInit(g *gin.Engine, hd *deps.HandlerDeps) {
	i := &web.Index{HD: hd}
	g.GET("/", i.Index)
}
