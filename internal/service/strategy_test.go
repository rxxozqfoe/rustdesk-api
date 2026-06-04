package service

import (
	"encoding/json"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with auto-migrated tables.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	err = db.AutoMigrate(
		&model.Strategy{},
		&model.StrategyPeer{},
		&model.StrategyUser{},
		&model.StrategyDeviceGroup{},
		&model.Peer{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

// newStrategyService creates a StrategyService backed by the given DB.
func newStrategyService(db *gorm.DB) *StrategyService {
	ctx := &ServiceContext{DB: db}
	return &StrategyService{ctx: ctx}
}

// makeJSON is a helper to build AutoJson from a map.
func makeJSON(m map[string]string) custom_types.AutoJson {
	b, _ := json.Marshal(m)
	return custom_types.AutoJson(b)
}

// --- CRUD Tests ---

func TestCreate_GeneratesGUID(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "test-strategy"}
	if err := ss.Create(s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if s.Guid == "" {
		t.Error("expected GUID to be generated, got empty string")
	}
	if s.Id == 0 {
		t.Error("expected Id to be set after Create")
	}
	// UUID format: 8-4-4-4-12
	if len(s.Guid) != 36 {
		t.Errorf("expected GUID length 36, got %d: %q", len(s.Guid), s.Guid)
	}
}

func TestCreate_DefaultsEmptyJSON(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "empty-json"}
	if err := ss.Create(s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Reload from DB
	loaded := ss.InfoById(s.Id)
	opts := ss.ConfigOptionsMap(loaded)
	extra := ss.ExtraMap(loaded)
	if len(opts) != 0 {
		t.Errorf("expected empty config_options map, got %v", opts)
	}
	if len(extra) != 0 {
		t.Errorf("expected empty extra map, got %v", extra)
	}
}

func TestCreate_WithConfigOptions(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	opts := map[string]string{
		"enable-file-transfer": "N",
		"enable-clipboard":     "N",
		"approve-mode":         "click",
	}
	s := &model.Strategy{
		Name:          "secure-policy",
		Enabled:       model.BoolPtr(true),
		ConfigOptions: makeJSON(opts),
	}
	if err := ss.Create(s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loaded := ss.InfoById(s.Id)
	got := ss.ConfigOptionsMap(loaded)
	for k, want := range opts {
		if got[k] != want {
			t.Errorf("config_options[%q] = %q, want %q", k, got[k], want)
		}
	}
}

func TestCreate_UniqueNameConstraint(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "duplicate"}
	if err := ss.Create(s1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	s2 := &model.Strategy{Name: "duplicate"}
	err := ss.Create(s2)
	if err == nil {
		t.Error("expected error for duplicate name, got nil")
	}
}

func TestInfoById(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "find-me"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	found := ss.InfoById(s.Id)
	if found.Name != "find-me" {
		t.Errorf("InfoById returned name %q, want %q", found.Name, "find-me")
	}
}

func TestInfoById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	found := ss.InfoById(99999)
	if found.Id != 0 {
		t.Errorf("expected Id=0 for non-existent strategy, got %d", found.Id)
	}
}

func TestInfoByGuid(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "guid-test"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	found := ss.InfoByGuid(s.Guid)
	if found.Id != s.Id {
		t.Errorf("InfoByGuid returned Id=%d, want %d", found.Id, s.Id)
	}
}

func TestInfoByName(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "name-lookup"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	found := ss.InfoByName("name-lookup")
	if found.Id != s.Id {
		t.Errorf("InfoByName returned Id=%d, want %d", found.Id, s.Id)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "original"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	s.Name = "updated"
	s.ConfigOptions = makeJSON(map[string]string{"enable-audio": "N"})
	if err := ss.Update(s); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	loaded := ss.InfoById(s.Id)
	if loaded.Name != "updated" {
		t.Errorf("name after update = %q, want %q", loaded.Name, "updated")
	}
	opts := ss.ConfigOptionsMap(loaded)
	if opts["enable-audio"] != "N" {
		t.Errorf("config_options[enable-audio] = %q, want %q", opts["enable-audio"], "N")
	}
}

func TestUpdate_DisableStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "will-disable", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	loaded := ss.InfoById(s.Id)
	if loaded.Enabled == nil || !*loaded.Enabled {
		t.Fatal("strategy should be enabled after creation")
	}

	s.Enabled = model.BoolPtr(false)
	if err := ss.Update(s); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	loaded = ss.InfoById(s.Id)
	if loaded.Enabled != nil && *loaded.Enabled {
		t.Error("strategy should be disabled after Update(enabled=false)")
	}
}

func TestSetEnabled(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "toggle", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := ss.SetEnabled(s.Id, false); err != nil {
		t.Fatalf("SetEnabled(false) failed: %v", err)
	}
	loaded := ss.InfoById(s.Id)
	if loaded.Enabled != nil && *loaded.Enabled {
		t.Error("expected enabled=false after SetEnabled(false)")
	}

	if err := ss.SetEnabled(s.Id, true); err != nil {
		t.Fatalf("SetEnabled(true) failed: %v", err)
	}
	loaded = ss.InfoById(s.Id)
	if loaded.Enabled == nil || !*loaded.Enabled {
		t.Error("expected enabled=true after SetEnabled(true)")
	}
}

func TestDelete_RemovesStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "to-delete"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := ss.Delete(s); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	loaded := ss.InfoById(s.Id)
	if loaded.Id != 0 {
		t.Error("strategy should not exist after deletion")
	}
}

func TestDelete_CascadesAssignments(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "cascade-test", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Create assignments of all three types
	if err := ss.AssignToPeer(s.Id, 100); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s.Id, 200); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(s.Id, 300); err != nil {
		t.Fatal(err)
	}

	if err := ss.Delete(s); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify all assignments are removed
	if peers := ss.PeerAssignments(s.Id); len(peers) != 0 {
		t.Errorf("expected 0 peer assignments after delete, got %d", len(peers))
	}
	if users := ss.UserAssignments(s.Id); len(users) != 0 {
		t.Errorf("expected 0 user assignments after delete, got %d", len(users))
	}
	if dgs := ss.DeviceGroupAssignments(s.Id); len(dgs) != 0 {
		t.Errorf("expected 0 device group assignments after delete, got %d", len(dgs))
	}
}

func TestList_Pagination(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	for i := 0; i < 5; i++ {
		if err := ss.Create(&model.Strategy{Name: "list-" + string(rune('A'+i))}); err != nil {
			t.Fatal(err)
		}
	}

	result := ss.List(1, 2, nil)
	if result.Total != 5 {
		t.Errorf("total = %d, want 5", result.Total)
	}
	if len(result.Strategies) != 2 {
		t.Errorf("page size = %d, want 2", len(result.Strategies))
	}
	if result.Page != 1 {
		t.Errorf("page = %d, want 1", result.Page)
	}
}

func TestCreate_EnabledFalse(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "want-disabled", Enabled: model.BoolPtr(false)}
	if err := ss.Create(s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loaded := ss.InfoById(s.Id)
	if loaded.Enabled != nil && *loaded.Enabled {
		t.Error("strategy should be disabled after Create with Enabled=false")
	}
}

func TestList_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "enabled-one", Enabled: model.BoolPtr(true)}
	s2 := &model.Strategy{Name: "disabled-one", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s1); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(s2); err != nil {
		t.Fatal(err)
	}
	// Use SetEnabled to reliably disable (avoids GORM zero-value bug)
	if err := ss.SetEnabled(s2.Id, false); err != nil {
		t.Fatal(err)
	}

	result := ss.List(1, 10, func(tx *gorm.DB) {
		tx.Where("enabled = ?", true)
	})
	if result.Total != 1 {
		t.Errorf("filtered total = %d, want 1", result.Total)
	}
}

