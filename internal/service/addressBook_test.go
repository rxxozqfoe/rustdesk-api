package service

import (
	"encoding/json"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAddressBookService(t *testing.T) (*AddressBookService, *gorm.DB) {
	t.Helper()
	svc, db := newServiceAggregate(t)
	return svc.AddressBookService, db
}

func TestAddressBook_CreateAndInfo(t *testing.T) {
	s, _ := newAddressBookService(t)
	ab := &model.AddressBook{Id: "dev-1", UserId: 1, Username: "u", Hostname: "h"}
	require.NoError(t, s.Create(ab))
	assert.NotZero(t, ab.RowId)

	assert.Equal(t, ab.RowId, s.Info("dev-1").RowId)
	assert.Equal(t, ab.RowId, s.InfoByRowId(ab.RowId).RowId)
	assert.Equal(t, ab.RowId, s.InfoByUserIdAndId(1, "dev-1").RowId)
	assert.Zero(t, s.InfoByUserIdAndId(2, "dev-1").RowId, "wrong owner -> not found")
}

func TestAddressBook_Update(t *testing.T) {
	s, _ := newAddressBookService(t)
	ab := &model.AddressBook{Id: "dev-2", UserId: 1, Alias: "old"}
	require.NoError(t, s.Create(ab))

	ab.Alias = "new"
	require.NoError(t, s.Update(ab))
	assert.Equal(t, "new", s.InfoByRowId(ab.RowId).Alias)
}

func TestAddressBook_Delete(t *testing.T) {
	s, _ := newAddressBookService(t)
	ab := &model.AddressBook{Id: "dev-3", UserId: 1}
	require.NoError(t, s.Create(ab))
	require.NoError(t, s.Delete(ab))
	assert.Zero(t, s.InfoByRowId(ab.RowId).RowId)
}

func TestAddressBook_ListByUserId(t *testing.T) {
	s, _ := newAddressBookService(t)
	require.NoError(t, s.Create(&model.AddressBook{Id: "a", UserId: 1}))
	require.NoError(t, s.Create(&model.AddressBook{Id: "b", UserId: 1}))
	require.NoError(t, s.Create(&model.AddressBook{Id: "c", UserId: 2}))

	res := s.ListByUserId(1, 1, 100)
	assert.EqualValues(t, 2, res.Total)

	res2 := s.ListByUserIds([]uint{1, 2}, 1, 100)
	assert.EqualValues(t, 3, res2.Total)
}

func TestAddressBook_ListPagination(t *testing.T) {
	s, _ := newAddressBookService(t)
	for _, id := range []string{"p1", "p2", "p3"} {
		require.NoError(t, s.Create(&model.AddressBook{Id: id, UserId: 9}))
	}
	res := s.List(1, 2, func(tx *gorm.DB) { tx.Where("user_id = ?", 9) })
	assert.EqualValues(t, 3, res.Total)
	assert.Len(t, res.AddressBooks, 2)
}

func TestPlatformFromOs(t *testing.T) {
	s, _ := newAddressBookService(t)
	tests := map[string]string{
		"Android 12":     "Android",
		"Windows 10":     "Windows",
		"Linux x86":      "Linux",
		"Mac OS X":       "Mac OS",
		"some-bsd-thing": "",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, s.PlatformFromOs(in))
		})
	}
}

func TestAddressBook_FromPeer(t *testing.T) {
	s, _ := newAddressBookService(t)
	p := &model.Peer{Id: "px", Username: "pu", Hostname: "ph", UserId: 3, Os: "Windows 11"}
	ab := s.FromPeer(p)
	assert.Equal(t, "px", ab.Id)
	assert.Equal(t, "pu", ab.Username)
	assert.Equal(t, "ph", ab.Hostname)
	assert.EqualValues(t, 3, ab.UserId)
	assert.Equal(t, "Windows", ab.Platform)
}

// UpdateAddressBook diffs incoming entries against the DB: add new, update
// existing, delete missing.
func TestUpdateAddressBook_AddUpdateDelete(t *testing.T) {
	s, _ := newAddressBookService(t)
	// seed two entries for user 1
	require.NoError(t, s.Create(&model.AddressBook{Id: "keep", UserId: 1, Alias: "before", Platform: "Linux", Username: "u", Hostname: "h"}))
	require.NoError(t, s.Create(&model.AddressBook{Id: "drop", UserId: 1, Platform: "Linux", Username: "u", Hostname: "h"}))

	// incoming: keep (updated), new (added); drop is omitted (deleted)
	incoming := []*model.AddressBook{
		{Id: "keep", Alias: "after", Platform: "Linux", Username: "u", Hostname: "h"},
		{Id: "new", Alias: "fresh", Platform: "Linux", Username: "u", Hostname: "h"},
	}
	require.NoError(t, s.UpdateAddressBook(incoming, 1))

	res := s.ListByUserId(1, 1, 100)
	assert.EqualValues(t, 2, res.Total, "drop removed, keep+new remain")
	assert.Equal(t, "after", s.InfoByUserIdAndId(1, "keep").Alias)
	assert.NotZero(t, s.InfoByUserIdAndId(1, "new").RowId)
	assert.Zero(t, s.InfoByUserIdAndId(1, "drop").RowId)
}

