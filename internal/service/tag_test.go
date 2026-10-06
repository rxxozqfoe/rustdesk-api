package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTagService(t *testing.T) (*TagService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.TagService, db
}

func TestTag_CreateAndInfo(t *testing.T) {
	s, _ := newTagService(t)
	tag := &model.Tag{Name: "prod", UserId: 1, Color: 0xFF0000FF}
	require.NoError(t, s.Create(tag))
	assert.NotZero(t, tag.Id)

	assert.Equal(t, "prod", s.Info(tag.Id).Name)
	assert.Equal(t, "prod", s.InfoById(tag.Id).Name)
	assert.Equal(t, tag.Id, s.InfoByUserIdAndNameAndCollectionId(1, "prod", 0).Id)
	assert.Zero(t, s.InfoByUserIdAndNameAndCollectionId(2, "prod", 0).Id, "wrong user -> not found")
}

func TestTag_Update(t *testing.T) {
	s, _ := newTagService(t)
	tag := &model.Tag{Name: "n", UserId: 1, Color: 1}
	require.NoError(t, s.Create(tag))

	tag.Color = 99
	require.NoError(t, s.Update(tag))
	assert.EqualValues(t, 99, s.InfoById(tag.Id).Color)
}

func TestTag_Delete(t *testing.T) {
	s, _ := newTagService(t)
	tag := &model.Tag{Name: "d", UserId: 1}
	require.NoError(t, s.Create(tag))
	require.NoError(t, s.Delete(tag))
	assert.Zero(t, s.InfoById(tag.Id).Id)
}

func TestTag_ListByUserId(t *testing.T) {
	s, _ := newTagService(t)
	require.NoError(t, s.Create(&model.Tag{Name: "a", UserId: 1}))
	require.NoError(t, s.Create(&model.Tag{Name: "b", UserId: 1}))
	require.NoError(t, s.Create(&model.Tag{Name: "c", UserId: 2}))

	assert.EqualValues(t, 2, s.ListByUserId(1).Total)
}

func TestTag_ListByUserIdAndCollectionId_Ordered(t *testing.T) {
	s, _ := newTagService(t)
	require.NoError(t, s.Create(&model.Tag{Name: "zeta", UserId: 1, CollectionId: 3}))
	require.NoError(t, s.Create(&model.Tag{Name: "alpha", UserId: 1, CollectionId: 3}))
	require.NoError(t, s.Create(&model.Tag{Name: "other", UserId: 1, CollectionId: 9}))

	res := s.ListByUserIdAndCollectionId(1, 3)
	require.EqualValues(t, 2, res.Total)
	// ordered by name asc
	assert.Equal(t, "alpha", res.Tags[0].Name)
	assert.Equal(t, "zeta", res.Tags[1].Name)
}

// UpdateTags reconciles a tag-name->color map against existing tags:
// add new, update changed colors, delete missing.
func TestUpdateTags_Reconcile(t *testing.T) {
	s, _ := newTagService(t)
	require.NoError(t, s.Create(&model.Tag{Name: "keep", UserId: 1, Color: 1}))
	require.NoError(t, s.Create(&model.Tag{Name: "recolor", UserId: 1, Color: 1}))
	require.NoError(t, s.Create(&model.Tag{Name: "remove", UserId: 1, Color: 1}))

	s.UpdateTags(1, map[string]uint{
		"keep":    1,  // unchanged
		"recolor": 42, // changed color
		"added":   7,  // new
		// "remove" omitted -> deleted
	})

	res := s.ListByUserId(1)
	assert.EqualValues(t, 3, res.Total)

	byName := map[string]uint{}
	for _, tg := range res.Tags {
		byName[tg.Name] = tg.Color
	}
	assert.EqualValues(t, 1, byName["keep"])
	assert.EqualValues(t, 42, byName["recolor"])
	assert.EqualValues(t, 7, byName["added"])
	_, hasRemoved := byName["remove"]
	assert.False(t, hasRemoved)
}
