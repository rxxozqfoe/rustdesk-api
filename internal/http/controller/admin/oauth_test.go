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

func TestOauth_Create_Github(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// github type: FormatOauthInfo sets Op = "github" automatically.
	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"oauth_type":"github","client_id":"cid","client_secret":"secret"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var oa model.Oauth
	require.NoError(t, kit.DB.Where("op = ?", model.OauthTypeGithub).First(&oa).Error)
	assert.Equal(t, "cid", oa.ClientId)
}

func TestOauth_Create_ValidationFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	// missing oauth_type, client_id, client_secret (all validate required).
	rec, env := doJSON(t, engine, http.MethodPost, "/create", `{}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
}

func TestOauth_Create_DuplicateOp(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/create", cont.Create)

	require.NoError(t, kit.DB.Create(&model.Oauth{Op: model.OauthTypeGithub, OauthType: model.OauthTypeGithub}).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/create",
		`{"oauth_type":"github","client_id":"cid","client_secret":"secret"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemExists")
}

func TestOauth_List(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/list", cont.List)

	require.NoError(t, kit.DB.Create(&model.Oauth{Op: "github", OauthType: "github", ClientId: "ghcid"}).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/list?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "ghcid")
}

func TestOauth_Detail_FoundAndNotFound(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/detail/:id", cont.Detail)

	oa := &model.Oauth{Op: "github", OauthType: "github", ClientId: "detail-cid"}
	require.NoError(t, kit.DB.Create(oa).Error)

	rec, env := doJSON(t, engine, http.MethodGet, "/detail/"+itoa(oa.Id), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)
	assert.Contains(t, rec.Body.String(), "detail-cid")

	_, env2 := doJSON(t, engine, http.MethodGet, "/detail/99999", "")
	assert.Equal(t, 101, env2.Code)
	assert.Contains(t, env2.Message, "ItemNotFound")
}

func TestOauth_Update(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	oa := &model.Oauth{Op: "github", OauthType: "github", ClientId: "old-cid"}
	require.NoError(t, kit.DB.Create(oa).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"id":`+itoa(oa.Id)+`,"oauth_type":"github","client_id":"new-cid","client_secret":"s"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var got model.Oauth
	require.NoError(t, kit.DB.First(&got, oa.Id).Error)
	assert.Equal(t, "new-cid", got.ClientId)
}

func TestOauth_Update_MissingIdFails(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/update", cont.Update)

	rec, env := doJSON(t, engine, http.MethodPost, "/update",
		`{"oauth_type":"github","client_id":"c","client_secret":"s"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestOauth_Delete(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	oa := &model.Oauth{Op: "github", OauthType: "github"}
	require.NoError(t, kit.DB.Create(oa).Error)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":`+itoa(oa.Id)+`}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, env.Code)

	var cnt int64
	require.NoError(t, kit.DB.Model(&model.Oauth{}).Where("id = ?", oa.Id).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestOauth_Delete_NotFound(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/delete", cont.Delete)

	rec, env := doJSON(t, engine, http.MethodPost, "/delete", `{"id":99999}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestOauth_Info_MissingCode(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/info", cont.Info)

	rec, env := doJSON(t, engine, http.MethodGet, "/info", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}

func TestOauth_Info_UnknownCode(t *testing.T) {
	hd, _ := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/info", cont.Info)

	rec, env := doJSON(t, engine, http.MethodGet, "/info?code=does-not-exist", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ItemNotFound")
}

func TestOauth_Confirm_ExpiredCode(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)

	u, token := seedTokenUser(t, kit)
	engine.POST("/confirm", withCurUser(u, token), cont.Confirm)

	// No cache entry for this code -> OauthExpired.
	rec, env := doJSON(t, engine, http.MethodPost, "/confirm", `{"code":"missing-code"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "OauthExpired")
}

func TestOauth_Confirm_MissingCode(t *testing.T) {
	hd, kit := newDeps(t)
	cont := &admin.Oauth{HD: hd}
	engine := testutil.NewEngine(t)

	u, token := seedTokenUser(t, kit)
	engine.POST("/confirm", withCurUser(u, token), cont.Confirm)

	rec, env := doJSON(t, engine, http.MethodPost, "/confirm", `{"code":""}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 101, env.Code)
	assert.Contains(t, env.Message, "ParamsError")
}
