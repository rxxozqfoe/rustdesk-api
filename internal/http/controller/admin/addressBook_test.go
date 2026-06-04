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

func TestAddressBook_Create_Success(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	u := testutil.CreateUser(t, kit.DB)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"id":"700700700","user_id":`+itoa(u.Id)+`,"hostname":"box","username":"peer"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var ab model.AddressBook
	require.NoError(t, kit.DB.Where("id = ? AND user_id = ?", "700700700", u.Id).First(&ab).Error)
	assert.Equal(t, "box", ab.Hostname)
}

func TestAddressBook_Create_MissingUserIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// id passes validation (required), but user_id == 0 -> ParamsError.
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"id":"700700700"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestAddressBook_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// id missing (validate required).
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"user_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestAddressBook_Create_Duplicate(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	u := testutil.CreateUser(t, kit.DB)
	testutil.CreateAddressBook(t, kit.DB, func(ab *model.AddressBook) {
		ab.Id = "700700700"
		ab.UserId = u.Id
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"id":"700700700","user_id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemExists")
}

func TestAddressBook_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	u := testutil.CreateUser(t, kit.DB)
	testutil.CreateAddressBook(t, kit.DB, func(ab *model.AddressBook) {
		ab.Id = "800800800"
		ab.UserId = u.Id
		ab.Hostname = "listed-host"
	})

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "listed-host")
}

func TestAddressBook_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	u := testutil.CreateUser(t, kit.DB)
	ab := testutil.CreateAddressBook(t, kit.DB, func(ab *model.AddressBook) {
		ab.Id = "900"
		ab.UserId = u.Id
		ab.Hostname = "before"
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"row_id":`+itoa(ab.RowId)+`,"id":"900","user_id":`+itoa(u.Id)+`,"hostname":"after"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.AddressBook
	require.NoError(t, kit.DB.First(&got, "row_id = ?", ab.RowId).Error)
	assert.Equal(t, "after", got.Hostname)
}

func TestAddressBook_Update_MissingRowIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":"900","user_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestAddressBook_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	u := testutil.CreateUser(t, kit.DB)
	ab := testutil.CreateAddressBook(t, kit.DB, func(ab *model.AddressBook) {
		ab.Id = "950"
		ab.UserId = u.Id
	})

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"row_id":`+itoa(ab.RowId)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.AddressBook{}).Where("row_id = ?", ab.RowId).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestAddressBook_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.AddressBook{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"row_id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}
