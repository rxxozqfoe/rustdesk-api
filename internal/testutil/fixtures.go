package testutil

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"gorm.io/gorm"
)

// boolPtr returns a pointer to b, for the *bool model fields.
func boolPtr(b bool) *bool { return &b }

// CreateUser inserts a user with sane defaults and returns it. Pass mutators
// to override fields before insertion, e.g. CreateUser(t, db, func(u *model.User){ u.Username = "bob" }).
// The default password is "password" (bcrypt-hashed).
func CreateUser(t testing.TB, db *gorm.DB, mutators ...func(*model.User)) *model.User {
	t.Helper()
	hash, err := utils.EncryptPassword("password")
	if err != nil {
		t.Fatalf("testutil: hash password: %v", err)
	}
	u := &model.User{
		Username: "tester",
		Email:    "tester@example.com",
		Password: hash,
		Nickname: "Tester",
		GroupId:  1,
		Status:   model.COMMON_STATUS_ENABLE,
		IsAdmin:  boolPtr(false),
	}
	for _, m := range mutators {
		m(u)
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("testutil: create user: %v", err)
	}
	return u
}

// CreateGroup inserts a group with the given name and returns it.
func CreateGroup(t testing.TB, db *gorm.DB, mutators ...func(*model.Group)) *model.Group {
	t.Helper()
	g := &model.Group{Name: "group", Type: model.GroupTypeDefault}
	for _, m := range mutators {
		m(g)
	}
	if err := db.Create(g).Error; err != nil {
		t.Fatalf("testutil: create group: %v", err)
	}
	return g
}

// CreatePeer inserts a peer owned by userID and returns it.
func CreatePeer(t testing.TB, db *gorm.DB, mutators ...func(*model.Peer)) *model.Peer {
	t.Helper()
	p := &model.Peer{
		Id:       "123456789",
		Uuid:     "uuid-123",
		Hostname: "host",
		Os:       "linux",
	}
	for _, m := range mutators {
		m(p)
	}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("testutil: create peer: %v", err)
	}
	return p
}

// CreateAddressBook inserts an address-book entry and returns it.
func CreateAddressBook(t testing.TB, db *gorm.DB, mutators ...func(*model.AddressBook)) *model.AddressBook {
	t.Helper()
	ab := &model.AddressBook{
		Id:       "123456789",
		Username: "peeruser",
		Hostname: "peerhost",
	}
	for _, m := range mutators {
		m(ab)
	}
	if err := db.Create(ab).Error; err != nil {
		t.Fatalf("testutil: create address book: %v", err)
	}
	return ab
}
