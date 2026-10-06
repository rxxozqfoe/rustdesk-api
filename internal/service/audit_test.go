package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAuditService(t *testing.T) (*AuditService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.AuditService, db
}

func TestAuditConn_CRUDAndLookup(t *testing.T) {
	s, _ := newAuditService(t)
	c := &model.AuditConn{PeerId: "peer-1", ConnId: 100}
	require.NoError(t, s.CreateAuditConn(c))
	assert.NotZero(t, c.Id)

	assert.Equal(t, c.Id, s.ConnInfoById(c.Id).Id)
	assert.Equal(t, c.Id, s.InfoByPeerIdAndConnId("peer-1", 100).Id)
	assert.Zero(t, s.InfoByPeerIdAndConnId("peer-1", 999).Id)

	c.PeerId = "peer-2"
	require.NoError(t, s.UpdateAuditConn(c))
	assert.Equal(t, "peer-2", s.ConnInfoById(c.Id).PeerId)

	require.NoError(t, s.DeleteAuditConn(c))
	assert.Zero(t, s.ConnInfoById(c.Id).Id)
}

func TestAuditConn_ListAndBatchDelete(t *testing.T) {
	s, _ := newAuditService(t)
	a := &model.AuditConn{PeerId: "p", ConnId: 1}
	b := &model.AuditConn{PeerId: "p", ConnId: 2}
	require.NoError(t, s.CreateAuditConn(a))
	require.NoError(t, s.CreateAuditConn(b))

	assert.EqualValues(t, 2, s.AuditConnList(1, 100, nil).Total)

	require.NoError(t, s.BatchDeleteAuditConn([]uint{a.Id, b.Id}))
	assert.EqualValues(t, 0, s.AuditConnList(1, 100, nil).Total)
}

func TestCloseStaleConns(t *testing.T) {
	s, db := newAuditService(t)
	open1 := &model.AuditConn{PeerId: "p", ConnId: 1, CloseTime: 0}
	open2 := &model.AuditConn{PeerId: "p", ConnId: 2, CloseTime: 0}
	closed := &model.AuditConn{PeerId: "p", ConnId: 3, CloseTime: 12345}
	require.NoError(t, s.CreateAuditConn(open1))
	require.NoError(t, s.CreateAuditConn(open2))
	require.NoError(t, s.CreateAuditConn(closed))

	require.NoError(t, s.CloseStaleConns())

	// the two open conns now have a non-zero close time; the already-closed one is unchanged
	var stillOpen int64
	db.Model(&model.AuditConn{}).Where("close_time = ?", 0).Count(&stillOpen)
	assert.EqualValues(t, 0, stillOpen)
	assert.EqualValues(t, 12345, s.ConnInfoById(closed.Id).CloseTime)
}

func TestAuditFile_CRUDAndBatchDelete(t *testing.T) {
	s, _ := newAuditService(t)
	f := &model.AuditFile{PeerId: "peer-f"}
	require.NoError(t, s.CreateAuditFile(f))
	assert.Equal(t, f.Id, s.FileInfoById(f.Id).Id)

	f.PeerId = "peer-f2"
	require.NoError(t, s.UpdateAuditFile(f))
	assert.Equal(t, "peer-f2", s.FileInfoById(f.Id).PeerId)

	assert.EqualValues(t, 1, s.AuditFileList(1, 100, nil).Total)

	require.NoError(t, s.BatchDeleteAuditFile([]uint{f.Id}))
	assert.Zero(t, s.FileInfoById(f.Id).Id)
}
