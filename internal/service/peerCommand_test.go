package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newPeerCommandService(t *testing.T) (*PeerCommandService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.PeerCommandService, db
}

func TestPeerCommand_CreateDisconnectAndPending(t *testing.T) {
	s, _ := newPeerCommandService(t)
	require.NoError(t, s.CreateDisconnect("peer-1", []int{1, 2, 3}))

	pending := s.PendingByPeerId("peer-1")
	require.Len(t, pending, 1)
	assert.Equal(t, model.PeerCommandDisconnect, pending[0].Command)
	assert.Equal(t, "[1,2,3]", pending[0].Payload)

	assert.Empty(t, s.PendingByPeerId("other-peer"))
}

func TestPeerCommand_CreateDisconnectAll(t *testing.T) {
	s, _ := newPeerCommandService(t)
	// empty conn ids -> payload is an empty json array (disconnect all)
	require.NoError(t, s.CreateDisconnect("peer-2", []int{}))
	pending := s.PendingByPeerId("peer-2")
	require.Len(t, pending, 1)
	assert.Equal(t, "[]", pending[0].Payload)
}

func TestPeerCommand_DeleteByIds(t *testing.T) {
	s, db := newPeerCommandService(t)
	require.NoError(t, s.CreateDisconnect("peer-3", []int{1}))
	require.NoError(t, s.CreateDisconnect("peer-3", []int{2}))

	var cmds []*model.PeerCommand
	require.NoError(t, db.Where("peer_id = ?", "peer-3").Find(&cmds).Error)
	require.Len(t, cmds, 2)

	s.DeleteByIds([]uint{cmds[0].Id, cmds[1].Id})
	assert.Empty(t, s.PendingByPeerId("peer-3"))
}

func TestPeerCommand_DeleteByIds_EmptyNoop(t *testing.T) {
	s, _ := newPeerCommandService(t)
	require.NoError(t, s.CreateDisconnect("peer-4", []int{1}))
	// empty slice must be a no-op (not delete everything)
	s.DeleteByIds(nil)
	assert.Len(t, s.PendingByPeerId("peer-4"), 1)
}
