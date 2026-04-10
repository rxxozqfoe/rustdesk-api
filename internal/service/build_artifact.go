package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type BuildArtifactService struct {
	ctx *ServiceContext
}

func (s *BuildArtifactService) InfoById(id uint) *model.BuildArtifact {
	ba := &model.BuildArtifact{}
	s.ctx.DB.Where("id = ?", id).First(ba)
	return ba
}

func (s *BuildArtifactService) FindByPlatformArchFormat(platform, arch, format string) *model.BuildArtifact {
	ba := &model.BuildArtifact{}
	s.ctx.DB.Where("platform = ? AND arch = ? AND format = ?", platform, arch, format).First(ba)
	return ba
}

func (s *BuildArtifactService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.BuildArtifactList {
	res := &model.BuildArtifactList{}
	queryList[model.BuildArtifact](s.ctx.DB, page, pageSize, res, &res.BuildArtifacts, where)
	return res
}

func (s *BuildArtifactService) Create(ba *model.BuildArtifact) error {
	return s.ctx.DB.Create(ba).Error
}

func (s *BuildArtifactService) Delete(ba *model.BuildArtifact) error {
	// Remove the file from disk if it exists
	if ba.FilePath != "" {
		os.Remove(ba.FilePath)
	}
	return s.ctx.DB.Delete(ba).Error
}

// SaveUploadedFile saves an uploaded file to the base-binaries directory and creates a DB record.
func (s *BuildArtifactService) SaveUploadedFile(src io.Reader, filename, platform, arch, format, version string) (*model.BuildArtifact, error) {
	baseDir := s.ctx.Config.CustomClient.BaseBinariesDir
	targetDir := filepath.Join(baseDir, fmt.Sprintf("%s-%s-%s", platform, arch, format))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, filename)
	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(dst, hasher)
	size, err := io.Copy(writer, src)
	if err != nil {
		os.Remove(targetPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	// Upsert: if same platform+arch+format exists, update it
	existing := s.FindByPlatformArchFormat(platform, arch, format)
	if existing.Id > 0 {
		// Remove old file
		if existing.FilePath != "" && existing.FilePath != targetPath {
			os.Remove(existing.FilePath)
		}
		existing.Version = version
		existing.FilePath = targetPath
		existing.FileSize = size
		existing.Sha256 = hash
		existing.Source = "uploaded"
		return existing, s.ctx.DB.Save(existing).Error
	}

	ba := &model.BuildArtifact{
		Platform: platform,
		Arch:     arch,
		Format:   format,
		Version:  version,
		FilePath: targetPath,
		FileSize: size,
		Sha256:   hash,
		Source:   "uploaded",
	}
	return ba, s.Create(ba)
}
