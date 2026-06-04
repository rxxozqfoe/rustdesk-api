package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil/servicekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestUserInfo(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "dave"
		u.Email = "dave@example.com"
		u.Nickname = "Dave"
	})

	uc := &api.User{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/user/info", authMiddleware(u), uc.Info)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/user/info", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		IsAdmin     *bool  `json:"is_admin"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, "dave", res.Name)
	assert.Equal(t, "dave@example.com", res.Email)
	assert.Equal(t, "Dave", res.DisplayName)
	require.NotNil(t, res.IsAdmin)
	assert.False(t, *res.IsAdmin)
}

func TestGroupUsers_NonAdminSeesSelfOnly(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	// A default (non-share) group; the non-admin user should only see themselves.
	g := testutil.CreateGroup(t, kit.DB, func(g *model.Group) { g.Type = model.GroupTypeDefault })
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "self"
		u.GroupId = g.Id
		u.IsAdmin = boolPtr(false)
	})
	// Another user in the same group that must NOT appear.
	testutil.CreateUser(t, kit.DB, func(o *model.User) {
		o.Username = "other"
		o.Email = "other@example.com"
		o.GroupId = g.Id
	})

	gc := &api.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/users", authMiddleware(u), gc.Users)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/users?pageSize=10", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Total uint `json:"total"`
		Data  []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, uint(1), res.Total)
	require.Len(t, res.Data, 1)
	assert.Equal(t, "self", res.Data[0].Name)
}

func TestGroupUsers_AdminSeesGroup(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	g := testutil.CreateGroup(t, kit.DB, func(g *model.Group) { g.Type = model.GroupTypeDefault })
	admin := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "admin"
		u.GroupId = g.Id
		u.IsAdmin = boolPtr(true)
	})
	testutil.CreateUser(t, kit.DB, func(o *model.User) {
		o.Username = "member"
		o.Email = "member@example.com"
		o.GroupId = g.Id
	})

	gc := &api.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/users", authMiddleware(admin), gc.Users)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/users?pageSize=10", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Total uint `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, uint(2), res.Total)
}

func TestGroupPeers_NonAdminSeesOwnPeers(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	g := testutil.CreateGroup(t, kit.DB, func(g *model.Group) { g.Type = model.GroupTypeDefault })
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "owner"
		u.GroupId = g.Id
		u.IsAdmin = boolPtr(false)
	})
	testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "mine"
		p.Uuid = "uuid-mine"
		p.UserId = u.Id
	})
	// A peer owned by someone else must not appear.
	other := testutil.CreateUser(t, kit.DB, func(o *model.User) {
		o.Username = "stranger"
		o.Email = "stranger@example.com"
		o.GroupId = g.Id
	})
	testutil.CreatePeer(t, kit.DB, func(p *model.Peer) {
		p.Id = "theirs"
		p.Uuid = "uuid-theirs"
		p.UserId = other.Id
	})

	gc := &api.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/peers", authMiddleware(u), gc.Peers)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/peers?pageSize=10", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Total uint `json:"total"`
		Data  []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Len(t, res.Data, 1)
	assert.Equal(t, "mine", res.Data[0].Id)
}

func TestGroupDevice_NonAdminDenied(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.IsAdmin = boolPtr(false) })

	gc := &api.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/device-group/accessible", authMiddleware(u), gc.Device)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/device-group/accessible", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Permission denied")
}

func TestGroupDevice_AdminOK(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) { u.IsAdmin = boolPtr(true) })
	require.NoError(t, kit.DB.Create(&model.DeviceGroup{Name: "dg1"}).Error)

	gc := &api.Group{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/device-group/accessible", authMiddleware(u), gc.Device)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/device-group/accessible", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Len(t, res.Data, 1)
	assert.Equal(t, "dg1", res.Data[0].Name)
}
