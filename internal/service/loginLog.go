package service

import (
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type LoginLogService struct {
	ctx *ServiceContext
}

// InfoById 根据用户id取用户信息
func (us *LoginLogService) InfoById(id uint) *model.LoginLog {
	u := &model.LoginLog{}
	us.ctx.DB.Where("id = ?", id).First(u)
	return u
}

func (us *LoginLogService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.LoginLogList {
	res := &model.LoginLogList{}
	queryList[model.LoginLog](us.ctx.DB, page, pageSize, res, &res.LoginLogs, where)
	return res
}

// Create 创建
func (us *LoginLogService) Create(u *model.LoginLog) error {
	res := us.ctx.DB.Create(u).Error
	return res
}
func (us *LoginLogService) Delete(u *model.LoginLog) error {
	return us.ctx.DB.Delete(u).Error
}

// Update 更新
func (us *LoginLogService) Update(u *model.LoginLog) error {
	return us.ctx.DB.Model(u).Updates(u).Error
}

func (us *LoginLogService) BatchDelete(ids []uint) error {
	return us.ctx.DB.Where("id in (?)", ids).Delete(&model.LoginLog{}).Error
}

func (us *LoginLogService) SoftDelete(l *model.LoginLog) error {
	l.IsDeleted = model.IsDeletedYes
	return us.Update(l)
}

func (us *LoginLogService) BatchSoftDelete(uid uint, ids []uint) error {
	return us.ctx.DB.Model(&model.LoginLog{}).Where("user_id = ? and id in (?)", uid, ids).Update("is_deleted", model.IsDeletedYes).Error
}
