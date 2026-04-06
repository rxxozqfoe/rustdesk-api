package api

type DeviceCliForm struct {
	Id                  string `json:"id" validate:"required"`
	Uuid                string `json:"uuid" validate:"required"`
	UserName            string `json:"user_name,omitempty"`
	StrategyName        string `json:"strategy_name,omitempty"`
	AddressBookName     string `json:"address_book_name,omitempty"`
	AddressBookTag      string `json:"address_book_tag,omitempty"`
	AddressBookAlias    string `json:"address_book_alias,omitempty"`
	AddressBookPassword string `json:"address_book_password,omitempty"`
	AddressBookNote     string `json:"address_book_note,omitempty"`
	DeviceGroupName     string `json:"device_group_name,omitempty"`
	Note                string `json:"note,omitempty"`
	DeviceUsername      string `json:"device_username,omitempty"`
	DeviceName          string `json:"device_name,omitempty"`
}
