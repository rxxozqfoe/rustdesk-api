package middleware_test

import (
	"sync"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// responseWired ensures the package-level localizer/logger used by
// response.Fail / response.TranslateMsg are wired exactly once for the whole
// test binary. Without this, middleware that calls response.Fail would
// nil-panic in TranslateMsg.
var responseWired sync.Once

// wireResponse installs a localizer backed by an empty bundle. Lookups always
// miss, so TranslateMsg logs a warning and falls back to returning the message
// id verbatim, which is deterministic and good enough for assertions.
func wireResponse(t testing.TB) {
	t.Helper()
	responseWired.Do(func() {
		bundle := i18n.NewBundle(language.English)
		response.SetLocalizer(func(lang string) *i18n.Localizer {
			return i18n.NewLocalizer(bundle, "en")
		})
		response.SetLogger(logger.New(&logger.Config{Level: "error"}))
	})
}

// seedTokenUser creates an enabled user and a matching, non-expired UserToken
// whose Token is minted by the service's GenerateToken (a JWT signed with the
// same key the middleware verifies against). It returns the user and token.
func seedTokenUser(t testing.TB, kit *servicekit.Kit, mutators ...func(*model.User)) (*model.User, string) {
	t.Helper()
	u := testutil.CreateUser(t, kit.DB, mutators...)
	token := kit.Services.GenerateToken(u)
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

// seedExpiredToken creates an enabled user and an already-expired UserToken.
func seedExpiredToken(t testing.TB, kit *servicekit.Kit) (*model.User, string) {
	t.Helper()
	u := testutil.CreateUser(t, kit.DB)
	token := kit.Services.GenerateToken(u)
	ut := &model.UserToken{
		UserId:    u.Id,
		Token:     token,
		ExpiredAt: time.Now().Add(-time.Hour).Unix(),
	}
	if err := kit.DB.Create(ut).Error; err != nil {
		t.Fatalf("seedExpiredToken: create token: %v", err)
	}
	return u, token
}
