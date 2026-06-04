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

func TestLoginLog_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	require.NoError(t, kit.DB.Create(&model.LoginLog{UserId: 1, Ip: "10.0.0.1", Client: "webadmin"}).Error)
	require.NoError(t, kit.DB.Create(&model.LoginLog{UserId: 2, Ip: "10.0.0.2", Client: "webadmin"}).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "10.0.0.1")
	assert.Contains(t, rec.Body.String(), "10.0.0.2")
}

func TestLoginLog_List_FilterByUser(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	require.NoError(t, kit.DB.Create(&model.LoginLog{UserId: 1, Ip: "10.0.0.1"}).Error)
	require.NoError(t, kit.DB.Create(&model.LoginLog{UserId: 2, Ip: "10.0.0.2"}).Error)

	rec, _ := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10&user_id=1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "10.0.0.1")
	assert.NotContains(t, rec.Body.String(), "10.0.0.2")
}

func TestLoginLog_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	ll := &model.LoginLog{UserId: 1, Ip: "10.0.0.9"}
	require.NoError(t, kit.DB.Create(ll).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(ll.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
}

func TestLoginLog_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestLoginLog_BatchDelete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/batchDelete", cont.BatchDelete)

	l1 := &model.LoginLog{UserId: 1, Ip: "1.1.1.1"}
	l2 := &model.LoginLog{UserId: 1, Ip: "2.2.2.2"}
	require.NoError(t, kit.DB.Create(l1).Error)
	require.NoError(t, kit.DB.Create(l2).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/batchDelete",
		`{"ids":[`+itoa(l1.Id)+`,`+itoa(l2.Id)+`]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
}

func TestLoginLog_BatchDelete_EmptyFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.LoginLog{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/batchDelete", cont.BatchDelete)

	rec, env := doJSON(t, engine, http.MethodPost, "/batchDelete", `{"ids":[]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}
