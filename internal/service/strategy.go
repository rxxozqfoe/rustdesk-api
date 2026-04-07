package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
	"gorm.io/gorm"
)

type StrategyService struct {
	ctx *ServiceContext
}

var emptyJsonObject = custom_types.AutoJson(json.RawMessage("{}"))

// InfoById finds a strategy by its uint ID.
func (ss *StrategyService) InfoById(id uint) *model.Strategy {
	s := &model.Strategy{}
	ss.ctx.DB.Where("id = ?", id).First(s)
	return s
}

// InfoByGuid finds a strategy by its GUID string.
func (ss *StrategyService) InfoByGuid(guid string) *model.Strategy {
	s := &model.Strategy{}
	ss.ctx.DB.Where("guid = ?", guid).First(s)
	return s
}

// InfoByName finds a strategy by its name.
func (ss *StrategyService) InfoByName(name string) *model.Strategy {
	s := &model.Strategy{}
	ss.ctx.DB.Where("name = ?", name).First(s)
	return s
}

// List returns a paginated list of strategies.
func (ss *StrategyService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.StrategyList {
	res := &model.StrategyList{}
	queryList[model.Strategy](ss.ctx.DB, page, pageSize, res, &res.Strategies, where)
	return res
}

// ListAll returns all strategies (no pagination).
func (ss *StrategyService) ListAll() []*model.Strategy {
	var strategies []*model.Strategy
	ss.ctx.DB.Find(&strategies)
	return strategies
}

// Create inserts a new strategy, generating a GUID automatically.
func (ss *StrategyService) Create(s *model.Strategy) error {
	s.Guid = uuid.New().String()
	if len(s.ConfigOptions) == 0 {
		s.ConfigOptions = emptyJsonObject
	}
	if len(s.Extra) == 0 {
		s.Extra = emptyJsonObject
	}
	return ss.ctx.DB.Create(s).Error
}

// Update modifies an existing strategy.
func (ss *StrategyService) Update(s *model.Strategy) error {
	return ss.ctx.DB.Model(s).Updates(s).Error
}

// Delete removes a strategy and all its associations.
func (ss *StrategyService) Delete(s *model.Strategy) error {
	return ss.ctx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("strategy_id = ?", s.Id).Delete(&model.StrategyPeer{}).Error; err != nil {
			return err
		}
		if err := tx.Where("strategy_id = ?", s.Id).Delete(&model.StrategyUser{}).Error; err != nil {
			return err
		}
		if err := tx.Where("strategy_id = ?", s.Id).Delete(&model.StrategyDeviceGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(s).Error
	})
}

// SetEnabled enables or disables a strategy.
func (ss *StrategyService) SetEnabled(id uint, enabled bool) error {
	return ss.ctx.DB.Model(&model.Strategy{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// AssignToPeer assigns a strategy to a peer. Removes any prior assignment for this peer.
func (ss *StrategyService) AssignToPeer(strategyId, peerRowId uint) error {
	return ss.ctx.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("peer_row_id = ?", peerRowId).Delete(&model.StrategyPeer{})
		return tx.Create(&model.StrategyPeer{
			StrategyId: strategyId,
			PeerRowId:  peerRowId,
		}).Error
	})
}

// AssignToUser assigns a strategy to a user. Removes any prior assignment for this user.
func (ss *StrategyService) AssignToUser(strategyId, userId uint) error {
	return ss.ctx.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("user_id = ?", userId).Delete(&model.StrategyUser{})
		return tx.Create(&model.StrategyUser{
			StrategyId: strategyId,
			UserId:     userId,
		}).Error
	})
}

// AssignToDeviceGroup assigns a strategy to a device group. Removes any prior assignment.
func (ss *StrategyService) AssignToDeviceGroup(strategyId, deviceGroupId uint) error {
	return ss.ctx.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("device_group_id = ?", deviceGroupId).Delete(&model.StrategyDeviceGroup{})
		return tx.Create(&model.StrategyDeviceGroup{
			StrategyId:    strategyId,
			DeviceGroupId: deviceGroupId,
		}).Error
	})
}

// UnassignPeer removes strategy assignment from a peer.
func (ss *StrategyService) UnassignPeer(peerRowId uint) error {
	return ss.ctx.DB.Where("peer_row_id = ?", peerRowId).Delete(&model.StrategyPeer{}).Error
}

// UnassignUser removes strategy assignment from a user.
func (ss *StrategyService) UnassignUser(userId uint) error {
	return ss.ctx.DB.Where("user_id = ?", userId).Delete(&model.StrategyUser{}).Error
}

// UnassignDeviceGroup removes strategy assignment from a device group.
func (ss *StrategyService) UnassignDeviceGroup(deviceGroupId uint) error {
	return ss.ctx.DB.Where("device_group_id = ?", deviceGroupId).Delete(&model.StrategyDeviceGroup{}).Error
}

// ResolveForPeer finds the effective enabled strategy for a peer.
// Priority: direct peer assignment > user assignment > device group assignment.
// Returns nil if no enabled strategy applies.
func (ss *StrategyService) ResolveForPeer(peer *model.Peer) *model.Strategy {
	// 1. Direct peer assignment
	s := &model.Strategy{}
	err := ss.ctx.DB.
		Joins("JOIN strategy_peers ON strategy_peers.strategy_id = strategies.id").
		Where("strategy_peers.peer_row_id = ? AND strategies.enabled = ?", peer.RowId, true).
		First(s).Error
	if err == nil && s.Id > 0 {
		return s
	}

	// 2. User assignment
	if peer.UserId > 0 {
		s = &model.Strategy{}
		err = ss.ctx.DB.
			Joins("JOIN strategy_users ON strategy_users.strategy_id = strategies.id").
			Where("strategy_users.user_id = ? AND strategies.enabled = ?", peer.UserId, true).
			First(s).Error
		if err == nil && s.Id > 0 {
			return s
		}
	}

	// 3. Device group assignment
	if peer.GroupId > 0 {
		s = &model.Strategy{}
		err = ss.ctx.DB.
			Joins("JOIN strategy_device_groups ON strategy_device_groups.strategy_id = strategies.id").
			Where("strategy_device_groups.device_group_id = ? AND strategies.enabled = ?", peer.GroupId, true).
			First(s).Error
		if err == nil && s.Id > 0 {
			return s
		}
	}

	return nil
}

// ConfigOptionsMap deserializes ConfigOptions from AutoJson to map[string]string.
func (ss *StrategyService) ConfigOptionsMap(s *model.Strategy) map[string]string {
	return autoJsonToStringMap(s.ConfigOptions)
}

// ExtraMap deserializes Extra from AutoJson to map[string]string.
func (ss *StrategyService) ExtraMap(s *model.Strategy) map[string]string {
	return autoJsonToStringMap(s.Extra)
}

func autoJsonToStringMap(j custom_types.AutoJson) map[string]string {
	m := make(map[string]string)
	if len(j) == 0 {
		return m
	}
	_ = json.Unmarshal(j, &m)
	return m
}
