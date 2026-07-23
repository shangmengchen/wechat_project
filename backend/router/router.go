package router

import (
	"net/http"

	"couple-mini/backend/api"
	"couple-mini/backend/configs"
	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/pkg/httpmw"
	"couple-mini/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func SetRouter(svc *service.Service) *gin.Engine {
	cfg := configs.GetGlobalConfig()
	if cfg.AppConfig.RunMode != "" {
		gin.SetMode(cfg.AppConfig.RunMode)
	}

	handler := api.New(svc)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(httpmw.RequestID(), httpmw.AccessLog(), httpmw.Recovery(), cors())
	r.Static("/uploads", "./uploads")

	r.GET("/healthz", handler.Health)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", handler.Login)
		v1.POST("/pair/code", handler.GeneratePairCode)
		v1.POST("/pair/confirm", handler.ConfirmPair)
		v1.PATCH("/couple/love-date", handler.UpdateLoveDate)
		v1.PATCH("/users/:id/profile", handler.UpdateUserProfile)
		v1.GET("/dashboard", handler.Dashboard)
		v1.POST("/uploads/images", handler.UploadImage)

		v1.GET("/moments", handler.Moments)
		v1.POST("/moments", handler.CreateMoment)
		v1.DELETE("/moments/:id", handler.DeleteMoment)
		v1.PATCH("/moments/:id/liked", handler.UpdateMomentLiked)

		v1.GET("/tasks", handler.Tasks)
		v1.POST("/tasks", handler.CreateTask)
		v1.DELETE("/tasks/:id", handler.DeleteTask)
		v1.POST("/tasks/:id/complete", handler.TaskAction(model.TaskReview))
		v1.POST("/tasks/:id/approve", handler.TaskAction(model.TaskDone))
		v1.POST("/tasks/:id/reject", handler.TaskAction(model.TaskTodo))

		v1.GET("/scheduled-tasks", handler.ScheduledTasks)
		v1.POST("/scheduled-tasks", handler.CreateScheduledTask)
		v1.DELETE("/scheduled-tasks/:id", handler.DeleteScheduledTask)
		v1.POST("/scheduled-tasks/:id/confirm", handler.ConfirmScheduledTask)

		v1.GET("/dishes", handler.Dishes)
		v1.POST("/dishes", handler.CreateDish)
		v1.DELETE("/dishes/:id", handler.DeleteDish)
		v1.PATCH("/dishes/:id/enabled", handler.UpdateDishEnabled)

		v1.GET("/orders", handler.Orders)
		v1.POST("/orders", handler.CreateOrder)

		v1.GET("/goals", handler.Goals)
		v1.POST("/goals", handler.CreateGoal)
		v1.PATCH("/goals/:id/value", handler.UpdateGoalValue)
		v1.PATCH("/goals/:id/status", handler.UpdateGoalStatus)
		v1.DELETE("/goals/:id", handler.DeleteGoal)
	}

	admin := r.Group("/admin")
	admin.Use(httpmw.AdminBasicAuth())
	{
		admin.Static("/assets", "./web/admin/assets")
		admin.GET("", handler.AdminPage)
		admin.GET("/", handler.AdminPage)
		admin.GET("/api/meta", handler.AdminMeta)
		admin.GET("/api/dashboard", handler.AdminDashboard)
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
