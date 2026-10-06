package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// okHandler writes a 200 "pong" body, used to confirm the request reached the
// final handler (i.e. the middleware called Next instead of aborting).
func okHandler(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func TestCors_SetsHeadersAndCallsNext(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Cors())
	engine.GET("/ping", okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/ping", "")
	req.Header.Set("Origin", "https://example.com")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "api-token,content-type,authorization ", rec.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, http.MethodGet, rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "pong", rec.Body.String())
}

func TestCors_OptionsPreflightShortCircuits(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Cors())

	handlerCalled := false
	engine.OPTIONS("/ping", func(c *gin.Context) { handlerCalled = true })

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodOptions, "/ping", "")
	req.Header.Set("Origin", "https://example.com")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, handlerCalled, "OPTIONS preflight must short-circuit before the handler")
	// CORS headers still set on the preflight response.
	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.MethodOptions, rec.Header().Get("Access-Control-Allow-Methods"))
}

func TestCors_EmptyOriginEchoesEmpty(t *testing.T) {
	engine := testutil.NewEngine(t)
	engine.Use(middleware.Cors())
	engine.GET("/ping", okHandler)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/ping", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}
