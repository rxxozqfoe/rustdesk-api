package service

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/rxxozqfoe/rustdesk-api/internal/lib/lock"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"github.com/rxxozqfoe/rustdesk-api/internal/utils"
	"gorm.io/gorm"
)

type UserService struct {
	ctx *ServiceContext
}

// InfoById 根据用户id取用户信息
func (us *UserService) InfoById(id uint) *model.User {
	u := &model.User{}
	us.ctx.DB.Where("id = ?", id).First(u)
	return u
}

// InfoByUsername 根据用户名取用户信息
func (us *UserService) InfoByUsername(un string) *model.User {
	u := &model.User{}
	us.ctx.DB.Where("username = ?", un).First(u)
	return u
}

// InfoByEmail 根据邮箱取用户信息
func (us *UserService) InfoByEmail(email string) *model.User {
	u := &model.User{}
	us.ctx.DB.Where("email = ?", email).First(u)
	return u
}

// InfoByOpenid 根据openid取用户信息
func (us *UserService) InfoByOpenid(openid string) *model.User {
	u := &model.User{}
	us.ctx.DB.Where("openid = ?", openid).First(u)
	return u
}

// InfoByUsernamePassword 根据用户名密码取用户信息
func (us *UserService) InfoByUsernamePassword(username, password string) *model.User {
	if us.ctx.Config.Ldap.Enable {
		u, err := us.ctx.Services.Authenticate(username, password)
		if err == nil {
			return u
		}
		us.ctx.Logger.Errorf("LDAP authentication failed, %v", err)
		us.ctx.Logger.Warn("Fallback to local database")
	}
	u := &model.User{}
	us.ctx.DB.Where("username = ?", username).First(u)
	if u.Id == 0 {
		return u
	}
	ok, newHash, err := utils.VerifyPassword(u.Password, password)
	if err != nil || !ok {
		return &model.User{}
	}
	if newHash != "" {
		us.ctx.DB.Model(u).Update("password", newHash)
		u.Password = newHash
	}
	return u
}

// InfoByAccesstoken 根据accesstoken取用户信息
func (us *UserService) InfoByAccessToken(token string) (*model.User, *model.UserToken) {
	u := &model.User{}
	ut := &model.UserToken{}
	us.ctx.DB.Where("token = ?", token).First(ut)
	if ut.Id == 0 {
		return u, ut
	}
	if ut.ExpiredAt < time.Now().Unix() {
		return u, ut
	}
	us.ctx.DB.Where("id = ?", ut.UserId).First(u)
	return u, ut
}

// GenerateToken 生成token
func (us *UserService) GenerateToken(u *model.User) string {
	if len(us.ctx.Jwt.Key) > 0 {
		return us.ctx.Jwt.GenerateToken(u.Id)
	}
	return utils.Md5(u.Username + time.Now().String())
}

// Login 登录
func (us *UserService) Login(u *model.User, llog *model.LoginLog) *model.UserToken {
	token := us.GenerateToken(u)
	ut := &model.UserToken{
		UserId:     u.Id,
		Token:      token,
		DeviceUuid: llog.Uuid,
		DeviceId:   llog.DeviceId,
		ExpiredAt:  us.UserTokenExpireTimestamp(),
	}
	us.ctx.DB.Create(ut)
	llog.UserTokenId = ut.UserId
	us.ctx.DB.Create(llog)
	if llog.Uuid != "" {
		us.ctx.Services.UuidBindUserId(llog.DeviceId, llog.Uuid, u.Id)
	}
	return ut
}

func (us *UserService) List(page, pageSize uint, where func(tx *gorm.DB)) *model.UserList {
	res := &model.UserList{}
	queryList[model.User](us.ctx.DB, page, pageSize, res, &res.Users, where)
	return res
}

func (us *UserService) ListByIds(ids []uint) (res []*model.User) {
	us.ctx.DB.Where("id in ?", ids).Find(&res)
	return res
}

// ListByGroupId 根据组id取用户列表
func (us *UserService) ListByGroupId(groupId, page, pageSize uint) (res *model.UserList) {
	res = us.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("group_id = ?", groupId)
	})
	return
}

