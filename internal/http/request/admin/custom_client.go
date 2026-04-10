package admin

import (
	"encoding/json"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type CustomClientForm struct {
	Id               uint              `json:"id"`
	Name             string            `json:"name" validate:"required"`
	AppName          string            `json:"app_name" validate:"required"`
	ServerHost       string            `json:"server_host"`
	ServerKey        string            `json:"server_key"`
	ApiServer        string            `json:"api_server"`
	RelayServer      string            `json:"relay_server"`
	DefaultSettings  map[string]string `json:"default_settings"`
	OverrideSettings map[string]string `json:"override_settings"`
	Enabled          *bool             `json:"enabled"`
}

func (f *CustomClientForm) ToCustomClient() *model.CustomClient {
	cc := &model.CustomClient{}
	cc.Id = f.Id
	cc.Name = f.Name
	cc.AppName = f.AppName
	cc.ServerHost = f.ServerHost
	cc.ServerKey = f.ServerKey
	cc.ApiServer = f.ApiServer
	cc.RelayServer = f.RelayServer
	if f.Enabled != nil {
		cc.Enabled = f.Enabled
	} else {
		cc.Enabled = model.BoolPtr(true)
	}
	if f.DefaultSettings != nil {
		b, _ := json.Marshal(f.DefaultSettings)
		cc.DefaultSettings = custom_types.AutoJson(b)
	}
	if f.OverrideSettings != nil {
		b, _ := json.Marshal(f.OverrideSettings)
		cc.OverrideSettings = custom_types.AutoJson(b)
	}
	return cc
}