func TestListAll(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	if err := ss.Create(&model.Strategy{Name: "all-1"}); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(&model.Strategy{Name: "all-2"}); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(&model.Strategy{Name: "all-3"}); err != nil {
		t.Fatal(err)
	}

	all := ss.ListAll()
	if len(all) != 3 {
		t.Errorf("ListAll returned %d, want 3", len(all))
	}
}

// --- Delete Error Path Tests ---

func TestDelete_ErrorOnPeerTable(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "del-err-peer", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Drop strategy_peers table to force error on first delete
	db.Exec("DROP TABLE strategy_peers")

	err := ss.Delete(s)
	if err == nil {
		t.Error("expected error when strategy_peers table is missing, got nil")
	}

	// Strategy should still exist (transaction rolled back)
	loaded := ss.InfoById(s.Id)
	if loaded.Id == 0 {
		t.Error("strategy should still exist after failed delete (transaction rollback)")
	}
}

func TestDelete_ErrorOnUserTable(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "del-err-user", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Drop strategy_users table to force error on second delete
	db.Exec("DROP TABLE strategy_users")

	err := ss.Delete(s)
	if err == nil {
		t.Error("expected error when strategy_users table is missing, got nil")
	}
}

func TestDelete_ErrorOnDeviceGroupTable(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "del-err-dg", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Drop strategy_device_groups table to force error on third delete
	db.Exec("DROP TABLE strategy_device_groups")

	err := ss.Delete(s)
	if err == nil {
		t.Error("expected error when strategy_device_groups table is missing, got nil")
	}
}

