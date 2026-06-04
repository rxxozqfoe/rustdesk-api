package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newCustomClientService(t *testing.T) (*CustomClientService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.CustomClientService, db
}

func TestCustomClient_CreateRequiresPrebuilt(t *testing.T) {
	s, _ := newCustomClientService(t)
	// no BuildArtifact -> Create should fail with a helpful error
	err := s.Create(&model.CustomClient{Platform: "linux", Arch: "x86_64", Version: "1.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pre-built binary")
}

func TestCustomClient_CreateSucceedsWithPrebuilt(t *testing.T) {
	s, db := newCustomClientService(t)
	require.NoError(t, db.Create(&model.BuildArtifact{Platform: "linux", Arch: "x86_64", Version: "1.0"}).Error)

	c := &model.CustomClient{Platform: "linux", Arch: "x86_64", Version: "1.0"}
	require.NoError(t, s.Create(c))
	assert.NotZero(t, c.Id)
	assert.Equal(t, model.BundleStatusBundling, c.Status, "Create sets status=bundling")

	assert.Equal(t, c.Id, s.InfoById(c.Id).Id)
	assert.EqualValues(t, 1, s.List(1, 100, nil).Total)
}

func TestCustomClient_GenerateCustomTxt_SignedAndVerifiable(t *testing.T) {
	s, _ := newCustomClientService(t)
	c := &model.CustomClient{
		ServerHost:  "rs.example.com",
		ServerKey:   "pubkey",
		ApiServer:   "https://api.example.com",
		RelayServer: "relay.example.com",
	}

	out, err := s.GenerateCustomTxt(c)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// decode the base64 signed blob: first 64 bytes are the signature, the rest is the JSON payload
	raw, err := base64.StdEncoding.DecodeString(out)
	require.NoError(t, err)
	require.Greater(t, len(raw), ed25519.SignatureSize)

	privBytes, err := base64.StdEncoding.DecodeString(signingPrivateKeyB64)
	require.NoError(t, err)
	pub := ed25519.PrivateKey(privBytes).Public().(ed25519.PublicKey)

	sig := raw[:ed25519.SignatureSize]
	msg := raw[ed25519.SignatureSize:]
	assert.True(t, ed25519.Verify(pub, msg, sig), "signature must verify against the embedded public key")

	// the payload must carry server settings under override-settings
	var payload map[string]any
	require.NoError(t, json.Unmarshal(msg, &payload))
	override, ok := payload["override-settings"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "rs.example.com", override["custom-rendezvous-server"])
	assert.Equal(t, "pubkey", override["key"])
	assert.Equal(t, "https://api.example.com", override["api-server"])
	assert.Equal(t, "relay.example.com", override["relay-server"])
}

func TestCustomClient_buildCustomClientJSON_MergesAndDefaults(t *testing.T) {
	s, _ := newCustomClientService(t)
	c := &model.CustomClient{
		ServerHost:       "host",
		OverrideSettings: custom_types.AutoJson(`{"theme":"dark"}`),
		DefaultSettings:  custom_types.AutoJson(`{"lang":"en"}`),
	}
	b, err := s.buildCustomClientJSON(c)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(b, &payload))

	override := payload["override-settings"].(map[string]any)
	assert.Equal(t, "dark", override["theme"], "user override settings are merged")
	assert.Equal(t, "host", override["custom-rendezvous-server"], "server host added to override settings")

	def := payload["default-settings"].(map[string]any)
	assert.Equal(t, "en", def["lang"])
}

func TestCustomClient_buildCustomClientJSON_EmptyOmitsKeys(t *testing.T) {
	s, _ := newCustomClientService(t)
	b, err := s.buildCustomClientJSON(&model.CustomClient{})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(b, &payload))
	_, hasOverride := payload["override-settings"]
	_, hasDefault := payload["default-settings"]
	assert.False(t, hasOverride, "no override-settings when nothing configured")
	assert.False(t, hasDefault)
}

func TestSignNaCl_Layout(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	msg := []byte("hello world")

	signed := SignNaCl(priv, msg)
	require.Len(t, signed, ed25519.SignatureSize+len(msg))
	assert.Equal(t, msg, signed[ed25519.SignatureSize:], "message appended after signature")
	assert.True(t, ed25519.Verify(pub, msg, signed[:ed25519.SignatureSize]))
}

func TestCustomClient_DeleteNoFile(t *testing.T) {
	s, db := newCustomClientService(t)
	require.NoError(t, db.Create(&model.BuildArtifact{Platform: "linux", Arch: "x86_64", Version: "1.0"}).Error)
	c := &model.CustomClient{Platform: "linux", Arch: "x86_64", Version: "1.0"}
	require.NoError(t, s.Create(c))

	// no FilePath / S3Key set -> Delete just removes the DB row
	require.NoError(t, s.Delete(c))
	assert.Zero(t, s.InfoById(c.Id).Id)
}
