package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lejianwen/rustdesk-api/v2/docs/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/admin/my"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Init registers all /api/admin routes. The shared HandlerDeps is threaded
// into every controller struct and middleware factory so nothing reaches for
// package-level state.
func Init(g *gin.Engine, hd *deps.HandlerDeps) {
	if hd.Config.App.ShowSwagger == 1 {
		g.GET("/admin/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("admin")))
	}

	users := hd.Services.UserService

	adg := g.Group("/api/admin")
	LoginBind(adg, hd)
	adg.POST("/user/register", (&admin.User{HD: hd}).Register)

	ConfigBind(adg, hd)

	// Public (no auth) endpoints
	CustomClientDownloadBind(adg, hd)

	adg.Use(middleware.BackendUserAuth(users))
	UserBind(adg, hd)
	GroupBind(adg, hd)
	TagBind(adg, hd)
	AddressBookBind(adg, hd)
	PeerBind(adg, hd)
	OauthBind(adg, hd)
	LoginLogBind(adg, hd)
	AuditBind(adg, hd)
	AddressBookCollectionBind(adg, hd)
	AddressBookCollectionRuleBind(adg, hd)
	UserTokenBind(adg, hd)

	ShareRecordBind(adg, hd)
	MyBind(adg, hd)

	RustdeskCmdBind(adg, hd)
	DeviceGroupBind(adg, hd)
	StrategyBind(adg, hd)
	CustomClientBind(adg, hd)
	BuildArtifactBind(adg, hd)
	PreBuildBind(adg, hd)
	WorkerStatusBind(adg, hd)
}

func RustdeskCmdBind(adg *gin.RouterGroup, hd *deps.HandlerDeps) {
	cont := &admin.Rustdesk{HD: hd}
	rg := adg.Group("/rustdesk")
	rg.POST("/sendCmd", cont.SendCmd)
	rg.GET("/cmdList", cont.CmdList)
	rg.POST("/cmdDelete", cont.CmdDelete)
	rg.POST("/cmdCreate", cont.CmdCreate)
}

func LoginBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	cont := &admin.Login{HD: hd}
	rg.POST("/login", cont.Login)
	rg.GET("/captcha", cont.Captcha)
	rg.POST("/logout", cont.Logout)
	rg.GET("/login-options", cont.LoginOptions)
	rg.POST("/oidc/auth", cont.OidcAuth)
	rg.GET("/oidc/auth-query", cont.OidcAuthQuery)
}

func UserBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/user")
	{
		cont := &admin.User{HD: hd}
		aR.GET("/current", cont.Current)
		aR.POST("/changeCurPwd", cont.ChangeCurPwd)
		aR.POST("/myOauth", cont.MyOauth)
		aR.POST("/groupUsers", cont.GroupUsers)
	}
	aRP := rg.Group("/user").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.User{HD: hd}
		aRP.GET("/list", cont.List)
		aRP.GET("/detail/:id", cont.Detail)
		aRP.POST("/create", cont.Create)
		aRP.POST("/update", cont.Update)
		aRP.POST("/delete", cont.Delete)
		aRP.POST("/changePwd", cont.UpdatePassword)
	}
}

func GroupBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/group").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.Group{HD: hd}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func DeviceGroupBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/device_group").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.DeviceGroup{HD: hd}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func StrategyBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.Strategy{HD: hd}
	aR := rg.Group("/strategy").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
		aR.POST("/assign", cont.Assign)
		aR.GET("/assignments/:id", cont.Assignments)
	}
}

func TagBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/tag").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.Tag{HD: hd}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func AddressBookBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/address_book")
	{
		cont := &admin.AddressBook{HD: hd}
		aR.POST("/shareByWebClient", cont.ShareByWebClient)

		arp := aR.Use(middleware.AdminPrivilege(users))
		arp.GET("/list", cont.List)
		arp.POST("/create", cont.Create)
		arp.POST("/update", cont.Update)
		arp.POST("/delete", cont.Delete)
		arp.POST("/batchCreate", cont.BatchCreate)
		arp.POST("/batchCreateFromPeers", cont.BatchCreateFromPeers)
	}
}

func PeerBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.Peer{HD: hd}
	rg.Group("/peer").POST("/simpleData", cont.SimpleData)
	aR := rg.Group("/peer").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
		aR.POST("/batchDelete", cont.BatchDelete)
	}
}

func OauthBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/oauth")
	{
		cont := &admin.Oauth{HD: hd}
		aR.POST("/confirm", cont.Confirm)
		aR.POST("/bind", cont.ToBind)
		aR.POST("/bindConfirm", cont.BindConfirm)
		aR.POST("/unbind", cont.Unbind)
		aR.GET("/info", cont.Info)
	}
	arp := aR.Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.Oauth{HD: hd}
		arp.GET("/list", cont.List)
		arp.GET("/detail/:id", cont.Detail)
		arp.POST("/create", cont.Create)
		arp.POST("/update", cont.Update)
		arp.POST("/delete", cont.Delete)
	}
}

func LoginLogBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.LoginLog{HD: hd}
	aR := rg.Group("/login_log").Use(middleware.AdminPrivilege(users))
	aR.GET("/list", cont.List)
	aR.POST("/delete", cont.Delete)
	aR.POST("/batchDelete", cont.BatchDelete)
}

func AuditBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.Audit{HD: hd}
	aR := rg.Group("/audit_conn").Use(middleware.AdminPrivilege(users))
	aR.GET("/list", cont.ConnList)
	aR.POST("/delete", cont.ConnDelete)
	aR.POST("/batchDelete", cont.BatchConnDelete)
	aR.POST("/disconnect", cont.ConnDisconnect)
	afR := rg.Group("/audit_file").Use(middleware.AdminPrivilege(users))
	afR.GET("/list", cont.FileList)
	afR.POST("/delete", cont.FileDelete)
	afR.POST("/batchDelete", cont.BatchFileDelete)
}

func AddressBookCollectionBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/address_book_collection").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.AddressBookCollection{HD: hd}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func AddressBookCollectionRuleBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/address_book_collection_rule").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.AddressBookCollectionRule{HD: hd}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func UserTokenBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/user_token").Use(middleware.AdminPrivilege(users))
	cont := &admin.UserToken{HD: hd}
	aR.GET("/list", cont.List)
	aR.POST("/delete", cont.Delete)
	aR.POST("/batchDelete", cont.BatchDelete)
}

func ConfigBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/config")
	rs := &admin.Config{HD: hd}

	aR.GET("/admin", rs.AdminConfig)

	aR.Use(middleware.BackendUserAuth(users))
	aR.GET("/server", rs.ServerConfig)
	aR.GET("/app", rs.AppConfig)
}

func MyBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	{
		cont := &my.ShareRecord{HD: hd}
		rg.GET("/my/share_record/list", cont.List)
		rg.POST("/my/share_record/delete", cont.Delete)
		rg.POST("/my/share_record/batchDelete", cont.BatchDelete)
	}

	{
		cont := &my.AddressBook{HD: hd}
		rg.GET("/my/address_book/list", cont.List)
		rg.POST("/my/address_book/create", cont.Create)
		rg.POST("/my/address_book/update", cont.Update)
		rg.POST("/my/address_book/delete", cont.Delete)
		rg.POST("/my/address_book/batchCreateFromPeers", cont.BatchCreateFromPeers)
		rg.POST("/my/address_book/batchUpdateTags", cont.BatchUpdateTags)
	}

	{
		cont := &my.Tag{HD: hd}
		rg.GET("/my/tag/list", cont.List)
		rg.POST("/my/tag/create", cont.Create)
		rg.POST("/my/tag/update", cont.Update)
		rg.POST("/my/tag/delete", cont.Delete)
	}

	{
		cont := &my.AddressBookCollection{HD: hd}
		rg.GET("/my/address_book_collection/list", cont.List)
		rg.POST("/my/address_book_collection/create", cont.Create)
		rg.POST("/my/address_book_collection/update", cont.Update)
		rg.POST("/my/address_book_collection/delete", cont.Delete)
	}

	{
		cont := &my.AddressBookCollectionRule{HD: hd}
		rg.GET("/my/address_book_collection_rule/list", cont.List)
		rg.POST("/my/address_book_collection_rule/create", cont.Create)
		rg.POST("/my/address_book_collection_rule/update", cont.Update)
		rg.POST("/my/address_book_collection_rule/delete", cont.Delete)
	}

	{
		cont := &my.Peer{HD: hd}
		rg.GET("/my/peer/list", cont.List)
	}

	{
		cont := &my.LoginLog{HD: hd}
		rg.GET("/my/login_log/list", cont.List)
		rg.POST("/my/login_log/delete", cont.Delete)
		rg.POST("/my/login_log/batchDelete", cont.BatchDelete)
	}
}

func ShareRecordBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	aR := rg.Group("/share_record").Use(middleware.AdminPrivilege(users))
	{
		cont := &admin.ShareRecord{HD: hd}
		aR.GET("/list", cont.List)
		aR.POST("/delete", cont.Delete)
		aR.POST("/batchDelete", cont.BatchDelete)
	}
}

func CustomClientDownloadBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	cont := &admin.CustomClient{HD: hd}
	rg.GET("/custom-client/download/:id", cont.Download)
}

func CustomClientBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.CustomClient{HD: hd}
	aR := rg.Group("/custom-client").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/delete", cont.Delete)
		aR.GET("/preview/:id", cont.Preview)
	}
}

func BuildArtifactBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.BuildArtifact{HD: hd}
	aR := rg.Group("/build-artifact").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/list", cont.List)
		aR.POST("/delete", cont.Delete)
	}
}

func PreBuildBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.PreBuild{HD: hd}
	aR := rg.Group("/pre-build").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/versions", cont.Versions)
		aR.POST("/trigger", cont.Trigger)
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.GET("/log/:id", cont.Log)
		aR.POST("/cancel/:id", cont.Cancel)
		aR.POST("/delete", cont.Delete)
	}
}

func WorkerStatusBind(rg *gin.RouterGroup, hd *deps.HandlerDeps) {
	users := hd.Services.UserService
	cont := &admin.WorkerStatus{HD: hd}
	aR := rg.Group("/worker").Use(middleware.AdminPrivilege(users))
	{
		aR.GET("/list", cont.List)
	}
}
