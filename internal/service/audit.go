package service

import (
	"time"

	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"gorm.io/gorm"
)

type AuditService struct {
	ctx *ServiceContext
}

func (as *AuditService) AuditConnList(page, pageSize uint, where func(tx *gorm.DB)) *model.AuditConnList {
	res := &model.AuditConnList{}
	queryList[model.AuditConn](as.ctx.DB, page, pageSize, res, &res.AuditConns, where)
	return res
}

// Create 创建
func (as *AuditService) CreateAuditConn(u *model.AuditConn) error {
	res := as.ctx.DB.Create(u).Error
	return res
}
func (as *AuditService) DeleteAuditConn(u *model.AuditConn) error {
	return as.ctx.DB.Delete(u).Error
}

// Update 更新
func (as *AuditService) UpdateAuditConn(u *model.AuditConn) error {
	return as.ctx.DB.Model(u).Updates(u).Error
}

// InfoByPeerIdAndConnId
func (as *AuditService) InfoByPeerIdAndConnId(peerId string, connId int64) (res *model.AuditConn) {
	res = &model.AuditConn{}
	as.ctx.DB.Where("peer_id = ? and conn_id = ?", peerId, connId).First(res)
	return
}

// ConnInfoById
func (as *AuditService) ConnInfoById(id uint) (res *model.AuditConn) {
	res = &model.AuditConn{}
	as.ctx.DB.Where("id = ?", id).First(res)
	return
}

// FileInfoById
func (as *AuditService) FileInfoById(id uint) (res *model.AuditFile) {
	res = &model.AuditFile{}
	as.ctx.DB.Where("id = ?", id).First(res)
	return
}

// CloseStaleConns closes all audit connections that were never properly closed
// (close_time = 0). This should be called on server startup because any
// connections from a previous run cannot still be active.
func (as *AuditService) CloseStaleConns() error {
	return as.ctx.DB.Model(&model.AuditConn{}).
		Where("close_time = ?", 0).
		Update("close_time", time.Now().Unix()).
		Error
}

func (as *AuditService) AuditFileList(page, pageSize uint, where func(tx *gorm.DB)) *model.AuditFileList {
	res := &model.AuditFileList{}
	queryList[model.AuditFile](as.ctx.DB, page, pageSize, res, &res.AuditFiles, where)
	return res
}

// CreateAuditFile
func (as *AuditService) CreateAuditFile(u *model.AuditFile) error {
	res := as.ctx.DB.Create(u).Error
	return res
}
func (as *AuditService) DeleteAuditFile(u *model.AuditFile) error {
	return as.ctx.DB.Delete(u).Error
}

// Update 更新
func (as *AuditService) UpdateAuditFile(u *model.AuditFile) error {
	return as.ctx.DB.Model(u).Updates(u).Error
}

func (as *AuditService) BatchDeleteAuditConn(ids []uint) error {
	return as.ctx.DB.Where("id in (?)", ids).Delete(&model.AuditConn{}).Error
}

func (as *AuditService) BatchDeleteAuditFile(ids []uint) error {
	return as.ctx.DB.Where("id in (?)", ids).Delete(&model.AuditFile{}).Error
}
