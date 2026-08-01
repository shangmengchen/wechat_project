package router

import (
	"net/http"

	"couple-mini/backend/api"
	"couple-mini/backend/configs"
	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/pkg/httpmw"
	"couple-mini/backend/internal/pkg/push"
	wechatcli "couple-mini/backend/internal/pkg/wechat"
	"couple-mini/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func SetRouter(svc *service.Service) *gin.Engine {
	cfg := configs.GetGlobalConfig()
	if cfg.AppConfig.RunMode != "" {
		gin.SetMode(cfg.AppConfig.RunMode)
	}

	pushHub := push.NewHub()
	handler := api.New(svc, wechatcli.NewClient(cfg.WeChatConfig.AppID, cfg.WeChatConfig.Secret), pushHub)
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20
	_ = r.SetTrustedProxies(nil)
	r.Use(httpmw.RequestID(), httpmw.AccessLog(), httpmw.Recovery(), securityHeaders(), cors())
	r.Static("/uploads", "./uploads")

	r.GET("/healthz", handler.Health)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", handler.Login)
		v1.GET("/events", handler.PairEvents)
		authV1 := v1.Group("")
		authV1.Use(httpmw.RequireAuth())
		authV1.GET("/sync/state", handler.SyncState)
		authV1.POST("/pair/code", handler.GeneratePairCode)
		authV1.POST("/pair/confirm", handler.ConfirmPair)
		authV1.POST("/pair/unbind", handler.Unpair)
		authV1.PATCH("/couple/love-date", handler.UpdateLoveDate)
		authV1.PATCH("/users/:id/profile", handler.UpdateUserProfile)
		authV1.GET("/dashboard", handler.Dashboard)
		authV1.POST("/uploads/images", handler.UploadImage)
		authV1.GET("/subscribe/templates", handler.SubscribeTemplates)
		authV1.GET("/notices/unread", handler.UnreadNotices)
		authV1.POST("/notices/read", handler.MarkNoticesRead)

		authV1.GET("/moments", handler.Moments)
		authV1.POST("/moments", handler.CreateMoment)
		authV1.DELETE("/moments/:id", handler.DeleteMoment)
		authV1.PATCH("/moments/:id/liked", handler.UpdateMomentLiked)

		authV1.GET("/tasks", handler.Tasks)
		authV1.POST("/tasks", handler.CreateTask)
		authV1.DELETE("/tasks/:id", handler.DeleteTask)
		authV1.POST("/tasks/:id/complete", handler.TaskAction(model.TaskReview))
		authV1.POST("/tasks/:id/approve", handler.TaskAction(model.TaskDone))
		authV1.POST("/tasks/:id/reject", handler.TaskAction(model.TaskTodo))

		authV1.GET("/scheduled-tasks", handler.ScheduledTasks)
		authV1.POST("/scheduled-tasks", handler.CreateScheduledTask)
		authV1.DELETE("/scheduled-tasks/:id", handler.DeleteScheduledTask)
		authV1.POST("/scheduled-tasks/:id/confirm", handler.ConfirmScheduledTask)

		authV1.GET("/dishes", handler.Dishes)
		authV1.POST("/dishes", handler.CreateDish)
		authV1.DELETE("/dishes/:id", handler.DeleteDish)
		authV1.PATCH("/dishes/:id/enabled", handler.UpdateDishEnabled)

		authV1.GET("/orders", handler.Orders)
		authV1.POST("/orders", handler.CreateOrder)

		authV1.GET("/goals", handler.Goals)
		authV1.POST("/goals", handler.CreateGoal)
		authV1.PATCH("/goals/:id/value", handler.UpdateGoalValue)
		authV1.PATCH("/goals/:id/status", handler.UpdateGoalStatus)
		authV1.DELETE("/goals/:id", handler.DeleteGoal)
	}

	admin := r.Group("/admin")
	admin.Use(httpmw.AdminBasicAuth())
	{
		admin.Static("/assets", "./web/admin/assets")
		admin.GET("", handler.AdminPage)
		admin.GET("/", handler.AdminPage)
		admin.GET("/api/meta", handler.AdminMeta)
		admin.GET("/api/dashboard", handler.AdminDashboard)
		admin.GET("/api/couples", handler.AdminCouples)
		admin.POST("/api/couples/:id/unbind", handler.AdminUnpairCouple)
		admin.GET("/api/errors", handler.AdminErrors)
	}

	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
