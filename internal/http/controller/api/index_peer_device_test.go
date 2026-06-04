package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Index / Heartbeat ---

func TestIndexIndex(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	i := &api.Index{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/", i.Index)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	// Standard envelope: {"code":0,"message":"success","data":"Hello Gwen"}
	assert.Contains(t, rec.Body.String(), `"code":0`)
	assert.Contains(t, rec.Body.String(), "Hello Gwen")
}

func TestHeartbeat_UnknownPeerReturnsEmpty(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	i := &api.Index{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/heartbeat", i.Heartbeat)

	rec := httptest.NewRecorder()
	body := `{"id":"ghost","uuid":"uuid-ghost"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/heartbeat", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{}`, rec.Body.String())
}

func TestHeartbeat_MissingUuidReturnsEmpty(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	i := &api.Index{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/heartbeat", i.Heartbeat)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/heartbeat", `{"id":"x"}`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{}`, rec.Body.String())
}

func TestHeartbeat_KnownPeerUpdatesLastOnline(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// Peer last online far in the past so the >=30s branch fires an update.
	p := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "live"
		p.Uuid = "uuid-live"
		p.Version = "1.2.3"
		p.Os = "linux"
		p.LastOnlineTime = 0
	})

	i := &api.Index{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/heartbeat", i.Heartbeat)

	rec := httptest.NewRecorder()
	body := `{"id":"live","uuid":"uuid-live","modified_at":0}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/heartbeat", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	// No strategy assigned => modified_at: 0 in the response.
	assert.Contains(t, rec.Body.String(), `"modified_at":0`)

	var reloaded model.Peer
	require.NoError(t, kit.DB.Where("row_id = ?", p.RowId).First(&reloaded).Error)
	assert.GreaterOrEqual(t, reloaded.LastOnlineTime, time.Now().Unix()-5)
}

func TestHeartbeat_IncompletePeerRequestsSysinfo(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// Empty Version/Os => handler sets "sysinfo": true.
	testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "bare"
		p.Uuid = "uuid-bare"
		p.Version = ""
		p.Os = ""
		p.LastOnlineTime = time.Now().Unix()
	})

	i := &api.Index{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/heartbeat", i.Heartbeat)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/heartbeat", `{"id":"bare","uuid":"uuid-bare"}`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"sysinfo":true`)
}

// --- Peer (sysinfo) ---

func TestSysInfo_CreatesPeer(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	p := &api.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/sysinfo", p.SysInfo)

	rec := httptest.NewRecorder()
	body := `{"id":"pc-1","uuid":"uuid-pc-1","hostname":"box","os":"linux","version":"1.0","username":"u"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/sysinfo", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "SYSINFO_UPDATED", rec.Body.String())

	var pe model.Peer
	require.NoError(t, kit.DB.Where("id = ?", "pc-1").First(&pe).Error)
	assert.Equal(t, "box", pe.Hostname)
	assert.Equal(t, "linux", pe.Os)
}

func TestSysInfo_UpdatesExistingPeer(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	testutil.CreatePeer(t, kit.DB, func(pe *model.Peer) {
		pe.Id = "pc-2"
		pe.Uuid = "uuid-pc-2"
		pe.Hostname = "old"
		pe.Os = "windows"
	})

	p := &api.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/sysinfo", p.SysInfo)

	rec := httptest.NewRecorder()
	body := `{"id":"pc-2","uuid":"uuid-pc-2","hostname":"new","os":"linux","version":"2.0"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/sysinfo", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "SYSINFO_UPDATED", rec.Body.String())

	var pe model.Peer
	require.NoError(t, kit.DB.Where("id = ?", "pc-2").First(&pe).Error)
	assert.Equal(t, "new", pe.Hostname)
	assert.Equal(t, "linux", pe.Os)
}

func TestSysInfo_BadJSON(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	p := &api.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/sysinfo", p.SysInfo)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/sysinfo", `{bad`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "ParamsError")
}

func TestSysInfoVer(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	p := &api.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/sysinfo_ver", p.SysInfoVer)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/sysinfo_ver", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	// Body is "<version>\n<startTime>"; startTime is always populated.
	assert.Contains(t, rec.Body.String(), "\n")
}

// --- Device (CLI) ---

func TestDeviceCli_CreatesPeerForUser(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	d := &api.Device{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/devices/cli", authMiddleware(u), d.Cli)

	rec := httptest.NewRecorder()
	body := `{"id":"cli-1","uuid":"uuid-cli-1","device_name":"laptop","device_username":"bob","note":"n"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/devices/cli", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var pe model.Peer
	require.NoError(t, kit.DB.Where("id = ?", "cli-1").First(&pe).Error)
	assert.Equal(t, u.Id, pe.UserId)
	assert.Equal(t, "laptop", pe.Hostname)
	assert.Equal(t, "bob", pe.Username)
	assert.Equal(t, "n", pe.Note)
}

func TestDeviceCli_ValidationError(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	d := &api.Device{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/devices/cli", authMiddleware(u), d.Cli)

	rec := httptest.NewRecorder()
	// Missing required id/uuid; ShouldBindJSON validate tags fail.
	req := testutil.JSONRequest(t, http.MethodPost, "/api/devices/cli", `{"note":"x"}`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "ParamsError")
}

var _ = json.Marshal
