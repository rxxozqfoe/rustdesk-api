package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rxxozqfoe/rustdesk-api/internal/app"
	"github.com/rxxozqfoe/rustdesk-api/internal/config"
	apphttp "github.com/rxxozqfoe/rustdesk-api/internal/http"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/deps"
	"github.com/rxxozqfoe/rustdesk-api/internal/http/response"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/cache"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/jwt"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/lock"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/logger"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/orm"
	libs3 "github.com/rxxozqfoe/rustdesk-api/internal/lib/s3"
	"github.com/rxxozqfoe/rustdesk-api/internal/lib/upload"
	"github.com/rxxozqfoe/rustdesk-api/internal/model"
	"github.com/rxxozqfoe/rustdesk-api/internal/service"
	"github.com/rxxozqfoe/rustdesk-api/internal/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

const DatabaseVersion = 267

// @title 管理系统API
// @version 1.0
// @description 接口
// @basePath /api
// @securityDefinitions.apikey token
// @in header
// @name api-token
// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization

var configPath string

// Wiring — populated by InitApp (this file is the composition root). These
// package-level vars hold the fully constructed dependency graph so both the
// HTTP server and the cobra reset commands can reach them. They are not
// globals in the architectural sense: `main` is the only package that touches
// them, and `InitApp` is the only place that assigns them.
var (
	appCtx    *app.AppContext
	services  *service.Service
	handlers  *deps.HandlerDeps
	localizer app.LocalizerFunc
)

var rootCmd = &cobra.Command{
	Use:   "apimain",
	Short: "RUSTDESK API SERVER",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		InitApp()
	},
	Run: func(cmd *cobra.Command, args []string) {
		appCtx.Logger.Info("API SERVER START")
		apphttp.ApiInit(handlers)
	},
}

var resetPwdCmd = &cobra.Command{
	Use:     "reset-admin-pwd [pwd]",
	Example: "reset-admin-pwd 123456",
	Short:   "Reset Admin Password",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwd := args[0]
		admin := services.UserService.InfoById(1)
		if admin.Id == 0 {
			appCtx.Logger.Warn("user not found! ")
			return
		}
		if err := services.UpdatePassword(admin, pwd); err != nil {
			appCtx.Logger.Error("reset password fail! ", err)
			return
		}
		appCtx.Logger.Info("reset password success! ")
	},
}

