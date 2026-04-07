package service

import (
	"encoding/json"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type PeerCommandService struct {
	ctx *ServiceContext
}

// PendingByPeerId returns all unprocessed commands for a given peer.
func (s *PeerCommandService) PendingByPeerId(peerId string) []*model.PeerCommand {
	var cmds []*model.PeerCommand
	s.ctx.DB.Where("peer_id = ?", peerId).Find(&cmds)
	return cmds
}

// DeleteByIds removes commands that have been delivered.
func (s *PeerCommandService) DeleteByIds(ids []uint) {
	if len(ids) == 0 {
		return
	}
	s.ctx.DB.Where("id IN ?", ids).Delete(&model.PeerCommand{})
}

// CreateDisconnect creates a disconnect command for a peer.
// connIds can be empty to disconnect all connections.
func (s *PeerCommandService) CreateDisconnect(peerId string, connIds []int) error {
	payload, err := json.Marshal(connIds)
	if err != nil {
		return err
	}
	cmd := &model.PeerCommand{
		PeerId:  peerId,
		Command: model.PeerCommandDisconnect,
		Payload: string(payload),
	}
	return s.ctx.DB.Create(cmd).Error
}