// --- Assignment Tests ---

func TestAssignToPeer(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "peer-assign"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := ss.AssignToPeer(s.Id, 10); err != nil {
		t.Fatalf("AssignToPeer failed: %v", err)
	}

	assignments := ss.PeerAssignments(s.Id)
	if len(assignments) != 1 {
		t.Fatalf("expected 1 peer assignment, got %d", len(assignments))
	}
	if assignments[0].PeerRowId != 10 {
		t.Errorf("PeerRowId = %d, want 10", assignments[0].PeerRowId)
	}
}

func TestAssignToPeer_ReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "old-strategy"}
	s2 := &model.Strategy{Name: "new-strategy"}
	if err := ss.Create(s1); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(s2); err != nil {
		t.Fatal(err)
	}

	// Assign peer 10 to strategy 1, then reassign to strategy 2
	if err := ss.AssignToPeer(s1.Id, 10); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToPeer(s2.Id, 10); err != nil {
		t.Fatal(err)
	}

	// Old strategy should have no peer assignments
	if peers := ss.PeerAssignments(s1.Id); len(peers) != 0 {
		t.Errorf("old strategy should have 0 peer assignments, got %d", len(peers))
	}
	// New strategy should have the peer
	peers := ss.PeerAssignments(s2.Id)
	if len(peers) != 1 {
		t.Fatalf("new strategy should have 1 peer assignment, got %d", len(peers))
	}
	if peers[0].PeerRowId != 10 {
		t.Errorf("PeerRowId = %d, want 10", peers[0].PeerRowId)
	}
}

func TestAssignToUser(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "user-assign"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s.Id, 20); err != nil {
		t.Fatal(err)
	}

	assignments := ss.UserAssignments(s.Id)
	if len(assignments) != 1 || assignments[0].UserId != 20 {
		t.Errorf("unexpected user assignments: %+v", assignments)
	}
}

func TestAssignToUser_ReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "old-user"}
	s2 := &model.Strategy{Name: "new-user"}
	if err := ss.Create(s1); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(s2); err != nil {
		t.Fatal(err)
	}

	if err := ss.AssignToUser(s1.Id, 20); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s2.Id, 20); err != nil {
		t.Fatal(err)
	}

	if users := ss.UserAssignments(s1.Id); len(users) != 0 {
		t.Errorf("old strategy should have 0 user assignments, got %d", len(users))
	}
	users := ss.UserAssignments(s2.Id)
	if len(users) != 1 || users[0].UserId != 20 {
		t.Errorf("unexpected: %+v", users)
	}
}

func TestAssignToDeviceGroup(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "dg-assign"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(s.Id, 30); err != nil {
		t.Fatal(err)
	}

	assignments := ss.DeviceGroupAssignments(s.Id)
	if len(assignments) != 1 || assignments[0].DeviceGroupId != 30 {
		t.Errorf("unexpected device group assignments: %+v", assignments)
	}
}

