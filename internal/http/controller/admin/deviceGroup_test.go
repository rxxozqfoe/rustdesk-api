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

func TestDeviceGroup_Create_And_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)
	engine.GET("/list", cont.List)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"name":"fleet"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var dg model.DeviceGroup
	require.NoError(t, kit.DB.Where("name = ?", "fleet").First(&dg).Error)

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 0, env2.Code)
	assert.Contains(t, rec2.Body.String(), "fleet")
}

func TestDeviceGroup_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestDeviceGroup_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	dg := &model.DeviceGroup{Name: "squad"}
	require.NoError(t, kit.DB.Create(dg).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(dg.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "squad")

	_, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestDeviceGroup_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	dg := &model.DeviceGroup{Name: "old-dg"}
	require.NoError(t, kit.DB.Create(dg).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(dg.Id)+`,"name":"new-dg"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.DeviceGroup
	require.NoError(t, kit.DB.First(&got, dg.Id).Error)
	assert.Equal(t, "new-dg", got.Name)
}

func TestDeviceGroup_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	dg := &model.DeviceGroup{Name: "gone-dg"}
	require.NoError(t, kit.DB.Create(dg).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(dg.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.DeviceGroup{}).Where("id = ?", dg.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestDeviceGroup_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.DeviceGroup{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}
