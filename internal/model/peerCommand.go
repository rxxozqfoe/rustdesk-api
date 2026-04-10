package model

const (
	PeerCommandDisconnect = "disconnect"
)

type PeerCommand struct {
	IdModel
	PeerId  string `json:"peer_id" gorm:"default:'';not null;index"`
	Command string `json:"command" gorm:"default:'';not null;"`
	Payload string `json:"payload" gorm:"default:'';not null;"` // JSON array, e.g. [1,2,3] for conn IDs
	TimeModel
}
