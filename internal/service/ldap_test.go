package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLdapService(t *testing.T) *LdapService {
	t.Helper()
	svc, _ := newServiceAggregate(t)
	return svc.LdapService
}

// --- LdapUser.Name / ToUser ---

func TestLdapUser_Name(t *testing.T) {
	lu := &LdapUser{FirstName: "Jane", LastName: "Doe"}
	assert.Equal(t, "Jane Doe", lu.Name())
}

func TestLdapUser_ToUser_Enabled(t *testing.T) {
	lu := &LdapUser{Username: "jdoe", Email: "j@x.com", FirstName: "J", LastName: "D", Enabled: true}
	u := lu.ToUser(nil)
	require.NotNil(t, u)
	assert.Equal(t, "jdoe", u.Username)
	assert.Equal(t, "j@x.com", u.Email)
	assert.Equal(t, "J D", u.Nickname)
	assert.Equal(t, model.COMMON_STATUS_ENABLE, u.Status)
}

func TestLdapUser_ToUser_DisabledMergesIntoExisting(t *testing.T) {
	lu := &LdapUser{Username: "u", Enabled: false}
	existing := &model.User{}
	existing.Id = 7
	out := lu.ToUser(existing)
	assert.Equal(t, existing, out, "merges into the passed-in user")
	assert.Equal(t, model.COMMON_STATUS_DISABLED, out.Status)
}

// --- field default resolution ---

func TestLdapFieldDefaults(t *testing.T) {
	ls := newLdapService(t)
	empty := &config.Ldap{}
	assert.Equal(t, "uid", ls.fieldUsername(empty))
	assert.Equal(t, "mail", ls.fieldEmail(empty))
	assert.Equal(t, "givenName", ls.fieldFirstName(empty))
	assert.Equal(t, "sn", ls.fieldLastName(empty))
	assert.Equal(t, "memberOf", ls.fieldMemberOf())
	assert.Equal(t, "userAccountControl", ls.fieldUserEnableAttr(empty))
}

func TestLdapFieldOverrides(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{}
	cfg.User.Username = "sAMAccountName"
	cfg.User.Email = "userPrincipalName"
	cfg.User.FirstName = "gn"
	cfg.User.LastName = "surname"
	cfg.User.EnableAttr = "myEnable"
	assert.Equal(t, "sAMAccountName", ls.fieldUsername(cfg))
	assert.Equal(t, "userPrincipalName", ls.fieldEmail(cfg))
	assert.Equal(t, "gn", ls.fieldFirstName(cfg))
	assert.Equal(t, "surname", ls.fieldLastName(cfg))
	assert.Equal(t, "myEnable", ls.fieldUserEnableAttr(cfg))
}

func TestBaseDnUser_FallbackAndOverride(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{BaseDn: "dc=base"}
	assert.Equal(t, "dc=base", ls.baseDnUser(cfg), "falls back to global base dn")
	cfg.User.BaseDn = "ou=users,dc=base"
	assert.Equal(t, "ou=users,dc=base", ls.baseDnUser(cfg))
}

func TestFilterField(t *testing.T) {
	ls := newLdapService(t)
	assert.Equal(t, "(uid=jdoe)", ls.filterField("uid", "jdoe"))
}

// --- isUserEnabled ---

func TestIsUserEnabled_NoAttrConfigured(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{} // EnableAttr + value empty => everyone enabled
	lu := &LdapUser{}
	assert.True(t, ls.isUserEnabled(cfg, lu))
	assert.True(t, lu.Enabled)
}

func TestIsUserEnabled_ActiveDirectory(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{}
	cfg.User.EnableAttr = "userAccountControl"
	cfg.User.EnableAttrValue = "512" // value present so the AD branch is taken

	// 512 = normal account (ACCOUNTDISABLE bit clear) -> enabled
	enabled := &LdapUser{EnableAttrValue: "512"}
	assert.True(t, ls.isUserEnabled(cfg, enabled))

	// 514 = 512 | 0x2 (ACCOUNTDISABLE) -> disabled
	disabled := &LdapUser{EnableAttrValue: "514"}
	assert.False(t, ls.isUserEnabled(cfg, disabled))

	// non-numeric value -> parse error -> disabled
	bad := &LdapUser{EnableAttrValue: "notanumber"}
	assert.False(t, ls.isUserEnabled(cfg, bad))
}

func TestIsUserEnabled_DirectComparison(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{}
	cfg.User.EnableAttr = "accountStatus"
	cfg.User.EnableAttrValue = "active"

	assert.True(t, ls.isUserEnabled(cfg, &LdapUser{EnableAttrValue: "active"}))
	assert.False(t, ls.isUserEnabled(cfg, &LdapUser{EnableAttrValue: "locked"}))
}

// --- isUserAdmin (memberOf branch; reverse-search branch needs a live server) ---

func TestIsUserAdmin_NoAdminGroupConfigured(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{}
	assert.False(t, ls.isUserAdmin(cfg, &LdapUser{}), "no admin group => never admin")
}

func TestIsUserAdmin_MemberOfMatch(t *testing.T) {
	ls := newLdapService(t)
	cfg := &config.Ldap{}
	cfg.User.AdminGroup = "cn=admins,dc=x"

	admin := &LdapUser{MemberOf: []string{"cn=users,dc=x", "CN=Admins,DC=X"}} // case-insensitive
	assert.True(t, ls.isUserAdmin(cfg, admin))

	nonAdmin := &LdapUser{MemberOf: []string{"cn=users,dc=x"}}
	assert.False(t, ls.isUserAdmin(cfg, nonAdmin))
}

// --- disabled-LDAP guards: should not attempt any network call ---

func TestLdapDisabled_ExistenceChecks(t *testing.T) {
	ls := newLdapService(t)
	require.False(t, ls.ctx.Config.Ldap.Enable, "LDAP disabled by default in test config")
	assert.False(t, ls.IsUsernameExists("anyone"))
	assert.False(t, ls.IsEmailExists("a@b.com"))
}

func TestLdapDisabled_GetUserInfoReturnsNotEnabled(t *testing.T) {
	ls := newLdapService(t)
	_, err := ls.GetUserInfoByUsernameLdap("x")
	assert.ErrorIs(t, err, ErrLdapNotEnabled)
	_, err = ls.GetUserInfoByEmailLdap("x@y.com")
	assert.ErrorIs(t, err, ErrLdapNotEnabled)
}

func TestLdapDisabled_GetUserInfoLocalReturnsEmptyUser(t *testing.T) {
	ls := newLdapService(t)
	u, err := ls.GetUserInfoByUsernameLocal("x")
	assert.ErrorIs(t, err, ErrLdapNotEnabled)
	assert.Zero(t, u.Id)
}

// Note: connectAndBind, verifyCredentials, Authenticate, search*, and the
// reverse-search branch of isUserAdmin/isUserInGroup all require a live LDAP
// server and are therefore not unit-tested here.
