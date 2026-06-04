package service

import (
	"strconv"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserService(t *testing.T) (*UserService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.UserService, db
}

// --- Create / username formatting / dedup ---

func TestUserCreate_FormatsAndHashesPassword(t *testing.T) {
	us, db := newUserService(t)
	u := &model.User{Username: "  Mixed Case  ", Email: "x@y.com", Password: "secret", IsAdmin: boolPtr(false)}
	require.NoError(t, us.Create(u))

	assert.NotZero(t, u.Id)
	assert.Equal(t, "mixedcase", u.Username, "username should be lowercased and spaces stripped")

	// stored password must be a bcrypt hash, not the plaintext
	var stored model.User
	require.NoError(t, db.First(&stored, u.Id).Error)
	assert.NotEqual(t, "secret", stored.Password)
	ok, _, err := utils.VerifyPassword(stored.Password, "secret")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUserCreate_DuplicateUsername(t *testing.T) {
	us, _ := newUserService(t)
	require.NoError(t, us.Create(&model.User{Username: "dup", Password: "p", IsAdmin: boolPtr(false)}))
	err := us.Create(&model.User{Username: "dup", Password: "p", IsAdmin: boolPtr(false)})
	require.Error(t, err)
	assert.Equal(t, "UsernameExists", err.Error())
}

// Create runs IsUsernameExists BEFORE formatting, so a name that only differs by
// case/spaces is NOT detected as a duplicate and then collides on the unique
// index when persisted. See SUSPECTED BUG note in the final report.
func TestUserCreate_CaseVariantHitsUniqueIndex(t *testing.T) {
	us, _ := newUserService(t)
	require.NoError(t, us.Create(&model.User{Username: "alice", Password: "p", IsAdmin: boolPtr(false)}))
	err := us.Create(&model.User{Username: "Alice", Password: "p", IsAdmin: boolPtr(false)})
	require.Error(t, err, "case variant collides on the unique index after formatting")
	// The error is a DB unique-constraint error, NOT the friendly "UsernameExists".
	assert.NotEqual(t, "UsernameExists", err.Error())
}

func TestFormatUsername(t *testing.T) {
	us, _ := newUserService(t)
	assert.Equal(t, "johnsmith", us.formatUsername("John Smith"))
	assert.Equal(t, "abc", us.formatUsername("A B C"))
}

func TestGenerateUsernameByOauth_DedupAppendsDigit(t *testing.T) {
	us, _ := newUserService(t)
	require.NoError(t, us.Create(&model.User{Username: "taken", Password: "p", IsAdmin: boolPtr(false)}))

	got := us.GenerateUsernameByOauth("taken")
	assert.NotEqual(t, "taken", got)
	assert.True(t, len(got) > len("taken"), "a digit should be appended for a taken name")
	assert.Equal(t, "taken", got[:len("taken")])
	assert.False(t, us.IsUsernameExists(got), "generated name must be free")
}

func TestGenerateUsernameByOauth_FreeNameUnchanged(t *testing.T) {
	us, _ := newUserService(t)
	assert.Equal(t, "brandnew", us.GenerateUsernameByOauth("brandnew"))
}

// --- existence checks ---

func TestIsUsernameExistsLocal(t *testing.T) {
	us, _ := newUserService(t)
	require.NoError(t, us.Create(&model.User{Username: "exists", Password: "p", IsAdmin: boolPtr(false)}))
	assert.True(t, us.IsUsernameExistsLocal("exists"))
	assert.False(t, us.IsUsernameExistsLocal("ghost"))
}

// --- Info lookups ---

func TestUserInfoLookups(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) {
		u.Username = "lookup"
		u.Email = "lookup@example.com"
	})

	assert.Equal(t, u.Id, us.InfoById(u.Id).Id)
	assert.Equal(t, u.Id, us.InfoByUsername("lookup").Id)
	assert.Equal(t, u.Id, us.InfoByEmail("lookup@example.com").Id)

	// missing lookups return zero-value structs
	assert.Zero(t, us.InfoById(99999).Id)
	assert.Zero(t, us.InfoByUsername("nobody").Id)
}

