package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DebugMode     = "debug"
	ReleaseMode   = "release"
	DefaultConfig = "conf/config.yaml"
	DevConfig     = "conf/config.dev.yaml"
)

type App struct {
	WebClient        int           `mapstructure:"web-client"`
	Register         bool          `mapstructure:"register"`
	RegisterStatus   int           `mapstructure:"register-status"`
	ShowSwagger      int           `mapstructure:"show-swagger"`
	TokenExpire      time.Duration `mapstructure:"token-expire"`
	WebSso           bool          `mapstructure:"web-sso"`
	DisablePwdLogin  bool          `mapstructure:"disable-pwd-login"`
	CaptchaThreshold int           `mapstructure:"captcha-threshold"`
	BanThreshold     int           `mapstructure:"ban-threshold"`
}
type Admin struct {
	Title           string `mapstructure:"title"`
	Hello           string `mapstructure:"hello"`
	HelloFile       string `mapstructure:"hello-file"`
	IdServerPort    int    `mapstructure:"id-server-port"`
	RelayServerPort int    `mapstructure:"relay-server-port"`
	// Password, when set, is the admin account's password: it is applied on
	// every start and cannot be changed through the api. PasswordFile reads
	// it from a file instead (e.g. a mounted Secret); Password wins.
	Password     string `mapstructure:"password"`
	PasswordFile string `mapstructure:"password-file"`
}

// Length bounds of a managed admin password, the same as the admin
// password forms accept.
const (
	AdminPasswordMinLen = 4
	AdminPasswordMaxLen = 32
)

type Config struct {
	Lang       string `mapstructure:"lang"`
	App        App
	Admin      Admin
	Gorm       Gorm
	Sqlite     Sqlite
	Mysql      Mysql
	Postgresql Postgresql
	Gin        Gin
	Logger     Logger
	Redis      Redis
	Cache      Cache
	Oss        Oss
	Jwt        Jwt
	Rustdesk   Rustdesk
	Proxy      Proxy
	Ldap       Ldap
	S3         S3     `mapstructure:"s3"`
	Worker     Worker `mapstructure:"worker"`
	Hbbs       Hbbs   `mapstructure:"hbbs"`
}

func (a *Admin) Init() {
	if a.IdServerPort == 0 {
		a.IdServerPort = DefaultIdServerPort
	}
	if a.RelayServerPort == 0 {
		a.RelayServerPort = DefaultRelayServerPort
	}
}

// LoadPassword resolves admin.password, reading admin.password-file when only
// the file is given (its trailing newline dropped). A password that is
// configured but unusable is an error rather than silently leaving the admin
// password unmanaged.
func (a *Admin) LoadPassword() error {
	if a.Password == "" && a.PasswordFile != "" {
		b, err := os.ReadFile(a.PasswordFile)
		if err != nil {
			return fmt.Errorf("admin.password-file: %w", err)
		}
		a.Password = strings.TrimRight(string(b), "\r\n")
		if a.Password == "" {
			return fmt.Errorf("admin.password-file %s is empty", a.PasswordFile)
		}
	}
	if a.Password == "" {
		return nil
	}
	if n := utf8.RuneCountInString(a.Password); n < AdminPasswordMinLen || n > AdminPasswordMaxLen {
		return fmt.Errorf("admin.password must be %d to %d characters, got %d", AdminPasswordMinLen, AdminPasswordMaxLen, n)
	}
	return nil
}

// Init 初始化配置
func Init(rowVal *Config, path string) *viper.Viper {
	if path == "" || path == DefaultConfig {
		if _, err := os.Stat(DevConfig); err == nil {
			path = DevConfig
		} else {
			path = DefaultConfig
		}
	}
	v := viper.GetViper()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.SetEnvPrefix("RUSTDESK_API")
	// Env overrides only reach keys viper knows of; declare these so the env
	// vars work with a config file that predates them.
	v.SetDefault("admin.password", "")
	v.SetDefault("admin.password-file", "")
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	/*
		v.WatchConfig()


			//监听配置修改没什么必要
			v.OnConfigChange(func(e fsnotify.Event) {
				//配置文件修改监听
				fmt.Println("config file changed:", e.Name)
				if err2 := v.Unmarshal(rowVal); err2 != nil {
					fmt.Println(err2)
				}
				rowVal.Rustdesk.LoadKeyFile()
				rowVal.Rustdesk.ParsePort()
			})
	*/
	if err := v.Unmarshal(rowVal); err != nil {
		panic(fmt.Errorf("fatal error config: %s", err))
	}
	rowVal.Rustdesk.LoadKeyFile()
	rowVal.Admin.Init()
	if err := rowVal.Admin.LoadPassword(); err != nil {
		panic(fmt.Errorf("fatal error config: %s", err))
	}
	return v
}

// ReadEnv 读取环境变量
func ReadEnv(rowVal interface{}) *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	if err := v.Unmarshal(rowVal); err != nil {
		fmt.Println(err)
	}
	return v
}
