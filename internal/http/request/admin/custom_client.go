package admin

import (
	"encoding/json"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type CustomClientForm struct {
	Id               uint              `json:"id"`
	Name             string            `json:"name" validate:"required"`
	ServerHost       string            `json:"server_host"`
	ServerKey        string            `json:"server_key"`
	ApiServer        string            `json:"api_server"`
	RelayServer      string            `json:"relay_server"`
	DefaultSettings  map[string]string `json:"default_settings"`
	OverrideSettings map[string]string `json:"override_settings"`
	Platform         string            `json:"platform" validate:"required"`
	Arch             string            `json:"arch" validate:"required"`
	Version          string            `json:"version" validate:"required"`
	Format           string            `json:"format" validate:"required"`
}

func (f *CustomClientForm) ToCustomClient() *model.CustomClient {
	cc := &model.CustomClient{}
	cc.Id = f.Id
	cc.Name = f.Name
	cc.ServerHost = f.ServerHost
	cc.ServerKey = f.ServerKey
	cc.ApiServer = f.ApiServer
	cc.RelayServer = f.RelayServer
	cc.Platform = f.Platform
	cc.Arch = f.Arch
	cc.Version = f.Version
	cc.Format = f.Format
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
