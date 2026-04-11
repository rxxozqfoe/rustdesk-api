package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type PreBuildService struct {
	ctx *ServiceContext
}

func NewPreBuildService(ctx *ServiceContext) *PreBuildService {
	svc := &PreBuildService{ctx: ctx}
	svc.recoverStaleJobs()
	return svc
}

func (s *PreBuildService) recoverStaleJobs() {
	s.ctx.DB.Model(&model.PreBuild{}).
		Where("status = ?", model.BuildStatusBuilding).
		Updates(map[string]any{
			"status": model.BuildStatusFailed,
			"error":  "interrupted by server restart",
		})
}

// ─── CRUD ─────────────────────────────────────────────────────────────────

func (s *PreBuildService) InfoById(id uint) *model.PreBuild {
	job := &model.PreBuild{}
	s.ctx.DB.Where("id = ?", id).First(job)
	return job
}

func (s *PreBuildService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.PreBuildList {
	res := &model.PreBuildList{}
	queryList[model.PreBuild](s.ctx.DB, page, pageSize, res, &res.PreBuilds, where)
	return res
}

func (s *PreBuildService) Delete(job *model.PreBuild) error {
	if job.LogPath != "" {
		os.Remove(job.LogPath)
	}
	return s.ctx.DB.Delete(job).Error
}

// ─── Trigger ──────────────────────────────────────────────────────────────

// Trigger creates a pending pre-build job for the build-worker to pick up.
func (s *PreBuildService) Trigger(version, platform, arch string) (*model.PreBuild, error) {
	if platform != "linux" {
		return nil, fmt.Errorf("only linux platform is supported for builds")
	}
	if arch != "x86_64" && arch != "aarch64" {
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}

	var activeCount int64
	s.ctx.DB.Model(&model.PreBuild{}).
		Where("status IN ?", []string{model.BuildStatusPending, model.BuildStatusBuilding}).
		Count(&activeCount)
	if activeCount > 0 {
		return nil, fmt.Errorf("a build is already in progress, please wait")
	}

	logDir := "./data/build-logs"
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, fmt.Sprintf("prebuild_%s_%s_%s_%d.log", version, platform, arch, time.Now().Unix()))

	job := &model.PreBuild{
		Version:  version,
		Platform: platform,
		Arch:     arch,
		Status:   model.BuildStatusPending,
		LogPath:  logPath,
	}
	if err := s.ctx.DB.Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create pre-build job: %w", err)
	}

	return job, nil
}
