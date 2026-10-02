package orm

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type SqliteConfig struct {
	Path         string
	MaxIdleConns int
	MaxOpenConns int
}

func NewSqlite(sqliteConf *SqliteConfig, logwriter logger.Writer) (*gorm.DB, error) {
	path := sqliteConf.Path
	if path == "" {
		path = "./data/rustdeskapi.db"
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Warn, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,        // Don't include params in the SQL log
				Colorful:                  false,
			},
		),
	})
	return finish(db, err, sqliteConf.MaxIdleConns, sqliteConf.MaxOpenConns)
}
