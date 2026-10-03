package config

import "time"

const (
	TypeSqlite     = "sqlite"
	TypeMysql      = "mysql"
	TypePostgresql = "postgresql"
)

// DefaultConnectTimeout is used when gorm.connect-timeout is unset or not positive.
const DefaultConnectTimeout = 60 * time.Second

type Gorm struct {
	Type         string `mapstructure:"type"`
	MaxIdleConns int    `mapstructure:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns"`
	// How long startup keeps retrying a database that does not accept
	// connections yet before the api exits.
	ConnectTimeout time.Duration `mapstructure:"connect-timeout"`
}

type Sqlite struct {
	Path string `mapstructure:"path"`
}

type Mysql struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
	Tls      string `mapstructure:"tls"` // true / false / skip-verify / custom
}

type Postgresql struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
	Sslmode  string `mapstructure:"sslmode"`   // "disable", "require", "verify-ca", "verify-full"
	TimeZone string `mapstructure:"time-zone"` // e.g., "Asia/Shanghai"
}
