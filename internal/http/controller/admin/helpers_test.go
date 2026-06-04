package admin_test

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/app"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

// responseWired ensures the package-level localizer/logger used by
// response.Fail / response.TranslateMsg are wired exactly once for the whole
// test binary. Without this, any handler that calls response.Fail would
// nil-panic in TranslateMsg.
var responseWired sync.Once

// wireResponse installs a localizer backed by an empty bundle. Lookups always
// miss, so TranslateMsg falls back to returning the message id verbatim, which
// is deterministic and good enough for assertions.
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

// newDeps builds a HandlerDeps backed by a servicekit kit, wiring a real
// validator, login limiter, localizer, and logger — everything the admin
// controllers reach for. Returns the deps and the kit so tests can seed and
// inspect the DB.
func newDeps(t testing.TB) (*deps.HandlerDeps, *servicekit.Kit) {
	t.Helper()
	wireResponse(t)
	kit := servicekit.New(t)
	hd := &deps.HandlerDeps{
		Config:    kit.Config,
		Logger:    testutil.NewLogger(t),
		Validator: app.NewValidator(kit.Config),
		Localizer: func(lang string) *i18n.Localizer {
			return i18n.NewLocalizer(i18n.NewBundle(language.English), "en")
		},
		LoginLimiter: utils.NewLoginLimiter(utils.SecurityPolicy{
			// Disable captcha/ban so login tests are deterministic.
			CaptchaThreshold: -1,
			BanThreshold:     0,
		}),
		Services: kit.Services,
	}
	return hd, kit
}

// withCurUser returns a middleware that sets the same context keys the real
// BackendUserAuth middleware sets (curUser + token), so handlers that read
// helper.CurUser see an authenticated user without minting a JWT.
func withCurUser(u *model.User, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("curUser", u)
		c.Set("token", token)
		c.Next()
	}
}

// envelope mirrors response.Response for decoding the standard admin JSON
// envelope {"code":..,"message":..,"data":..}.
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// doJSON serves an HTTP request against engine and decodes the envelope.
func doJSON(t testing.TB, engine *gin.Engine, method, target, body string) (*httptest.ResponseRecorder, envelope) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, method, target, body)
	engine.ServeHTTP(rec, req)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "body: %s", rec.Body.String())
	return rec, env
}

// seedTokenUser creates an enabled user plus a matching, non-expired UserToken
// whose Token is signed by the service. Returns the user and token string.
func seedTokenUser(t testing.TB, kit *servicekit.Kit, mutators ...func(*model.User)) (*model.User, string) {
	t.Helper()
	u := testutil.CreateUser(t, kit.DB, mutators...)
	token := kit.Services.UserService.GenerateToken(u)
	ut := &model.UserToken{
		UserId:    u.Id,
		Token:     token,
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, kit.DB.Create(ut).Error)
	return u, token
}

// testKit wraps a servicekit.Kit so per-controller engine helpers can return a
// single value alongside their controller.
type testKit struct {
	kit *servicekit.Kit
}

// itoa is a tiny convenience for building URL path segments / JSON ids.
func itoa[T ~uint | ~int](v T) string {
	return strconv.FormatInt(int64(v), 10)
}
