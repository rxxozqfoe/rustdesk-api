package service

import (
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

type TagService struct {
	ctx *ServiceContext
}

func (s *TagService) Info(id uint) *model.Tag {
	p := &model.Tag{}
	s.ctx.DB.Where("id = ?", id).First(p)
	return p
}
func (s *TagService) InfoByUserIdAndNameAndCollectionId(userid uint, name string, cid uint) *model.Tag {
	p := &model.Tag{}
	s.ctx.DB.Where("user_id = ? and name = ? and collection_id = ?", userid, name, cid).First(p)
	return p
}

func (s *TagService) ListByUserId(userId uint) (res *model.TagList) {
	res = s.List(1, 1000, func(tx *gorm.DB) {
		tx.Where("user_id = ?", userId)
	})
	return
}
func (s *TagService) ListByUserIdAndCollectionId(userId, cid uint) (res *model.TagList) {
	res = s.List(1, 1000, func(tx *gorm.DB) {
		tx.Where("user_id = ? and collection_id = ?", userId, cid)
		tx.Order("name asc")
	})
	return
}
func (s *TagService) UpdateTags(userId uint, tags map[string]uint) {
	tx := s.ctx.DB.Begin()
	//先查询所有tag
	var allTags []*model.Tag
	tx.Where("user_id = ?", userId).Find(&allTags)
	for _, t := range allTags {
		if _, ok := tags[t.Name]; !ok {
			//删除
			tx.Delete(t)
		} else {
			if tags[t.Name] != t.Color {
				//更新
				t.Color = tags[t.Name]
				tx.Save(t)
			}
			//移除
			delete(tags, t.Name)
		}
	}
	//新增
	for tag, color := range tags {
		t := &model.Tag{}
		t.Name = tag
		t.Color = color
		t.UserId = userId
		tx.Create(t)
	}
	tx.Commit()
}

// InfoById 根据用户id取用户信息
func (s *TagService) InfoById(id uint) *model.Tag {
	u := &model.Tag{}
	s.ctx.DB.Where("id = ?", id).First(u)
	return u
}

func (s *TagService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.TagList {
	res := &model.TagList{}
	queryList[model.Tag](s.ctx.DB, page, pageSize, res, &res.Tags, where)
	return res
}

// Create 创建
func (s *TagService) Create(u *model.Tag) error {
	res := s.ctx.DB.Create(u).Error
	return res
}
func (s *TagService) Delete(u *model.Tag) error {
	return s.ctx.DB.Delete(u).Error
}

// Update 更新
func (s *TagService) Update(u *model.Tag) error {
	return s.ctx.DB.Model(u).Select("*").Omit("created_at").Updates(u).Error
}
