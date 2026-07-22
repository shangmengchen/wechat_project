package bootstrap

import (
	"context"
	"fmt"
	"time"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/pkg/gormcli"
	"couple-mini/backend/internal/repo"
	"couple-mini/backend/internal/service"
	"couple-mini/backend/router"
)

func Run() error {
	configs.InitGlobalConfig()
	cfg := configs.GetGlobalConfig()

	repository := repo.New(gormcli.GetDB())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.EnsureSchema(ctx); err != nil {
		return err
	}

	svc := service.New(repository)
	r := router.SetRouter(svc)
	return r.Run(fmt.Sprintf(":%d", cfg.AppConfig.Port))
}
