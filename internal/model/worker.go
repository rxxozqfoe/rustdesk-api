package model

import (
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type WorkerPlatform struct {
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
}

type Worker struct {
	IdModel
	Name           string                 `json:"name" gorm:"type:varchar(255);not null;uniqueIndex"`
	Platforms      custom_types.AutoJson  `json:"platforms" gorm:"type:text" swaggertype:"array,object"`
	Versions       custom_types.AutoJson  `json:"versions" gorm:"type:text" swaggertype:"array,string"`
	LastSeenAt     *custom_types.AutoTime `json:"last_seen_at" gorm:"type:timestamp"`
	StatusComputed string                 `json:"status" gorm:"-"`
	TimeModel
}

// ComputeStatus sets the StatusComputed field based on last heartbeat time.
func (w *Worker) ComputeStatus(now time.Time, timeout time.Duration) {
	if w.LastSeenAt != nil && now.Sub(time.Time(*w.LastSeenAt)) < timeout {
		w.StatusComputed = "online"
	} else {
		w.StatusComputed = "offline"
	}
}

type WorkerList struct {
	Workers []*Worker `json:"list"`
	Pagination
}
