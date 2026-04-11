package model

import "github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"

// CustomClient represents a bundled custom client installer.
// Each record = one config + one platform/version/format → one downloadable file.
type CustomClient struct {
	IdModel
	// Config
	Name             string                `json:"name" gorm:"type:varchar(255);not null"`
	AppName          string                `json:"app_name" gorm:"type:varchar(255);not null"`
	ServerHost       string                `json:"server_host" gorm:"type:varchar(255)"`
	ServerKey        string                `json:"server_key" gorm:"type:varchar(255)"`
	ApiServer        string                `json:"api_server" gorm:"type:varchar(255)"`
	RelayServer      string                `json:"relay_server" gorm:"type:varchar(255)"`
	DefaultSettings  custom_types.AutoJson `json:"default_settings" gorm:"type:text" swaggertype:"object"`
	OverrideSettings custom_types.AutoJson `json:"override_settings" gorm:"type:text" swaggertype:"object"`
	// Target
	Platform string `json:"platform" gorm:"type:varchar(50);not null"` // linux, windows, macos, android
	Arch     string `json:"arch" gorm:"type:varchar(50);not null"`     // x86_64, aarch64
	Version  string `json:"version" gorm:"type:varchar(50);not null"`  // pre-build version tag
	Format   string `json:"format" gorm:"type:varchar(50);not null"`   // deb, zip
	// Bundle status
	Status   string `json:"status" gorm:"type:varchar(20);not null;index"` // bundling, completed, failed
	FilePath string `json:"file_path" gorm:"type:varchar(500)"`           // path to bundled installer
	FileSize int64  `json:"file_size"`
	Error    string `json:"error" gorm:"type:text"`
	TimeModel
}

const (
	BundleStatusBundling  = "bundling"
	BundleStatusCompleted = "completed"
	BundleStatusFailed    = "failed"
)

type CustomClientList struct {
	CustomClients []*CustomClient `json:"list"`
	Pagination
}

// BuildArtifact tracks a pre-compiled build output folder that can be
// packaged into various formats (deb, rpm, zip, etc.) with custom.txt injection.
type BuildArtifact struct {
	IdModel
	Platform string `json:"platform" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_ver"` // linux, windows, macos, android
	Arch     string `json:"arch" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_ver"`     // x86_64, aarch64
	Version  string `json:"version" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_ver"`
	DirPath  string `json:"dir_path" gorm:"type:varchar(500);not null"` // path to the build output folder
	Source   string `json:"source" gorm:"type:varchar(50)"`             // "local_build", "uploaded"
	TimeModel
}

type BuildArtifactList struct {
	BuildArtifacts []*BuildArtifact `json:"list"`
	Pagination
}

// PreBuild represents an async build task that compiles the RustDesk client
// from source and produces a BuildArtifact (build output folder).
type PreBuild struct {
	IdModel
	Version     string                 `json:"version" gorm:"type:varchar(50);not null"`
	Platform    string                 `json:"platform" gorm:"type:varchar(50);not null"`
	Arch        string                 `json:"arch" gorm:"type:varchar(50);not null"`
	Status      string                 `json:"status" gorm:"type:varchar(20);not null;index"` // pending, building, completed, failed
	LogPath     string                 `json:"log_path" gorm:"type:varchar(500)"`
	Error       string                 `json:"error" gorm:"type:text"`
	ArtifactId  uint                   `json:"artifact_id"`
	StartedAt   *custom_types.AutoTime `json:"started_at" gorm:"type:timestamp"`
	CompletedAt *custom_types.AutoTime `json:"completed_at" gorm:"type:timestamp"`
	TimeModel
}

const (
	BuildStatusPending   = "pending"
	BuildStatusBuilding  = "building"
	BuildStatusCompleted = "completed"
	BuildStatusFailed    = "failed"
)

type PreBuildList struct {
	PreBuilds []*PreBuild `json:"list"`
	Pagination
}
