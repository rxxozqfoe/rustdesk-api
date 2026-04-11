package config

type Worker struct {
	Token   string `mapstructure:"token"`    // shared secret for worker authentication
	BaseURL string `mapstructure:"base-url"` // worker HTTP base URL for proxying (e.g. "http://worker:8080")
}

func (w *Worker) Enabled() bool {
	return w.Token != ""
}