// ListIdsByGroupId 根据组id取用户id列表
func (us *UserService) ListIdsByGroupId(groupId uint) (ids []uint) {
	us.ctx.DB.Model(&model.User{}).Where("group_id = ?", groupId).Pluck("id", &ids)
	return ids

}

// ListIdAndNameByGroupId 根据组id取用户id和用户名列表
func (us *UserService) ListIdAndNameByGroupId(groupId uint) (res []*model.User) {
	us.ctx.DB.Model(&model.User{}).Where("group_id = ?", groupId).Select("id, username").Find(&res)
	return res
}

// CheckUserEnable 判断用户是否禁用
func (us *UserService) CheckUserEnable(u *model.User) bool {
	return u.Status == model.COMMON_STATUS_ENABLE
}

// Create 创建
func (us *UserService) Create(u *model.User) error {
	// The initial username should be formatted, and the username should be unique
	if us.IsUsernameExists(u.Username) {
		return errors.New("UsernameExists")
	}
	u.Username = us.formatUsername(u.Username)
	var err error
	u.Password, err = utils.EncryptPassword(u.Password)
	if err != nil {
		return err
	}
	res := us.ctx.DB.Create(u).Error
	return res
}

// GetUuidByToken 根据token和user取uuid
func (us *UserService) GetUuidByToken(u *model.User, token string) string {
	ut := &model.UserToken{}
	err := us.ctx.DB.Where("user_id = ? and token = ?", u.Id, token).First(ut).Error
	if err != nil {
		return ""
	}
	return ut.DeviceUuid
}

// Logout 退出登录 -> 删除token, 解绑uuid
func (us *UserService) Logout(u *model.User, token string) error {
	uuid := us.GetUuidByToken(u, token)
	err := us.ctx.DB.Where("user_id = ? and token = ?", u.Id, token).Delete(&model.UserToken{}).Error
	if err != nil {
		return err
	}
	if uuid != "" {
		us.ctx.Services.UuidUnbindUserId(uuid, u.Id)
	}
	return nil
}

