package config

const (
	DefaultBaseBinariesDir  = "./data/base-binaries"
	DefaultCacheDir         = "./data/custom-client-cache"
	DefaultBuildWorktreeDir = "./data/build-worktree"
	DefaultBuildLogDir      = "./data/build-logs"
)

type CustomClient struct {
	BaseBinariesDir  string `mapstructure:"base-binaries-dir"`  // directory for pre-built base binaries
	CacheDir         string `mapstructure:"cache-dir"`          // directory for repackaged binary cache
	RustdeskSrcDir   string `mapstructure:"rustdesk-src-dir"`   // path to rustdesk/ source tree
	BuildWorktreeDir string `mapstructure:"build-worktree-dir"` // git worktree for builds (isolated from main source)
	BuildLogDir      string `mapstructure:"build-log-dir"`      // directory for build log files
}

func (cc *CustomClient) Init() {
	if cc.BaseBinariesDir == "" {
		cc.BaseBinariesDir = DefaultBaseBinariesDir
	}
	if cc.CacheDir == "" {
		cc.CacheDir = DefaultCacheDir
	}
	if cc.BuildWorktreeDir == "" {
		cc.BuildWorktreeDir = DefaultBuildWorktreeDir
	}
	if cc.BuildLogDir == "" {
		cc.BuildLogDir = DefaultBuildLogDir
	}
}
