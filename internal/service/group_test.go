package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newGroupService(t *testing.T) (*GroupService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.GroupService, db
}

func TestGroup_CRUD(t *testing.T) {
	s, _ := newGroupService(t)
	g := &model.Group{Name: "team", Type: model.GroupTypeDefault}
	require.NoError(t, s.Create(g))
	assert.NotZero(t, g.Id)
	assert.Equal(t, "team", s.InfoById(g.Id).Name)

	g.Name = "renamed"
	require.NoError(t, s.Update(g))
	assert.Equal(t, "renamed", s.InfoById(g.Id).Name)

	require.NoError(t, s.Delete(g))
	assert.Zero(t, s.InfoById(g.Id).Id)
}

func TestGroup_ListPaginationAndFilter(t *testing.T) {
	s, _ := newGroupService(t)
	require.NoError(t, s.Create(&model.Group{Name: "g1", Type: model.GroupTypeDefault}))
	require.NoError(t, s.Create(&model.Group{Name: "g2", Type: model.GroupTypeShare}))
	require.NoError(t, s.Create(&model.Group{Name: "g3", Type: model.GroupTypeShare}))

	all := s.List(1, 2, nil)
	assert.EqualValues(t, 3, all.Total)
	assert.Len(t, all.Groups, 2)

	shared := s.List(1, 100, func(tx *gorm.DB) { tx.Where("type = ?", model.GroupTypeShare) })
	assert.EqualValues(t, 2, shared.Total)
}

func TestDeviceGroup_CRUD(t *testing.T) {
	s, _ := newGroupService(t)
	dg := &model.DeviceGroup{Name: "devices"}
	require.NoError(t, s.DeviceGroupCreate(dg))
	assert.NotZero(t, dg.Id)
	assert.Equal(t, "devices", s.DeviceGroupInfoById(dg.Id).Name)

	dg.Name = "updated"
	require.NoError(t, s.DeviceGroupUpdate(dg))
	assert.Equal(t, "updated", s.DeviceGroupInfoById(dg.Id).Name)

	res := s.DeviceGroupList(1, 100, nil)
	assert.EqualValues(t, 1, res.Total)

	require.NoError(t, s.DeviceGroupDelete(dg))
	assert.Zero(t, s.DeviceGroupInfoById(dg.Id).Id)
}
