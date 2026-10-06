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

func TestAb_Fetch(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)
	// One address-book entry and one tag belonging to the user, collection 0.
	testutil.CreateAddressBook(t, kit.DB, func(ab *model.AddressBook) {
		ab.UserId = u.Id
		ab.Id = "peer-1"
		ab.CollectionId = 0
	})
	require.NoError(t, kit.DB.Create(&model.Tag{UserId: u.Id, Name: "work", Color: 42, CollectionId: 0}).Error)

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.GET("/api/ab", authMiddleware(u), a.Ab)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/ab", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Envelope: {"data": "<json string>", "licensed_devices": 999}
	var outer struct {
		Data            string `json:"data"`
		LicensedDevices int    `json:"licensed_devices"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outer))
	assert.Equal(t, 999, outer.LicensedDevices)

	var inner struct {
		Peers     []map[string]any `json:"peers"`
		Tags      []string         `json:"tags"`
		TagColors string           `json:"tag_colors"`
	}
	require.NoError(t, json.Unmarshal([]byte(outer.Data), &inner))
	require.Len(t, inner.Peers, 1)
	assert.Equal(t, "peer-1", inner.Peers[0]["id"])
	assert.Contains(t, inner.Tags, "work")
	assert.Contains(t, inner.TagColors, "work")
}

func TestUpAb_PersistsPeersAndTags(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/ab", authMiddleware(u), a.UpAb)

	// The outer form has a single "data" string field that itself is JSON.
	dataObj := map[string]any{
		"tags": []string{"t1"},
		"peers": []map[string]any{
			{"id": "newpeer", "username": "bob", "hostname": "h1", "platform": "Linux"},
		},
		"tag_colors": `{"t1":7}`,
	}
	dataBytes, _ := json.Marshal(dataObj)
	outer, _ := json.Marshal(map[string]string{"data": string(dataBytes)})

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/ab", string(outer))
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// The peer should now exist in the user's address book.
	var ab model.AddressBook
	require.NoError(t, kit.DB.Where("user_id = ? AND id = ?", u.Id, "newpeer").First(&ab).Error)
	assert.Equal(t, "bob", ab.Username)

	// The tag should have been created/updated for the user.
	var tag model.Tag
	require.NoError(t, kit.DB.Where("user_id = ? AND name = ?", u.Id, "t1").First(&tag).Error)
	assert.Equal(t, uint(7), tag.Color)
}

func TestUpAb_BadJSON(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/ab", authMiddleware(u), a.UpAb)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/ab", `{not-json`)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "ParamsError")
}

func TestAbSettings(t *testing.T) {
	kit := servicekit.New(t)
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/ab/settings", authMiddleware(u), a.Settings)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/ab/settings", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "max_peer_one_ab")
}

func TestAbPersonal_Enabled(t *testing.T) {
	kit := servicekit.New(t)
	kit.Config.Rustdesk.Personal = 1
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB, func(u *model.User) {
		u.Username = "carol"
		u.GroupId = 1
	})

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/ab/personal", authMiddleware(u), a.Personal)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/ab/personal", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Guid string `json:"guid"`
		Name string `json:"name"`
		Rule int    `json:"rule"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, "carol", res.Name)
	assert.Equal(t, 3, res.Rule)
	// ComposeGuid => "<groupId>-<userId>-0"
	assert.Equal(t, "1-"+itoa(u.Id)+"-0", res.Guid)
}

func TestAbPersonal_Disabled(t *testing.T) {
	kit := servicekit.New(t)
	kit.Config.Rustdesk.Personal = 0
	hd := newDeps(t, kit)
	u := testutil.CreateUser(t, kit.DB)

	a := &api.Ab{HD: hd}
	engine := testutil.NewEngine(t)
	engine.POST("/api/ab/personal", authMiddleware(u), a.Personal)

	rec := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodPost, "/api/ab/personal", "")
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "null", rec.Body.String())
}

// itoa is a tiny uint-to-decimal helper to avoid importing strconv just for
// one assertion.
func itoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// silence unused import linter if assertions above ever shrink.
var _ = model.AddressBook{}