func TestAssignToDeviceGroup_ReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "old-dg"}
	s2 := &model.Strategy{Name: "new-dg"}
	if err := ss.Create(s1); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(s2); err != nil {
		t.Fatal(err)
	}

	if err := ss.AssignToDeviceGroup(s1.Id, 30); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(s2.Id, 30); err != nil {
		t.Fatal(err)
	}

	if dgs := ss.DeviceGroupAssignments(s1.Id); len(dgs) != 0 {
		t.Errorf("old strategy should have 0 dg assignments, got %d", len(dgs))
	}
	dgs := ss.DeviceGroupAssignments(s2.Id)
	if len(dgs) != 1 || dgs[0].DeviceGroupId != 30 {
		t.Errorf("unexpected: %+v", dgs)
	}
}

func TestUnassignPeer(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "unassign-peer"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToPeer(s.Id, 10); err != nil {
		t.Fatal(err)
	}

	if err := ss.UnassignPeer(10); err != nil {
		t.Fatalf("UnassignPeer failed: %v", err)
	}
	if peers := ss.PeerAssignments(s.Id); len(peers) != 0 {
		t.Errorf("expected 0 peer assignments after unassign, got %d", len(peers))
	}
}

func TestUnassignUser(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "unassign-user"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s.Id, 20); err != nil {
		t.Fatal(err)
	}

	if err := ss.UnassignUser(20); err != nil {
		t.Fatalf("UnassignUser failed: %v", err)
	}
	if users := ss.UserAssignments(s.Id); len(users) != 0 {
		t.Errorf("expected 0 user assignments after unassign, got %d", len(users))
	}
}

func TestUnassignDeviceGroup(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "unassign-dg"}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(s.Id, 30); err != nil {
		t.Fatal(err)
	}

	if err := ss.UnassignDeviceGroup(30); err != nil {
		t.Fatalf("UnassignDeviceGroup failed: %v", err)
	}
	if dgs := ss.DeviceGroupAssignments(s.Id); len(dgs) != 0 {
		t.Errorf("expected 0 dg assignments after unassign, got %d", len(dgs))
	}
}

// --- ResolveForPeer Tests (priority resolution) ---

func TestResolveForPeer_DirectPeerAssignment(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "peer-direct", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-1", UserId: 0, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToPeer(s.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}

	resolved := ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil, expected strategy")
	}
	if resolved.Id != s.Id {
		t.Errorf("resolved strategy Id=%d, want %d", resolved.Id, s.Id)
	}
}

func TestResolveForPeer_UserAssignment(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "user-level", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-2", UserId: 50, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToUser(s.Id, 50); err != nil {
		t.Fatal(err)
	}

	resolved := ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil, expected user-level strategy")
	}
	if resolved.Id != s.Id {
		t.Errorf("resolved strategy Id=%d, want %d", resolved.Id, s.Id)
	}
}

func TestResolveForPeer_DeviceGroupAssignment(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "group-level", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-3", UserId: 0, GroupId: 70}
	db.Create(peer)
	if err := ss.AssignToDeviceGroup(s.Id, 70); err != nil {
		t.Fatal(err)
	}

	resolved := ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil, expected group-level strategy")
	}
	if resolved.Id != s.Id {
		t.Errorf("resolved strategy Id=%d, want %d", resolved.Id, s.Id)
	}
}

