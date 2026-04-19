package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/router"
)

// ApiInit builds the gin engine and registers all routes. It takes the shared
// HandlerDeps constructed by the composition root (cmd/apimain.InitApp), so
// nothing in this package reaches for package-level state.
func ApiInit(hd *deps.HandlerDeps) {
	gin.SetMode(hd.Config.Gin.Mode)

	// Route gin's own writers (panic recovery, debug warnings) into our
	// structured slog-backed logger so nothing escapes to raw stdout.
	if hd.Logger != nil {
		gin.DefaultWriter = hd.Logger.Writer(slog.LevelInfo)
		gin.DefaultErrorWriter = hd.Logger.Writer(slog.LevelError)
	}

	g := gin.New()

	// [WARNING] You trusted all proxies, this is NOT safe. We recommend you set a value.
	// See https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
	if hd.Config.Gin.TrustProxy != "" {
		pro := strings.Split(hd.Config.Gin.TrustProxy, ",")
		if err := g.SetTrustedProxies(pro); err != nil {
			panic(err)
		}
	}

	g.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "404 not found")
	})
	g.Use(middleware.Logger(hd.Logger, hd.Config.Logger), middleware.Limiter(hd.LoginLimiter), gin.Recovery())
	router.WebInit(g, hd)
	router.Init(g, hd)
	router.ApiInit(g, hd)
	router.WorkerInit(g, hd)
	Run(g, hd.Config.Gin.ApiAddr)
}