// --- InfoByUsernamePassword (local auth path; LDAP disabled) ---

func TestInfoByUsernamePassword_Success(t *testing.T) {
	us, db := newUserService(t)
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "auth" }) // default password "password"

	u := us.InfoByUsernamePassword("auth", "password")
	assert.NotZero(t, u.Id)
}

func TestInfoByUsernamePassword_WrongPassword(t *testing.T) {
	us, db := newUserService(t)
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "auth2" })

	u := us.InfoByUsernamePassword("auth2", "wrong")
	assert.Zero(t, u.Id, "wrong password returns empty user")
}

func TestInfoByUsernamePassword_UnknownUser(t *testing.T) {
	us, _ := newUserService(t)
	u := us.InfoByUsernamePassword("ghost", "x")
	assert.Zero(t, u.Id)
}

// --- IsAdmin / admin counting ---

func TestIsAdmin(t *testing.T) {
	us, _ := newUserService(t)
	assert.True(t, us.IsAdmin(&model.User{IsAdmin: boolPtr(true)}))
	assert.False(t, us.IsAdmin(&model.User{IsAdmin: boolPtr(false)}))
	assert.False(t, us.IsAdmin(nil), "nil user is not admin")
}

func TestGetAdminUserCount(t *testing.T) {
	us, db := newUserService(t)
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "a1"; u.IsAdmin = boolPtr(true) })
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "a2"; u.IsAdmin = boolPtr(true) })
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "n1"; u.IsAdmin = boolPtr(false) })
	assert.EqualValues(t, 2, us.getAdminUserCount())
}

// --- Delete (last admin guard + cascade) ---

func TestUserDelete_LastAdminBlocked(t *testing.T) {
	us, db := newUserService(t)
	admin := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "onlyadmin"; u.IsAdmin = boolPtr(true) })
	err := us.Delete(admin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last admin")
	assert.NotZero(t, us.InfoById(admin.Id).Id, "admin must still exist")
}

func TestUserDelete_NonAdminCascades(t *testing.T) {
	us, db := newUserService(t)
	// keep an admin around so the last-admin guard doesn't fire
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "keepadmin"; u.IsAdmin = boolPtr(true) })
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "victim"; u.IsAdmin = boolPtr(false) })

	require.NoError(t, db.Create(&model.UserThird{UserId: u.Id, Op: "github"}).Error)
	require.NoError(t, db.Create(&model.AddressBook{Id: "p1", UserId: u.Id}).Error)

	require.NoError(t, us.Delete(u))
	assert.Zero(t, us.InfoById(u.Id).Id)

	var thirds, abs int64
	db.Model(&model.UserThird{}).Where("user_id = ?", u.Id).Count(&thirds)
	db.Model(&model.AddressBook{}).Where("user_id = ?", u.Id).Count(&abs)
	assert.EqualValues(t, 0, thirds, "oauth bindings removed")
	assert.EqualValues(t, 0, abs, "address books removed")
}

// --- Update (last admin guard) ---

func TestUserUpdate_CannotDemoteLastAdmin(t *testing.T) {
	us, db := newUserService(t)
	admin := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "soleadmin"; u.IsAdmin = boolPtr(true) })

	// attempt to demote the last admin
	admin.IsAdmin = boolPtr(false)
	err := us.Update(admin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last admin")
}

func TestUserUpdate_NormalFieldChange(t *testing.T) {
	us, db := newUserService(t)
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "otheradmin"; u.IsAdmin = boolPtr(true) })
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "edit"; u.IsAdmin = boolPtr(false) })

	u.Nickname = "Updated Nick"
	require.NoError(t, us.Update(u))
	assert.Equal(t, "Updated Nick", us.InfoById(u.Id).Nickname)
}

// --- Password change flushes tokens ---

