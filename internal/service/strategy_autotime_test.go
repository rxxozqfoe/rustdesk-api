package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

// Test that AutoTime round-trips correctly through DB (Create → Read).
// AutoTime has Value() for writing but NO Scan() for reading —
// GORM may fall back to time.Time scanning, which might work or silently zero out.
func TestAutoTime_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "time-test", Enabled: model.BoolPtr(true)}
	ss.Create(s)

	loaded := ss.InfoById(s.Id)
	updatedAt := time.Time(loaded.UpdatedAt)

	if updatedAt.IsZero() {
		t.Error("BUG: UpdatedAt is zero after round-trip. " +
			"AutoTime lacks Scan() — GORM cannot read it back from DB correctly.")
	} else {
		t.Logf("UpdatedAt = %v (Unix: %d)", updatedAt, updatedAt.Unix())
	}
}

// Test the exact conversion used in heartbeat handler:
//   serverModifiedAt := time.Time(strategy.UpdatedAt).Unix()
// If UpdatedAt is zero, this returns a large negative number or epoch 0,
// which will never match the client's modified_at, causing infinite re-sync.
// Verifies that after an admin updates a strategy, the Unix-second timestamp
// changes so the heartbeat comparison detects the update.
func TestHeartbeat_ModifiedAtConversion(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "heartbeat-ts", Enabled: model.BoolPtr(true)}
	ss.Create(s)

	loaded := ss.InfoById(s.Id)
	serverModifiedAt := time.Time(loaded.UpdatedAt).Unix()

	t.Logf("serverModifiedAt = %d", serverModifiedAt)

	if serverModifiedAt <= 0 {
		t.Error("serverModifiedAt <= 0")
	}

	clientModifiedAt := serverModifiedAt

	// Wait >1s to ensure Unix timestamp (second precision) changes
	time.Sleep(1100 * time.Millisecond)

	// Admin updates the strategy
	loaded.ConfigOptions = makeJSON(map[string]string{"enable-audio": "N"})
	ss.Update(loaded)

	reloaded := ss.InfoById(s.Id)
	newServerModifiedAt := time.Time(reloaded.UpdatedAt).Unix()

	t.Logf("after update: newServerModifiedAt = %d", newServerModifiedAt)

	if newServerModifiedAt == clientModifiedAt {
		t.Error("BUG: UpdatedAt did not change after Update(). " +
			"Client will never re-sync because timestamps match.")
	}
}

// Test that UpdatedAt actually changes when strategy is modified.
// This is critical: if GORM doesn't auto-update UpdatedAt, the heartbeat
// comparison will never detect changes.
func TestAutoTime_UpdateChangesTimestamp(t *testing.T) {
	db := setupTestDB(t)
	ss := newStrategyService(db)

	s := &model.Strategy{Name: "ts-change", Enabled: model.BoolPtr(true)}
	ss.Create(s)

	loaded := ss.InfoById(s.Id)
	originalTs := time.Time(loaded.UpdatedAt)

	// Sleep briefly to ensure timestamp changes
	time.Sleep(1100 * time.Millisecond)

	loaded.Name = "ts-changed"
	ss.Update(loaded)

	reloaded := ss.InfoById(s.Id)
	newTs := time.Time(reloaded.UpdatedAt)

	t.Logf("original UpdatedAt = %v", originalTs)
	t.Logf("new UpdatedAt      = %v", newTs)

	if !newTs.After(originalTs) {
		t.Error("BUG: UpdatedAt did not advance after Update(). " +
			"Heartbeat will never detect strategy changes.")
	}
}

// Verify the zero-value behavior of AutoTime when converted to Unix timestamp.
func TestAutoTime_ZeroValue(t *testing.T) {
	var at custom_types.AutoTime
	ts := time.Time(at).Unix()
	t.Logf("Zero AutoTime.Unix() = %d", ts)

	// time.Time zero value is year 0001, Unix() returns a large negative number
	if ts == 0 {
		t.Log("Zero AutoTime maps to Unix 0 — client's default modified_at=0 will match, " +
			"preventing strategy from being sent on first heartbeat")
	} else if ts < 0 {
		t.Log("Zero AutoTime maps to negative Unix — will not match client's 0, " +
			"so strategy will be sent (correct behavior)")
	}
}
