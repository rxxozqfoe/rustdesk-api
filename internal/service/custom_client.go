package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

// Create saves the record with status=bundling, then starts async bundling.
func (s *CustomClientService) Create(c *model.CustomClient) error {
	// Validate pre-built artifact exists
	ba := s.ctx.Services.BuildArtifactService.FindByPlatformArchVersion(c.Platform, c.Arch, c.Version)
	if ba.Id == 0 {
		return fmt.Errorf("no pre-built binary for %s/%s v%s — please run a pre-build job first", c.Platform, c.Arch, c.Version)
	}

	c.Status = model.BundleStatusBundling
	if err := s.ctx.DB.Create(c).Error; err != nil {
		return err
	}

	// Start async bundle
	go s.executeBundle(c.Id)
	return nil
}

func (s *CustomClientService) Delete(c *model.CustomClient) error {
	// Remove bundled file
	if c.FilePath != "" {
		os.Remove(c.FilePath)
	}
	return s.ctx.DB.Delete(c).Error
}

// executeBundle runs the repackaging in background.
func (s *CustomClientService) executeBundle(id uint) {
	c := s.InfoById(id)
	if c.Id == 0 {
		return
	}

	// Generate custom.txt
	customTxt, err := s.GenerateCustomTxt(c)
	if err != nil {
		s.ctx.DB.Model(c).Updates(map[string]any{
			"status": model.BundleStatusFailed,
			"error":  fmt.Sprintf("failed to generate custom.txt: %v", err),
		})
		return
	}

	// Find pre-built artifact
	ba := s.ctx.Services.BuildArtifactService.FindByPlatformArchVersion(c.Platform, c.Arch, c.Version)
	if ba.Id == 0 {
		s.ctx.DB.Model(c).Updates(map[string]any{
			"status": model.BundleStatusFailed,
			"error":  "pre-built binary not found",
		})
		return
	}

	// Repackage
	result, err := s.ctx.Services.RepackagerService.Repackage(ba.DirPath, c.Format, customTxt)
	if err != nil {
		s.ctx.DB.Model(c).Updates(map[string]any{
			"status": model.BundleStatusFailed,
			"error":  fmt.Sprintf("repackaging failed: %v", err),
		})
		return
	}

	// Move bundled file to persistent location
	cacheDir, _ := filepath.Abs(s.ctx.Config.CustomClient.CacheDir)
	os.MkdirAll(cacheDir, 0755)

	appName := c.AppName
	if appName == "" {
		appName = "rustdesk"
	}
	filename := fmt.Sprintf("%s-%s-%s-%s.%s", appName, c.Version, c.Platform, c.Arch, c.Format)
	destPath := filepath.Join(cacheDir, fmt.Sprintf("%d-%s", c.Id, filename))

	if err := copyFile(result.FilePath, destPath); err != nil {
		result.Cleanup()
		s.ctx.DB.Model(c).Updates(map[string]any{
			"status": model.BundleStatusFailed,
			"error":  fmt.Sprintf("failed to save bundled file: %v", err),
		})
		return
	}
	result.Cleanup()

	// Get file size
	info, _ := os.Stat(destPath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}

	s.ctx.DB.Model(c).Updates(map[string]any{
		"status":    model.BundleStatusCompleted,
		"file_path": destPath,
		"file_size": fileSize,
	})
}

// ─── Signing ──────────────────────────────────────────────────────────────

// Hardcoded Ed25519 keypair for custom client signing.
// The public key must match the KEY constant in rustdesk/src/common.rs:2187.
// Public key:  fjMWQpn+Kvu2hO6hRjIWyS8n55JITpevt0OzMBRIn4Q=
const signingPrivateKeyB64 = "K+hPH7thulZob58hBy3OX5a22uo9sdHhhRGUhXd/aip+MxZCmf4q+7aE7qFGMhbJLyfnkkhOl6+3Q7MwFEifhA=="

func (s *CustomClientService) GenerateCustomTxt(c *model.CustomClient) (string, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(signingPrivateKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode signing key: %w", err)
	}
	privKey := ed25519.PrivateKey(privKeyBytes)

	jsonPayload, err := s.buildCustomClientJSON(c)
	if err != nil {
		return "", fmt.Errorf("failed to build custom client JSON: %w", err)
	}

	signedMessage := SignNaCl(privKey, jsonPayload)
	return base64.StdEncoding.EncodeToString(signedMessage), nil
}

func (s *CustomClientService) buildCustomClientJSON(c *model.CustomClient) ([]byte, error) {
	payload := make(map[string]any)

	if c.AppName != "" {
		payload["app-name"] = c.AppName
	}
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

	if len(c.DefaultSettings) > 0 {
		var ds map[string]string
		if err := json.Unmarshal(c.DefaultSettings, &ds); err == nil && len(ds) > 0 {
			payload["default-settings"] = ds
		}
	}
	if len(c.OverrideSettings) > 0 {
		var os map[string]string
		if err := json.Unmarshal(c.OverrideSettings, &os); err == nil && len(os) > 0 {
			payload["override-settings"] = os
		}
	}

	return json.Marshal(payload)
}

func SignNaCl(privKey ed25519.PrivateKey, message []byte) []byte {
	sig := ed25519.Sign(privKey, message)
	signed := make([]byte, len(sig)+len(message))
	copy(signed, sig)
	copy(signed[len(sig):], message)
	return signed
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func DerivePublicKeyB64() (string, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(signingPrivateKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode signing key: %w", err)
	}
	pubKey := ed25519.PrivateKey(privKeyBytes).Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(pubKey), nil
}
