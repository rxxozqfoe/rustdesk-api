package admin

import (
	"encoding/json"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type StrategyForm struct {
	Id            uint              `json:"id"`
	Name          string            `json:"name" validate:"required"`
	Enabled       *bool             `json:"enabled"`
	ConfigOptions map[string]string `json:"config_options"`
	Extra         map[string]string `json:"extra"`
}

func (sf *StrategyForm) ToStrategy() *model.Strategy {
	s := &model.Strategy{}
	s.Id = sf.Id
	s.Name = sf.Name
	if sf.Enabled != nil {
		s.Enabled = sf.Enabled
	} else {
		s.Enabled = model.BoolPtr(true)
	}
	if sf.ConfigOptions != nil {
		b, _ := json.Marshal(sf.ConfigOptions)
		s.ConfigOptions = custom_types.AutoJson(b)
	}
	if sf.Extra != nil {
		b, _ := json.Marshal(sf.Extra)
		s.Extra = custom_types.AutoJson(b)
	}
	return s
}

type StrategyAssignForm struct {
	Strategy string   `json:"strategy"` // strategy GUID (empty = unassign)
	Peers    []string `json:"peers"`    // peer device IDs (rustdesk ID strings)
	Users    []uint   `json:"users"`    // user IDs
	Groups   []uint   `json:"groups"`   // device group IDs
}