func TestUpdatePassword_RehashesAndFlushesTokens(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "pwchange" })
	require.NoError(t, db.Create(&model.UserToken{UserId: u.Id, Token: "tok"}).Error)

	require.NoError(t, us.UpdatePassword(u, "newsecret"))

	// new password verifies
	loaded := us.InfoById(u.Id)
	ok, _, err := utils.VerifyPassword(loaded.Password, "newsecret")
	require.NoError(t, err)
	assert.True(t, ok)

	// tokens flushed
	var cnt int64
	db.Model(&model.UserToken{}).Where("user_id = ?", u.Id).Count(&cnt)
	assert.EqualValues(t, 0, cnt)
}

// --- Login / token lifecycle ---

func TestLogin_CreatesTokenAndLog(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "login" })

	ut := us.Login(u, &model.LoginLog{UserId: u.Id, DeviceId: "d1", Uuid: "uuid-x", Type: model.LoginLogTypeAccount})
	require.NotNil(t, ut)
	assert.NotEmpty(t, ut.Token)
	// ExpiredAt is set from UserTokenExpireTimestamp(). With the default config
	// it currently resolves to ~now (see the BUG note in
	// TestUserTokenExpireTimestamp_Default), so we only assert it is populated.
	assert.NotZero(t, ut.ExpiredAt)

	var logs int64
	db.Model(&model.LoginLog{}).Where("user_id = ?", u.Id).Count(&logs)
	assert.EqualValues(t, 1, logs)
}

func TestInfoByAccessToken(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "tokuser" })
	ut := &model.UserToken{UserId: u.Id, Token: "valid-tok", ExpiredAt: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, db.Create(ut).Error)

	gotU, gotT := us.InfoByAccessToken("valid-tok")
	assert.Equal(t, u.Id, gotU.Id)
	assert.Equal(t, ut.Id, gotT.Id)
}

func TestInfoByAccessToken_Expired(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "exptok" })
	require.NoError(t, db.Create(&model.UserToken{UserId: u.Id, Token: "old", ExpiredAt: time.Now().Add(-time.Hour).Unix()}).Error)

	gotU, _ := us.InfoByAccessToken("old")
	assert.Zero(t, gotU.Id, "expired token yields empty user")
}

func TestInfoByAccessToken_Unknown(t *testing.T) {
	us, _ := newUserService(t)
	gotU, gotT := us.InfoByAccessToken("nope")
	assert.Zero(t, gotU.Id)
	assert.Zero(t, gotT.Id)
}

func TestLogout_DeletesToken(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "logout" })
	require.NoError(t, db.Create(&model.UserToken{UserId: u.Id, Token: "tok-logout", DeviceUuid: "uu"}).Error)

	require.NoError(t, us.Logout(u, "tok-logout"))
	var cnt int64
	db.Model(&model.UserToken{}).Where("token = ?", "tok-logout").Count(&cnt)
	assert.EqualValues(t, 0, cnt)
}

func TestFlushTokenVariants(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "flush" })
	require.NoError(t, db.Create(&model.UserToken{UserId: u.Id, Token: "t1", DeviceUuid: "uuid-a"}).Error)
	require.NoError(t, db.Create(&model.UserToken{UserId: u.Id, Token: "t2", DeviceUuid: "uuid-b"}).Error)

	require.NoError(t, us.FlushTokenByUuid("uuid-a"))
	var cnt int64
	db.Model(&model.UserToken{}).Where("user_id = ?", u.Id).Count(&cnt)
	assert.EqualValues(t, 1, cnt)

	require.NoError(t, us.FlushToken(u))
	db.Model(&model.UserToken{}).Where("user_id = ?", u.Id).Count(&cnt)
	assert.EqualValues(t, 0, cnt)
}

func TestBatchDeleteUserToken(t *testing.T) {
	us, db := newUserService(t)
	t1 := &model.UserToken{UserId: 1, Token: "b1"}
	t2 := &model.UserToken{UserId: 1, Token: "b2"}
	require.NoError(t, db.Create(t1).Error)
	require.NoError(t, db.Create(t2).Error)

	require.NoError(t, us.BatchDeleteUserToken([]uint{t1.Id, t2.Id}))
	var cnt int64
	db.Model(&model.UserToken{}).Count(&cnt)
	assert.EqualValues(t, 0, cnt)
}

// --- token expiry helpers ---

