package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type CustomClientService struct {
	ctx *ServiceContext
}

func (s *CustomClientService) InfoById(id uint) *model.CustomClient {
	c := &model.CustomClient{}
	s.ctx.DB.Where("id = ?", id).First(c)
	return c
}

func (s *CustomClientService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.CustomClientList {
	res := &model.CustomClientList{}
	queryList[model.CustomClient](s.ctx.DB, page, pageSize, res, &res.CustomClients, where)
	return res
}

func (s *CustomClientService) Create(c *model.CustomClient) error {
	if c.Enabled == nil {
		c.Enabled = model.BoolPtr(true)
	}
	return s.ctx.DB.Create(c).Error
}

func (s *CustomClientService) Update(c *model.CustomClient) error {
	return s.ctx.DB.Model(c).Updates(c).Error
}

func (s *CustomClientService) Delete(c *model.CustomClient) error {
	return s.ctx.DB.Delete(c).Error
}

// GenerateCustomTxt builds the custom.txt content for the given config.
// It produces a base64-encoded NaCl crypto_sign signed message.
func (s *CustomClientService) GenerateCustomTxt(c *model.CustomClient) (string, error) {
	signingKeyB64 := s.ctx.Config.CustomClient.SigningKey
	if signingKeyB64 == "" {
		return "", fmt.Errorf("custom-client signing-key is not configured")
	}

	privKeyBytes, err := base64.StdEncoding.DecodeString(signingKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode signing key: %w", err)
	}
	if len(privKeyBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("signing key must be %d bytes, got %d", ed25519.PrivateKeySize, len(privKeyBytes))
	}
	privKey := ed25519.PrivateKey(privKeyBytes)

	jsonPayload, err := s.buildCustomClientJSON(c)
	if err != nil {
		return "", fmt.Errorf("failed to build custom client JSON: %w", err)
	}

	signedMessage := SignNaCl(privKey, jsonPayload)
	return base64.StdEncoding.EncodeToString(signedMessage), nil
}

// buildCustomClientJSON constructs the JSON payload that gets embedded in custom.txt.
func (s *CustomClientService) buildCustomClientJSON(c *model.CustomClient) ([]byte, error) {
	payload := make(map[string]interface{})

	if c.AppName != "" {
		payload["app-name"] = c.AppName
	}

	// Server settings go as top-level keys (they end up in HARD_SETTINGS)
	if c.ServerHost != "" {
		payload["custom-rendezvous-server"] = c.ServerHost
	}
	if c.ServerKey != "" {
		payload["key"] = c.ServerKey
	}
	if c.ApiServer != "" {
		payload["api-server"] = c.ApiServer
	}
	if c.RelayServer != "" {
		payload["relay-server"] = c.RelayServer
	}

	// Default settings (user can override)
	if len(c.DefaultSettings) > 0 {
		var ds map[string]string
		if err := json.Unmarshal(c.DefaultSettings, &ds); err == nil && len(ds) > 0 {
			payload["default-settings"] = ds
		}
	}

	// Override settings (user cannot change)
	if len(c.OverrideSettings) > 0 {
		var os map[string]string
		if err := json.Unmarshal(c.OverrideSettings, &os); err == nil && len(os) > 0 {
			payload["override-settings"] = os
		}
	}

	return json.Marshal(payload)
}

// SignNaCl produces a NaCl crypto_sign "signed message": signature (64 bytes) || message.
// This matches what sodiumoxide::crypto::sign::verify() expects on the Rust side.
func SignNaCl(privKey ed25519.PrivateKey, message []byte) []byte {
	sig := ed25519.Sign(privKey, message)
	// ed25519.Sign returns a 64-byte detached signature.
	// NaCl signed message format = signature || original message
	signed := make([]byte, len(sig)+len(message))
	copy(signed, sig)
	copy(signed[len(sig):], message)
	return signed
}

// VerifyNaCl verifies a NaCl crypto_sign "signed message" and returns the original message.
// Useful for testing.
func VerifyNaCl(pubKey ed25519.PublicKey, signedMessage []byte) ([]byte, bool) {
	if len(signedMessage) < ed25519.SignatureSize {
		return nil, false
	}
	sig := signedMessage[:ed25519.SignatureSize]
	message := signedMessage[ed25519.SignatureSize:]
	if ed25519.Verify(pubKey, message, sig) {
		return message, true
	}
	return nil, false
}
