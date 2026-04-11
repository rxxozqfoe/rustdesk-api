package config

type Worker struct {
	Token            string `mapstructure:"token"`
	HeartbeatTimeout int    `mapstructure:"heartbeat-timeout"`
}

func (w *Worker) Enabled() bool {
	return w.Token != ""
}
