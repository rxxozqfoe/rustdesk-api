package admin_test

import (
	"net/http"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeer_Create_And_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)
	engine.GET("/list", cont.List)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"id":"900900900","hostname":"workstation","os":"linux","uuid":"u-1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var p model.Peer
	require.NoError(t, kit.DB.Where("id = ?", "900900900").First(&p).Error)

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 0, env2.Code)
	assert.Contains(t, rec2.Body.String(), "workstation")
}

func TestPeer_List_FilterByHostname(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "111"
		p.Uuid = "ua"
		p.Hostname = "alpha-box"
	})
	testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "222"
		p.Uuid = "ub"
		p.Hostname = "beta-box"
	})

	rec, _ := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10&hostname=alpha", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "alpha-box")
	assert.NotContains(t, rec.Body.String(), "beta-box")
}

func TestPeer_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	p := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "333"
		p.Uuid = "uc"
		p.Hostname = "gamma"
	})

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(p.RowId), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "gamma")

	_, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestPeer_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	p := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "444"
		p.Uuid = "ud"
		p.Hostname = "old-host"
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"row_id":`+itoa(p.RowId)+`,"id":"444","hostname":"new-host"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.Peer
	require.NoError(t, kit.DB.First(&got, p.RowId).Error)
	assert.Equal(t, "new-host", got.Hostname)
}

func TestPeer_Update_MissingRowIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	rec, env := doJSON(t, engine, http.MethodPost, "/update", `{"id":"444"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestPeer_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	p := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "555"
		p.Uuid = "ue"
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"row_id":`+itoa(p.RowId)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Peer{}).Where("row_id = ?", p.RowId).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestPeer_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"row_id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestPeer_BatchDelete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/batchDelete", cont.BatchDelete)

	p1 := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) { p.Id = "601"; p.Uuid = "u601" })
	p2 := testutil.CreatePeer(t, kit.DB, func(p *model.Peer) { p.Id = "602"; p.Uuid = "u602" })

	rec, env := doJSON(t, engine, http.MethodPost, "/batchDelete",
		`{"row_ids":[`+itoa(p1.RowId)+`,`+itoa(p2.RowId)+`]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Peer{}).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestPeer_BatchDelete_EmptyFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Peer{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/batchDelete", cont.BatchDelete)

	rec, env := doJSON(t, engine, http.MethodPost, "/batchDelete", `{"row_ids":[]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}
