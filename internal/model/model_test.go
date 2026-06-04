package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination_SetPagination(t *testing.T) {
	tests := []struct {
		name                     string
		page, pageSize, total    int64
		wantPage, wantSize, wTot int64
	}{
		{"typical", 2, 20, 100, 2, 20, 100},
		{"zero values", 0, 0, 0, 0, 0, 0},
		{"large total", 1, 10, 1_000_000, 1, 10, 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Pagination
			p.SetPagination(tt.page, tt.pageSize, tt.total)
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantSize, p.PageSize)
			assert.Equal(t, tt.wTot, p.Total)
		})
	}
}

func TestStatusCode_Constants(t *testing.T) {
	// Guards against accidental reordering of the status enum, which is
	// persisted to the DB and compared in service code.
	assert.Equal(t, StatusCode(1), COMMON_STATUS_ENABLE)
	assert.Equal(t, StatusCode(2), COMMON_STATUS_DISABLED)
	assert.NotEqual(t, COMMON_STATUS_ENABLE, COMMON_STATUS_DISABLED)
}