// Design doc: "直接裝置指派 > 使用者指派 > 裝置群組指派"
func TestResolveForPeer_PriorityOrder(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	sPeer := &model.Strategy{Name: "priority-peer", Enabled: model.BoolPtr(true)}
	sUser := &model.Strategy{Name: "priority-user", Enabled: model.BoolPtr(true)}
	sGroup := &model.Strategy{Name: "priority-group", Enabled: model.BoolPtr(true)}
	if err := ss.Create(sPeer); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(sUser); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(sGroup); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-prio", UserId: 50, GroupId: 70}
	db.Create(peer)

	// Assign all three levels
	if err := ss.AssignToPeer(sPeer.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(sUser.Id, 50); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(sGroup.Id, 70); err != nil {
		t.Fatal(err)
	}

	// Should pick peer-level (highest priority)
	resolved := ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil")
	}
	if resolved.Id != sPeer.Id {
		t.Errorf("expected peer-level strategy (Id=%d), got Id=%d (%s)",
			sPeer.Id, resolved.Id, resolved.Name)
	}

	// Remove peer assignment → should fall through to user-level
	if err := ss.UnassignPeer(peer.RowId); err != nil {
		t.Fatal(err)
	}
	resolved = ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil after removing peer assignment")
	}
	if resolved.Id != sUser.Id {
		t.Errorf("expected user-level strategy (Id=%d), got Id=%d (%s)",
			sUser.Id, resolved.Id, resolved.Name)
	}

	// Remove user assignment → should fall through to device group
	if err := ss.UnassignUser(50); err != nil {
		t.Fatal(err)
	}
	resolved = ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("ResolveForPeer returned nil after removing user assignment")
	}
	if resolved.Id != sGroup.Id {
		t.Errorf("expected group-level strategy (Id=%d), got Id=%d (%s)",
			sGroup.Id, resolved.Id, resolved.Name)
	}

	// Remove group assignment → should return nil
	if err := ss.UnassignDeviceGroup(70); err != nil {
		t.Fatal(err)
	}
	resolved = ss.ResolveForPeer(peer)
	if resolved != nil {
		t.Errorf("expected nil after removing all assignments, got Id=%d", resolved.Id)
	}
}

// Design doc: "只回傳 enabled = true 的策略"
func TestResolveForPeer_SkipsDisabledStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "disabled", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-disabled", UserId: 0, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToPeer(s.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}

	// Disable the strategy
	if err := ss.SetEnabled(s.Id, false); err != nil {
		t.Fatal(err)
	}

	resolved := ss.ResolveForPeer(peer)
	if resolved != nil {
		t.Errorf("expected nil for disabled strategy, got Id=%d", resolved.Id)
	}
}

// When peer-level strategy is disabled, should fall through to user-level.
func TestResolveForPeer_FallsThroughDisabledPeerToUser(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	sPeer := &model.Strategy{Name: "disabled-peer", Enabled: model.BoolPtr(true)}
	sUser := &model.Strategy{Name: "enabled-user", Enabled: model.BoolPtr(true)}
	if err := ss.Create(sPeer); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(sUser); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-fallthrough", UserId: 50, GroupId: 0}
	db.Create(peer)

	if err := ss.AssignToPeer(sPeer.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(sUser.Id, 50); err != nil {
		t.Fatal(err)
	}

	// Disable peer-level strategy
	if err := ss.SetEnabled(sPeer.Id, false); err != nil {
		t.Fatal(err)
	}

	resolved := ss.ResolveForPeer(peer)
	if resolved == nil {
		t.Fatal("expected user-level strategy, got nil")
	}
	if resolved.Id != sUser.Id {
		t.Errorf("expected fallthrough to user-level strategy Id=%d, got Id=%d",
			sUser.Id, resolved.Id)
	}
}

// Design doc: "若無任何策略適用，回傳 nil"
func TestResolveForPeer_NoAssignment(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	peer := &model.Peer{Id: "peer-none", UserId: 0, GroupId: 0}
	db.Create(peer)

	resolved := ss.ResolveForPeer(peer)
	if resolved != nil {
		t.Errorf("expected nil for peer with no assignment, got Id=%d", resolved.Id)
	}
}

// Peer with UserId=0 should skip user-level lookup.
func TestResolveForPeer_SkipsUserWhenNoUserId(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "user-only", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s.Id, 50); err != nil {
		t.Fatal(err)
	}

	// Peer has no user association
	peer := &model.Peer{Id: "peer-no-user", UserId: 0, GroupId: 0}
	db.Create(peer)

	resolved := ss.ResolveForPeer(peer)
	if resolved != nil {
		t.Errorf("expected nil for peer without UserId, got Id=%d", resolved.Id)
	}
}

// Peer with GroupId=0 should skip device-group-level lookup.
func TestResolveForPeer_SkipsGroupWhenNoGroupId(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "group-only", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToDeviceGroup(s.Id, 70); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "peer-no-group", UserId: 0, GroupId: 0}
	db.Create(peer)

	resolved := ss.ResolveForPeer(peer)
	if resolved != nil {
		t.Errorf("expected nil for peer without GroupId, got Id=%d", resolved.Id)
	}
}

// --- JSON Deserialization Tests ---

