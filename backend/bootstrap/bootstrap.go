package bootstrap

import (
	"context"
	"fmt"
	"time"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/pkg/adminview"
	"couple-mini/backend/internal/pkg/gormcli"
	applog "couple-mini/backend/internal/pkg/logger"
	"couple-mini/backend/internal/repo"
	"couple-mini/backend/internal/service"
	"couple-mini/backend/router"
)

func Run() error {
	configs.InitGlobalConfig()
	cfg := configs.GetGlobalConfig()

	if err := applog.Init(cfg.LogConfig); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer applog.Close()

	applog.L().Info("backend starting",
		"app", cfg.AppConfig.AppName,
		"version", cfg.AppConfig.Version,
		"run_mode", cfg.AppConfig.RunMode,
		"port", cfg.AppConfig.Port,
	)
	adminview.Start(
		cfg.AppConfig.AppName,
		cfg.AppConfig.Version,
		cfg.AppConfig.RunMode,
		time.Duration(cfg.AdminConfig.SampleIntervalSec)*time.Second,
		cfg.AdminConfig.HistoryLimit,
	)
	defer adminview.Stop()

	db, err := gormcli.GetDB()
	if err != nil {
		return err
	}

	repository := repo.New(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.EnsureSchema(ctx, cfg.DbConfig.AutoMigrate, cfg.DbConfig.AutoSeed); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	applog.L().Info("database schema ready",
		"auto_migrate", cfg.DbConfig.AutoMigrate,
		"auto_seed", cfg.DbConfig.AutoSeed,
	)

	svc := service.New(repository)
	r := router.SetRouter(svc)
	addr := fmt.Sprintf(":%d", cfg.AppConfig.Port)
	applog.L().Info("http server listening", "addr", addr)
	return r.Run(addr)
}
