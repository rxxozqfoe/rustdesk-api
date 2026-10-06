package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// limiterEngine wires the Limiter middleware in front of okHandler.
func limiterEngine(t *testing.T, ll *utils.LoginLimiter) *gin.Engine {
	engine := testutil.NewEngine(t)
	engine.GET("/login", middleware.Limiter(ll), okHandler)
	return engine
}

// httptest.NewRequest defaults RemoteAddr to this; ClientIP derives from it.
const testClientIP = "192.0.2.1"

func newLimiterPolicy() utils.SecurityPolicy {
	return utils.SecurityPolicy{
		CaptchaThreshold: 3,
		BanThreshold:     2,
		AttemptsWindow:   5 * time.Minute,
		BanDuration:      30 * time.Minute,
	}
}

func TestLimiter_UnderLimitAllows(t *testing.T) {
	wireResponse(t)
	ll := utils.NewLoginLimiter(newLimiterPolicy())
	engine := limiterEngine(t, ll)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/login", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestLimiter_OverLimitBlocks(t *testing.T) {
	wireResponse(t)
	ll := utils.NewLoginLimiter(newLimiterPolicy())

	// Drive the client IP past the ban threshold so CheckSecurityStatus
	// reports banned == true.
	ll.RecordFailedAttempt(testClientIP)
	ll.RecordFailedAttempt(testClientIP)
	banned, _ := ll.CheckSecurityStatus(testClientIP)
	require.True(t, banned, "precondition: IP should be banned after threshold")

	engine := limiterEngine(t, ll)
	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/login", "")
	engine.ServeHTTP(rec, req)

	// Limiter calls response.Fail with StatusLocked as the business code;
	// the HTTP envelope stays 200 but the body carries 423 and aborts.
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":423`)
	assert.Contains(t, rec.Body.String(), "Banned")
	assert.NotContains(t, rec.Body.String(), "pong")
}

func TestLimiter_BelowBanThresholdStillAllows(t *testing.T) {
	wireResponse(t)
	ll := utils.NewLoginLimiter(newLimiterPolicy())

	// One failed attempt is below BanThreshold (2): not banned yet.
	ll.RecordFailedAttempt(testClientIP)
	banned, _ := ll.CheckSecurityStatus(testClientIP)
	require.False(t, banned)

	engine := limiterEngine(t, ll)
	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/login", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}
