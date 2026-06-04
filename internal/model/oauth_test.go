package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOauthType(t *testing.T) {
	valid := []string{
		OauthTypeGithub,
		OauthTypeGoogle,
		OauthTypeOidc,
		OauthTypeWebauth,
		OauthTypeLinuxdo,
	}
	for _, ot := range valid {
		t.Run("valid/"+ot, func(t *testing.T) {
			assert.NoError(t, ValidateOauthType(ot))
		})
	}

	invalid := []string{"", "GitHub", "facebook", "OIDC", " github"}
	for _, ot := range invalid {
		t.Run("invalid/"+ot, func(t *testing.T) {
			err := ValidateOauthType(ot)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid Oauth type")
		})
	}
}

func TestOauthUser_ToUser(t *testing.T) {
	ou := &OauthUser{
		Username: "newname",
		Email:    "a@b.com",
		Name:     "Alice",
		Picture:  "http://pic",
	}

	t.Run("override username", func(t *testing.T) {
		u := &User{Username: "original"}
		ou.ToUser(u, true)
		assert.Equal(t, "newname", u.Username)
		assert.Equal(t, "a@b.com", u.Email)
		assert.Equal(t, "Alice", u.Nickname)
		assert.Equal(t, "http://pic", u.Avatar)
	})

	t.Run("keep existing username", func(t *testing.T) {
		u := &User{Username: "original"}
		ou.ToUser(u, false)
		assert.Equal(t, "original", u.Username, "username must be preserved when override=false")
		assert.Equal(t, "a@b.com", u.Email)
		assert.Equal(t, "Alice", u.Nickname)
		assert.Equal(t, "http://pic", u.Avatar)
	})
}

func TestOidcUser_ToOauthUser(t *testing.T) {
	t.Run("uses preferred username when present", func(t *testing.T) {
		oidc := &OidcUser{
			OauthUserBase:     OauthUserBase{Name: "Bob", Email: "Bob@Example.COM"},
			Sub:               "sub-1",
			PreferredUsername: "bobby",
			VerifiedEmail:     true,
			Picture:           "pic",
		}
		ou := oidc.ToOauthUser()
		assert.Equal(t, "sub-1", ou.OpenId)
		assert.Equal(t, "bobby", ou.Username, "preferred username is used verbatim")
		assert.Equal(t, "Bob", ou.Name)
		assert.Equal(t, "Bob@Example.COM", ou.Email)
		assert.True(t, ou.VerifiedEmail)
		assert.Equal(t, "pic", ou.Picture)
	})

	t.Run("falls back to lowercased email when no preferred username", func(t *testing.T) {
		oidc := &OidcUser{
			OauthUserBase: OauthUserBase{Name: "Carol", Email: "Carol@Example.COM"},
			Sub:           "sub-2",
		}
		ou := oidc.ToOauthUser()
		assert.Equal(t, "carol@example.com", ou.Username,
			"fallback username is the full lowercased email, not just the local part")
	})
}

func TestGithubUser_ToOauthUser(t *testing.T) {
	gu := &GithubUser{
		OauthUserBase: OauthUserBase{Name: "Dave", Email: "d@e.com"},
		Id:            12345,
		Login:         "DaveLogin",
		AvatarUrl:     "http://avatar",
		VerifiedEmail: true,
	}
	ou := gu.ToOauthUser()
	assert.Equal(t, "12345", ou.OpenId, "OpenId is the stringified numeric id")
	assert.Equal(t, "davelogin", ou.Username, "login is lowercased")
	assert.Equal(t, "Dave", ou.Name)
	assert.Equal(t, "d@e.com", ou.Email)
	assert.True(t, ou.VerifiedEmail)
	assert.Equal(t, "http://avatar", ou.Picture)
}

func TestLinuxdoUser_ToOauthUser(t *testing.T) {
	lu := &LinuxdoUser{
		OauthUserBase: OauthUserBase{Name: "Eve", Email: "e@f.com"},
		Id:            777,
		Username:      "EveUser",
		Avatar:        "http://av",
	}
	ou := lu.ToOauthUser()
	assert.Equal(t, "777", ou.OpenId)
	assert.Equal(t, "eveuser", ou.Username, "username is lowercased")
	assert.Equal(t, "Eve", ou.Name)
	assert.Equal(t, "e@f.com", ou.Email)
	assert.True(t, ou.VerifiedEmail, "linux.do emails are treated as verified")
	assert.Equal(t, "http://av", ou.Picture)
}
