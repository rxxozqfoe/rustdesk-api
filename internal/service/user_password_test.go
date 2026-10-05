package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newPasswordTestService returns a UserService on an in-memory database with
// admin.password set to managed ("" = unmanaged).
func newPasswordTestService(t *testing.T, managed string) (*UserService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserToken{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	cfg := &config.Config{}
	cfg.Admin.Password = managed
	return &UserService{ctx: &ServiceContext{DB: db, Config: cfg}}, db
}

// createPasswordTestUser stores an admin with the given id and password, plus
// one login token.
func createPasswordTestUser(t *testing.T, db *gorm.DB, id uint, password string) {
	t.Helper()
	hash, err := utils.EncryptPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	isAdmin := true
	u := &model.User{IdModel: model.IdModel{Id: id}, Username: fmt.Sprintf("user%d", id), Password: hash, IsAdmin: &isAdmin}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.UserToken{UserId: id, Token: fmt.Sprintf("token%d", id)}).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
}

func assertPassword(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()
	u := &model.User{}
	db.First(u, id)
	if ok, _, _ := utils.VerifyPassword(u.Password, want); !ok {
		t.Errorf("user %d: password is not %q", id, want)
	}
}

func tokenCount(db *gorm.DB, id uint) int64 {
	var n int64
	db.Model(&model.UserToken{}).Where("user_id = ?", id).Count(&n)
	return n
}

func TestSyncManagedAdminPassword_Unset(t *testing.T) {
	us, db := newPasswordTestService(t, "")
	createPasswordTestUser(t, db, ManagedAdminId, "oldpass")

	changed, err := us.SyncManagedAdminPassword()
	if err != nil || changed {
		t.Fatalf("got changed=%v err=%v, want false, nil", changed, err)
	}
	assertPassword(t, db, ManagedAdminId, "oldpass")
}

func TestSyncManagedAdminPassword_AppliesAndLogsOut(t *testing.T) {
	us, db := newPasswordTestService(t, "fromsecret")
	createPasswordTestUser(t, db, ManagedAdminId, "oldpass")

	changed, err := us.SyncManagedAdminPassword()
	if err != nil || !changed {
		t.Fatalf("got changed=%v err=%v, want true, nil", changed, err)
	}
	assertPassword(t, db, ManagedAdminId, "fromsecret")
	if n := tokenCount(db, ManagedAdminId); n != 0 {
		t.Errorf("admin still has %d tokens after the password changed", n)
	}
}

func TestSyncManagedAdminPassword_AlreadyMatchesKeepsSessions(t *testing.T) {
	us, db := newPasswordTestService(t, "fromsecret")
	createPasswordTestUser(t, db, ManagedAdminId, "fromsecret")

	changed, err := us.SyncManagedAdminPassword()
	if err != nil || changed {
		t.Fatalf("got changed=%v err=%v, want false, nil", changed, err)
	}
	if n := tokenCount(db, ManagedAdminId); n != 1 {
		t.Errorf("admin has %d tokens, want the 1 it had", n)
	}
}

func TestSyncManagedAdminPassword_AdminMissing(t *testing.T) {
	us, db := newPasswordTestService(t, "fromsecret")
	createPasswordTestUser(t, db, 2, "otherpass")

	_, err := us.SyncManagedAdminPassword()
	if !errors.Is(err, ErrManagedAdminNotFound) {
		t.Fatalf("got err=%v, want ErrManagedAdminNotFound", err)
	}
	assertPassword(t, db, 2, "otherpass")
}

func TestUpdatePassword_RejectsManagedAdmin(t *testing.T) {
	us, db := newPasswordTestService(t, "fromsecret")
	createPasswordTestUser(t, db, ManagedAdminId, "fromsecret")
	createPasswordTestUser(t, db, 2, "otherpass")

	if err := us.UpdatePassword(us.InfoById(ManagedAdminId), "changed"); !errors.Is(err, ErrAdminPasswordManaged) {
		t.Fatalf("managed admin: got err=%v, want ErrAdminPasswordManaged", err)
	}
	assertPassword(t, db, ManagedAdminId, "fromsecret")

	if err := us.UpdatePassword(us.InfoById(2), "changed"); err != nil {
		t.Fatalf("other user: %v", err)
	}
	assertPassword(t, db, 2, "changed")
}

func TestUpdatePassword_UnmanagedAdmin(t *testing.T) {
	us, db := newPasswordTestService(t, "")
	createPasswordTestUser(t, db, ManagedAdminId, "oldpass")

	if err := us.UpdatePassword(us.InfoById(ManagedAdminId), "changed"); err != nil {
		t.Fatalf("got err=%v, want nil", err)
	}
	assertPassword(t, db, ManagedAdminId, "changed")
}

func TestIsPasswordManaged(t *testing.T) {
	managed, _ := newPasswordTestService(t, "fromsecret")
	unmanaged, _ := newPasswordTestService(t, "")
	admin := &model.User{IdModel: model.IdModel{Id: ManagedAdminId}}
	other := &model.User{IdModel: model.IdModel{Id: 2}}

	if !managed.IsPasswordManaged(admin) {
		t.Error("admin should be managed when admin.password is set")
	}
	if managed.IsPasswordManaged(other) {
		t.Error("only the admin account is managed")
	}
	if unmanaged.IsPasswordManaged(admin) {
		t.Error("nothing is managed when admin.password is unset")
	}
	if managed.IsPasswordManaged(nil) {
		t.Error("nil user is not managed")
	}
}