func TestConfigOptionsMap(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	opts := map[string]string{
		"enable-keyboard":          "N",
		"approve-mode":             "click",
		"whitelist":                "192.168.1.0,10.0.0.0",
		"custom-rendezvous-server": "my.server.com",
	}
	s := &model.Strategy{
		Name:          "json-test",
		ConfigOptions: makeJSON(opts),
	}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	loaded := ss.InfoById(s.Id)
	got := ss.ConfigOptionsMap(loaded)
	for k, want := range opts {
		if got[k] != want {
			t.Errorf("ConfigOptionsMap[%q] = %q, want %q", k, got[k], want)
		}
	}
}

func TestExtraMap(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	extra := map[string]string{"custom-key": "custom-value"}
	s := &model.Strategy{
		Name:  "extra-test",
		Extra: makeJSON(extra),
	}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	loaded := ss.InfoById(s.Id)
	got := ss.ExtraMap(loaded)
	if got["custom-key"] != "custom-value" {
		t.Errorf("ExtraMap[custom-key] = %q, want %q", got["custom-key"], "custom-value")
	}
}

func TestConfigOptionsMap_EmptyJSON(t *testing.T) {
	ss := &StrategyService{}
	s := &model.Strategy{ConfigOptions: custom_types.AutoJson([]byte("{}"))}
	got := ss.ConfigOptionsMap(s)
	if len(got) != 0 {
		t.Errorf("expected empty map for {}, got %v", got)
	}
}

func TestConfigOptionsMap_NilJSON(t *testing.T) {
	ss := &StrategyService{}
	s := &model.Strategy{ConfigOptions: nil}
	got := ss.ConfigOptionsMap(s)
	if len(got) != 0 {
		t.Errorf("expected empty map for nil, got %v", got)
	}
}

// Verify AutoJson.Scan defaults empty data to "{}" (object),
// which is correct for Strategy config_options/extra fields.
func TestAutoJson_ScanEmptyDefaultsToObject(t *testing.T) {
	var j custom_types.AutoJson
	err := j.Scan("")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	raw := string(j)
	if raw != "{}" {
		t.Errorf("Scan empty default = %q, want %q", raw, "{}")
	}

	// Should unmarshal cleanly into map[string]string
	m := make(map[string]string)
	if err := json.Unmarshal(j, &m); err != nil {
		t.Errorf("json.Unmarshal of '{}' into map should succeed, got: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

// --- Multiple Assignments to Same Strategy ---

func TestMultiplePeersAssignedToSameStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "multi-peer", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := ss.AssignToPeer(s.Id, 10); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToPeer(s.Id, 20); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToPeer(s.Id, 30); err != nil {
		t.Fatal(err)
	}

	peers := ss.PeerAssignments(s.Id)
	if len(peers) != 3 {
		t.Errorf("expected 3 peer assignments, got %d", len(peers))
	}
}

func TestMultipleUsersAssignedToSameStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "multi-user", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := ss.AssignToUser(s.Id, 10); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToUser(s.Id, 20); err != nil {
		t.Fatal(err)
	}

	users := ss.UserAssignments(s.Id)
	if len(users) != 2 {
		t.Errorf("expected 2 user assignments, got %d", len(users))
	}
}

// --- Assignment Error Handling ---

// In AssignToPeer/AssignToUser/AssignToDeviceGroup, the delete operation's
// error is not checked before proceeding to create. This test verifies
// the transaction still works correctly.
func TestAssignToPeer_TransactionIntegrity(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s1 := &model.Strategy{Name: "txn-1", Enabled: model.BoolPtr(true)}
	s2 := &model.Strategy{Name: "txn-2", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s1); err != nil {
		t.Fatal(err)
	}
	if err := ss.Create(s2); err != nil {
		t.Fatal(err)
	}

	// Assign, then reassign — verify exactly one record exists
	if err := ss.AssignToPeer(s1.Id, 10); err != nil {
		t.Fatal(err)
	}
	if err := ss.AssignToPeer(s2.Id, 10); err != nil {
		t.Fatal(err)
	}

	var count int64
	db.Model(&model.StrategyPeer{}).Where("peer_row_id = ?", 10).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 StrategyPeer record for peer 10, got %d", count)
	}
}
