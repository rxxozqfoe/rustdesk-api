package http

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/router"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

// ApiInit builds the gin engine and registers all routes. It takes the shared
// HandlerDeps constructed by the composition root (cmd/apimain.InitApp), so
// nothing in this package reaches for package-level state.
func ApiInit(hd *deps.HandlerDeps) {
	gin.SetMode(hd.Config.Gin.Mode)
	g := gin.New()

	// [WARNING] You trusted all proxies, this is NOT safe. We recommend you set a value.
	// See https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
	if hd.Config.Gin.TrustProxy != "" {
		pro := strings.Split(hd.Config.Gin.TrustProxy, ",")
		if err := g.SetTrustedProxies(pro); err != nil {
			panic(err)
		}
	}

	if hd.Config.Gin.Mode == gin.ReleaseMode && hd.Logger != nil {
		// Route gin's Recovery error output into our structured logger.
		gin.DefaultErrorWriter = hd.Logger.WriterLevel(logrus.ErrorLevel)
	}
	g.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "404 not found")
	})
	g.Use(middleware.Logger(hd.Logger), middleware.Limiter(hd.LoginLimiter), gin.Recovery())
	router.WebInit(g, hd)
	router.Init(g, hd)
	router.ApiInit(g, hd)
	Run(g, hd.Config.Gin.ApiAddr)
}
