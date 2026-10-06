package testutil

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
)

// NewLogger returns a logger that writes to stdout at error level, quiet
// enough for tests but still surfacing real problems.
func NewLogger(t testing.TB) *logger.Logger {
	t.Helper()
	return logger.New(&logger.Config{Level: "error"})
}

// NewJwt returns a Jwt signer with a fixed test key and a one-hour expiry.
func NewJwt(t testing.TB) *jwt.Jwt {
	t.Helper()
	return jwt.NewJwt("test-secret-key", time.Hour)
}

// NewLock returns an in-process lock, the same implementation used when no
// distributed lock is configured.
func NewLock() lock.Locker {
	return lock.NewLocal()
}

// NewConfig returns a minimal Config populated with the fields services read
// during tests. Callers may mutate the returned value to exercise specific
// branches (LDAP, OSS, Rustdesk API server, etc.).
func NewConfig() *config.Config {
	c := &config.Config{Lang: "en"}
	c.Rustdesk.ApiServer = "http://127.0.0.1:21114"
	c.Jwt.Key = "test-secret-key"
	return c
}
