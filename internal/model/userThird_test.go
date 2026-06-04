package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserThird_FromOauthUser(t *testing.T) {
	ou := &OauthUser{
		OpenId:   "open-1",
		Name:     "Frank",
		Username: "frank",
		Email:    "Frank@Example.COM",
		Picture:  "pic",
	}
	var ut UserThird
	ut.FromOauthUser(42, ou, OauthTypeGithub, "op-x")

	assert.Equal(t, uint(42), ut.UserId)
	assert.Equal(t, OauthTypeGithub, ut.OauthType)
	assert.Equal(t, "op-x", ut.Op)
	// Embedded OauthUser is copied wholesale...
	assert.Equal(t, "open-1", ut.OpenId)
	assert.Equal(t, "Frank", ut.Name)
	assert.Equal(t, "frank", ut.Username)
	// ...but Email is normalized to lower case.
	assert.Equal(t, "frank@example.com", ut.Email)
}
