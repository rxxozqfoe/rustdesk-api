package service

import (
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/s3"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"gorm.io/gorm"
)

// ServiceContext holds shared dependencies for all services.
// Services is a back-pointer to the Service aggregate, assigned by New() after
// the aggregate is constructed. It lets a service reach its siblings (e.g.
// UserService → LdapService) without a package-level singleton.
type ServiceContext struct {
	Config   *config.Config
	DB       *gorm.DB
	Logger   *logger.Logger
	Jwt      *jwt.Jwt
	Lock     lock.Locker
	S3       *s3.Client // nil when S3 is not configured
	Services *Service
}

type Service struct {
	*UserService
	*AddressBookService
	*TagService
	*PeerService
	*GroupService
	*OauthService
	*LoginLogService
	*AuditService
	*ShareRecordService
	*ServerCmdService
	*LdapService
	*AppService
	*StrategyService
	*PeerCommandService
	*CustomClientService
	*BuildArtifactService
	*PreBuildService
	*WorkerService
	*WorkerRegistryService
}

func New(c *config.Config, g *gorm.DB, l *logger.Logger, j *jwt.Jwt, lo lock.Locker, s3c *s3.Client) *Service {
	sc := &ServiceContext{Config: c, DB: g, Logger: l, Jwt: j, Lock: lo, S3: s3c}
	s := &Service{
		UserService:        &UserService{ctx: sc},
		AddressBookService: &AddressBookService{ctx: sc},
		TagService:         &TagService{ctx: sc},
		PeerService:        &PeerService{ctx: sc},
		GroupService:       &GroupService{ctx: sc},
		OauthService:       &OauthService{ctx: sc},
		LoginLogService:    &LoginLogService{ctx: sc},
		AuditService:       &AuditService{ctx: sc},
		ShareRecordService: &ShareRecordService{ctx: sc},
		ServerCmdService:   &ServerCmdService{ctx: sc},
		LdapService:        &LdapService{ctx: sc},
		AppService:         &AppService{},
		StrategyService:      &StrategyService{ctx: sc},
		PeerCommandService:   &PeerCommandService{ctx: sc},
		CustomClientService:  &CustomClientService{ctx: sc},
		BuildArtifactService: &BuildArtifactService{ctx: sc},
		PreBuildService:      NewPreBuildService(sc),
		WorkerService:        &WorkerService{ctx: sc},
		WorkerRegistryService: &WorkerRegistryService{ctx: sc},
	}
	sc.Services = s // tie the knot so siblings can reach each other via ctx.Services
	return s
}

func Paginate(page, pageSize uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page == 0 {
			page = 1
		}
		if pageSize == 0 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		return db.Offset(int(offset)).Limit(int(pageSize))
	}
}

func CommonEnable() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", model.COMMON_STATUS_ENABLE)
	}
}

// Paginator is implemented by any list type that embeds model.Pagination.
type Paginator interface {
	SetPagination(page, pageSize, total int64)
}

// queryList is a generic helper that eliminates the repeated pagination+query pattern.
func queryList[T any](db *gorm.DB, page, pageSize uint, list Paginator, dest *[]*T, where func(*gorm.DB)) {
	tx := db.Model(new(T))
	if where != nil {
		where(tx)
	}
	var total int64
	tx.Count(&total)
	tx.Scopes(Paginate(page, pageSize)).Find(dest)
	list.SetPagination(int64(page), int64(pageSize), total)
}
