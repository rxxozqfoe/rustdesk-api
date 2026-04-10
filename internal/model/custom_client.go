package model

import "github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"

// CustomClient stores a named custom client configuration that can be used
// to generate signed custom.txt files and repackaged binaries.
type CustomClient struct {
	IdModel
	Name             string                `json:"name" gorm:"type:varchar(255);uniqueIndex;not null"`
	AppName          string                `json:"app_name" gorm:"type:varchar(255);not null"`
	ServerHost       string                `json:"server_host" gorm:"type:varchar(255)"`
	ServerKey        string                `json:"server_key" gorm:"type:varchar(255)"`
	ApiServer        string                `json:"api_server" gorm:"type:varchar(255)"`
	RelayServer      string                `json:"relay_server" gorm:"type:varchar(255)"`
	DefaultSettings  custom_types.AutoJson `json:"default_settings" gorm:"type:text" swaggertype:"object"`
	OverrideSettings custom_types.AutoJson `json:"override_settings" gorm:"type:text" swaggertype:"object"`
	Enabled          *bool                 `json:"enabled" gorm:"not null"`
	TimeModel
}

type CustomClientList struct {
	CustomClients []*CustomClient `json:"list"`
	Pagination
}

// BuildArtifact tracks a pre-compiled build output (Flutter build folder or
// final package) that can be repackaged with a custom.txt injection.
type BuildArtifact struct {
	IdModel
	Platform string `json:"platform" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_format"` // linux, windows, macos, android
	Arch     string `json:"arch" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_format"`     // x86_64, aarch64
	Format   string `json:"format" gorm:"type:varchar(50);not null;uniqueIndex:idx_ba_platform_arch_format"`   // deb, exe, dmg, apk
	Version  string `json:"version" gorm:"type:varchar(50)"`
	FilePath string `json:"file_path" gorm:"type:varchar(500);not null"` // path to the base binary file
	FileSize int64  `json:"file_size"`
	Sha256   string `json:"sha256" gorm:"type:varchar(64)"`
	Source   string `json:"source" gorm:"type:varchar(50)"` // "local_build", "uploaded"
	TimeModel
}

type BuildArtifactList struct {
	BuildArtifacts []*BuildArtifact `json:"list"`
	Pagination
}

// BuildJob represents an async build task that compiles the RustDesk client
// from source and produces a BuildArtifact.
type BuildJob struct {
	IdModel
	Version     string                `json:"version" gorm:"type:varchar(50);not null"`
	Platform    string                `json:"platform" gorm:"type:varchar(50);not null"`
	Arch        string                `json:"arch" gorm:"type:varchar(50);not null"`
	Format      string                `json:"format" gorm:"type:varchar(50);not null"`
	Status      string                `json:"status" gorm:"type:varchar(20);not null;index"` // pending, building, completed, failed
	LogPath     string                `json:"log_path" gorm:"type:varchar(500)"`
	Error       string                `json:"error" gorm:"type:text"`
	ArtifactId  uint                  `json:"artifact_id"`
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

type BuildJobList struct {
	BuildJobs []*BuildJob `json:"list"`
	Pagination
}
