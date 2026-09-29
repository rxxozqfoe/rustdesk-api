package service

import (
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"gorm.io/gorm"
)

type ShareRecordService struct {
	ctx *ServiceContext
}

// InfoById 根据用户id取用户信息
func (srs *ShareRecordService) InfoById(id uint) *model.ShareRecord {
	u := &model.ShareRecord{}
	srs.ctx.DB.Where("id = ?", id).First(u)
	return u
}

func (srs *ShareRecordService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.ShareRecordList {
	res := &model.ShareRecordList{}
	queryList[model.ShareRecord](srs.ctx.DB, page, pageSize, res, &res.ShareRecords, where)
	return res
}

// Create 创建
func (srs *ShareRecordService) Create(u *model.ShareRecord) error {
	res := srs.ctx.DB.Create(u).Error
	return res
}
func (srs *ShareRecordService) Delete(u *model.ShareRecord) error {
	return srs.ctx.DB.Delete(u).Error
}

// Update 更新
func (srs *ShareRecordService) Update(u *model.ShareRecord) error {
	return srs.ctx.DB.Model(u).Updates(u).Error
}

func (srs *ShareRecordService) BatchDelete(ids []uint) error {
	return srs.ctx.DB.Where("id in (?)", ids).Delete(&model.ShareRecord{}).Error
}
