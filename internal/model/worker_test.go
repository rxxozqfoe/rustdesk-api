package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"github.com/lejianwen/rustdesk-api/v2/internal/testutil"
)

func TestWorker_ComputeStatus(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeout := 5 * time.Minute

	tests := []struct {
		name     string
		lastSeen *time.Time
		want     string
	}{
		{
			name: "nil last seen is offline",
			want: "offline",
		},
		{
			name:     "within timeout is online",
			lastSeen: ptrTime(now.Add(-1 * time.Minute)),
			want:     "online",
		},
		{
			name:     "exactly at timeout boundary is offline",
			lastSeen: ptrTime(now.Add(-5 * time.Minute)),
			want:     "offline", // strict < timeout
		},
		{
			name:     "beyond timeout is offline",
			lastSeen: ptrTime(now.Add(-10 * time.Minute)),
			want:     "offline",
		},
		{
			name:     "future last seen is online",
			lastSeen: ptrTime(now.Add(1 * time.Minute)),
			want:     "online",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &model.Worker{}
			if tt.lastSeen != nil {
				at := custom_types.AutoTime(*tt.lastSeen)
				w.LastSeenAt = &at
			}
			w.ComputeStatus(now, timeout)
			assert.Equal(t, tt.want, w.StatusComputed)
		})
	}
}

// Round-trips a Worker (including AutoJson Platforms/Versions and an
// AutoTime LastSeenAt) through the in-memory DB to confirm the custom
// types persist and scan back correctly.
func TestWorker_DBRoundTrip(t *testing.T) {
	db := testutil.NewMemDB(t)

	platforms, err := json.Marshal([]model.WorkerPlatform{{Platform: "linux", Arch: "x86_64"}})
	require.NoError(t, err)
	seen := custom_types.AutoTime(time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC))

	w := &model.Worker{
		Name:       "worker-1",
		Platforms:  custom_types.AutoJson(platforms),
		Versions:   custom_types.AutoJson(json.RawMessage(`["1.0.0","1.0.1"]`)),
		LastSeenAt: &seen,
	}
	require.NoError(t, db.Create(w).Error)

	var loaded model.Worker
	require.NoError(t, db.First(&loaded, w.Id).Error)

	assert.Equal(t, "worker-1", loaded.Name)
	assert.JSONEq(t, string(platforms), string(loaded.Platforms))
	assert.JSONEq(t, `["1.0.0","1.0.1"]`, string(loaded.Versions))
	require.NotNil(t, loaded.LastSeenAt)
	assert.True(t, time.Time(seen).Equal(time.Time(*loaded.LastSeenAt)),
		"LastSeenAt should survive the DB round-trip")
}

func ptrTime(t time.Time) *time.Time { return &t }
