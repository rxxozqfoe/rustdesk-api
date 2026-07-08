package service

import (
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm/clause"
)

type ConnAuditRefService struct {
	ctx *ServiceContext
}

// Upsert stores (or refreshes) a ref -> controller-user snapshot. The rendezvous
// server calls this through the internal hbbs API when it mints a conn_audit_ref.
func (s *ConnAuditRefService) Upsert(ref string, userId uint, username string, ttl time.Duration) error {
	if ref == "" {
		return nil
	}
	row := &model.ConnAuditRef{
		Ref:       ref,
		UserId:    userId,
		Username:  username,
		ExpiredAt: time.Now().Add(ttl).Unix(),
	}
	// Upsert on the unique ref so re-connections refresh the snapshot in place.
	return s.ctx.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ref"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "username", "expired_at", "updated_at"}),
	}).Create(row).Error
}

// Lookup returns the non-expired snapshot for ref, or nil.
func (s *ConnAuditRefService) Lookup(ref string) *model.ConnAuditRef {
	if ref == "" {
		return nil
	}
	row := &model.ConnAuditRef{}
	s.ctx.DB.Where("ref = ?", ref).First(row)
	if row.Id == 0 || (row.ExpiredAt != 0 && row.ExpiredAt < time.Now().Unix()) {
		return nil
	}
	return row
}

// CleanExpired removes stale snapshots. Safe to call periodically / on startup.
func (s *ConnAuditRefService) CleanExpired() error {
	return s.ctx.DB.Where("expired_at != 0 and expired_at < ?", time.Now().Unix()).
		Delete(&model.ConnAuditRef{}).Error
}
