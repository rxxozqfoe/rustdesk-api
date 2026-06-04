package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newPeerService(t *testing.T) (*PeerService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.PeerService, db
}

func TestPeer_CreateAndFinders(t *testing.T) {
	s, _ := newPeerService(t)
	p := &model.Peer{Id: "id-1", Uuid: "uuid-1", UserId: 5}
	require.NoError(t, s.Create(p))
	assert.NotZero(t, p.RowId)

	assert.Equal(t, p.RowId, s.FindById("id-1").RowId)
	assert.Equal(t, p.RowId, s.FindByUuid("uuid-1").RowId)
	assert.Equal(t, p.RowId, s.InfoByRowId(p.RowId).RowId)
	assert.Equal(t, p.RowId, s.FindByUserIdAndUuid("uuid-1", 5).RowId)
	assert.Zero(t, s.FindByUserIdAndUuid("uuid-1", 6).RowId, "wrong user -> not found")
	assert.Zero(t, s.FindById("missing").RowId)
}

func TestPeer_Update(t *testing.T) {
	s, _ := newPeerService(t)
	p := &model.Peer{Id: "id-2", Uuid: "uuid-2", Alias: "old"}
	require.NoError(t, s.Create(p))

	p.Alias = "new"
	require.NoError(t, s.Update(p))
	assert.Equal(t, "new", s.InfoByRowId(p.RowId).Alias)
}

func TestPeer_DeleteAlsoFlushesTokens(t *testing.T) {
	s, db := newPeerService(t)
	p := &model.Peer{Id: "id-3", Uuid: "uuid-3"}
	require.NoError(t, s.Create(p))
	require.NoError(t, db.Create(&model.UserToken{UserId: 1, Token: "tk", DeviceUuid: "uuid-3"}).Error)

	require.NoError(t, s.Delete(p))
	assert.Zero(t, s.InfoByRowId(p.RowId).RowId)

	var tokens int64
	db.Model(&model.UserToken{}).Where("device_uuid = ?", "uuid-3").Count(&tokens)
	assert.EqualValues(t, 0, tokens, "tokens for the peer uuid should be flushed")
}

func TestPeer_UuidBindUserId(t *testing.T) {
	s, _ := newPeerService(t)
	p := &model.Peer{Id: "id-4", Uuid: "uuid-4", UserId: 0}
	require.NoError(t, s.Create(p))

	s.UuidBindUserId("id-4", "uuid-4", 42)
	assert.EqualValues(t, 42, s.FindByUuid("uuid-4").UserId)
}

func TestPeer_UuidBindUserId_NoPeerNoCreate(t *testing.T) {
	s, db := newPeerService(t)
	// binding a uuid that has no peer must NOT create a peer
	s.UuidBindUserId("ghost-id", "ghost-uuid", 1)
	var cnt int64
	db.Model(&model.Peer{}).Count(&cnt)
	assert.EqualValues(t, 0, cnt)
}

func TestPeer_UuidUnbindUserId(t *testing.T) {
	s, _ := newPeerService(t)
	p := &model.Peer{Id: "id-5", Uuid: "uuid-5", UserId: 7}
	require.NoError(t, s.Create(p))

	s.UuidUnbindUserId("uuid-5", 7)
	assert.EqualValues(t, 0, s.FindByUuid("uuid-5").UserId)
}

func TestPeer_EraseUserId(t *testing.T) {
	s, _ := newPeerService(t)
	require.NoError(t, s.Create(&model.Peer{Id: "e1", Uuid: "eu1", UserId: 9}))
	require.NoError(t, s.Create(&model.Peer{Id: "e2", Uuid: "eu2", UserId: 9}))
	require.NoError(t, s.Create(&model.Peer{Id: "e3", Uuid: "eu3", UserId: 8}))

	require.NoError(t, s.EraseUserId(9))
	assert.EqualValues(t, 0, s.FindByUuid("eu1").UserId)
	assert.EqualValues(t, 0, s.FindByUuid("eu2").UserId)
	assert.EqualValues(t, 8, s.FindByUuid("eu3").UserId, "other users untouched")
}

func TestPeer_ListByUserIdsAndFilter(t *testing.T) {
	s, _ := newPeerService(t)
	require.NoError(t, s.Create(&model.Peer{Id: "l1", Uuid: "lu1", UserId: 1}))
	require.NoError(t, s.Create(&model.Peer{Id: "l2", Uuid: "lu2", UserId: 1}))
	require.NoError(t, s.Create(&model.Peer{Id: "l3", Uuid: "lu3", UserId: 2}))

	assert.EqualValues(t, 2, s.ListByUserIds([]uint{1}, 1, 100).Total)
	assert.EqualValues(t, 3, s.ListByUserIds([]uint{1, 2}, 1, 100).Total)

	// ListFilterByUserId combines the user filter with an extra predicate
	res := s.ListFilterByUserId(1, 100, func(tx *gorm.DB) { tx.Where("id = ?", "l1") }, 1)
	assert.EqualValues(t, 1, res.Total)
}

func TestPeer_GetUuidListByIDsFiltersEmpty(t *testing.T) {
	s, _ := newPeerService(t)
	p1 := &model.Peer{Id: "u1", Uuid: "has-uuid"}
	p2 := &model.Peer{Id: "u2", Uuid: ""} // empty uuid filtered out
	require.NoError(t, s.Create(p1))
	require.NoError(t, s.Create(p2))

	uuids, err := s.GetUuidListByIDs([]uint{p1.RowId, p2.RowId})
	require.NoError(t, err)
	assert.Equal(t, []string{"has-uuid"}, uuids)
}

func TestPeer_BatchDelete(t *testing.T) {
	s, db := newPeerService(t)
	p1 := &model.Peer{Id: "bd1", Uuid: "bdu1"}
	p2 := &model.Peer{Id: "bd2", Uuid: "bdu2"}
	require.NoError(t, s.Create(p1))
	require.NoError(t, s.Create(p2))
	require.NoError(t, db.Create(&model.UserToken{UserId: 1, Token: "x", DeviceUuid: "bdu1"}).Error)

	require.NoError(t, s.BatchDelete([]uint{p1.RowId, p2.RowId}))

	var peers, tokens int64
	db.Model(&model.Peer{}).Count(&peers)
	db.Model(&model.UserToken{}).Where("device_uuid = ?", "bdu1").Count(&tokens)
	assert.EqualValues(t, 0, peers)
	assert.EqualValues(t, 0, tokens)
}