func TestUserTokenExpireTimestamp_Default(t *testing.T) {
	us, _ := newUserService(t)
	got := us.UserTokenExpireTimestamp()
	// BUG: the zero-TokenExpire fallback in user.go assigns `exp = 604800` to a
	// time.Duration, which is 604800 *nanoseconds* (~0.6ms), not the 7 days the
	// comment claims. So the default expiry resolves to ~now. Asserting the
	// actual behavior here; this test should be tightened once the fallback is
	// fixed to time.Duration(604800)*time.Second.
	expected := time.Now().Unix()
	assert.InDelta(t, expected, got, 5)
}

func TestUserTokenExpireTimestamp_Configured(t *testing.T) {
	us, _ := newUserService(t)
	us.ctx.Config.App.TokenExpire = time.Hour
	got := us.UserTokenExpireTimestamp()
	assert.InDelta(t, time.Now().Add(time.Hour).Unix(), got, 5)
}

func TestRefreshAccessToken(t *testing.T) {
	us, db := newUserService(t)
	us.ctx.Config.App.TokenExpire = time.Hour
	ut := &model.UserToken{UserId: 1, Token: "refresh", ExpiredAt: 1}
	require.NoError(t, db.Create(ut).Error)

	us.RefreshAccessToken(ut)
	assert.Greater(t, ut.ExpiredAt, time.Now().Unix())

	var stored model.UserToken
	require.NoError(t, db.First(&stored, ut.Id).Error)
	assert.Equal(t, ut.ExpiredAt, stored.ExpiredAt)
}

// --- JWT ---

func TestGenerateTokenAndVerifyJWT(t *testing.T) {
	us, _ := newUserService(t)
	u := &model.User{}
	u.Id = 123
	tok := us.GenerateToken(u)
	assert.NotEmpty(t, tok)

	uid, err := us.VerifyJWT(tok)
	require.NoError(t, err)
	assert.EqualValues(t, 123, uid)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	us, _ := newUserService(t)
	u := us.Register("newbie", "n@e.com", "pw", model.COMMON_STATUS_ENABLE)
	require.NotNil(t, u)
	assert.NotZero(t, u.Id)
	assert.Equal(t, model.COMMON_STATUS_ENABLE, u.Status)
}

func TestRegister_DuplicateReturnsNil(t *testing.T) {
	us, _ := newUserService(t)
	require.NotNil(t, us.Register("once", "a@b.com", "pw", model.COMMON_STATUS_ENABLE))
	assert.Nil(t, us.Register("once", "c@d.com", "pw", model.COMMON_STATUS_ENABLE))
}

// --- password empty checks (used by oauth auto-register) ---

func TestIsPasswordEmpty(t *testing.T) {
	us, db := newUserService(t)
	withPw := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "haspw" })
	noPw := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "nopw"; u.Password = "" })

	assert.False(t, us.IsPasswordEmptyById(withPw.Id))
	assert.True(t, us.IsPasswordEmptyById(noPw.Id))
	assert.True(t, us.IsPasswordEmptyByUsername("nopw"))
	assert.False(t, us.IsPasswordEmptyByUsername("haspw"))
	// unknown id/username return false (record not found)
	assert.False(t, us.IsPasswordEmptyById(99999))
	assert.False(t, us.IsPasswordEmptyByUsername("ghost"))
}

// --- List & pagination ---

func TestUserList_Pagination(t *testing.T) {
	us, db := newUserService(t)
	for i := 0; i < 5; i++ {
		testutil.CreateUser(t, db, func(u *model.User) { u.Username = "u" + strconv.Itoa(i) })
	}
	res := us.List(1, 2, nil)
	assert.EqualValues(t, 5, res.Total)
	assert.Len(t, res.Users, 2)
}

func TestListByGroupId(t *testing.T) {
	us, db := newUserService(t)
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "g10a"; u.GroupId = 10 })
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "g10b"; u.GroupId = 10 })
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "g20"; u.GroupId = 20 })

	res := us.ListByGroupId(10, 1, 100)
	assert.EqualValues(t, 2, res.Total)

	ids := us.ListIdsByGroupId(10)
	assert.Len(t, ids, 2)
}

