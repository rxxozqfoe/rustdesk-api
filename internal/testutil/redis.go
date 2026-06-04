package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// NewMiniRedis starts an embedded in-process Redis and returns it together
// with the client options pointing at it. The server is closed automatically
// when the test ends, so cache/Redis tests run in CI without an external
// dependency.
func NewMiniRedis(t testing.TB) (*miniredis.Miniredis, *redis.Options) {
	t.Helper()
	mr := miniredis.RunT(t)
	return mr, &redis.Options{Addr: mr.Addr()}
}
