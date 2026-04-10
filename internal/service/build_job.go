package service

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"gorm.io/gorm"
)

type BuildJobService struct {
	ctx       *ServiceContext
	buildChan chan struct{} // capacity 1: only one build at a time
	mu        sync.Mutex
	cancelCmd *exec.Cmd // current running build command (for cancellation)
}

func NewBuildJobService(ctx *ServiceContext) *BuildJobService {
	svc := &BuildJobService{
		ctx:       ctx,
		buildChan: make(chan struct{}, 1),
	}
	// On startup, mark any leftover "building" jobs as failed
	svc.recoverStaleJobs()
	return svc
}

func (s *BuildJobService) recoverStaleJobs() {
	s.ctx.DB.Model(&model.BuildJob{}).
		Where("status = ?", model.BuildStatusBuilding).
		Updates(map[string]interface{}{
			"status": model.BuildStatusFailed,
			"error":  "interrupted by server restart",
		})
}

// ─── CRUD ─────────────────────────────────────────────────────────────────

func (s *BuildJobService) InfoById(id uint) *model.BuildJob {
	job := &model.BuildJob{}
	s.ctx.DB.Where("id = ?", id).First(job)
	return job
}

func (s *BuildJobService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.BuildJobList {
	res := &model.BuildJobList{}
	queryList[model.BuildJob](s.ctx.DB, page, pageSize, res, &res.BuildJobs, where)
	return res
}

func (s *BuildJobService) Delete(job *model.BuildJob) error {
	if job.LogPath != "" {
		os.Remove(job.LogPath)
	}
	return s.ctx.DB.Delete(job).Error
}

// ─── Versions ─────────────────────────────────────────────────────────────

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

// ListVersions returns git tags from the rustdesk source tree, filtered to semver.
func (s *BuildJobService) ListVersions() ([]string, error) {
	srcDir := s.ctx.Config.CustomClient.RustdeskSrcDir
	if srcDir == "" {
		return nil, fmt.Errorf("rustdesk-src-dir is not configured")
	}

	cmd := exec.Command("git", "-C", srcDir, "tag", "--list", "--sort=-version:refname")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git tag failed: %w", err)
	}

	var versions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		tag := strings.TrimSpace(line)
		if tag != "" && semverRe.MatchString(tag) {
			versions = append(versions, tag)
		}
	}
	return versions, nil
}

// ─── Trigger ──────────────────────────────────────────────────────────────

func (s *BuildJobService) Trigger(version, platform, arch, format string) (*model.BuildJob, error) {
	// Validate platform (only linux supported for now)
	if platform != "linux" {
		return nil, fmt.Errorf("only linux platform is supported for builds")
	}
	if arch != "x86_64" && arch != "aarch64" {
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}
	if format != "deb" {
		return nil, fmt.Errorf("only deb format is supported for builds")
	}

	srcDir := s.ctx.Config.CustomClient.RustdeskSrcDir
	if srcDir == "" {
		return nil, fmt.Errorf("rustdesk-src-dir is not configured")
	}

	// Check no active build
	var activeCount int64
	s.ctx.DB.Model(&model.BuildJob{}).
		Where("status IN ?", []string{model.BuildStatusPending, model.BuildStatusBuilding}).
		Count(&activeCount)
	if activeCount > 0 {
		return nil, fmt.Errorf("a build is already in progress, please wait")
	}

	// Create log directory
	logDir := s.ctx.Config.CustomClient.BuildLogDir
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, fmt.Sprintf("build_%s_%s_%s_%d.log", version, platform, arch, time.Now().Unix()))

	job := &model.BuildJob{
		Version:  version,
		Platform: platform,
		Arch:     arch,
		Format:   format,
		Status:   model.BuildStatusPending,
		LogPath:  logPath,
	}
	if err := s.ctx.DB.Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create build job: %w", err)
	}

	// Start async build
	go s.executeBuild(job.Id)

	return job, nil
}

// ─── Cancel ───────────────────────────────────────────────────────────────

func (s *BuildJobService) Cancel(id uint) error {
	job := s.InfoById(id)
	if job.Id == 0 {
		return fmt.Errorf("job not found")
	}
	if job.Status != model.BuildStatusPending && job.Status != model.BuildStatusBuilding {
		return fmt.Errorf("job is not cancellable (status: %s)", job.Status)
	}

	// If building, kill the process group
	s.mu.Lock()
	cmd := s.cancelCmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		// Kill the entire process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}

	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]interface{}{
		"status":       model.BuildStatusFailed,
		"error":        "cancelled by user",
		"completed_at": &now,
	})
	return nil
}

// ─── Log Reader ───────────────────────────────────────────────────────────

