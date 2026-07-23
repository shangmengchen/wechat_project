package api

import (
	"path/filepath"
	"strconv"

	"couple-mini/backend/configs"

	"github.com/gin-gonic/gin"
)

func (api *API) AdminPage(c *gin.Context) {
	c.File(filepath.Join("web", "admin", "index.html"))
}

func (api *API) AdminDashboard(c *gin.Context) {
	data, err := api.service.AdminDashboard()
	respond(c, data, err)
}

func (api *API) AdminErrors(c *gin.Context) {
	limit := 20
	if value := c.Query("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	data, err := api.service.AdminErrors(limit)
	respond(c, data, err)
}

func (api *API) AdminMeta(c *gin.Context) {
	cfg := configs.GetGlobalConfig()
	ok(c, gin.H{
		"title":   cfg.AdminConfig.Title,
		"appName": cfg.AppConfig.AppName,
		"version": cfg.AppConfig.Version,
		"runMode": cfg.AppConfig.RunMode,
	})
}
