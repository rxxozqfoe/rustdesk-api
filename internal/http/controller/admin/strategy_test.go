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

func TestStrategy_Create_And_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Strategy{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)
	engine.GET("/list", cont.List)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"name":"default-strategy"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var s model.Strategy
	require.NoError(t, kit.DB.Where("name = ?", "default-strategy").First(&s).Error)
	// Create should have populated a guid.
	assert.NotEmpty(t, s.Guid)

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 0, env2.Code)
	assert.Contains(t, rec2.Body.String(), "default-strategy")
}

func TestStrategy_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Strategy{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestStrategy_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Strategy{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	s := &model.Strategy{Name: "detail-strategy", Guid: "guid-1", Enabled: model.BoolPtr(true)}
	require.NoError(t, kit.DB.Create(s).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(s.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "detail-strategy")

	_, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestStrategy_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Strategy{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	s := &model.Strategy{Name: "gone-strategy", Guid: "guid-2", Enabled: model.BoolPtr(true)}
	require.NoError(t, kit.DB.Create(s).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(s.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Strategy{}).Where("id = ?", s.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}
