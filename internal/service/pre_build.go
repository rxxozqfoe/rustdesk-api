package service

import (
	"bufio"
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

type PreBuildService struct {
	ctx       *ServiceContext
	buildChan chan struct{}
	mu        sync.Mutex
	cancelCmd *exec.Cmd
}

func NewPreBuildService(ctx *ServiceContext) *PreBuildService {
	svc := &PreBuildService{
		ctx:       ctx,
		buildChan: make(chan struct{}, 1),
	}
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

// ─── Versions ─────────────────────────────────────────────────────────────

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func (s *PreBuildService) ListVersions() ([]string, error) {
	srcDir := s.ctx.Config.CustomClient.RustdeskSrcDir
	if srcDir == "" {
		return nil, fmt.Errorf("rustdesk-src-dir is not configured")
	}
	srcDir, _ = filepath.Abs(srcDir)

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

func (s *PreBuildService) Trigger(version, platform, arch string) (*model.PreBuild, error) {
	if platform != "linux" {
		return nil, fmt.Errorf("only linux platform is supported for builds")
	}
	if arch != "x86_64" && arch != "aarch64" {
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}

	srcDir := s.ctx.Config.CustomClient.RustdeskSrcDir
	if srcDir == "" {
		return nil, fmt.Errorf("rustdesk-src-dir is not configured")
	}
	srcDir, _ = filepath.Abs(srcDir)

	var activeCount int64
	s.ctx.DB.Model(&model.PreBuild{}).
		Where("status IN ?", []string{model.BuildStatusPending, model.BuildStatusBuilding}).
		Count(&activeCount)
	if activeCount > 0 {
		return nil, fmt.Errorf("a build is already in progress, please wait")
	}

	logDir := s.ctx.Config.CustomClient.BuildLogDir
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

	go s.executeBuild(job.Id)
	return job, nil
}

// ─── Cancel ───────────────────────────────────────────────────────────────

func (s *PreBuildService) Cancel(id uint) error {
	job := s.InfoById(id)
	if job.Id == 0 {
		return fmt.Errorf("job not found")
	}
	if job.Status != model.BuildStatusPending && job.Status != model.BuildStatusBuilding {
		return fmt.Errorf("job is not cancellable (status: %s)", job.Status)
	}

	s.mu.Lock()
	cmd := s.cancelCmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}

	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]any{
		"status":       model.BuildStatusFailed,
		"error":        "cancelled by user",
		"completed_at": &now,
	})
	return nil
}

// ─── Log Reader ───────────────────────────────────────────────────────────

