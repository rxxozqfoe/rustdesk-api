package service

import (
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type GroupService struct {
	ctx *ServiceContext
}

// InfoById 根据用户id取用户信息
func (us *GroupService) InfoById(id uint) *model.Group {
	u := &model.Group{}
	us.ctx.DB.Where("id = ?", id).First(u)
	return u
}

func (us *GroupService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.GroupList {
	res := &model.GroupList{}
	queryList[model.Group](us.ctx.DB, page, pageSize, res, &res.Groups, where)
	return res
}

// Create 创建
func (us *GroupService) Create(u *model.Group) error {
	res := us.ctx.DB.Create(u).Error
	return res
}
func (us *GroupService) Delete(u *model.Group) error {
	return us.ctx.DB.Delete(u).Error
}

// Update 更新
func (us *GroupService) Update(u *model.Group) error {
	return us.ctx.DB.Model(u).Updates(u).Error
}

// DeviceGroupInfoById 根据用户id取用户信息
func (us *GroupService) DeviceGroupInfoById(id uint) *model.DeviceGroup {
	u := &model.DeviceGroup{}
	us.ctx.DB.Where("id = ?", id).First(u)
	return u
}

func (us *GroupService) DeviceGroupList(page, pageSize uint, where func(tx *gorm.DB)) *model.DeviceGroupList {
	res := &model.DeviceGroupList{}
	queryList[model.DeviceGroup](us.ctx.DB, page, pageSize, res, &res.DeviceGroups, where)
	return res
}

func (us *GroupService) DeviceGroupCreate(u *model.DeviceGroup) error {
	res := us.ctx.DB.Create(u).Error
	return res
}
func (us *GroupService) DeviceGroupDelete(u *model.DeviceGroup) error {
	return us.ctx.DB.Delete(u).Error
}

func (us *GroupService) DeviceGroupUpdate(u *model.DeviceGroup) error {
	return us.ctx.DB.Model(u).Updates(u).Error
}
