// Package servicekit assembles a fully-wired *service.Service backed by an
// in-memory database for HTTP-layer tests. It lives in its own package because
// it imports internal/service; keeping it out of internal/testutil lets the
// service package's own (internal) tests use testutil without an import cycle.
//
// Import this only from controller/middleware test packages, never from the
// service package's tests.
package servicekit

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/service"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"gorm.io/gorm"
)

// Kit bundles the dependencies an HTTP test typically needs.
type Kit struct {
	Services *service.Service
	DB       *gorm.DB
	Config   *config.Config
}

// New builds a Service aggregate over a fresh in-memory database, using the
// fake logger/jwt/lock from testutil and no S3 client. Mutate the returned
// Config before exercising config-dependent routes.
func New(t testing.TB) *Kit {
	t.Helper()
	db := testutil.NewMemDB(t)
	cfg := testutil.NewConfig()
	svcs := service.New(cfg, db, testutil.NewLogger(t), testutil.NewJwt(t), testutil.NewLock(), nil)
	return &Kit{Services: svcs, DB: db, Config: cfg}
}
