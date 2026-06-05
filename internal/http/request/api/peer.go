package api

import "github.com/rxxozqfoe/rustdesk-api/internal/model"

type AddressBookFormData struct {
	Tags      []string             `json:"tags"`
	Peers     []*model.AddressBook `json:"peers"`
	TagColors string               `json:"tag_colors"`
}

type AddressBookForm struct {
	Data string `json:"data" example:"{\"tags\":[\"tag1\",\"tag2\",\"tag3\"],\"peers\":[{\"id\":\"abc\",\"username\":\"abv-l\",\"hostname\":\"\",\"platform\":\"Windows\",\"alias\":\"\",\"tags\":[\"tag1\",\"tag2\"],\"hash\":\"hash\"}],\"tag_colors\":\"{\\\"tag1\\\":4288585374,\\\"tag2\\\":4278238420,\\\"tag3\\\":4291681337}\"}"`
}

type PeerForm struct {
	Cpu                       string `json:"cpu"`
	Hostname                  string `json:"hostname"`
	Id                        string `json:"id"`
	Memory                    string `json:"memory"`
	Os                        string `json:"os"`
	Username                  string `json:"username"`
	Uuid                      string `json:"uuid"`
	Version                   string `json:"version"`
	PresetAddressBookName     string `json:"preset-address-book-name,omitempty"`
	PresetAddressBookTag      string `json:"preset-address-book-tag,omitempty"`
	PresetAddressBookAlias    string `json:"preset-address-book-alias,omitempty"`
	PresetAddressBookPassword string `json:"preset-address-book-password,omitempty"`
	PresetAddressBookNote     string `json:"preset-address-book-note,omitempty"`
	PresetUsername            string `json:"preset-user-name,omitempty"`
	PresetStrategyName        string `json:"preset-strategy-name,omitempty"`
	PresetDeviceGroupName     string `json:"preset-device-group-name,omitempty"`
	PresetNote                string `json:"preset-note,omitempty"`
}

func (pf *PeerForm) ToPeer() *model.Peer {
	return &model.Peer{
		Cpu:      pf.Cpu,
		Hostname: pf.Hostname,
		Id:       pf.Id,
		Memory:   pf.Memory,
		Os:       pf.Os,
		Username: pf.Username,
		Uuid:     pf.Uuid,
		Version:  pf.Version,
		Note:     pf.PresetNote,
	}
}

// PersonalAddressBookForm 个人地址簿表单
type PersonalAddressBookForm struct {
	model.AddressBook
	ForceAlwaysRelay string `json:"forceAlwaysRelay"`
}

func (pabf *PersonalAddressBookForm) ToAddressBook() *model.AddressBook {
	return &model.AddressBook{
		RowId:            pabf.RowId,
		Id:               pabf.Id,
		Username:         pabf.Username,
		Password:         pabf.Password,
		Hostname:         pabf.Hostname,
		Alias:            pabf.Alias,
		Platform:         pabf.Platform,
		Tags:             pabf.Tags,
		Hash:             pabf.Hash,
		UserId:           pabf.UserId,
		ForceAlwaysRelay: pabf.ForceAlwaysRelay == "true",
		RdpPort:          pabf.RdpPort,
		RdpUsername:      pabf.RdpUsername,
		Online:           pabf.Online,
		LoginName:        pabf.LoginName,
		SameServer:       pabf.SameServer,
		Note:             pabf.Note,
	}
}

type TagRenameForm struct {
	Old string `json:"old"`
	New string `json:"new"`
}
type TagColorForm struct {
	Name  string `json:"name"`
	Color uint   `json:"color"`
}

type PeerInfoInHeartbeat struct {
	Id         string  `json:"id"`
	Uuid       string  `json:"uuid"`
	Ver        int     `json:"ver"`
	Conns      []int64 `json:"conns,omitempty"`
	ModifiedAt int64   `json:"modified_at"`
}
