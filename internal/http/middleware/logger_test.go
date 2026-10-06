package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	return logger.New(&logger.Config{Level: "error"})
}

func TestLogger_CallsNextAndPreservesStatus(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Logger(newTestLogger(t), config.Logger{LogHeartbeat: true}))

	called := false
	engine.POST("/echo", func(c *gin.Context) {
		called = true
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/echo?x=1", `{"a":1}`)
	require.NotPanics(t, func() { engine.ServeHTTP(rec, req) })

	assert.True(t, called, "Logger must call the downstream handler")
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"ok":true`)
}

func TestLogger_4xxAnd5xxDoNotPanic(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Logger(newTestLogger(t), config.Logger{LogHeartbeat: true}))
	engine.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
	engine.GET("/boom", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/bad", http.StatusBadRequest},
		{"/boom", http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		req := testutil.JSONRequest(t, http.MethodGet, tc.path, "")
		require.NotPanics(t, func() { engine.ServeHTTP(rec, req) })
		assert.Equal(t, tc.want, rec.Code)
	}
}

func TestLogger_HeartbeatSkippedWhenDisabled(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Logger(newTestLogger(t), config.Logger{LogHeartbeat: false}))

	called := false
	engine.GET("/api/heartbeat", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/heartbeat", "")
	require.NotPanics(t, func() { engine.ServeHTTP(rec, req) })

	// Even when logging is skipped, the request must still reach the handler.
	assert.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestLogger_DownloadPathSkippedButHandlerRuns(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Logger(newTestLogger(t), config.Logger{LogHeartbeat: true}))

	called := false
	engine.GET("/api/custom-client/download/abc", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "binary")
	})

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/custom-client/download/abc", "")
	require.NotPanics(t, func() { engine.ServeHTTP(rec, req) })

	assert.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "binary", rec.Body.String())
}

func TestLogger_CapturesResponseBodyWithoutCorruptingIt(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Logger(newTestLogger(t), config.Logger{LogHeartbeat: true}))
	engine.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "hello-body")
	})

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/data", "")
	engine.ServeHTTP(rec, req)

	// The wrapped writer must forward the body untouched to the client.
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello-body", rec.Body.String())
}