var resetUserPwdCmd = &cobra.Command{
	Use:     "reset-pwd [userId] [pwd]",
	Example: "reset-pwd 2 123456",
	Short:   "Reset User Password",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		userId := args[0]
		pwd := args[1]
		uid, err := strconv.Atoi(userId)
		if err != nil {
			appCtx.Logger.Warn("userId must be int!")
			return
		}
		if uid <= 0 {
			appCtx.Logger.Warn("userId must be greater than 0! ")
			return
		}
		u := services.UserService.InfoById(uint(uid))
		if u.Id == 0 {
			appCtx.Logger.Warn("user not found! ")
			return
		}
		if err := services.UpdatePassword(u, pwd); err != nil {
			appCtx.Logger.Warn("reset password fail! ", err)
			return
		}
		appCtx.Logger.Info("reset password success!")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "conf/config.yaml", "choose config file")
	rootCmd.AddCommand(resetPwdCmd, resetUserPwdCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// InitApp is the composition root. It builds AppContext, constructs every
// dependency explicitly, wires the service aggregate, assembles HandlerDeps,
// and runs the database migration. No package-level singletons are set.
func InitApp() {
	a := &app.AppContext{ConfigPath: configPath}

	// Config
	a.Viper = config.Init(&a.Config, a.ConfigPath)

	// Logger
	a.Logger = logger.New(&logger.Config{
		Path:         a.Config.Logger.Path,
		Level:        a.Config.Logger.Level,
		ReportCaller: a.Config.Logger.ReportCaller,
	})

	// I18n localizer (constructor — returns a value, no global mutation)
	localizer = app.NewI18n(&a.Config)

	// Validator
	validator := app.NewValidator(&a.Config)

	// Redis
	a.Redis = redis.NewClient(&redis.Options{
		Addr:     a.Config.Redis.Addr,
		Password: a.Config.Redis.Password,
		DB:       a.Config.Redis.Db,
	})

	// Cache
	switch a.Config.Cache.Type {
	case cache.TypeFile:
		fc := cache.NewFileCache()
		fc.SetDir(a.Config.Cache.FileDir)
		a.Cache = fc
	case cache.TypeRedis:
		a.Cache = cache.NewRedis(&redis.Options{
			Addr:     a.Config.Cache.RedisAddr,
			Password: a.Config.Cache.RedisPwd,
			DB:       a.Config.Cache.RedisDb,
		})
	}

	// Database
	var db *gorm.DB
	switch a.Config.Gorm.Type {
	case config.TypeMysql:
		dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			a.Config.Mysql.Username,
			a.Config.Mysql.Password,
			a.Config.Mysql.Addr,
			a.Config.Mysql.Dbname,
			a.Config.Mysql.Tls,
		)
		db = orm.NewMysql(&orm.MysqlConfig{
			Dsn:          dsn,
			MaxIdleConns: a.Config.Gorm.MaxIdleConns,
			MaxOpenConns: a.Config.Gorm.MaxOpenConns,
		}, a.Logger)

	case config.TypePostgresql:
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			a.Config.Postgresql.Host,
			a.Config.Postgresql.Port,
			a.Config.Postgresql.User,
			a.Config.Postgresql.Password,
			a.Config.Postgresql.Dbname,
			a.Config.Postgresql.Sslmode,
			a.Config.Postgresql.TimeZone,
		)
		db = orm.NewPostgresql(&orm.PostgresqlConfig{
			Dsn:          dsn,
			MaxIdleConns: a.Config.Gorm.MaxIdleConns,
			MaxOpenConns: a.Config.Gorm.MaxOpenConns,
		}, a.Logger)

	case config.TypeSqlite:
		db = orm.NewSqlite(&orm.SqliteConfig{
			Path:         a.Config.Sqlite.Path,
			MaxIdleConns: a.Config.Gorm.MaxIdleConns,
			MaxOpenConns: a.Config.Gorm.MaxOpenConns,
		}, a.Logger)

	default:
		a.Logger.Fatalf("unsupported database type: %s", a.Config.Gorm.Type)
	}

	// OSS
	oss := &upload.Oss{
		AccessKeyId:     a.Config.Oss.AccessKeyId,
		AccessKeySecret: a.Config.Oss.AccessKeySecret,
		Host:            a.Config.Oss.Host,
		CallbackUrl:     a.Config.Oss.CallbackUrl,
		ExpireTime:      a.Config.Oss.ExpireTime,
		MaxByte:         a.Config.Oss.MaxByte,
	}

	// JWT & Lock
	jwtHandler := jwt.NewJwt(a.Config.Jwt.Key, a.Config.Jwt.ExpireDuration)
	locker := lock.NewLocal()

	// S3 client (nil when S3 is not configured)
	s3Client, err := libs3.New(&a.Config.S3)
	if err != nil {
		a.Logger.Fatal("S3 init failed: " + err.Error())
	}
	if s3Client != nil {
		if err := s3Client.EnsureBucket(context.Background(), a.Config.S3.Region); err != nil {
			a.Logger.Warn("S3 bucket check failed (will retry on use): " + err.Error())
		} else {
			a.Logger.Info("S3 storage enabled, bucket: " + a.Config.S3.Bucket)
		}
	}

	// Validate worker config at startup
	if a.Config.Worker.Enabled() {
		if a.Config.Worker.LogCacheDir == "" {
			a.Logger.Fatal("worker.token is set but worker.log-cache-dir is not configured")
		}
	}

	// Service aggregate (with back-pointer — see service.New)
	svcs := service.New(&a.Config, db, a.Logger, jwtHandler, locker, s3Client)

	// Login limiter (HTTP-layer concern)
	a.Logger.Info(fmt.Sprintf("CaptchaThreshold: %d, BanThreshold: %d", a.Config.App.CaptchaThreshold, a.Config.App.BanThreshold))
	limiter := utils.NewLoginLimiter(utils.SecurityPolicy{
		CaptchaThreshold: a.Config.App.CaptchaThreshold,
		BanThreshold:     a.Config.App.BanThreshold,
		AttemptsWindow:   10 * time.Minute,
		BanDuration:      30 * time.Minute,
	})
	limiter.RegisterProvider(utils.B64StringCaptchaProvider{})

	// HandlerDeps — the aggregate threaded into every HTTP controller,
	// middleware, and router bind function.
	hd := &deps.HandlerDeps{
		Config:       &a.Config,
		Logger:       a.Logger,
		Validator:    validator,
		Localizer:    deps.LocalizerFunc(localizer),
		LoginLimiter: limiter,
		Oss:          oss,
		S3:           s3Client,
		Services:     svcs,
	}

	// Wire http/response's single permitted func-pointer hooks so
	// TranslateMsg and friends work without pulling from a global.
	response.SetLocalizer(func(lang string) *i18n.Localizer { return localizer(lang) })
	response.SetLogger(a.Logger)

	// Database migration (explicit params — no globals)
	DatabaseAutoUpdate(db, a, svcs, localizer)

	// Close stale audit connections from previous server runs
	if err := svcs.CloseStaleConns(); err != nil {
		a.Logger.Errorf("failed to close stale audit connections: %v", err)
	}

	// Publish to package-level wiring vars so cobra commands can reach them.
	appCtx = a
	services = svcs
	handlers = hd
}