// GetLog reads the build log from the given byte offset. Returns the new content and new offset.
func (s *BuildJobService) GetLog(id uint, offset int64) (string, int64, error) {
	job := s.InfoById(id)
	if job.Id == 0 {
		return "", 0, fmt.Errorf("job not found")
	}
	if job.LogPath == "" {
		return "", 0, nil
	}

	f, err := os.Open(job.LogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, nil
		}
		return "", 0, err
	}
	defer f.Close()

	if offset > 0 {
		f.Seek(offset, io.SeekStart)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return "", offset, err
	}

	return string(data), offset + int64(len(data)), nil
}

// ─── Async Build Execution ────────────────────────────────────────────────

func (s *BuildJobService) executeBuild(jobId uint) {
	// Acquire build slot
	s.buildChan <- struct{}{}
	defer func() { <-s.buildChan }()

	job := s.InfoById(jobId)
	if job.Id == 0 || job.Status != model.BuildStatusPending {
		return
	}

	// Update status to building
	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]interface{}{
		"status":     model.BuildStatusBuilding,
		"started_at": &now,
	})

	// Open log file
	logFile, err := os.Create(job.LogPath)
	if err != nil {
		s.failJob(job, fmt.Sprintf("failed to create log file: %v", err))
		return
	}
	defer logFile.Close()

	logger := bufio.NewWriter(logFile)
	writeLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		logger.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg))
		logger.Flush()
	}

	writeLog("Build started: version=%s platform=%s arch=%s format=%s", job.Version, job.Platform, job.Arch, job.Format)

	// Ensure worktree exists
	worktreeDir := s.ctx.Config.CustomClient.BuildWorktreeDir
	srcDir := s.ctx.Config.CustomClient.RustdeskSrcDir
	if err := s.ensureWorktree(srcDir, worktreeDir, writeLog); err != nil {
		s.failJob(job, fmt.Sprintf("worktree setup failed: %v", err))
		return
	}

	// Checkout the target version
	writeLog("Fetching tags and checking out version %s...", job.Version)
	if err := s.runInDir(worktreeDir, logFile, "git", "fetch", "origin", "--tags"); err != nil {
		s.failJob(job, fmt.Sprintf("git fetch failed: %v", err))
		return
	}
	if err := s.runInDir(worktreeDir, logFile, "git", "checkout", "--force", job.Version); err != nil {
		s.failJob(job, fmt.Sprintf("git checkout %s failed: %v", job.Version, err))
		return
	}
	if err := s.runInDir(worktreeDir, logFile, "git", "clean", "-fd"); err != nil {
		writeLog("Warning: git clean failed: %v", err)
	}

	// Patch the signing public key
	pubKey := s.ctx.Config.CustomClient.SigningPublicKey
	if pubKey != "" {
		writeLog("Patching signing public key in common.rs...")
		commonRsPath := filepath.Join(worktreeDir, "src", "common.rs")
		if err := s.patchPublicKey(commonRsPath, pubKey); err != nil {
			s.failJob(job, fmt.Sprintf("failed to patch public key: %v", err))
			return
		}
	} else {
		writeLog("Warning: signing-public-key not configured, skipping patch")
	}

	// Run the build
	writeLog("Starting build (this may take 10-30 minutes)...")
	buildCmd := exec.Command("python3", "build.py", "--flutter", "--hwcodec")
	buildCmd.Dir = worktreeDir
	buildCmd.Stdout = logFile
	buildCmd.Stderr = logFile
	buildCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s.mu.Lock()
	s.cancelCmd = buildCmd
	s.mu.Unlock()

	buildErr := buildCmd.Run()

	s.mu.Lock()
	s.cancelCmd = nil
	s.mu.Unlock()

	if buildErr != nil {
		s.failJob(job, fmt.Sprintf("build command failed: %v", buildErr))
		return
	}

	// Find the output .deb file
	writeLog("Build completed, looking for output .deb...")
	debPath, err := s.findBuildOutput(worktreeDir, job.Version, job.Format)
	if err != nil {
		s.failJob(job, fmt.Sprintf("failed to find build output: %v", err))
		return
	}
	writeLog("Found: %s", debPath)

	// Move to base-binaries-dir and register as BuildArtifact
	artifact, err := s.registerArtifact(debPath, job, writeLog)
	if err != nil {
		s.failJob(job, fmt.Sprintf("failed to register artifact: %v", err))
		return
	}

	// Mark job as completed
	completedAt := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]interface{}{
		"status":       model.BuildStatusCompleted,
		"artifact_id":  artifact.Id,
		"completed_at": &completedAt,
	})
	writeLog("Build job completed successfully. Artifact ID: %d", artifact.Id)
}

