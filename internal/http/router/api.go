package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lejianwen/rustdesk-api/v2/docs/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"net/http"
)

// ApiInit registers all /api routes (the RustDesk PC-client surface). The
// shared HandlerDeps is threaded into every controller and middleware factory.
func ApiInit(g *gin.Engine, hd *deps.HandlerDeps) {
	if hd.Config.App.ShowSwagger == 1 {
		g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("api")))
	}
	g.LoadHTMLGlob("resources/templates/*")

	users := hd.Services.UserService

	frg := g.Group("/api")

	{
		i := &api.Index{HD: hd}
		frg.GET("/", i.Index)
		frg.GET("/version", i.Version)
		frg.POST("/heartbeat", i.Heartbeat)
	}

	{
		l := &api.Login{HD: hd}
		frg.GET("/login-options", l.LoginOptions)
		frg.POST("/login", l.Login)
	}

	{
		o := &api.Oauth{HD: hd}
		frg.POST("/oidc/auth", o.OidcAuth)
		frg.GET("/oidc/auth-query", o.OidcAuthQuery)
		frg.GET("/oauth/callback", o.OauthCallback)
		frg.GET("/oauth/login", o.OauthCallback)
		frg.GET("/oauth/msg", o.Message)
		frg.GET("/oidc/callback", o.OauthCallback)
		frg.GET("/oidc/login", o.OauthCallback)
		frg.GET("/oidc/msg", o.Message)
	}
	{
		pe := &api.Peer{HD: hd}
		frg.POST("/sysinfo", pe.SysInfo)
		frg.POST("/sysinfo_ver", pe.SysInfoVer)
	}

	if hd.Config.App.WebClient == 1 {
		WebClientRoutes(frg, hd)
	}

	{
		au := &api.Audit{HD: hd}
		frg.POST("/audit/conn", au.AuditConn)
		frg.POST("/audit/file", au.AuditFile)
	}

	frg.Use(middleware.RustAuth(hd.Config, users))
	{
		u := &api.User{HD: hd}
		frg.GET("/user/info", u.Info)
		frg.POST("/currentUser", u.Info)
	}
	{
		l := &api.Login{HD: hd}
		frg.POST("/logout", l.Logout)
	}
	{
		gr := &api.Group{HD: hd}
		frg.GET("/users", gr.Users)
		frg.GET("/peers", gr.Peers)
		frg.GET("/device-group/accessible", gr.Device)
	}

	{
		ab := &api.Ab{HD: hd}
		frg.GET("/ab", ab.Ab)
		frg.POST("/ab", ab.UpAb)
	}

	PersonalRoutes(frg, hd)
	g.StaticFS("/upload", http.Dir(hd.Config.Gin.ResourcesPath+"/public/upload"))
}

func PersonalRoutes(frg *gin.RouterGroup, hd *deps.HandlerDeps) {
	ab := &api.Ab{HD: hd}
	frg.POST("/ab/personal", ab.Personal)
	frg.POST("/ab/settings", ab.Settings)
	frg.POST("/ab/shared/profiles", ab.SharedProfiles)
	frg.POST("/ab/peers", ab.Peers)
	frg.POST("/ab/tags/:guid", ab.PTags)
	frg.POST("/ab/peer/add/:guid", ab.PeerAdd)
	frg.DELETE("/ab/peer/:guid", ab.PeerDel)
	frg.PUT("/ab/peer/update/:guid", ab.PeerUpdate)
	frg.POST("/ab/tag/add/:guid", ab.TagAdd)
	frg.PUT("/ab/tag/rename/:guid", ab.TagRename)
	frg.PUT("/ab/tag/update/:guid", ab.TagUpdate)
	frg.DELETE("/ab/tag/:guid", ab.TagDel)
}

func WebClientRoutes(frg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	w := &api.WebClient{HD: hd}
	frg.POST("/shared-peer", w.SharedPeer)
	frg.POST("/server-config", middleware.RustAuth(hd.Config, users), w.ServerConfig)
	frg.POST("/server-config-v2", middleware.RustAuth(hd.Config, users), w.ServerConfigV2)
}
