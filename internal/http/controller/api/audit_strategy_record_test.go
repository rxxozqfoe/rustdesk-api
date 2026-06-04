package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Audit ---

func TestAuditConn_New(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	a := &api.Audit{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/audit/conn", a.AuditConn)

	rec := httptest.NewRecorder()
	body := `{"action":"new","conn_id":7,"id":"peer-a","peer":["peer-b","Bob"],"ip":"1.2.3.4","type":1,"uuid":"u1"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/audit/conn", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":0`)

	var ac model.AuditConn
	require.NoError(t, kit.DB.Where("peer_id = ? AND conn_id = ?", "peer-a", 7).First(&ac).Error)
	assert.Equal(t, "peer-b", ac.FromPeer)
	assert.Equal(t, "Bob", ac.FromName)
}

func TestAuditConn_Close(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// Seed an existing open conn.
	require.NoError(t, kit.DB.Create(&model.AuditConn{
		Action: model.AuditActionNew, ConnId: 9, PeerId: "peer-c",
	}).Error)

	a := &api.Audit{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/audit/conn", a.AuditConn)

	rec := httptest.NewRecorder()
	body := `{"action":"close","conn_id":9,"id":"peer-c"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/audit/conn", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var ac model.AuditConn
	require.NoError(t, kit.DB.Where("peer_id = ? AND conn_id = ?", "peer-c", 9).First(&ac).Error)
	assert.NotZero(t, ac.CloseTime)
}

func TestAuditConn_BadJSON(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	a := &api.Audit{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/audit/conn", a.AuditConn)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/audit/conn", `{bad`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "ParamsError")
}

func TestAuditFile_Creates(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	a := &api.Audit{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/audit/file", a.AuditFile)

	rec := httptest.NewRecorder()
	body := `{"id":"peer-f","is_file":true,"path":"/tmp/x","peer_id":"peer-g","type":1,"uuid":"uf","info":"{\"ip\":\"9.9.9.9\",\"name\":\"alice\",\"num\":2}"}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/audit/file", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":0`)

	var af model.AuditFile
	require.NoError(t, kit.DB.Where("peer_id = ?", "peer-f").First(&af).Error)
	assert.Equal(t, "alice", af.FromName)
	assert.Equal(t, "9.9.9.9", af.Ip)
	assert.Equal(t, 2, af.Num)
}

// --- Strategy ---

func TestStrategyList(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	require.NoError(t, kit.Services.StrategyService.Create(&model.Strategy{Name: "s1"}))
	require.NoError(t, kit.Services.StrategyService.Create(&model.Strategy{Name: "s2"}))

	sc := &api.StrategyController{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/strategies", sc.List)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/strategies", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &arr))
	assert.Len(t, arr, 2)
}

func TestStrategyDetail_FoundAndNotFound(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	s := &model.Strategy{Name: "s1"}
	require.NoError(t, kit.Services.StrategyService.Create(s))

	sc := &api.StrategyController{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/strategies/:guid", sc.Detail)

	// Found
	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/strategies/"+s.Guid, "")
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"name":"s1"`)

	// Not found
	rec2 := httptest.NewRecorder()
	req2 := testutil.JSONRequest(t, http.MethodGet, "/api/strategies/nonexistent-guid", "")
	engine.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "Strategy not found")
}

func TestStrategyUpdateStatus(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	s := &model.Strategy{Name: "s1"}
	require.NoError(t, kit.Services.StrategyService.Create(s))

	sc := &api.StrategyController{HD: hd}
	engine := testutil.NewEngine(t)
	engine.PUT("/api/strategies/:guid/status", sc.UpdateStatus)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPut, "/api/strategies/"+s.Guid+"/status", `false`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	reloaded := kit.Services.StrategyService.InfoById(s.Id)
	require.NotNil(t, reloaded.Enabled)
	assert.False(t, *reloaded.Enabled)
}

func TestStrategyAssign_ToPeer(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	s := &model.Strategy{Name: "s1"}
	require.NoError(t, kit.Services.StrategyService.Create(s))
	p := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "assignpeer"
		p.Uuid = "uuid-assignpeer"
	})

	sc := &api.StrategyController{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/strategies/assign", sc.Assign)

	rec := httptest.NewRecorder()
	body := `{"strategy":"` + s.Guid + `","peers":["assignpeer"]}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/strategies/assign", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var count int64
	kit.DB.Model(&model.StrategyPeer{}).Where("peer_row_id = ? AND strategy_id = ?", p.RowId, s.Id).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestStrategyAssign_BadGuid(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	sc := &api.StrategyController{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/strategies/assign", sc.Assign)

	rec := httptest.NewRecorder()
	body := `{"strategy":"missing-guid","peers":["x"]}`
	req := testutil.JSONRequest(t, http.MethodPost, "/api/strategies/assign", body)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Strategy not found")
}

// --- Record ---

func TestRecord_NewCreatesFile(t *testing.T) {
	kit := servicekit.New(t)
	dir := t.TempDir()
	kit.Config.Gin.ResourcesPath = dir
	hd := newDeps(t, kit)

	r := &api.Record{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/record", r.Upload)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=new&file=clip.mp4", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	_, err := os.Stat(filepath.Join(dir, "public", "records", "clip.mp4"))
	require.NoError(t, err)
}

func TestRecord_MissingFileParam(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	r := &api.Record{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/record", r.Upload)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=new", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "file parameter is required")
}

func TestRecord_InvalidType(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	r := &api.Record{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/record", r.Upload)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=bogus&file=x", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid type parameter")
}

func TestRecord_PartThenRemove(t *testing.T) {
	kit := servicekit.New(t)
	dir := t.TempDir()
	kit.Config.Gin.ResourcesPath = dir
	hd := newDeps(t, kit)

	r := &api.Record{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/record", r.Upload)

	// "new" first: only this op creates the records directory (the "part" op
	// opens with O_CREATE but never MkdirAll), mirroring the real client flow.
	recNew := httptest.NewRecorder()
	reqNew := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=new&file=seg.bin", "")
	engine.ServeHTTP(recNew, reqNew)
	require.Equal(t, http.StatusOK, recNew.Code)

	// Write a part at offset 0.
	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=part&file=seg.bin&offset=0", "hello")
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	fp := filepath.Join(dir, "public", "records", "seg.bin")
	data, err := os.ReadFile(fp)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	// Remove it.
	rec2 := httptest.NewRecorder()
	req2 := testutil.JSONRequest(t, http.MethodPost, "/api/record?type=remove&file=seg.bin", "")
	engine.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	_, err = os.Stat(fp)
	assert.True(t, os.IsNotExist(err))
}