// Delete 删除用户和oauth信息
func (us *UserService) Delete(u *model.User) error {
	userCount := us.getAdminUserCount()
	if userCount <= 1 && us.IsAdmin(u) {
		return errors.New("the last admin user cannot be deleted")
	}
	tx := us.ctx.DB.Begin()
	// 删除用户
	if err := tx.Delete(u).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 删除关联的 OAuth 信息
	if err := tx.Where("user_id = ?", u.Id).Delete(&model.UserThird{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	//  删除关联的ab
	if err := tx.Where("user_id = ?", u.Id).Delete(&model.AddressBook{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	//  删除关联的abc
	if err := tx.Where("user_id = ?", u.Id).Delete(&model.AddressBookCollection{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	//  删除关联的abcr
	if err := tx.Where("user_id = ?", u.Id).Delete(&model.AddressBookCollectionRule{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	// 删除关联的peer
	if err := us.ctx.Services.EraseUserId(u.Id); err != nil {
		us.ctx.Logger.Warn("User deleted successfully, but failed to unlink peer.")
		return nil
	}
	return nil
}

// Update 更新
func (us *UserService) Update(u *model.User) error {
	currentUser := us.InfoById(u.Id)
	// 如果当前用户是管理员并且 IsAdmin 不为空，进行检查
	if us.IsAdmin(currentUser) {
		adminCount := us.getAdminUserCount()
		// 如果这是唯一的管理员，确保不能禁用或取消管理员权限
		if adminCount <= 1 && (!us.IsAdmin(u) || u.Status == model.COMMON_STATUS_DISABLED) {
			return errors.New("the last admin user cannot be disabled or demoted")
		}
	}
	return us.ctx.DB.Model(u).Updates(u).Error
}

// FlushToken 清空token
func (us *UserService) FlushToken(u *model.User) error {
	return us.ctx.DB.Where("user_id = ?", u.Id).Delete(&model.UserToken{}).Error
}

// FlushTokenByUuid 清空token
func (us *UserService) FlushTokenByUuid(uuid string) error {
	return us.ctx.DB.Where("device_uuid = ?", uuid).Delete(&model.UserToken{}).Error
}

// FlushTokenByUuids 清空token
func (us *UserService) FlushTokenByUuids(uuids []string) error {
	return us.ctx.DB.Where("device_uuid in (?)", uuids).Delete(&model.UserToken{}).Error
}

// UpdatePassword 更新密码
func (us *UserService) UpdatePassword(u *model.User, password string) error {
	var err error
	u.Password, err = utils.EncryptPassword(password)
	if err != nil {
		return err
	}
	err = us.ctx.DB.Model(u).Update("password", u.Password).Error
	if err != nil {
		return err
	}
	err = us.FlushToken(u)
	return err
}

// IsAdmin 是否管理员
func (us *UserService) IsAdmin(u *model.User) bool {
	return u != nil && *u.IsAdmin
}

// InfoByOauthId 根据oauth的name和openId取用户信息
func (us *UserService) InfoByOauthId(op string, openId string) *model.User {
	ut := us.ctx.Services.OauthService.UserThirdInfo(op, openId)
	if ut.Id == 0 {
		return nil
	}
	u := us.InfoById(ut.UserId)
	if u.Id == 0 {
		return nil
	}
	return u
}

// RegisterByOauth 注册
func (us *UserService) RegisterByOauth(oauthUser *model.OauthUser, op string) (*model.User, error) {
	us.ctx.Lock.Lock(lock.LockRegisterByOauth)
	defer us.ctx.Lock.UnLock(lock.LockRegisterByOauth)
	ut := us.ctx.Services.OauthService.UserThirdInfo(op, oauthUser.OpenId)
	if ut.Id != 0 {
		return us.InfoById(ut.UserId), nil
	}
	oauthType, err := us.ctx.Services.GetTypeByOp(op)
	if err != nil {
		return nil, err
	}
	//check if this email has been registered
	email := oauthUser.Email
	// only email is not empty
	if email != "" {
		email = strings.ToLower(email)
		// update email to oauthUser, in case it contain upper case
		oauthUser.Email = email
		// call this, if find user by email, it will update the email to local database
		user, ldapErr := us.ctx.Services.GetUserInfoByEmailLocal(email)
		// If we enable ldap, and the error is not ErrLdapUserNotFound, return the error because we could not sure if the user is not found in ldap
		if !errors.Is(ldapErr, ErrLdapNotEnabled) && !errors.Is(ldapErr, ErrLdapUserNotFound) && ldapErr != nil {
			return user, ldapErr
		}
		if user.Id == 0 {
			// this means the user is not found in ldap, maybe ldao is not enabled
			user = us.InfoByEmail(email)
		}
		if user.Id != 0 {
			ut.FromOauthUser(user.Id, oauthUser, oauthType, op)
			us.ctx.DB.Create(ut)
			return user, nil
		}
	}

	tx := us.ctx.DB.Begin()
	ut = &model.UserThird{}
	ut.FromOauthUser(0, oauthUser, oauthType, op)
	// The initial username should be formatted
	username := us.formatUsername(oauthUser.Username)
	usernameUnique := us.GenerateUsernameByOauth(username)
	user := &model.User{
		Username: usernameUnique,
		GroupId:  1,
	}
	oauthUser.ToUser(user, false)
	tx.Create(user)
	if user.Id == 0 {
		tx.Rollback()
		return user, errors.New("OauthRegisterFailed")
	}
	ut.UserId = user.Id
	tx.Create(ut)
	tx.Commit()
	return user, nil
}

// GenerateUsernameByOauth 生成用户名
func (us *UserService) GenerateUsernameByOauth(name string) string {
	for us.IsUsernameExists(name) {
		name += strconv.Itoa(rand.Intn(10)) // Append a random digit (0-9)
	}
	return name
}

// UserThirdsByUserId
func (us *UserService) UserThirdsByUserId(userId uint) (res []*model.UserThird) {
	us.ctx.DB.Where("user_id = ?", userId).Find(&res)
	return res
}

func (us *UserService) UserThirdInfo(userId uint, op string) *model.UserThird {
	ut := &model.UserThird{}
	us.ctx.DB.Where("user_id = ? and op = ?", userId, op).First(ut)
	return ut
}

// FindLatestUserIdFromLoginLogByUuid 根据uuid和设备id查找最后登录的用户id
func (us *UserService) FindLatestUserIdFromLoginLogByUuid(uuid string, deviceId string) uint {
	llog := &model.LoginLog{}
	us.ctx.DB.Where("uuid = ? and device_id = ?", uuid, deviceId).Order("id desc").First(llog)
	return llog.UserId
}

// IsPasswordEmptyById 根据用户id判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyById(id uint) bool {
	u := &model.User{}
	if us.ctx.DB.Where("id = ?", id).First(u).Error != nil {
		return false
	}
	return u.Password == ""
}

// IsPasswordEmptyByUsername 根据用户id判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyByUsername(username string) bool {
	u := &model.User{}
	if us.ctx.DB.Where("username = ?", username).First(u).Error != nil {
		return false
	}
	return u.Password == ""
}

// IsPasswordEmptyByUser 判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyByUser(u *model.User) bool {
	return us.IsPasswordEmptyById(u.Id)
}

// Register 注册, 如果用户名已存在则返回nil
func (us *UserService) Register(username string, email string, password string, status model.StatusCode) *model.User {
	u := &model.User{
		Username: username,
		Email:    email,
		Password: password,
		GroupId:  1,
		Status:   status,
	}
	err := us.Create(u)
	if err != nil {
		return nil
	}
	return u
}

func (us *UserService) TokenList(page uint, size uint, f func(tx *gorm.DB)) *model.UserTokenList {
	res := &model.UserTokenList{}
	res.Page = int64(page)
	res.PageSize = int64(size)
	tx := us.ctx.DB.Model(&model.UserToken{})
	if f != nil {
		f(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, size))
	tx.Find(&res.UserTokens)
	return res
}

func (us *UserService) TokenInfoById(id uint) *model.UserToken {
	ut := &model.UserToken{}
	us.ctx.DB.Where("id = ?", id).First(ut)
	return ut
}

func (us *UserService) DeleteToken(l *model.UserToken) error {
	return us.ctx.DB.Delete(l).Error
}

// Helper functions, used for formatting username
func (us *UserService) formatUsername(username string) string {
	username = strings.ReplaceAll(username, " ", "")
	username = strings.ToLower(username)
	return username
}

// helper functions, getAdminUserCount
func (us *UserService) getAdminUserCount() int64 {
	var count int64
	us.ctx.DB.Model(&model.User{}).Where("is_admin = ?", true).Count(&count)
	return count
}

// UserTokenExpireTimestamp 生成用户token过期时间
func (us *UserService) UserTokenExpireTimestamp() int64 {
	exp := us.ctx.Config.App.TokenExpire
	if exp == 0 {
		//默认七天
		exp = 604800
	}
	return time.Now().Add(exp).Unix()
}

func (us *UserService) RefreshAccessToken(ut *model.UserToken) {
	ut.ExpiredAt = us.UserTokenExpireTimestamp()
	us.ctx.DB.Model(ut).Update("expired_at", ut.ExpiredAt)
}

func (us *UserService) AutoRefreshAccessToken(ut *model.UserToken) {
	if ut.ExpiredAt-time.Now().Unix() < us.ctx.Config.App.TokenExpire.Milliseconds()/3000 {
		us.RefreshAccessToken(ut)
	}
}

func (us *UserService) BatchDeleteUserToken(ids []uint) error {
	return us.ctx.DB.Where("id in ?", ids).Delete(&model.UserToken{}).Error
}

func (us *UserService) VerifyJWT(token string) (uint, error) {
	return us.ctx.Jwt.ParseToken(token)
}

// IsUsernameExists 判断用户名是否存在, it will check the internal database and LDAP(if enabled)
func (us *UserService) IsUsernameExists(username string) bool {
	return us.IsUsernameExistsLocal(username) || us.ctx.Services.LdapService.IsUsernameExists(username)
}

func (us *UserService) IsUsernameExistsLocal(username string) bool {
	u := &model.User{}
	us.ctx.DB.Where("username = ?", username).First(u)
	return u.Id != 0
}

func (us *UserService) IsEmailExistsLdap(email string) bool {
	return us.ctx.Services.IsEmailExists(email)
}
