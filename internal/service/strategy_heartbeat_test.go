package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
)

// Simulate the exact heartbeat handler logic from index.go:69-82
// to verify the response the client would receive.
func TestHeartbeat_FullSimulation(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	// Setup: create strategy with config_options
	opts := map[string]string{
		"enable-file-transfer": "N",
		"enable-clipboard":     "N",
		"approve-mode":         "click",
	}
	s := &model.Strategy{
		Name:          "office-policy",
		Enabled:       model.BoolPtr(true),
		ConfigOptions: makeJSON(opts),
	}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Setup: create peer and assign strategy
	peer := &model.Peer{Id: "test-peer-1", UserId: 0, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToPeer(s.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}

	// Simulate heartbeat: client sends modified_at = 0 (first boot)
	clientModifiedAt := int64(0)

	// -- Replicate index.go:69-82 --
	strategy := ss.ResolveForPeer(peer)
	resp := gin.H{}

	if strategy != nil {
		serverModifiedAt := time.Time(strategy.UpdatedAt).Unix()
		resp["modified_at"] = serverModifiedAt
		if clientModifiedAt != serverModifiedAt {
			resp["strategy"] = gin.H{
				"config_options": ss.ConfigOptionsMap(strategy),
				"extra":          ss.ExtraMap(strategy),
			}
		}
	} else {
		resp["modified_at"] = 0
	}

	// Verify: strategy should be present in response
	if _, ok := resp["strategy"]; !ok {
		t.Fatal("BUG: first heartbeat (modified_at=0) should include strategy, but it's missing")
	}

	// Verify: config_options should match
	strategyResp := resp["strategy"].(gin.H)
	configOpts := strategyResp["config_options"].(map[string]string)
	if configOpts["enable-file-transfer"] != "N" {
		t.Errorf("config_options[enable-file-transfer] = %q, want %q",
			configOpts["enable-file-transfer"], "N")
	}
	if configOpts["enable-clipboard"] != "N" {
		t.Errorf("config_options[enable-clipboard] = %q, want %q",
			configOpts["enable-clipboard"], "N")
	}
	if configOpts["approve-mode"] != "click" {
		t.Errorf("config_options[approve-mode] = %q, want %q",
			configOpts["approve-mode"], "click")
	}

	// Serialize to JSON and verify it matches what client expects
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	t.Logf("Heartbeat response JSON: %s", string(jsonBytes))

	// Parse back like the client does
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("client cannot parse response: %v", err)
	}

	// Client parses modified_at as i64
	var modifiedAt int64
	if err := json.Unmarshal(parsed["modified_at"], &modifiedAt); err != nil {
		t.Fatalf("client cannot parse modified_at: %v", err)
	}
	if modifiedAt <= 0 {
		t.Errorf("modified_at = %d, should be positive Unix timestamp", modifiedAt)
	}

	// Client parses strategy as StrategyOptions { config_options, extra }
	type StrategyOptions struct {
		ConfigOptions map[string]string `json:"config_options"`
		Extra         map[string]string `json:"extra"`
	}
	var so StrategyOptions
	if err := json.Unmarshal(parsed["strategy"], &so); err != nil {
		t.Fatalf("client cannot parse strategy: %v", err)
	}
	if so.ConfigOptions["enable-file-transfer"] != "N" {
		t.Errorf("parsed config_options[enable-file-transfer] = %q, want %q",
			so.ConfigOptions["enable-file-transfer"], "N")
	}

	t.Logf("Client would receive: modified_at=%d, config_options=%v, extra=%v",
		modifiedAt, so.ConfigOptions, so.Extra)
}

// Simulate second heartbeat: client sends back the modified_at it received.
// Strategy should NOT be re-sent (bandwidth optimization).
func TestHeartbeat_SecondCallSkipsStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "skip-test", Enabled: model.BoolPtr(true), ConfigOptions: makeJSON(map[string]string{"enable-audio": "N"})}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "test-peer-2", UserId: 0, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToPeer(s.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}

	// First heartbeat
	strategy := ss.ResolveForPeer(peer)
	serverModifiedAt := time.Time(strategy.UpdatedAt).Unix()

	// Second heartbeat: client sends back the same modified_at
	resp := gin.H{}
	strategy = ss.ResolveForPeer(peer)
	if strategy != nil {
		newServerModifiedAt := time.Time(strategy.UpdatedAt).Unix()
		resp["modified_at"] = newServerModifiedAt
		if serverModifiedAt != newServerModifiedAt {
			resp["strategy"] = gin.H{
				"config_options": ss.ConfigOptionsMap(strategy),
				"extra":          ss.ExtraMap(strategy),
			}
		}
	}

	if _, ok := resp["strategy"]; ok {
		t.Error("BUG: second heartbeat with same modified_at should NOT include strategy")
	}
}

// Simulate: strategy not assigned to peer → response should have modified_at=0
func TestHeartbeat_NoStrategyAssigned(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	peer := &model.Peer{Id: "test-peer-3", UserId: 0, GroupId: 0}
	db.Create(peer)

	strategy := ss.ResolveForPeer(peer)
	resp := gin.H{}
	if strategy != nil {
		t.Fatal("should not find strategy for unassigned peer")
	}
	resp["modified_at"] = 0

	jsonBytes, _ := json.Marshal(resp)
	t.Logf("No-strategy response: %s", string(jsonBytes))

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	var modifiedAt int64
	if err := json.Unmarshal(parsed["modified_at"], &modifiedAt); err != nil {
		t.Fatal(err)
	}
	if modifiedAt != 0 {
		t.Errorf("modified_at should be 0 when no strategy, got %d", modifiedAt)
	}
}

// Simulate: strategy assigned via user (not direct peer)
func TestHeartbeat_UserLevelStrategy(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{
		Name:          "user-policy",
		Enabled:       model.BoolPtr(true),
		ConfigOptions: makeJSON(map[string]string{"enable-terminal": "N"}),
	}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	// Peer belongs to user 50
	peer := &model.Peer{Id: "test-peer-4", UserId: 50, GroupId: 0}
	db.Create(peer)

	// Assign strategy to user 50 (not directly to peer)
	if err := ss.AssignToUser(s.Id, 50); err != nil {
		t.Fatal(err)
	}

	strategy := ss.ResolveForPeer(peer)
	if strategy == nil {
		t.Fatal("should resolve user-level strategy for peer")
	}

	configOpts := ss.ConfigOptionsMap(strategy)
	if configOpts["enable-terminal"] != "N" {
		t.Errorf("config_options[enable-terminal] = %q, want %q",
			configOpts["enable-terminal"], "N")
	}
}

// Simulate: disabled strategy should not be sent
func TestHeartbeat_DisabledStrategyNotSent(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "disabled-policy", Enabled: model.BoolPtr(true)}
	if err := ss.Create(s); err != nil {
		t.Fatal(err)
	}

	peer := &model.Peer{Id: "test-peer-5", UserId: 0, GroupId: 0}
	db.Create(peer)
	if err := ss.AssignToPeer(s.Id, peer.RowId); err != nil {
		t.Fatal(err)
	}

	// Disable the strategy
	if err := ss.SetEnabled(s.Id, false); err != nil {
		t.Fatal(err)
	}

	strategy := ss.ResolveForPeer(peer)
	resp := gin.H{}
	if strategy != nil {
		resp["modified_at"] = time.Time(strategy.UpdatedAt).Unix()
	} else {
		resp["modified_at"] = 0
	}

	modifiedAt := resp["modified_at"]
	if modifiedAt != 0 {
		t.Errorf("disabled strategy should result in modified_at=0, got %v", modifiedAt)
	}
}
