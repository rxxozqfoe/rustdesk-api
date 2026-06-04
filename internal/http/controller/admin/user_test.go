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

func userEngine(t *testing.T) (*admin.User, *testKit) {
	hd, kit := newDeps(t)
	return &admin.User{HD: hd}, &testKit{kit: kit}
}

func TestUser_Create_Success(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"username":"newuser","group_id":1,"status":1}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var u model.User
	require.NoError(t, k.kit.DB.Where("username = ?", "newuser").First(&u).Error)
	assert.Equal(t, "newuser", u.Username)
}

func TestUser_Create_ValidationFails(t *testing.T) {
	cont, _ := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// Missing username (validate required, gte=2) and group_id (required).
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{"status":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestUser_Create_DuplicateUsername(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "dup" })

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"username":"dup","group_id":1,"status":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "OperationFailed")
}

func TestUser_List(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "alice" })
	testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "bob" })

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "alice")
	assert.Contains(t, rec.Body.String(), "bob")
}

func TestUser_List_FilterByUsername(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "alice" })
	testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "bob" })

	rec, _ := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10&username=ali", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "alice")
	assert.NotContains(t, rec.Body.String(), "bob")
}

func TestUser_Detail_FoundAndNotFound(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	u := testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "carol" })

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(u.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "carol")

	rec2, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestUser_Update(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	u := testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "dave" })

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(u.Id)+`,"username":"dave","group_id":1,"status":1,"nickname":"Updated"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.User
	require.NoError(t, k.kit.DB.First(&got, u.Id).Error)
	assert.Equal(t, "Updated", got.Nickname)
}

func TestUser_Update_MissingIdFails(t *testing.T) {
	cont, _ := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"username":"x","group_id":1,"status":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestUser_Delete(t *testing.T) {
	cont, k := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	u := testutil.CreateUser(t, k.kit.DB, func(u *model.User) { u.Username = "erin" })

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(u.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, k.kit.DB.Model(&model.User{}).Where("id = ?", u.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestUser_Delete_NotFound(t *testing.T) {
	cont, _ := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestUser_Delete_ZeroIdFails(t *testing.T) {
	cont, _ := userEngine(t)
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":0}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestUser_Current(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.User{HD: hd}
	engine := testutil.NewEngine(t)

	u, token := seedTokenUser(t, kit, func(u *model.User) { u.Username = "frank" })
	engine.GET("/current", withCurUser(u, token), cont.Current)

	rec, env := doJSON(t, engine, http.MethodGet, "/current", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), token)
}
