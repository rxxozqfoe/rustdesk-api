// Package api_test exercises the PC-client-facing HTTP handlers in
// internal/http/controller/api. Tests build the minimal HandlerDeps each
// handler reads (the same fields the router threads in), register only the
// route under test on a fresh Gin engine, and assert both the JSON envelope
// and the resulting DB state.
package api_test

import (
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/app"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// responseWired ensures the package-level localizer/logger used by
// response.Error / response.TranslateMsg are wired exactly once for the whole
// test binary. Without this, handlers that call those helpers nil-panic.
var responseWired sync.Once

// emptyLocalizer returns a localizer backed by an empty bundle, so every lookup
// misses and TranslateMsg falls back to returning the message id verbatim —
// deterministic and good enough for assertions.
func emptyLocalizer(string) *i18n.Localizer {
	bundle := i18n.NewBundle(language.English)
	return i18n.NewLocalizer(bundle, "en")
}

// wireResponse installs the package-level response helpers exactly once.
func wireResponse(t testing.TB) {
	t.Helper()
	responseWired.Do(func() {
		response.SetLocalizer(emptyLocalizer)
		response.SetLogger(logger.New(&logger.Config{Level: "error"}))
	})
}

// newDeps builds a HandlerDeps over the kit, populating the fields the api
// handlers actually read: Config, Logger, Services, Validator, Localizer and a
// LoginLimiter. It is the test-side mirror of the router's composition root.
func newDeps(t testing.TB, kit *servicekit.Kit) *deps.HandlerDeps {
	t.Helper()
	wireResponse(t)
	v := app.NewValidator(kit.Config)
	return &deps.HandlerDeps{
		Config:    kit.Config,
		Logger:    testutil.NewLogger(t),
		Services:  kit.Services,
		Validator: v,
		Localizer: func(lang string) *i18n.Localizer { return emptyLocalizer(lang) },
		LoginLimiter: utils.NewLoginLimiter(utils.SecurityPolicy{
			CaptchaThreshold: -1, // disabled: no captcha / ban side-effects in tests
			BanThreshold:     0,
		}),
	}
}

// authMiddleware injects the given user into the Gin context under the same key
// RustAuth uses ("curUser"), plus a token, so authenticated handlers run
// without minting a real token. Use realAuth helpers when the auth path itself
// is what is being tested.
func authMiddleware(u *model.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("curUser", u)
		c.Set("token", "test-token")
		c.Next()
	}
}

// realAuth wires the production RustAuth middleware against the kit's
// UserService, for tests that go through the actual token verification path.
func realAuth(kit *servicekit.Kit) gin.HandlerFunc {
	return middleware.RustAuth(kit.Config, kit.Services.UserService)
}

// seedTokenUser creates an enabled user and a matching, non-expired UserToken
// whose Token is a JWT signed with the key RustAuth verifies against. Returns
// the user and its bearer token.
func seedTokenUser(t testing.TB, kit *servicekit.Kit, mutators ...func(*model.User)) (*model.User, string) {
	t.Helper()
	u := testutil.CreateUser(t, kit.DB, mutators...)
	token := kit.Services.UserService.GenerateToken(u)
	ut := &model.UserToken{
		UserId:    u.Id,
		Token:     token,
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	}
	if err := kit.DB.Create(ut).Error; err != nil {
		t.Fatalf("seedTokenUser: create token: %v", err)
	}
	return u, token
}
