package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type WorkerService struct {
	ctx *ServiceContext
}

// WorkerJob is the unified task payload sent to the build-worker.
type WorkerJob struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"` // "pre-build" or "bundle"
	Version  string `json:"version,omitempty"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	// Bundle fields
	Format        string `json:"format,omitempty"`
	AppName       string `json:"app_name,omitempty"`
	CustomTxt     string `json:"custom_txt,omitempty"`      // pre-signed custom.txt content
	ArtifactS3Key string `json:"artifact_s3_key,omitempty"` // pre-build artifact S3 key for bundle
	ArtifactDir   string `json:"artifact_dir,omitempty"`    // pre-build artifact local dir for bundle
}

// FetchPendingJob returns one pending pre-build or bundle job for the worker to execute.
// Pre-build jobs take priority.
func (s *WorkerService) FetchPendingJob() (*WorkerJob, error) {
	// Try pre-build first
	pb := &model.PreBuild{}
	s.ctx.DB.Where("status = ?", model.BuildStatusPending).Order("id ASC").First(pb)
	if pb.Id > 0 {
		return &WorkerJob{
			ID:       pb.Id,
			Type:     "pre-build",
			Version:  pb.Version,
			Platform: pb.Platform,
			Arch:     pb.Arch,
		}, nil
	}

	// Then try bundle
	cc := &model.CustomClient{}
	s.ctx.DB.Where("status = ?", model.BundleStatusBundling).Order("id ASC").First(cc)
	if cc.Id > 0 {
		// Generate custom.txt (signed by API server — worker just injects it)
		customTxt, err := s.ctx.Services.CustomClientService.GenerateCustomTxt(cc)
		if err != nil {
			// Mark the job as failed so we don't retry it forever
			s.ctx.DB.Model(cc).Updates(map[string]any{
				"status": model.BundleStatusFailed,
				"error":  fmt.Sprintf("failed to generate custom.txt: %v", err),
			})
			return nil, nil // return nil so worker polls next job
		}

		// Find artifact
		ba := s.ctx.Services.BuildArtifactService.FindByPlatformArchVersion(cc.Platform, cc.Arch, cc.Version)
		var artifactS3Key, artifactDir string
		if ba.Id > 0 {
			artifactS3Key = ba.S3Key
			artifactDir = ba.DirPath
		}

		return &WorkerJob{
			ID:            cc.Id,
			Type:          "bundle",
			Version:       cc.Version,
			Platform:      cc.Platform,
			Arch:          cc.Arch,
			Format:        cc.Format,
			AppName:       cc.AppName,
			CustomTxt:     customTxt,
			ArtifactS3Key: artifactS3Key,
			ArtifactDir:   artifactDir,
		}, nil
	}

	return nil, nil // no pending jobs
}

// StartJob marks a job as started.
func (s *WorkerService) StartJob(jobID uint, jobType string) error {
	now := custom_types.AutoTime(time.Now())
	switch jobType {
	case "pre-build":
		return s.ctx.DB.Model(&model.PreBuild{}).Where("id = ?", jobID).Updates(map[string]any{
			"status":     model.BuildStatusBuilding,
			"started_at": &now,
		}).Error
	case "bundle":
		// bundle status is already "bundling" from Create
		return nil
	default:
		return fmt.Errorf("unknown job type: %s", jobType)
	}
}

// AppendLog appends log content to the pre-build job's log.
// For pre-builds, we store log incrementally. Bundles don't have per-job logs.
func (s *WorkerService) AppendLog(jobID uint, content string) error {
	pb := &model.PreBuild{}
	s.ctx.DB.Where("id = ?", jobID).First(pb)
	if pb.Id == 0 {
		return fmt.Errorf("job not found")
	}
	if pb.LogPath == "" {
		return nil
	}
	f, err := openOrCreateFile(pb.LogPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// CompletePreBuild marks a pre-build job as completed and registers the artifact.
func (s *WorkerService) CompletePreBuild(jobID uint, s3Key string) error {
	pb := &model.PreBuild{}
	s.ctx.DB.Where("id = ?", jobID).First(pb)
	if pb.Id == 0 {
		return fmt.Errorf("job not found")
	}

	// Register artifact with S3 key
	artifact := &model.BuildArtifact{
		Platform: pb.Platform,
		Arch:     pb.Arch,
		Version:  pb.Version,
		S3Key:    s3Key,
		Source:   "worker_build",
	}
	existing := s.ctx.Services.BuildArtifactService.FindByPlatformArchVersion(pb.Platform, pb.Arch, pb.Version)
	if existing.Id > 0 {
		existing.S3Key = s3Key
		existing.Source = "worker_build"
		s.ctx.DB.Save(existing)
		artifact = existing
	} else {
		s.ctx.DB.Create(artifact)
	}

	now := custom_types.AutoTime(time.Now())
	return s.ctx.DB.Model(pb).Updates(map[string]any{
		"status":       model.BuildStatusCompleted,
		"artifact_id":  artifact.Id,
		"completed_at": &now,
	}).Error
}

// CompleteBundle marks a bundle job as completed.
func (s *WorkerService) CompleteBundle(jobID uint, s3Key string, fileSize int64) error {
	return s.ctx.DB.Model(&model.CustomClient{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":    model.BundleStatusCompleted,
		"s3_key":    s3Key,
		"file_size": fileSize,
	}).Error
}

// FailJob marks a job as failed.
func (s *WorkerService) FailJob(jobID uint, jobType string, errMsg string) error {
	now := custom_types.AutoTime(time.Now())
	switch jobType {
	case "pre-build":
		return s.ctx.DB.Model(&model.PreBuild{}).Where("id = ?", jobID).Updates(map[string]any{
			"status":       model.BuildStatusFailed,
			"error":        errMsg,
			"completed_at": &now,
		}).Error
	case "bundle":
		return s.ctx.DB.Model(&model.CustomClient{}).Where("id = ?", jobID).Updates(map[string]any{
			"status": model.BundleStatusFailed,
			"error":  errMsg,
		}).Error
	default:
		return fmt.Errorf("unknown job type: %s", jobType)
	}
}

func openOrCreateFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
}

// ProxyVersions fetches available versions from the worker's HTTP server.
func (s *WorkerService) ProxyVersions() ([]string, error) {
	baseURL := s.ctx.Config.Worker.BaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("worker base-url is not configured")
	}

	resp, err := s.workerGet(baseURL + "/api/worker/versions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode worker response: %w", err)
	}
	return result.Data, nil
}

// ProxyLog fetches build log from the worker's HTTP server.
func (s *WorkerService) ProxyLog(jobID uint, offset int64) (string, int64, error) {
	baseURL := s.ctx.Config.Worker.BaseURL
	if baseURL == "" {
		return "", 0, fmt.Errorf("worker base-url is not configured")
	}

	url := fmt.Sprintf("%s/api/worker/jobs/%d/log?offset=%d", baseURL, jobID, offset)
	resp, err := s.workerGet(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Content   string `json:"content"`
			NewOffset int64  `json:"new_offset"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", offset, fmt.Errorf("failed to decode worker response: %w", err)
	}
	return result.Data.Content, result.Data.NewOffset, nil
}

func (s *WorkerService) workerGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.ctx.Config.Worker.Token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("worker request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("worker returned %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
