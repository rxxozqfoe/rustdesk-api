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

func TestGroup_Create_And_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)
	engine.GET("/list", cont.List)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"name":"devs","type":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var g model.Group
	require.NoError(t, kit.DB.Where("name = ?", "devs").First(&g).Error)

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 0, env2.Code)
	assert.Contains(t, rec2.Body.String(), "devs")
}

func TestGroup_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"type":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestGroup_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	g := testutil.CreateGroup(t, kit.DB, func(g *model.Group) { g.Name = "qa" })

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(g.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "qa")

	_, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestGroup_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	g := testutil.CreateGroup(t, kit.DB, func(g *model.Group) { g.Name = "old" })

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(g.Id)+`,"name":"new"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.Group
	require.NoError(t, kit.DB.First(&got, g.Id).Error)
	assert.Equal(t, "new", got.Name)
}

func TestGroup_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	g := testutil.CreateGroup(t, kit.DB)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(g.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Group{}).Where("id = ?", g.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestGroup_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}
