package admin

type AuditQuery struct {
	PeerId   string `form:"peer_id"`
	FromPeer string `form:"from_peer"`
	PageQuery
}

type AuditConnLogIds struct {
	Ids []uint `json:"ids" validate:"required"`
}
type AuditFileLogIds struct {
	Ids []uint `json:"ids" validate:"required"`
}

type AuditConnDisconnectForm struct {
	PeerId  string `json:"peer_id" validate:"required"`
	ConnIds []int  `json:"conn_ids" validate:"required"`
}
