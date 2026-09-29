package model

// ConnAuditRef stores a short-lived snapshot mapping a connection-audit
// reference token (conn_audit_ref) to the controlling user. The rendezvous
// server (hbbs) writes it when it injects ControlledContext into a connection
// request; the controlled client later echoes the same ref in its connection
// audit, letting the API attribute the audit to the controller user.
type ConnAuditRef struct {
	IdModel
	Ref       string `json:"ref" gorm:"default:'';not null;uniqueIndex"`
	UserId    uint   `json:"user_id" gorm:"default:0;not null;index"`
	Username  string `json:"username" gorm:"default:'';not null;"`
	ExpiredAt int64  `json:"expired_at" gorm:"default:0;not null;index"`
	TimeModel
}
