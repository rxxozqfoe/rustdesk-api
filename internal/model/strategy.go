package model

import "github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"

// Strategy represents a configuration strategy that can be pushed to devices.
type Strategy struct {
	IdModel
	Guid          string                `json:"guid" gorm:"type:varchar(36);uniqueIndex;not null"`
	Name          string                `json:"name" gorm:"type:varchar(255);uniqueIndex;not null"`
	Enabled       *bool                 `json:"enabled" gorm:"not null"`
	ConfigOptions custom_types.AutoJson `json:"config_options" gorm:"type:text" swaggertype:"object"`
	Extra         custom_types.AutoJson `json:"extra" gorm:"type:text" swaggertype:"object"`
	TimeModel
}

// BoolPtr is a helper to create a *bool for struct literals.
func BoolPtr(b bool) *bool { return &b }

type StrategyList struct {
	Strategies []*Strategy `json:"list"`
	Pagination
}

// StrategyPeer assigns a strategy directly to a peer (device).
type StrategyPeer struct {
	IdModel
	StrategyId uint `json:"strategy_id" gorm:"not null;index;uniqueIndex:idx_sp_peer"`
	PeerRowId  uint `json:"peer_row_id" gorm:"not null;uniqueIndex:idx_sp_peer"`
	TimeModel
}

// StrategyUser assigns a strategy to a user (applies to all their devices).
type StrategyUser struct {
	IdModel
	StrategyId uint `json:"strategy_id" gorm:"not null;index;uniqueIndex:idx_su_user"`
	UserId     uint `json:"user_id" gorm:"not null;uniqueIndex:idx_su_user"`
	TimeModel
}

// StrategyDeviceGroup assigns a strategy to a device group.
type StrategyDeviceGroup struct {
	IdModel
	StrategyId    uint `json:"strategy_id" gorm:"not null;index;uniqueIndex:idx_sdg_dg"`
	DeviceGroupId uint `json:"device_group_id" gorm:"not null;uniqueIndex:idx_sdg_dg"`
	TimeModel
}
