package config

type Worker struct {
	Token            string `mapstructure:"token"`
	HeartbeatTimeout int    `mapstructure:"heartbeat-timeout"`
	LogCacheDir      string `mapstructure:"log-cache-dir"`
}

func (w *Worker) GetLogCacheDir() string {
	if w.LogCacheDir != "" {
		return w.LogCacheDir
	}
	return "./data/build-logs"
}

func (w *Worker) Enabled() bool {
	return w.Token != ""
}