func TestBatchUpdateTags(t *testing.T) {
	s, _ := newAddressBookService(t)
	a := &model.AddressBook{Id: "t1", UserId: 1}
	b := &model.AddressBook{Id: "t2", UserId: 1}
	require.NoError(t, s.Create(a))
	require.NoError(t, s.Create(b))

	require.NoError(t, s.BatchUpdateTags([]*model.AddressBook{a, b}, []string{"red", "blue"}))

	got := s.InfoByRowId(a.RowId)
	var tags []string
	require.NoError(t, json.Unmarshal(got.Tags, &tags))
	assert.ElementsMatch(t, []string{"red", "blue"}, tags)
}

// --- Collections & rules ---

func TestCollection_CRUDAndOwner(t *testing.T) {
	s, _ := newAddressBookService(t)
	c := &model.AddressBookCollection{UserId: 5, Name: "Work"}
	require.NoError(t, s.CreateCollection(c))
	assert.NotZero(t, c.Id)

	assert.Equal(t, "Work", s.CollectionInfoById(c.Id).Name)
	assert.True(t, s.CheckCollectionOwner(5, c.Id))
	assert.False(t, s.CheckCollectionOwner(6, c.Id))

	c.Name = "Home"
	require.NoError(t, s.UpdateCollection(c))
	assert.Equal(t, "Home", s.CollectionInfoById(c.Id).Name)

	res := s.ListCollectionByUserId(5)
	assert.EqualValues(t, 1, res.Total)
}

func TestDeleteCollection_CascadesRulesAndBooks(t *testing.T) {
	s, db := newAddressBookService(t)
	c := &model.AddressBookCollection{UserId: 5, Name: "C"}
	require.NoError(t, s.CreateCollection(c))
	require.NoError(t, s.Create(&model.AddressBook{Id: "cab", UserId: 5, CollectionId: c.Id}))
	require.NoError(t, s.CreateRule(&model.AddressBookCollectionRule{CollectionId: c.Id, Type: model.ShareAddressBookRuleTypePersonal, ToId: 9, Rule: model.ShareAddressBookRuleRuleRead}))

	require.NoError(t, s.DeleteCollection(c))
	assert.Zero(t, s.CollectionInfoById(c.Id).Id)

	var abs, rules int64
	db.Model(&model.AddressBook{}).Where("collection_id = ?", c.Id).Count(&abs)
	db.Model(&model.AddressBookCollectionRule{}).Where("collection_id = ?", c.Id).Count(&rules)
	assert.EqualValues(t, 0, abs)
	assert.EqualValues(t, 0, rules)
}

// --- privilege resolution via rules ---

func TestUserMaxRule_OwnerIsFullControl(t *testing.T) {
	s, _ := newAddressBookService(t)
	user := &model.User{GroupId: 1}
	user.Id = 10
	// owner (uid == user.Id) always gets full control
	assert.Equal(t, model.ShareAddressBookRuleRuleFullControl, s.UserMaxRule(user, 10, 1))
}

func TestUserMaxRule_PersonalAndGroupRules(t *testing.T) {
	s, _ := newAddressBookService(t)
	user := &model.User{GroupId: 7}
	user.Id = 10
	cid := uint(3)

	// no rules yet -> 0
	assert.Equal(t, 0, s.UserMaxRule(user, 99, cid))

	// personal read rule
	require.NoError(t, s.CreateRule(&model.AddressBookCollectionRule{
		CollectionId: cid, Type: model.ShareAddressBookRuleTypePersonal, ToId: 10, Rule: model.ShareAddressBookRuleRuleRead,
	}))
	assert.Equal(t, model.ShareAddressBookRuleRuleRead, s.UserMaxRule(user, 99, cid))
	assert.True(t, s.CheckUserReadPrivilege(user, 99, cid))
	assert.False(t, s.CheckUserWritePrivilege(user, 99, cid))

	// group rule with higher privilege wins
	require.NoError(t, s.CreateRule(&model.AddressBookCollectionRule{
		CollectionId: cid, Type: model.ShareAddressBookRuleTypeGroup, ToId: 7, Rule: model.ShareAddressBookRuleRuleReadWrite,
	}))
	assert.Equal(t, model.ShareAddressBookRuleRuleReadWrite, s.UserMaxRule(user, 99, cid))
	assert.True(t, s.CheckUserWritePrivilege(user, 99, cid))
}

func TestRule_CRUD(t *testing.T) {
	s, _ := newAddressBookService(t)
	r := &model.AddressBookCollectionRule{CollectionId: 1, Type: model.ShareAddressBookRuleTypePersonal, ToId: 4, Rule: model.ShareAddressBookRuleRuleRead}
	require.NoError(t, s.CreateRule(r))
	assert.Equal(t, r.Id, s.RuleInfoById(r.Id).Id)
	assert.Equal(t, r.Id, s.RulePersonalInfoByToIdAndCid(4, 1).Id)

	r.Rule = model.ShareAddressBookRuleRuleReadWrite
	require.NoError(t, s.UpdateRule(r))
	assert.Equal(t, model.ShareAddressBookRuleRuleReadWrite, s.RuleInfoById(r.Id).Rule)

	require.NoError(t, s.DeleteRule(r))
	assert.Zero(t, s.RuleInfoById(r.Id).Id)
}