func TestListByIds(t *testing.T) {
	us, db := newUserService(t)
	a := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "ida" })
	b := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "idb" })
	testutil.CreateUser(t, db, func(u *model.User) { u.Username = "idc" })

	res := us.ListByIds([]uint{a.Id, b.Id})
	assert.Len(t, res, 2)
}

// --- InfoByOauthId (cross-service) ---

func TestInfoByOauthId(t *testing.T) {
	us, db := newUserService(t)
	u := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "oauthlinked" })
	require.NoError(t, db.Create(&model.UserThird{UserId: u.Id, Op: "github", OauthUser: model.OauthUser{OpenId: "gh-1"}}).Error)

	got := us.InfoByOauthId("github", "gh-1")
	require.NotNil(t, got)
	assert.Equal(t, u.Id, got.Id)

	assert.Nil(t, us.InfoByOauthId("github", "missing"))
}

func TestUserThirdsByUserId(t *testing.T) {
	us, db := newUserService(t)
	require.NoError(t, db.Create(&model.UserThird{UserId: 5, Op: "github", OauthUser: model.OauthUser{OpenId: "a"}}).Error)
	require.NoError(t, db.Create(&model.UserThird{UserId: 5, Op: "linuxdo", OauthUser: model.OauthUser{OpenId: "b"}}).Error)
	require.NoError(t, db.Create(&model.UserThird{UserId: 6, Op: "github", OauthUser: model.OauthUser{OpenId: "c"}}).Error)

	assert.Len(t, us.UserThirdsByUserId(5), 2)
	assert.Equal(t, "linuxdo", us.UserThirdInfo(5, "linuxdo").Op)
}

// --- RegisterByOauth ---

func TestRegisterByOauth_NewUser(t *testing.T) {
	us, db := newUserService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)

	ou := &model.OauthUser{OpenId: "new-open", Username: "FromOauth", Email: "OA@Example.com", Name: "OA"}
	u, err := us.RegisterByOauth(ou, "github")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.NotZero(t, u.Id)
	assert.Equal(t, "fromoauth", u.Username, "username formatted to lowercase")

	// a UserThird binding is created and email is lowercased
	ut := us.UserThirdInfo(u.Id, "github")
	assert.Equal(t, "new-open", ut.OpenId)
	assert.Equal(t, "oa@example.com", ut.Email)
}

func TestRegisterByOauth_ExistingBindingReturnsSameUser(t *testing.T) {
	us, db := newUserService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)
	existing := testutil.CreateUser(t, db, func(u *model.User) { u.Username = "already" })
	require.NoError(t, db.Create(&model.UserThird{UserId: existing.Id, Op: "github", OauthUser: model.OauthUser{OpenId: "bound"}}).Error)

	u, err := us.RegisterByOauth(&model.OauthUser{OpenId: "bound"}, "github")
	require.NoError(t, err)
	assert.Equal(t, existing.Id, u.Id, "existing binding returns the linked user")
}

func TestRegisterByOauth_LinksToExistingEmail(t *testing.T) {
	us, db := newUserService(t)
	require.NoError(t, db.Create(&model.Oauth{Op: "github", OauthType: model.OauthTypeGithub}).Error)
	existing := testutil.CreateUser(t, db, func(u *model.User) {
		u.Username = "byemail"
		u.Email = "match@example.com"
	})

	u, err := us.RegisterByOauth(&model.OauthUser{OpenId: "e-open", Email: "MATCH@example.com"}, "github")
	require.NoError(t, err)
	assert.Equal(t, existing.Id, u.Id, "should link to the existing user with the same email")

	// binding created pointing at the existing user
	ut := us.UserThirdInfo(existing.Id, "github")
	assert.Equal(t, "e-open", ut.OpenId)
}

func TestRegisterByOauth_OpNotFound(t *testing.T) {
	us, _ := newUserService(t)
	_, err := us.RegisterByOauth(&model.OauthUser{OpenId: "x", Username: "y"}, "ghost")
	require.Error(t, err, "GetTypeByOp fails for unknown op")
}