func (s *PreBuildService) GetLog(id uint, offset int64) (string, int64, error) {
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

func (s *PreBuildService) executeBuild(jobId uint) {
	s.buildChan <- struct{}{}
	defer func() { <-s.buildChan }()

	job := s.InfoById(jobId)
	if job.Id == 0 || job.Status != model.BuildStatusPending {
		return
	}

	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]any{
		"status":     model.BuildStatusBuilding,
		"started_at": &now,
	})

	logFile, err := os.Create(job.LogPath)
	if err != nil {
		s.failJob(job, fmt.Sprintf("failed to create log file: %v", err))
		return
	}
	defer logFile.Close()

	logger := bufio.NewWriter(logFile)
	writeLog := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(logger, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
		logger.Flush()
	}

	writeLog("Pre-build started: version=%s platform=%s arch=%s", job.Version, job.Platform, job.Arch)

	// Resolve to absolute paths to avoid cwd-relative issues
	worktreeDir, _ := filepath.Abs(s.ctx.Config.CustomClient.BuildWorktreeDir)
	srcDir, _ := filepath.Abs(s.ctx.Config.CustomClient.RustdeskSrcDir)
	if err := s.ensureWorktree(srcDir, worktreeDir, writeLog); err != nil {
		s.failJob(job, fmt.Sprintf("worktree setup failed: %v", err))
		return
	}

	writeLog("Fetching tags and checking out version %s...", job.Version)
	if err := s.runInDir(worktreeDir, logFile, "git", "fetch", "origin", "--tags"); err != nil {
		s.failJob(job, fmt.Sprintf("git fetch failed: %v", err))
		return
	}
	if err := s.runInDir(worktreeDir, logFile, "git", "checkout", "--force", job.Version); err != nil {
		s.failJob(job, fmt.Sprintf("git checkout %s failed: %v", job.Version, err))
		return
	}
	s.runInDir(worktreeDir, logFile, "git", "clean", "-fd")

	// Init submodules (hbb_common is a git submodule required for build)
	writeLog("Initializing git submodules...")
	if err := s.runInDir(worktreeDir, logFile, "git", "submodule", "update", "--init", "--recursive"); err != nil {
		s.failJob(job, fmt.Sprintf("git submodule update failed: %v", err))
		return
	}

	pubKey, _ := DerivePublicKeyB64()
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

	// Pre-flight checks (ref: rustdesk/README.md)
	// - VCPKG_ROOT must be set (for libvpx, libyuv, opus, aom)
	// - cargo, flutter must be in PATH
	// - System deps: libgtk-3-dev, clang, nasm, yasm, etc.
	vcpkgRoot := os.Getenv("VCPKG_ROOT")
	if vcpkgRoot == "" {
		s.failJob(job, "VCPKG_ROOT environment variable is not set. Install vcpkg and set VCPKG_ROOT first. See rustdesk/README.md")
		return
	}
	writeLog("VCPKG_ROOT=%s", vcpkgRoot)

	if _, err := exec.LookPath("cargo"); err != nil {
		s.failJob(job, "cargo not found in PATH. Install Rust toolchain first.")
		return
	}
	if _, err := exec.LookPath("flutter"); err != nil {
		s.failJob(job, "flutter not found in PATH. Install Flutter SDK first.")
		return
	}

	// Step 1: Generate flutter_rust_bridge code
	// Ref: .github/workflows/bridge.yml — required before cargo build
	writeLog("Step 1/4: Generating flutter_rust_bridge code...")
	// Workaround: downgrade extended_text for bridge codegen (same as CI bridge.yml)
	pubspecPath := filepath.Join(worktreeDir, "flutter", "pubspec.yaml")
	s.runBuildCmd(worktreeDir, logFile, "sed", "-i",
		"s/extended_text: 14.0.0/extended_text: 13.0.0/g", pubspecPath)
	// flutter pub get (required by codegen)
	if err := s.runBuildCmd(filepath.Join(worktreeDir, "flutter"), logFile, "flutter", "pub", "get"); err != nil {
		s.failJob(job, fmt.Sprintf("flutter pub get failed: %v", err))
		return
	}
	if err := s.runBuildCmd(worktreeDir, logFile,
		"flutter_rust_bridge_codegen",
		"--rust-input", "./src/flutter_ffi.rs",
		"--dart-output", "./flutter/lib/generated_bridge.dart",
		"--c-output", "./flutter/macos/Runner/bridge_generated.h",
	); err != nil {
		s.failJob(job, fmt.Sprintf("flutter_rust_bridge_codegen failed: %v (install with: cargo install flutter_rust_bridge_codegen --version 1.80.1 --features uuid)", err))
		return
	}

	// Step 2: Compile Rust library
	// Ref: build.py build_flutter_deb() — cargo build --features {features} --lib --release
	features := "flutter"
	writeLog("Step 2/4: Compiling Rust library (cargo build --features %s --lib --release)...", features)
	if err := s.runBuildCmd(worktreeDir, logFile, "cargo", "build", "--features", features, "--lib", "--release"); err != nil {
		s.failJob(job, fmt.Sprintf("cargo build failed: %v", err))
		return
	}

	// Step 3: FFI bindgen workaround
	// Ref: build.py ffi_bindgen_function_refactor() — required after cargo build
	writeLog("Step 3/4: Applying FFI bindgen workaround...")
	bridgeDart := filepath.Join(worktreeDir, "flutter", "lib", "generated_bridge.dart")
	if err := s.runBuildCmd(worktreeDir, logFile, "sed", "-i",
		"s/ffi.NativeFunction<ffi.Bool Function(DartPort/ffi.NativeFunction<ffi.Uint8 Function(DartPort/g",
		bridgeDart); err != nil {
		writeLog("Warning: ffi_bindgen_function_refactor failed (may not be needed): %v", err)
	}

	// Step 4: Build Flutter
	// Restore extended_text version for actual build (codegen step may have downgraded it)
	s.runBuildCmd(worktreeDir, logFile, "git", "checkout", "--", "flutter/pubspec.yaml")
	writeLog("Step 4/4: Building Flutter UI (flutter build linux --release)...")
	if err := s.runBuildCmd(filepath.Join(worktreeDir, "flutter"), logFile, "flutter", "build", "linux", "--release"); err != nil {
		s.failJob(job, fmt.Sprintf("flutter build failed: %v", err))
		return
	}

	// Locate build output folder
	buildOutputDir := GetBuildOutputDir(worktreeDir, job.Platform)
	if _, err := os.Stat(buildOutputDir); err != nil {
		s.failJob(job, fmt.Sprintf("build output folder not found: %s", buildOutputDir))
		return
	}
	writeLog("Build output folder: %s", buildOutputDir)

	// Copy to base-binaries-dir so worktree can be reused
	baseDir, _ := filepath.Abs(s.ctx.Config.CustomClient.BaseBinariesDir)
	targetDir := filepath.Join(baseDir, fmt.Sprintf("%s-%s-%s", job.Platform, job.Arch, job.Version))
	writeLog("Copying build output to %s...", targetDir)
	os.RemoveAll(targetDir)
	os.MkdirAll(baseDir, 0755)
	if err := s.copyDir(buildOutputDir, targetDir); err != nil {
		s.failJob(job, fmt.Sprintf("failed to copy build output: %v", err))
		return
	}

	// Register as BuildArtifact
	artifact, err := s.ctx.Services.BuildArtifactService.RegisterBuildFolder(
		job.Platform, job.Arch, job.Version, targetDir, "local_build",
	)
	if err != nil {
		s.failJob(job, fmt.Sprintf("failed to register artifact: %v", err))
		return
	}

	completedAt := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]any{
		"status":       model.BuildStatusCompleted,
		"artifact_id":  artifact.Id,
		"completed_at": &completedAt,
	})
	writeLog("Pre-build completed. Artifact ID: %d, output: %s", artifact.Id, targetDir)
}

