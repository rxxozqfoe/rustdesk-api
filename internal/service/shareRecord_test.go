package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newShareRecordService(t *testing.T) (*ShareRecordService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.ShareRecordService, db
}

func TestShareRecord_CRUD(t *testing.T) {
	s, _ := newShareRecordService(t)
	r := &model.ShareRecord{UserId: 1, ShareToken: "tok-1"}
	require.NoError(t, s.Create(r))
	assert.NotZero(t, r.Id)
	assert.Equal(t, r.Id, s.InfoById(r.Id).Id)

	r.ShareToken = "tok-2"
	require.NoError(t, s.Update(r))
	assert.Equal(t, "tok-2", s.InfoById(r.Id).ShareToken)

	require.NoError(t, s.Delete(r))
	assert.Zero(t, s.InfoById(r.Id).Id)
}

func TestShareRecord_ListAndBatchDelete(t *testing.T) {
	s, _ := newShareRecordService(t)
	a := &model.ShareRecord{UserId: 1, ShareToken: "a"}
	b := &model.ShareRecord{UserId: 2, ShareToken: "b"}
	require.NoError(t, s.Create(a))
	require.NoError(t, s.Create(b))

	assert.EqualValues(t, 2, s.List(1, 100, nil).Total)
	assert.EqualValues(t, 1, s.List(1, 100, func(tx *gorm.DB) { tx.Where("user_id = ?", 1) }).Total)

	require.NoError(t, s.BatchDelete([]uint{a.Id, b.Id}))
	assert.EqualValues(t, 0, s.List(1, 100, nil).Total)
}

// AddressBookService.ShareByWebClient assigns a uuid token; SharedPeer fetches by token.
func TestAddressBook_ShareByWebClientAndSharedPeer(t *testing.T) {
	svc, _ := newServiceAggregate(t)
	abs := svc.AddressBookService
	rec := &model.ShareRecord{UserId: 1}
	require.NoError(t, abs.ShareByWebClient(rec))
	assert.NotEmpty(t, rec.ShareToken, "ShareByWebClient generates a uuid token")

	found := abs.SharedPeer(rec.ShareToken)
	assert.Equal(t, rec.Id, found.Id)
}
