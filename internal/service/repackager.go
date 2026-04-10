package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RepackagerService struct {
	ctx *ServiceContext
}

// RepackageResult holds the path to the repackaged file and a cleanup function.
type RepackageResult struct {
	FilePath string   // path to the generated file
	Cleanup  func()   // call when done to remove temp files
}

// RepackageLinuxDeb injects custom.txt into a .deb package.
// It extracts the deb, adds custom.txt to /usr/share/rustdesk/, and rebuilds.
func (s *RepackagerService) RepackageLinuxDeb(baseDeb string, customTxtContent string) (*RepackageResult, error) {
	// Verify dpkg-deb is available
	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		return nil, fmt.Errorf("dpkg-deb not found: %w (install dpkg on this server)", err)
	}

	// Create temp working directory
	workDir, err := os.MkdirTemp("", "rustdesk-repack-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(workDir)
	}

	extractDir := filepath.Join(workDir, "extracted")
	outputFile := filepath.Join(workDir, "output.deb")

	// Extract the deb
	cmd := exec.Command("dpkg-deb", "-R", baseDeb, extractDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("dpkg-deb extract failed: %w\n%s", err, string(out))
	}

	// Write custom.txt into the rustdesk data directory
	customTxtPath := filepath.Join(extractDir, "usr", "share", "rustdesk", "custom.txt")
	if err := os.WriteFile(customTxtPath, []byte(customTxtContent), 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to write custom.txt: %w", err)
	}

	// Rebuild the deb
	cmd = exec.Command("dpkg-deb", "-b", extractDir, outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("dpkg-deb build failed: %w\n%s", err, string(out))
	}

	return &RepackageResult{
		FilePath: outputFile,
		Cleanup:  cleanup,
	}, nil
}

// RepackageWindowsExe injects custom.txt into a Windows build folder and creates a zip.
// For full portable EXE packing, the portable packer binary + Python/Brotli are needed.
// This simplified version creates a zip with custom.txt included.
func (s *RepackagerService) RepackageWindowsExe(baseFile string, customTxtContent string) (*RepackageResult, error) {
	// For .exe base files, we create a zip containing the exe + custom.txt
	if _, err := exec.LookPath("zip"); err != nil {
		return nil, fmt.Errorf("zip not found: %w", err)
	}

	workDir, err := os.MkdirTemp("", "rustdesk-repack-win-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(workDir)
	}

	// Copy the base exe to work dir
	baseName := filepath.Base(baseFile)
	destExe := filepath.Join(workDir, baseName)
	if out, err := exec.Command("cp", baseFile, destExe).CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to copy base exe: %w\n%s", err, string(out))
	}

	// Write custom.txt
	if err := os.WriteFile(filepath.Join(workDir, "custom.txt"), []byte(customTxtContent), 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to write custom.txt: %w", err)
	}

	// Create zip
	outputFile := filepath.Join(workDir, strings.TrimSuffix(baseName, ".exe")+".zip")
	cmd := exec.Command("zip", "-j", outputFile, destExe, filepath.Join(workDir, "custom.txt"))
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("zip failed: %w\n%s", err, string(out))
	}

	return &RepackageResult{
		FilePath: outputFile,
		Cleanup:  cleanup,
	}, nil
}

// Repackage dispatches to the appropriate platform-specific repackager.
func (s *RepackagerService) Repackage(baseFilePath string, format string, customTxtContent string) (*RepackageResult, error) {
	switch format {
	case "deb":
		return s.RepackageLinuxDeb(baseFilePath, customTxtContent)
	case "exe":
		return s.RepackageWindowsExe(baseFilePath, customTxtContent)
	default:
		return nil, fmt.Errorf("repackaging not yet supported for format: %s", format)
	}
}