func (s *PreBuildService) failJob(job *model.PreBuild, errMsg string) {
	now := custom_types.AutoTime(time.Now())
	s.ctx.DB.Model(job).Updates(map[string]any{
		"status":       model.BuildStatusFailed,
		"error":        errMsg,
		"completed_at": &now,
	})
	if job.LogPath != "" {
		if f, err := os.OpenFile(job.LogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err == nil {
			fmt.Fprintf(f, "[%s] ERROR: %s\n", time.Now().Format("15:04:05"), errMsg)
			f.Close()
		}
	}
}

func (s *PreBuildService) ensureWorktree(srcDir, worktreeDir string, writeLog func(string, ...any)) error {
	// Ensure both paths are absolute
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("failed to resolve srcDir: %w", err)
	}
	absWt, err := filepath.Abs(worktreeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve worktreeDir: %w", err)
	}

	if _, err := os.Stat(absWt); err == nil {
		if err := s.runInDir(absWt, nil, "git", "status"); err != nil {
			writeLog("Existing worktree seems broken, removing and recreating...")
			os.RemoveAll(absWt)
			exec.Command("git", "-C", absSrc, "worktree", "remove", "--force", absWt).Run()
		} else {
			return nil
		}
	}

	writeLog("Creating build worktree at %s (source: %s)...", absWt, absSrc)
	os.MkdirAll(filepath.Dir(absWt), 0755)
	cmd := exec.Command("git", "-C", absSrc, "worktree", "add", "--detach", absWt)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %w\n%s", err, string(out))
	}
	return nil
}

// runBuildCmd runs a command with process-group isolation, cancel support,
// and inherits the full parent environment (VCPKG_ROOT, PATH with cargo/flutter, etc.).
func (s *PreBuildService) runBuildCmd(dir string, logWriter io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ() // inherit all env vars (VCPKG_ROOT, PATH, etc.)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s.mu.Lock()
	s.cancelCmd = cmd
	s.mu.Unlock()

	err := cmd.Run()

	s.mu.Lock()
	s.cancelCmd = nil
	s.mu.Unlock()

	return err
}

func (s *PreBuildService) runInDir(dir string, logWriter io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if logWriter != nil {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}
	return cmd.Run()
}

func (s *PreBuildService) patchPublicKey(commonRsPath, pubKey string) error {
	data, err := os.ReadFile(commonRsPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", commonRsPath, err)
	}
	re := regexp.MustCompile(`const KEY: &str = "[^"]+";`)
	if !re.MatchString(string(data)) {
		return fmt.Errorf("could not find KEY constant in %s", commonRsPath)
	}
	newContent := re.ReplaceAllString(string(data), fmt.Sprintf(`const KEY: &str = "%s";`, pubKey))
	return os.WriteFile(commonRsPath, []byte(newContent), 0644)
}

func (s *PreBuildService) copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-a", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cp failed: %w\n%s", err, string(out))
	}
	return nil
}
