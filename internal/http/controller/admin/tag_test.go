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

func TestTag_Create_Success(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	u := testutil.CreateUser(t, kit.DB)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"name":"prod","color":4278190080,"user_id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var tg model.Tag
	require.NoError(t, kit.DB.Where("name = ?", "prod").First(&tg).Error)
	assert.Equal(t, u.Id, tg.UserId)
}

func TestTag_Create_MissingUserIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// color provided so validation passes, but user_id == 0 triggers ParamsError.
	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"name":"prod","color":4278190080}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestTag_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// Missing name and color (both validate required).
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"user_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestTag_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	u := testutil.CreateUser(t, kit.DB)
	require.NoError(t, kit.DB.Create(&model.Tag{Name: "alpha", Color: 1, UserId: u.Id}).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "alpha")
}

func TestTag_Detail_AdminAccess(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)

	owner := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "owner" })
	tg := &model.Tag{Name: "secret", Color: 1, UserId: owner.Id}
	require.NoError(t, kit.DB.Create(tg).Error)

	admN := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "boss"
		b := true
		u.IsAdmin = &b
	})
	engine.GET("/detail/:id", withCurUser(admN, "tok"), cont.Detail)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(tg.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "secret")
}

func TestTag_Detail_NonOwnerDenied(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)

	owner := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "owner" })
	tg := &model.Tag{Name: "secret", Color: 1, UserId: owner.Id}
	require.NoError(t, kit.DB.Create(tg).Error)

	other := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.Username = "other" })
	engine.GET("/detail/:id", withCurUser(other, "tok"), cont.Detail)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(tg.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "NoAccess")
}

func TestTag_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	u := testutil.CreateUser(t, kit.DB)
	tg := &model.Tag{Name: "old", Color: 1, UserId: u.Id}
	require.NoError(t, kit.DB.Create(tg).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(tg.Id)+`,"name":"new","color":2,"user_id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.Tag
	require.NoError(t, kit.DB.First(&got, tg.Id).Error)
	assert.Equal(t, "new", got.Name)
}

func TestTag_Update_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":99999,"name":"x","color":1,"user_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestTag_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	u := testutil.CreateUser(t, kit.DB)
	tg := &model.Tag{Name: "gone", Color: 1, UserId: u.Id}
	require.NoError(t, kit.DB.Create(tg).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(tg.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Tag{}).Where("id = ?", tg.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestTag_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Tag{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}
