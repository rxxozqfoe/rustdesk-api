// Package deps defines HandlerDeps, the dependency aggregate injected into
// every HTTP controller, middleware factory, and router bind function.
//
// It lives in its own sub-package so that both the parent http package and
// its http/router sub-package can import it without creating an import cycle.
package deps

import (
	"github.com/rxxozqfoe/rustdesk-api/internal/app"
	"github.com/rxxozqfoe/rustdesk-api/internal/config"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/logger"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/s3"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/upload"
	"github.com/rxxozqfoe/rustdesk-api/internal/service"
	"github.com/rxxozqfoe/rustdesk-api/internal/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// LocalizerFunc returns a localizer for the given Accept-Language header value.
type LocalizerFunc func(lang string) *i18n.Localizer

// HandlerDeps is the single aggregate that HTTP handlers receive via
// constructor injection. Every controller struct holds a *HandlerDeps field.
// Middleware factories typically take narrower slices of these fields.
type HandlerDeps struct {
	Config       *config.Config
	Logger       *logger.Logger
	Validator    app.AppValidator
	Localizer    LocalizerFunc
	LoginLimiter *utils.LoginLimiter
	Oss          *upload.Oss
	S3           *s3.Client // nil when S3 is not configured
	Services     *service.Service
}
