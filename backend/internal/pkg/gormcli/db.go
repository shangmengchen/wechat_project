package gormcli

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"couple-mini/backend/configs"
	applog "couple-mini/backend/internal/pkg/logger"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	db      *gorm.DB
	initErr error
	once    sync.Once
)

func GetDB() (*gorm.DB, error) {
	once.Do(func() {
		db, initErr = openDB()
	})
	return db, initErr
}

func openDB() (*gorm.DB, error) {
	dbConfig := configs.GetGlobalConfig().DbConfig
	logConfig := configs.GetGlobalConfig().LogConfig

	dsn, driverCfg, err := resolveDSN(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve mysql dsn: %w", err)
	}

	if dbConfig.CreateDB {
		if err := ensureDatabase(driverCfg); err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newGORMLogger(logConfig),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("fetch sql db: %w", err)
	}
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConn)
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConn)
	sqlDB.SetConnMaxIdleTime(time.Duration(dbConfig.MaxIdleTime) * time.Second)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	applog.L().Info("database connected",
		"host", driverCfg.Addr,
		"database", driverCfg.DBName,
		"max_idle_conn", dbConfig.MaxIdleConn,
		"max_open_conn", dbConfig.MaxOpenConn,
	)
	return db, nil
}

func resolveDSN(cfg configs.DbConfig) (string, *mysqldriver.Config, error) {
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		driverCfg, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			return "", nil, err
		}
		driverCfg.ParseTime = true
		driverCfg.Loc = time.Local
		if driverCfg.Params == nil {
			driverCfg.Params = map[string]string{}
		}
		driverCfg.Params["charset"] = "utf8mb4"
		if driverCfg.DBName == "" {
			driverCfg.DBName = cfg.DBName
		}
		return formatDSN(driverCfg), driverCfg, nil
	}

	driverCfg := mysqldriver.NewConfig()
	driverCfg.User = cfg.User
	driverCfg.Passwd = cfg.Password
	driverCfg.Net = "tcp"
	driverCfg.Addr = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	driverCfg.DBName = cfg.DBName
	driverCfg.ParseTime = true
	driverCfg.Loc = time.Local
	driverCfg.Params = map[string]string{
		"charset": "utf8mb4",
	}
	return formatDSN(driverCfg), driverCfg, nil
}

func ensureDatabase(driverCfg *mysqldriver.Config) error {
	if driverCfg == nil || driverCfg.DBName == "" {
		return nil
	}

	adminCfg := *driverCfg
	adminCfg.DBName = ""
	rawDB, err := sql.Open("mysql", formatDSN(&adminCfg))
	if err != nil {
		return fmt.Errorf("open mysql admin connection: %w", err)
	}
	defer rawDB.Close()

	if err := rawDB.Ping(); err != nil {
		return fmt.Errorf("ping mysql admin connection: %w", err)
	}

	dbName := strings.ReplaceAll(driverCfg.DBName, "`", "``")
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	if _, err := rawDB.Exec(query); err != nil {
		return fmt.Errorf("create database %s: %w", driverCfg.DBName, err)
	}

	applog.L().Info("database ensured", "database", driverCfg.DBName)
	return nil
}

func newGORMLogger(cfg configs.LogConfig) gormlogger.Interface {
	level := parseGORMLevel(cfg.SQLLevel)
	writerLevel := slog.LevelInfo
	switch level {
	case gormlogger.Error:
		writerLevel = slog.LevelError
	case gormlogger.Warn:
		writerLevel = slog.LevelWarn
	}

	return gormlogger.New(
		log.New(&slogWriter{
			logger: applog.L().With("component", "gorm"),
			level:  writerLevel,
		}, "", 0),
		gormlogger.Config{
			SlowThreshold:             time.Duration(cfg.SQLSlowMS) * time.Millisecond,
			IgnoreRecordNotFoundError: true,
			LogLevel:                  level,
			Colorful:                  false,
		},
	)
}

func parseGORMLevel(value string) gormlogger.LogLevel {
	switch strings.ToLower(value) {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

type slogWriter struct {
	logger *slog.Logger
	level  slog.Level
}

func (w *slogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		w.logger.Log(context.Background(), w.level, msg)
	}
	return len(p), nil
}

func formatDSN(cfg *mysqldriver.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.Passwd == "" {
		dbName := ""
		if cfg.DBName != "" {
			dbName = "/" + cfg.DBName
		} else {
			dbName = "/"
		}
		return fmt.Sprintf("%s@tcp(%s)%s?charset=utf8mb4&parseTime=true&loc=Local", cfg.User, cfg.Addr, dbName)
	}
	return cfg.FormatDSN()
}
