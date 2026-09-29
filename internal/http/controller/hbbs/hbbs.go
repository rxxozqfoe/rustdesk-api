package hbbs

import (
	"time"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
)

// Hbbs holds the internal endpoints the RustDesk rendezvous server calls to
// implement 1.4.9 Pro features. All routes are protected by the hbbs.token
// shared secret (see middleware.HbbsAuth).
type Hbbs struct {
	HD *deps.HandlerDeps
}

type connAuditRefForm struct {
	Ref   string `json:"ref"`
	Token string `json:"token"`
}

// ConnAuditRef records a ref -> controller-user snapshot. The rendezvous server
// calls this when it mints a conn_audit_ref for an outgoing connection so the
// controlled client's later audit can be attributed to the controlling user.
func (h *Hbbs) ConnAuditRef(c *gin.Context) {
	f := &connAuditRefForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 400, "invalid body")
		return
	}
	if f.Ref == "" || f.Token == "" {
		response.Fail(c, 400, "ref and token are required")
		return
	}
	u, _ := h.HD.Services.InfoByAccessToken(f.Token)
	if u == nil || u.Id == 0 {
		response.Fail(c, 401, "invalid user token")
		return
	}
	ttl := time.Duration(h.HD.Config.Hbbs.RefTTLSeconds()) * time.Second
	if err := h.HD.Services.Upsert(f.Ref, u.Id, u.Username, ttl); err != nil {
		h.HD.Logger.Warnf("hbbs ConnAuditRef upsert fail: %v", err)
		response.Fail(c, 500, "store failed")
		return
	}
	response.Success(c, gin.H{"user_id": u.Id, "username": u.Username})
}

// DeviceDeployed reports whether a device is provisioned, for the rendezvous
// NOT_DEPLOYED gate. hbbs.deploy-enabled is the single switch: with it off
// every device counts as deployed, whatever hbbs's DEPLOY_ENABLED says.
// hbbs asks only after its own uuid check, so the uuid/pk it sends are the
// device's; they must match the ones recorded when the id was deployed.
func (h *Hbbs) DeviceDeployed(c *gin.Context) {
	if !h.HD.Config.Hbbs.DeployEnabled {
		response.Success(c, gin.H{"deployed": true})
		return
	}
	uuid := c.Query("uuid")
	pk := c.Query("pk")
	peer := h.HD.Services.FindById(c.Query("id"))
	deployed := peer.RowId != 0 && peer.Deployed &&
		(uuid == "" || peer.DeployedUuid == "" || peer.DeployedUuid == uuid) &&
		(pk == "" || peer.DeployedPk == "" || peer.DeployedPk == pk)
	response.Success(c, gin.H{"deployed": deployed})
}
