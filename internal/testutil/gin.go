package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Keep Gin quiet and in test mode for the whole test binary.
	gin.SetMode(gin.TestMode)
}

// NewContext returns a Gin context wired to a fresh ResponseRecorder, for
// unit-testing a single handler or middleware in isolation.
func NewContext(t testing.TB) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}

// NewEngine returns a fresh Gin engine in test mode for end-to-end route
// testing via httptest.
func NewEngine(t testing.TB) *gin.Engine {
	t.Helper()
	return gin.New()
}

// JSONRequest builds an *http.Request with a JSON body and content-type set.
// body may be empty for GET/DELETE-style calls.
func JSONRequest(t testing.TB, method, target, body string) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}
