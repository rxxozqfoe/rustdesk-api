package custom_types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoTime_Value(t *testing.T) {
	t.Run("zero value returns nil", func(t *testing.T) {
		var zero AutoTime // zero time.Time
		v, err := zero.Value()
		require.NoError(t, err)
		assert.Nil(t, v, "zero AutoTime should marshal to nil so the DB stores NULL")
	})

	t.Run("non-zero value returns the underlying time.Time", func(t *testing.T) {
		now := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
		at := AutoTime(now)
		v, err := at.Value()
		require.NoError(t, err)
		got, ok := v.(time.Time)
		require.True(t, ok, "Value should be a time.Time, got %T", v)
		assert.True(t, now.Equal(got))
	})
}

func TestAutoTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "formatted timestamp",
			in:   time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC),
			want: `"2024-03-15 10:30:45"`,
		},
		{
			name: "zero time still formats (no nil handling in MarshalJSON)",
			in:   time.Time{},
			want: `"0001-01-01 00:00:00"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := AutoTime(tt.in)
			b, err := at.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}