func DatabaseAutoUpdate(db *gorm.DB, a *app.AppContext, svcs *service.Service, localizer app.LocalizerFunc) {
	version := DatabaseVersion

	if a.Config.Gorm.Type == config.TypeMysql {
		dbName := db.Migrator().CurrentDatabase()
		if dbName == "" {
			dbName = a.Config.Mysql.Dbname
			dsnWithoutDB := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				a.Config.Mysql.Username,
				a.Config.Mysql.Password,
				a.Config.Mysql.Addr,
				"",
			)
			dbWithoutDB := orm.NewMysql(&orm.MysqlConfig{
				Dsn: dsnWithoutDB,
			}, a.Logger)
			sqlDBWithoutDB, err := dbWithoutDB.DB()
			if err != nil {
				a.Logger.Errorf("获取底层 *sql.DB 对象失败: %v", err)
				return
			}
			defer func() {
				if err := sqlDBWithoutDB.Close(); err != nil {
					a.Logger.Errorf("关闭连接失败: %v", err)
				}
			}()

			if err := dbWithoutDB.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + " DEFAULT CHARSET utf8mb4").Error; err != nil {
				a.Logger.Error(err)
				return
			}
		}
	}

	if !db.Migrator().HasTable(&model.Version{}) {
		Migrate(db, a, svcs, localizer, uint(version))
	} else {
		var v model.Version
		db.Last(&v)
		if v.Version < uint(version) {
			Migrate(db, a, svcs, localizer, uint(version))
		}

		if v.Version < 245 {
			db.Exec("update oauths set oauth_type = op")
			db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google'")
			db.Exec("update user_thirds set oauth_type = third_type, op = third_type")
			uts := make([]model.UserThird, 0)
			db.Where("oauth_type = ?", "google").Find(&uts)
			for _, ut := range uts {
				if ut.UserId > 0 {
					db.Model(&model.User{}).Where("id = ?", ut.UserId).Update("email", ut.OpenId)
				}
			}
		}
		if v.Version < 246 {
			db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google' and issuer is null")
		}
	}
}

func Migrate(db *gorm.DB, a *app.AppContext, svcs *service.Service, localizer app.LocalizerFunc, version uint) {
	a.Logger.Info("Migrating....", version)
	err := db.AutoMigrate(
		&model.Version{},
		&model.User{},
		&model.UserToken{},
		&model.Tag{},
		&model.AddressBook{},
		&model.Peer{},
		&model.Group{},
		&model.UserThird{},
		&model.Oauth{},
		&model.LoginLog{},
		&model.ShareRecord{},
		&model.AuditConn{},
		&model.AuditFile{},
		&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},
		&model.ServerCmd{},
		&model.DeviceGroup{},
		&model.PeerCommand{},
		&model.Strategy{},
		&model.StrategyPeer{},
		&model.StrategyUser{},
		&model.StrategyDeviceGroup{},
		&model.CustomClient{},
		&model.BuildArtifact{},
		&model.PreBuild{},
		&model.Worker{},
	)
	if err != nil {
		a.Logger.Error("migrate err :=>", err)
	}
	db.Create(&model.Version{Version: version})
	var vc int64
	db.Model(&model.Version{}).Count(&vc)
	if vc == 1 {
		loc := localizer("")
		defaultGroup, _ := loc.LocalizeMessage(&i18n.Message{ID: "DefaultGroup"})
		if err := svcs.GroupService.Create(&model.Group{
			Name: defaultGroup,
			Type: model.GroupTypeDefault,
		}); err != nil {
			a.Logger.Error("create default group err :=>", err)
		}

		shareGroup, _ := loc.LocalizeMessage(&i18n.Message{ID: "ShareGroup"})
		if err := svcs.GroupService.Create(&model.Group{
			Name: shareGroup,
			Type: model.GroupTypeShare,
		}); err != nil {
			a.Logger.Error("create share group err :=>", err)
		}

		isAdmin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "Admin",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &isAdmin,
			GroupId:  1,
		}
		pwd := utils.RandomString(8)
		a.Logger.Info("Admin Password Is: ", pwd)
		var pwdErr error
		admin.Password, pwdErr = utils.EncryptPassword(pwd)
		if pwdErr != nil {
			a.Logger.Fatalf("failed to generate admin password: %v", pwdErr)
		}
		db.Create(admin)
	}
}
