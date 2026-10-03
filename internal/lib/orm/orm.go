package orm

import (
	"time"

	"gorm.io/gorm"
)

const (
	retryFirstWait = time.Second
	retryMaxWait   = 10 * time.Second
)

// finish applies the pool limits to a database that opened, and closes one
// that failed to (gorm.Open keeps the pool open when only the ping fails), so
// that retrying the open does not leak connection pools.
func finish(db *gorm.DB, err error, maxIdleConns, maxOpenConns int) (*gorm.DB, error) {
	if err != nil {
		if db != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(maxIdleConns)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(maxOpenConns)

	return db, nil
}

// OpenWithRetry calls open until it succeeds or timeout has passed, waiting
// 1s, 2s, 4s ... (at most 10s) between attempts, and returns the last error
// once the time is spent. A database that starts alongside the api (Docker
// Compose, Kubernetes) is usually not accepting connections yet. retrying,
// if non-nil, is called before each wait.
func OpenWithRetry(timeout time.Duration, open func() (*gorm.DB, error), retrying func(err error, wait time.Duration)) (*gorm.DB, error) {
	deadline := time.Now().Add(timeout)
	wait := retryFirstWait
	for {
		db, err := open()
		if err == nil {
			return db, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, err
		}
		wait = min(wait, remaining)
		if retrying != nil {
			retrying(err, wait)
		}
		time.Sleep(wait)
		wait = min(wait*2, retryMaxWait)
	}
}
