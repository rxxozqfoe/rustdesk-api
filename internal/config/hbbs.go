package config

// Hbbs configures the internal integration surface used by the RustDesk
// rendezvous server (hbbs) to implement 1.4.9 Pro features (controller-user
// audit attribution and device-deployment gating).
type Hbbs struct {
	// Token is the shared secret protecting the internal /api/hbbs/* endpoints.
	// Empty disables the internal API entirely.
	Token string `mapstructure:"token"`
	// DeployEnabled gates the device deployment flow. When true,
	// /api/devices/deploy provisions devices and the rendezvous server may
	// reject not-yet-deployed devices with NOT_DEPLOYED.
	DeployEnabled bool `mapstructure:"deploy-enabled"`
	// ConnAuditRefTTL is the lifetime, in seconds, of conn_audit_ref snapshots.
	// Defaults to 86400 (1 day) when unset.
	ConnAuditRefTTL int `mapstructure:"conn-audit-ref-ttl"`
}

func (h *Hbbs) Enabled() bool {
	return h.Token != ""
}

func (h *Hbbs) RefTTLSeconds() int {
	if h.ConnAuditRefTTL <= 0 {
		return 86400
	}
	return h.ConnAuditRefTTL
}
