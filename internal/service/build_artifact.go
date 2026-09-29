package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rxxozqfoe/rustdesk-api/internal/model"
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

func (s *BuildArtifactService) FindByPlatformArchVersion(platform, arch, version string) *model.BuildArtifact {
	ba := &model.BuildArtifact{}
	s.ctx.DB.Where("platform = ? AND arch = ? AND version = ?", platform, arch, version).First(ba)
	return ba
}

func (s *BuildArtifactService) FindByPlatformArch(platform, arch string) *model.BuildArtifact {
	ba := &model.BuildArtifact{}
	s.ctx.DB.Where("platform = ? AND arch = ?", platform, arch).Order("created_at DESC").First(ba)
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
	if ba.DirPath != "" {
		if err := os.RemoveAll(ba.DirPath); err != nil {
			s.ctx.Logger.Warnf("remove build artifact dir fail: %s %v", ba.DirPath, err)
		}
	}
	if ba.S3Key != "" && s.ctx.S3 != nil {
		if err := s.ctx.S3.Delete(context.Background(), ba.S3Key); err != nil {
			s.ctx.Logger.Warnf("delete build artifact from S3 fail: %s %v", ba.S3Key, err)
		}
	}
	return s.ctx.DB.Delete(ba).Error
}

// RegisterBuildFolder registers a build output folder as a BuildArtifact.
// Upserts: if same platform+arch+version exists, updates it.
func (s *BuildArtifactService) RegisterBuildFolder(platform, arch, version, dirPath, source string) (*model.BuildArtifact, error) {
	if _, err := os.Stat(dirPath); err != nil {
		return nil, fmt.Errorf("build output folder does not exist: %s", dirPath)
	}

	existing := s.FindByPlatformArchVersion(platform, arch, version)
	if existing.Id > 0 {
		if existing.DirPath != dirPath && existing.DirPath != "" {
			if err := os.RemoveAll(existing.DirPath); err != nil {
				s.ctx.Logger.Warnf("remove stale build artifact dir fail: %s %v", existing.DirPath, err)
			}
		}
		existing.DirPath = dirPath
		existing.Source = source
		return existing, s.ctx.DB.Save(existing).Error
	}

	artifact := &model.BuildArtifact{
		Platform: platform,
		Arch:     arch,
		Version:  version,
		DirPath:  dirPath,
		Source:   source,
	}
	return artifact, s.Create(artifact)
}

// GetBuildOutputDir returns the expected Flutter build output directory path.
func GetBuildOutputDir(worktreeDir, platform string) string {
	switch platform {
	case "linux":
		return filepath.Join(worktreeDir, "flutter", "build", "linux", "x64", "release", "bundle")
	case "windows":
		return filepath.Join(worktreeDir, "flutter", "build", "windows", "x64", "runner", "Release")
	case "macos":
		return filepath.Join(worktreeDir, "flutter", "build", "macos", "Build", "Products", "Release")
	default:
		return filepath.Join(worktreeDir, "flutter", "build")
	}
}
