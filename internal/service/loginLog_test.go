package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newLoginLogService(t *testing.T) (*LoginLogService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.LoginLogService, db
}

func TestLoginLog_CRUD(t *testing.T) {
	s, _ := newLoginLogService(t)
	l := &model.LoginLog{UserId: 1, DeviceId: "d1", Type: model.LoginLogTypeAccount}
	require.NoError(t, s.Create(l))
	assert.NotZero(t, l.Id)
	assert.Equal(t, l.Id, s.InfoById(l.Id).Id)

	require.NoError(t, s.Delete(l))
	assert.Zero(t, s.InfoById(l.Id).Id)
}

func TestLoginLog_ListAndBatchDelete(t *testing.T) {
	s, _ := newLoginLogService(t)
	a := &model.LoginLog{UserId: 1, Type: model.LoginLogTypeAccount}
	b := &model.LoginLog{UserId: 1, Type: model.LoginLogTypeOauth}
	require.NoError(t, s.Create(a))
	require.NoError(t, s.Create(b))

	assert.EqualValues(t, 2, s.List(1, 100, nil).Total)
	assert.EqualValues(t, 1, s.List(1, 100, func(tx *gorm.DB) { tx.Where("type = ?", model.LoginLogTypeOauth) }).Total)

	require.NoError(t, s.BatchDelete([]uint{a.Id, b.Id}))
	assert.EqualValues(t, 0, s.List(1, 100, nil).Total)
}

func TestLoginLog_SoftDelete(t *testing.T) {
	s, _ := newLoginLogService(t)
	l := &model.LoginLog{UserId: 1, Type: model.LoginLogTypeAccount}
	require.NoError(t, s.Create(l))

	require.NoError(t, s.SoftDelete(l))
	// row still present, but flagged deleted
	assert.EqualValues(t, model.IsDeletedYes, s.InfoById(l.Id).IsDeleted)
}

func TestLoginLog_BatchSoftDelete_ScopedToUser(t *testing.T) {
	s, _ := newLoginLogService(t)
	mine := &model.LoginLog{UserId: 1, Type: model.LoginLogTypeAccount}
	other := &model.LoginLog{UserId: 2, Type: model.LoginLogTypeAccount}
	require.NoError(t, s.Create(mine))
	require.NoError(t, s.Create(other))

	// attempt to soft-delete both ids but as user 1 only
	require.NoError(t, s.BatchSoftDelete(1, []uint{mine.Id, other.Id}))

	assert.EqualValues(t, model.IsDeletedYes, s.InfoById(mine.Id).IsDeleted)
	assert.EqualValues(t, model.IsDeletedNo, s.InfoById(other.Id).IsDeleted, "other user's log must not be touched")
}
