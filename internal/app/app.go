package app

import (
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/rxxozqfoe/rustdesk-api/internal/config"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/cache"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/logger"
	"github.com/spf13/viper"
)

// AppContext holds infrastructure-level dependencies shared across the whole
// application: configuration, logger, DB handles, caches, etc.
//
// HTTP-layer concerns (validator, localizer, login limiter, OSS uploader) used
// to live on this struct but now live on http/deps.HandlerDeps because they
// operate on gin.Context / per-request state.
//
// AppContext is constructed once in cmd/apimain.InitApp (the composition root).
// There is no package-level singleton — dependencies flow explicitly from main.
type AppContext struct {
	Config     config.Config
	Logger     *logger.Logger
	ConfigPath string
	Viper      *viper.Viper
	Redis      *redis.Client
	Cache      cache.Handler
}

// AppValidator wraps validation logic with i18n support. Built by NewValidator
// and stored on http/deps.HandlerDeps.
type AppValidator struct {
	Validate    *validator.Validate
	UT          *ut.UniversalTranslator
	VTrans      ut.Translator
	ValidStruct func(*gin.Context, interface{}) []string
	ValidVar    func(ctx *gin.Context, field interface{}, tag string) []string
}
