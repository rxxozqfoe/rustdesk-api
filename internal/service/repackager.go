package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type RepackagerService struct {
	ctx *ServiceContext
}

// RepackageResult holds the path to the repackaged file and a cleanup function.
type RepackageResult struct {
	FilePath string
	Cleanup  func()
}

// Repackage takes a build output folder, injects custom.txt, and packages into the requested format.
func (s *RepackagerService) Repackage(buildDir string, format string, customTxtContent string) (*RepackageResult, error) {
	switch format {
	case "deb":
		return s.packageDeb(buildDir, customTxtContent)
	case "zip":
		return s.packageZip(buildDir, customTxtContent)
	default:
		return nil, fmt.Errorf("packaging format not yet supported: %s", format)
	}
}

// packageDeb creates a .deb from a build output folder with custom.txt injected.
// Uses the same directory structure as build.py's build_deb_from_folder().
func (s *RepackagerService) packageDeb(buildDir string, customTxtContent string) (*RepackageResult, error) {
	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		return nil, fmt.Errorf("dpkg-deb not found: %w (install dpkg on this server)", err)
	}

	workDir, err := os.MkdirTemp("", "rustdesk-repack-deb-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(workDir) }

	// Build the deb directory structure
	debRoot := filepath.Join(workDir, "deb")
	dataDir := filepath.Join(debRoot, "usr", "share", "rustdesk")
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(filepath.Join(debRoot, "usr", "bin"), 0755)

	// Copy build output into the data directory
	cmd := exec.Command("cp", "-a", buildDir+"/.", dataDir+"/")
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to copy build output: %w\n%s", err, string(out))
	}

	// Inject custom.txt
	if err := os.WriteFile(filepath.Join(dataDir, "custom.txt"), []byte(customTxtContent), 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to write custom.txt: %w", err)
	}

	// Create symlink: /usr/bin/rustdesk -> /usr/share/rustdesk/rustdesk
	os.Symlink("/usr/share/rustdesk/rustdesk", filepath.Join(debRoot, "usr", "bin", "rustdesk"))

	// Create minimal DEBIAN/control
	debianDir := filepath.Join(debRoot, "DEBIAN")
	os.MkdirAll(debianDir, 0755)
	control := `Package: rustdesk
Architecture: amd64
Version: 0.0.0
Depends: libgtk-3-0, libxcb-randr0, libxdo3 | libxdo4, libxfixes3, libxcb-shape0, libxcb-xfixes0, libasound2, libsystemd0, curl, libva2, libva-drm2, libva-x11-2, libpam0g
Description: RustDesk custom client
`
	if err := os.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to write control file: %w", err)
	}

	// Build the deb
	outputFile := filepath.Join(workDir, "output.deb")
	cmd = exec.Command("dpkg-deb", "-b", debRoot, outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("dpkg-deb build failed: %w\n%s", err, string(out))
	}

	return &RepackageResult{FilePath: outputFile, Cleanup: cleanup}, nil
}

// packageZip creates a zip from a build output folder with custom.txt injected.
func (s *RepackagerService) packageZip(buildDir string, customTxtContent string) (*RepackageResult, error) {
	if _, err := exec.LookPath("zip"); err != nil {
		return nil, fmt.Errorf("zip not found: %w", err)
	}

	workDir, err := os.MkdirTemp("", "rustdesk-repack-zip-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(workDir) }

	// Copy build output to work dir
	stageDir := filepath.Join(workDir, "rustdesk")
	cmd := exec.Command("cp", "-a", buildDir, stageDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to copy build output: %w\n%s", err, string(out))
	}

	// Inject custom.txt
	if err := os.WriteFile(filepath.Join(stageDir, "custom.txt"), []byte(customTxtContent), 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to write custom.txt: %w", err)
	}

	// Create zip
	outputFile := filepath.Join(workDir, "output.zip")
	cmd = exec.Command("zip", "-r", outputFile, "rustdesk")
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("zip failed: %w\n%s", err, string(out))
	}

	return &RepackageResult{FilePath: outputFile, Cleanup: cleanup}, nil
}
