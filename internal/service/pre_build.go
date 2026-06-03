package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
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
		if strings.HasPrefix(job.LogPath, "logs/") {
			// Completed job: LogPath is an S3 key
			if s.ctx.S3 != nil {
				if err := s.ctx.S3.Delete(context.Background(), job.LogPath); err != nil {
					s.ctx.Logger.Warnf("delete pre-build log from S3 fail: %s %v", job.LogPath, err)
				}
			}
		} else {
			// In-progress job: LogPath is a local file
			if err := os.Remove(job.LogPath); err != nil && !os.IsNotExist(err) {
				s.ctx.Logger.Warnf("remove pre-build log fail: %s %v", job.LogPath, err)
			}
		}
	}
	return s.ctx.DB.Delete(job).Error
}

// ─── Cancel ──────────────────────────────────────────────────────────────

// Cancel marks a pending or building job as failed with "cancelled by user".
// The build-worker checks job status periodically and will abort if cancelled.
func (s *PreBuildService) Cancel(id uint) error {
	job := s.InfoById(id)
	if job.Id == 0 {
		return fmt.Errorf("job not found")
	}
	if job.Status != model.BuildStatusPending && job.Status != model.BuildStatusBuilding {
		return fmt.Errorf("job is not cancellable (status: %s)", job.Status)
	}

	now := custom_types.AutoTime(time.Now())
	return s.ctx.DB.Model(job).Updates(map[string]any{
		"status":       model.BuildStatusFailed,
		"error":        "cancelled by user",
		"completed_at": &now,
	}).Error
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
		Where("platform = ? AND arch = ? AND status IN ?", platform, arch,
			[]string{model.BuildStatusPending, model.BuildStatusBuilding}).
		Count(&activeCount)
	if activeCount > 0 {
		return nil, fmt.Errorf("a build for %s/%s is already in progress, please wait", platform, arch)
	}

	logDir := s.ctx.Config.Worker.LogCacheDir
	if logDir == "" {
		return nil, fmt.Errorf("worker.log-cache-dir is not configured")
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log cache dir %s: %w", logDir, err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("prebuild_%s_%s_%s_%d.tmp.log", version, platform, arch, time.Now().Unix()))

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

// ─── Log Reading ─────────────────────────────────────────────────────────

// GetLog returns the build log content starting from the given byte offset.
func (s *PreBuildService) GetLog(id uint, offset int64) (string, int64, error) {
	job := s.InfoById(id)
	if job.Id == 0 {
		return "", 0, fmt.Errorf("job not found")
	}
	if job.LogPath == "" {
		return "", 0, nil
	}

	// In-progress jobs: read from local cache file
	if job.Status == model.BuildStatusPending || job.Status == model.BuildStatusBuilding {
		return readLocalLog(job.LogPath, offset)
	}

	// Completed/failed jobs: read from S3
	if s.ctx.S3 != nil {
		return s.readS3Log(job.LogPath, offset)
	}

	// Fallback to local if S3 not configured
	return readLocalLog(job.LogPath, offset)
}

func readLocalLog(path string, offset int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, nil
		}
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return "", 0, err
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return "", offset, err
	}
	return string(data), offset + int64(len(data)), nil
}

func (s *PreBuildService) readS3Log(s3Key string, offset int64) (string, int64, error) {
	reader, err := s.ctx.S3.GetObject(context.Background(), s3Key)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read log from S3: %w", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, err
	}

	content := string(data)
	if offset > 0 && int64(len(content)) > offset {
		content = content[offset:]
	} else if offset >= int64(len(content)) {
		return "", int64(len(data)), nil
	}
	return content, int64(len(data)), nil
}
