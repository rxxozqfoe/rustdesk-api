package config

type Worker struct {
	Token            string `mapstructure:"token"`
	HeartbeatTimeout int    `mapstructure:"heartbeat-timeout"`
	LogCacheDir      string `mapstructure:"log-cache-dir"`
}

func (w *Worker) Enabled() bool {
	return w.Token != ""
}
