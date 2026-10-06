package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerAuth_ValidTokenPasses(t *testing.T) {
	wireResponse(t)
	engine := testutil.NewEngine(t)
	engine.GET("/worker/ping", middleware.WorkerAuth("secret-worker-token"), okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/worker/ping", "")
	req.Header.Set("Authorization", "Bearer secret-worker-token")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestWorkerAuth_NotConfiguredRejected(t *testing.T) {
	wireResponse(t)
	engine := testutil.NewEngine(t)
	engine.GET("/worker/ping", middleware.WorkerAuth(""), okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/worker/ping", "")
	req.Header.Set("Authorization", "Bearer anything")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":403`)
	assert.Contains(t, rec.Body.String(), "worker API is not configured")
	assert.NotContains(t, rec.Body.String(), "pong")
}

func TestWorkerAuth_WrongTokenRejected(t *testing.T) {
	wireResponse(t)
	engine := testutil.NewEngine(t)
	engine.GET("/worker/ping", middleware.WorkerAuth("secret-worker-token"), okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/worker/ping", "")
	req.Header.Set("Authorization", "Bearer wrong-token")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":401`)
	assert.Contains(t, rec.Body.String(), "invalid worker token")
}

func TestWorkerAuth_MissingHeaderRejected(t *testing.T) {
	wireResponse(t)
	engine := testutil.NewEngine(t)
	engine.GET("/worker/ping", middleware.WorkerAuth("secret-worker-token"), okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/worker/ping", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":401`)
	assert.Contains(t, rec.Body.String(), "invalid worker token")
}

func TestWorkerAuth_BareTokenWithoutBearerRejected(t *testing.T) {
	wireResponse(t)
	engine := testutil.NewEngine(t)
	engine.GET("/worker/ping", middleware.WorkerAuth("secret-worker-token"), okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/worker/ping", "")
	// Missing "Bearer " prefix must not match the expected value.
	req.Header.Set("Authorization", "secret-worker-token")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":401`)
}
