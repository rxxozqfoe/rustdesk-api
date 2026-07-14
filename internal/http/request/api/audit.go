package api

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type AuditConnForm struct {
	Action       string   `json:"action"`
	ConnId       int64    `json:"conn_id"`
	Id           string   `json:"id"`
	Peer         []string `json:"peer"`
	Ip           string   `json:"ip"`
	SessionId    float64  `json:"session_id"`
	Type         int      `json:"type"`
	Uuid         string   `json:"uuid"`
	PrimaryAuth  int      `json:"primary_auth"`
	TwoFactor    int      `json:"two_factor"`
	ConnAuditRef string   `json:"conn_audit_ref"`
}

func (a *AuditConnForm) ToAuditConn() *model.AuditConn {
	fp := ""
	fn := ""
	if len(a.Peer) >= 1 {
		fp = a.Peer[0]
		if len(a.Peer) == 2 {
			fn = a.Peer[1]
		}
	}
	ssid := strconv.FormatFloat(a.SessionId, 'f', -1, 64)
	return &model.AuditConn{
		Action:       a.Action,
		ConnId:       a.ConnId,
		PeerId:       a.Id,
		FromPeer:     fp,
		FromName:     fn,
		Ip:           a.Ip,
		SessionId:    ssid,
		Type:         a.Type,
		Uuid:         a.Uuid,
		PrimaryAuth:  a.PrimaryAuth,
		TwoFactor:    a.TwoFactor,
		ConnAuditRef: a.ConnAuditRef,
	}
}

type AuditFileForm struct {
	Id     string `json:"id"`
	Info   string `json:"info"`
	IsFile bool   `json:"is_file"`
	Path   string `json:"path"`
	PeerId string `json:"peer_id"`
	Type   int    `json:"type"`
	Uuid   string `json:"uuid"`
}
type AuditFileInfo struct {
	Ip   string `json:"ip"`
	Name string `json:"name"`
	Num  int    `json:"num"`
}

func (a *AuditFileForm) ToAuditFile() *model.AuditFile {
	fi := &AuditFileInfo{}
	err := json.Unmarshal([]byte(a.Info), fi)
	if err != nil {
		// Fall through with a zero-valued fi; callers don't depend on these fields
		// when info is malformed. DTO files intentionally do not reach for an
		// application logger — errors are logged in the controller layer.
		log.Println("ToAuditFile:", err)
	}

	return &model.AuditFile{
		PeerId:   a.Id,
		Info:     a.Info,
		IsFile:   a.IsFile,
		FromPeer: a.PeerId,
		Path:     a.Path,
		Type:     a.Type,
		Uuid:     a.Uuid,
		FromName: fi.Name,
		Ip:       fi.Ip,
		Num:      fi.Num,
	}
}
