package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"gorm.io/gorm"
)

type WorkerRegistryService struct {
	ctx *ServiceContext
}

func (s *WorkerRegistryService) Register(name string, platforms []model.WorkerPlatform) (*model.Worker, error) {
	platformsJSON, _ := json.Marshal(platforms)
	now := custom_types.AutoTime(time.Now())

	w := &model.Worker{}
	err := s.ctx.DB.Where("name = ?", name).First(w).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to query worker: %w", err)
	}

	if w.Id > 0 {
		// Update existing
		w.Platforms = platformsJSON
		w.LastSeenAt = &now
		if err := s.ctx.DB.Save(w).Error; err != nil {
			return nil, fmt.Errorf("failed to update worker: %w", err)
		}
		return w, nil
	}

	// Create new
	w = &model.Worker{
		Name:       name,
		Platforms:  platformsJSON,
		LastSeenAt: &now,
	}
	if err := s.ctx.DB.Create(w).Error; err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}
	return w, nil
}

func (s *WorkerRegistryService) Heartbeat(name string) error {
	now := custom_types.AutoTime(time.Now())
	result := s.ctx.DB.Model(&model.Worker{}).Where("name = ?", name).Update("last_seen_at", &now)
	if result.RowsAffected == 0 {
		return fmt.Errorf("worker %q not found", name)
	}
	return nil
}

func (s *WorkerRegistryService) PushVersions(name string, versions []string) error {
	versionsJSON, _ := json.Marshal(versions)
	result := s.ctx.DB.Model(&model.Worker{}).Where("name = ?", name).Update("versions", versionsJSON)
	if result.RowsAffected == 0 {
		return fmt.Errorf("worker %q not found", name)
	}
	return nil
}

func (s *WorkerRegistryService) List(page, pageSize uint) *model.WorkerList {
	res := &model.WorkerList{}
	queryList[model.Worker](s.ctx.DB, page, pageSize, res, &res.Workers, nil)
	// Compute status for each worker
	timeout := s.heartbeatTimeout()
	now := time.Now()
	for _, w := range res.Workers {
		w.ComputeStatus(now, timeout)
	}
	return res
}

func (s *WorkerRegistryService) GetAllVersions() []string {
	var workers []*model.Worker
	timeout := s.heartbeatTimeout()
	cutoff := time.Now().Add(-timeout)
	s.ctx.DB.Where("last_seen_at > ?", cutoff).Find(&workers)

	seen := make(map[string]bool)
	var result []string
	for _, w := range workers {
		var versions []string
		if err := json.Unmarshal(w.Versions, &versions); err == nil {
			for _, v := range versions {
				if !seen[v] {
					seen[v] = true
					result = append(result, v)
				}
			}
		}
	}
	return result
}

func (s *WorkerRegistryService) heartbeatTimeout() time.Duration {
	t := s.ctx.Config.Worker.HeartbeatTimeout
	if t <= 0 {
		t = 15
	}
	return time.Duration(t) * time.Second
}
