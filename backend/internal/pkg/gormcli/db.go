package gormcli

import (
	"fmt"
	"os"
	"sync"
	"time"

	"couple-mini/backend/configs"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func GetDB() *gorm.DB {
	once.Do(openDB)
	return db
}

func openDB() {
	dbConfig := configs.GetGlobalConfig().DbConfig
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = buildDSN(dbConfig)
	}

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("open db err: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("fetch db err: %v", err))
	}
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConn)
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConn)
	sqlDB.SetConnMaxIdleTime(time.Duration(dbConfig.MaxIdleTime) * time.Second)
}

func buildDSN(cfg configs.DbConfig) string {
	if cfg.Password == "" {
		return fmt.Sprintf("%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			cfg.User, cfg.Host, cfg.Port, cfg.DBName)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
