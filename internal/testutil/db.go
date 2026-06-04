// Package testutil provides shared helpers for the API server's tests:
// an in-memory database, fake dependency constructors, Gin request helpers,
// an embedded Redis, and model fixtures. It is imported only by *_test.go
// files and never by production code.
package testutil

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// migrateModels mirrors the production AutoMigrate list in cmd/apimain.go so
// tests run against the same schema the server creates at startup.
var migrateModels = []any{
	&model.Version{},
	&model.User{},
	&model.UserToken{},
	&model.Tag{},
	&model.AddressBook{},
	&model.Peer{},
	&model.Group{},
	&model.UserThird{},
	&model.Oauth{},
	&model.LoginLog{},
	&model.ShareRecord{},
	&model.AuditConn{},
	&model.AuditFile{},
	&model.AddressBookCollection{},
	&model.AddressBookCollectionRule{},
	&model.ServerCmd{},
	&model.DeviceGroup{},
	&model.PeerCommand{},
	&model.Strategy{},
	&model.StrategyPeer{},
	&model.StrategyUser{},
	&model.StrategyDeviceGroup{},
	&model.CustomClient{},
	&model.BuildArtifact{},
	&model.PreBuild{},
	&model.Worker{},
}

// NewMemDB returns a fresh in-memory SQLite database with every model
// migrated. Each call gets its own isolated database, so tests never share
// state. The GORM logger is silenced to keep test output readable.
func NewMemDB(t testing.TB) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("testutil: open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(migrateModels...); err != nil {
		t.Fatalf("testutil: auto-migrate: %v", err)
	}
	return db
}
