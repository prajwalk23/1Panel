package common

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func LoadDBConnByPath(fullPath, dbName string) *gorm.DB {
	db, err := GetDBWithPath(fullPath)
	if err != nil {
		panic(fmt.Errorf("init %s db failed, err: %w", dbName, err))
	}
	return db
}

func LoadDBConnByPathWithErr(fullPath, dbName string) (*gorm.DB, error) {
	db, err := GetDBWithPath(fullPath)
	if err != nil {
		return nil, fmt.Errorf("init %s db failed, err: %v", dbName, err)
	}
	return db, nil
}

func CloseDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func GetDBWithPath(dbPath string) (*gorm.DB, error) {
	dbDir := path.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return nil, fmt.Errorf("init db dir failed, err: %w", err)
	}
	if err := os.Chmod(dbDir, 0700); err != nil {
		return nil, fmt.Errorf("set db dir permissions failed, err: %w", err)
	}
	file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("init db file failed, err: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("set db file permissions failed, err: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.Chmod(dbPath+suffix, 0600); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("set db sidecar permissions failed, err: %w", err)
		}
	}

	// fork: pragmas go in the DSN so every pooled connection gets them (db.Exec only reaches one),
	// and cache_size is capped at 1 MiB per connection.
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-1024)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   newLogger(),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(2) // fork: was 4
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetConnMaxIdleTime(0)

	return db, nil
}

func newLogger() logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}
