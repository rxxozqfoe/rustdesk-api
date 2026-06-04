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

func TestAddressBookCollection_Create_Success(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	u := testutil.CreateUser(t, kit.DB)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"name":"work","user_id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var col model.AddressBookCollection
	require.NoError(t, kit.DB.Where("name = ?", "work").First(&col).Error)
	assert.Equal(t, u.Id, col.UserId)
}

func TestAddressBookCollection_Create_MissingUserIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// name passes validation (required), user_id == 0 -> ParamsError.
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"name":"work"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestAddressBookCollection_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// name missing (validate required).
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"user_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestAddressBookCollection_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	u := testutil.CreateUser(t, kit.DB)
	require.NoError(t, kit.DB.Create(&model.AddressBookCollection{Name: "listed-col", UserId: u.Id}).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "listed-col")
}

func TestAddressBookCollection_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	u := testutil.CreateUser(t, kit.DB)
	col := &model.AddressBookCollection{Name: "detail-col", UserId: u.Id}
	require.NoError(t, kit.DB.Create(col).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(col.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "detail-col")

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestAddressBookCollection_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	u := testutil.CreateUser(t, kit.DB)
	col := &model.AddressBookCollection{Name: "old-col", UserId: u.Id}
	require.NoError(t, kit.DB.Create(col).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(col.Id)+`,"name":"new-col","user_id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.AddressBookCollection
	require.NoError(t, kit.DB.First(&got, col.Id).Error)
	assert.Equal(t, "new-col", got.Name)
}

func TestAddressBookCollection_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	u := testutil.CreateUser(t, kit.DB)
	col := &model.AddressBookCollection{Name: "gone-col", UserId: u.Id}
	require.NoError(t, kit.DB.Create(col).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(col.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.AddressBookCollection{}).Where("id = ?", col.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestAddressBookCollection_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBookCollection{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}