func (s *BuildJobService) failJob(job *model.BuildJob, errMsg string) {
	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]interface{}{
		"status":       model.BuildStatusFailed,
		"error":        errMsg,
		"completed_at": &now,
	})
	// Also write to log file
	if job.LogPath != "" {
		if f, err := os.OpenFile(job.LogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err == nil {
			fmt.Fprintf(f, "[%s] ERROR: %s\n", time.Now().Format("15:04:05"), errMsg)
			f.Close()
		}
	}
}

func (s *BuildJobService) ensureWorktree(srcDir, worktreeDir string, writeLog func(string, ...interface{})) error {
	if _, err := os.Stat(worktreeDir); err == nil {
		// Worktree exists, verify it's valid
		if err := s.runInDir(worktreeDir, nil, "git", "status"); err != nil {
			writeLog("Existing worktree seems broken, removing and recreating...")
			os.RemoveAll(worktreeDir)
			// Also remove from git worktree list
			exec.Command("git", "-C", srcDir, "worktree", "remove", "--force", worktreeDir).Run()
		} else {
			return nil
		}
	}

	writeLog("Creating build worktree at %s...", worktreeDir)
	os.MkdirAll(filepath.Dir(worktreeDir), 0755)
	cmd := exec.Command("git", "-C", srcDir, "worktree", "add", "--detach", worktreeDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %w\n%s", err, string(out))
	}
	return nil
}

func (s *BuildJobService) runInDir(dir string, logWriter io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if logWriter != nil {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}
	return cmd.Run()
}

func (s *BuildJobService) patchPublicKey(commonRsPath, pubKey string) error {
	data, err := os.ReadFile(commonRsPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", commonRsPath, err)
	}

	content := string(data)
	// Match the hardcoded KEY constant line
	re := regexp.MustCompile(`const KEY: &str = "[^"]+";`)
	if !re.MatchString(content) {
		return fmt.Errorf("could not find KEY constant in %s", commonRsPath)
	}

	newContent := re.ReplaceAllString(content, fmt.Sprintf(`const KEY: &str = "%s";`, pubKey))
	return os.WriteFile(commonRsPath, []byte(newContent), 0644)
}

func (s *BuildJobService) findBuildOutput(worktreeDir, version, format string) (string, error) {
	if format != "deb" {
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	// build.py puts the .deb in the worktree root (parent of flutter/)
	pattern := filepath.Join(worktreeDir, fmt.Sprintf("rustdesk-*.deb"))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		// Also check flutter/ directory
		pattern = filepath.Join(worktreeDir, "flutter", fmt.Sprintf("rustdesk-*.deb"))
		matches, _ = filepath.Glob(pattern)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no .deb file found in %s", worktreeDir)
	}

	// Return the most recently modified one
	var newest string
	var newestTime time.Time
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newest = m
			newestTime = info.ModTime()
		}
	}
	return newest, nil
}

func (s *BuildJobService) registerArtifact(debPath string, job *model.BuildJob, writeLog func(string, ...interface{})) (*model.BuildArtifact, error) {
	baseDir := s.ctx.Config.CustomClient.BaseBinariesDir
	targetDir := filepath.Join(baseDir, fmt.Sprintf("%s-%s-%s", job.Platform, job.Arch, job.Format))
	os.MkdirAll(targetDir, 0755)

	filename := filepath.Base(debPath)
	targetPath := filepath.Join(targetDir, filename)

	writeLog("Moving %s to %s...", debPath, targetPath)

	// Compute SHA256
	src, err := os.Open(debPath)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	size, err := io.Copy(hasher, src)
	src.Close()
	if err != nil {
		return nil, err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Move file
	if err := os.Rename(debPath, targetPath); err != nil {
		// If rename fails (cross-device), do copy + delete
		if err := copyFile(debPath, targetPath); err != nil {
			return nil, fmt.Errorf("failed to move build output: %w", err)
		}
		os.Remove(debPath)
	}

	// Upsert BuildArtifact
	existing := &model.BuildArtifact{}
	s.ctx.DB.Where("platform = ? AND arch = ? AND format = ?", job.Platform, job.Arch, job.Format).First(existing)

	if existing.Id > 0 {
		// Remove old file if different
		if existing.FilePath != targetPath {
			os.Remove(existing.FilePath)
		}
		existing.Version = job.Version
		existing.FilePath = targetPath
		existing.FileSize = size
		existing.Sha256 = hash
		existing.Source = "local_build"
		return existing, s.ctx.DB.Save(existing).Error
	}

	artifact := &model.BuildArtifact{
		Platform: job.Platform,
		Arch:     job.Arch,
		Format:   job.Format,
		Version:  job.Version,
		FilePath: targetPath,
		FileSize: size,
		Sha256:   hash,
		Source:   "local_build",
	}
	return artifact, s.ctx.DB.Create(artifact).Error
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
