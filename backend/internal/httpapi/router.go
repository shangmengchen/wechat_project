package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"couple-mini/backend/internal/domain"
	"couple-mini/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type Store interface {
	Login(openid, nickname string) (domain.User, error)
	GeneratePairCode(userID string) (domain.Couple, error)
	PairByCode(userID, code, loveDate string) (domain.Couple, error)
	UpdateLoveDate(loveDate string) (domain.Couple, error)
	UpdateUserProfile(user domain.User) (domain.User, error)
	Dashboard(userID string) (domain.DashboardPayload, error)
	Moments() ([]domain.Moment, error)
	AddMoment(moment domain.Moment) (domain.Moment, error)
	DeleteMoment(id string) error
	UpdateMomentLiked(id string, liked bool) (domain.Moment, error)
	Tasks() ([]domain.Task, error)
	AddTask(task domain.Task) (domain.Task, error)
	DeleteTask(id string) error
	UpdateTaskStatus(id string, status domain.TaskStatus) (domain.Task, error)
	ScheduledTasks() ([]domain.ScheduledTask, error)
	AddScheduledTask(task domain.ScheduledTask) (domain.ScheduledTask, error)
	DeleteScheduledTask(id string) error
	ConfirmScheduledTask(id string) (domain.ScheduledTask, error)
	Dishes() ([]domain.Dish, error)
	AddDish(dish domain.Dish) (domain.Dish, error)
	DeleteDish(id string) error
	UpdateDishEnabled(id string, enabled bool) (domain.Dish, error)
	Orders() ([]domain.Order, error)
	AddOrder(order domain.Order) (domain.Order, error)
	Goals() ([]domain.Goal, error)
	AddGoal(goal domain.Goal) (domain.Goal, error)
	UpdateGoalValue(id string, currentValue int) (domain.Goal, error)
	UpdateGoalStatus(id, status string) (domain.Goal, error)
	DeleteGoal(id string) error
}

type API struct {
	store Store
}

func NewRouter(store Store) http.Handler {
	api := API{store: store}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors())
	router.Static("/uploads", "./uploads")

	router.GET("/healthz", api.health)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", api.login)
		v1.POST("/pair/code", api.generatePairCode)
		v1.POST("/pair/confirm", api.confirmPair)
		v1.PATCH("/couple/love-date", api.updateLoveDate)
		v1.PATCH("/users/:id/profile", api.updateUserProfile)
		v1.GET("/dashboard", api.dashboard)
		v1.POST("/uploads/images", api.uploadImage)

		v1.GET("/moments", api.moments)
		v1.POST("/moments", api.createMoment)
		v1.DELETE("/moments/:id", api.deleteMoment)
		v1.PATCH("/moments/:id/liked", api.updateMomentLiked)

		v1.GET("/tasks", api.tasks)
		v1.POST("/tasks", api.createTask)
		v1.DELETE("/tasks/:id", api.deleteTask)
		v1.POST("/tasks/:id/complete", api.taskAction(domain.TaskReview))
		v1.POST("/tasks/:id/approve", api.taskAction(domain.TaskDone))
		v1.POST("/tasks/:id/reject", api.taskAction(domain.TaskTodo))

		v1.GET("/scheduled-tasks", api.scheduledTasks)
		v1.POST("/scheduled-tasks", api.createScheduledTask)
		v1.DELETE("/scheduled-tasks/:id", api.deleteScheduledTask)
		v1.POST("/scheduled-tasks/:id/confirm", api.confirmScheduledTask)

		v1.GET("/dishes", api.dishes)
		v1.POST("/dishes", api.createDish)
		v1.DELETE("/dishes/:id", api.deleteDish)
		v1.PATCH("/dishes/:id/enabled", api.updateDishEnabled)

		v1.GET("/orders", api.orders)
		v1.POST("/orders", api.createOrder)

		v1.GET("/goals", api.goals)
		v1.POST("/goals", api.createGoal)
		v1.PATCH("/goals/:id/value", api.updateGoalValue)
		v1.PATCH("/goals/:id/status", api.updateGoalStatus)
		v1.DELETE("/goals/:id", api.deleteGoal)
	}

	return router
}

func (api API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (api API) login(c *gin.Context) {
	var req struct {
		Code     string `json:"code"`
		OpenID   string `json:"openid"`
		Nickname string `json:"nickname"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.OpenID == "" {
		req.OpenID = "mock-openid-" + req.Code
	}
	if req.Nickname == "" {
		req.Nickname = "微信用户"
	}
	user, err := api.store.Login(req.OpenID, req.Nickname)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"token": "demo-token-" + user.ID, "user": user})
}

func (api API) generatePairCode(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		badRequest(c, "userId required")
		return
	}
	couple, err := api.store.GeneratePairCode(defaultUser(req.UserID))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, couple)
}

func (api API) confirmPair(c *gin.Context) {
	var req struct {
		UserID   string `json:"userId"`
		Code     string `json:"code"`
		LoveDate string `json:"loveDate"`
	}
	if !bindJSON(c, &req) {
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if !regexp.MustCompile(`^\d{6}$`).MatchString(req.Code) {
		badRequest(c, "分享码输入错误")
		return
	}
	couple, err := api.store.PairByCode(defaultUser(req.UserID), req.Code, req.LoveDate)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, couple)
}

func (api API) updateLoveDate(c *gin.Context) {
	var req struct {
		LoveDate string `json:"loveDate"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.LoveDate) == "" {
		badRequest(c, "loveDate required")
		return
	}
	couple, err := api.store.UpdateLoveDate(req.LoveDate)
	respond(c, couple, err)
}

func (api API) updateUserProfile(c *gin.Context) {
	var req domain.User
	if !bindJSON(c, &req) {
		return
	}
	req.ID = c.Param("id")
	if strings.TrimSpace(req.Nickname) == "" {
		badRequest(c, "nickname required")
		return
	}
	user, err := api.store.UpdateUserProfile(req)
	respond(c, user, err)
}

func (api API) dashboard(c *gin.Context) {
	data, err := api.store.Dashboard(c.Query("userId"))
	respond(c, data, err)
}

func (api API) uploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "file required")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		badRequest(c, "unsupported image type")
		return
	}
	dir := filepath.Join("uploads", "images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fail(c, err)
		return
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relative := "/uploads/images/" + name
	if err := c.SaveUploadedFile(file, filepath.Join(dir, name)); err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"url": absoluteURL(c, relative)})
}

func (api API) moments(c *gin.Context) {
	data, err := api.store.Moments()
	respond(c, data, err)
}

func (api API) createMoment(c *gin.Context) {
	var req domain.Moment
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		badRequest(c, "content required")
		return
	}
	if req.Author == "" {
		req.Author = "小雨"
	}
	if req.Avatar == "" {
		req.Avatar = "小"
	}
	req.TimeLabel = "刚刚"
	data, err := api.store.AddMoment(req)
	respond(c, data, err)
}

func (api API) deleteMoment(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.store.DeleteMoment(c.Param("id")))
}

func (api API) updateMomentLiked(c *gin.Context) {
	var req struct {
		Liked bool `json:"liked"`
	}
	if !bindJSON(c, &req) {
		return
	}
	data, err := api.store.UpdateMomentLiked(c.Param("id"), req.Liked)
	respond(c, data, err)
}

func (api API) tasks(c *gin.Context) {
	data, err := api.store.Tasks()
	respond(c, data, err)
}

func (api API) createTask(c *gin.Context) {
	var req domain.Task
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Owner == "" {
		req.Owner = "双方"
	}
	if req.Type == "" {
		req.Type = "一次性"
	}
	if strings.TrimSpace(req.Tag) == "" {
		req.Tag = "生活"
	}
	data, err := api.store.AddTask(req)
	respond(c, data, err)
}

func (api API) deleteTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.store.DeleteTask(c.Param("id")))
}

func (api API) taskAction(status domain.TaskStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := api.store.UpdateTaskStatus(c.Param("id"), status)
		respond(c, data, err)
	}
}

func (api API) scheduledTasks(c *gin.Context) {
	data, err := api.store.ScheduledTasks()
	respond(c, data, err)
}

func (api API) createScheduledTask(c *gin.Context) {
	var req domain.ScheduledTask
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Cycle == "" {
		req.Cycle = "每天"
	}
	if req.Assignee == "" {
		req.Assignee = "双方"
	}
	if req.Time == "" {
		req.Time = "20:00"
	}
	if req.Next == "" {
		req.Next = "今天 " + req.Time
	}
	data, err := api.store.AddScheduledTask(req)
	respond(c, data, err)
}

func (api API) deleteScheduledTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.store.DeleteScheduledTask(c.Param("id")))
}

func (api API) confirmScheduledTask(c *gin.Context) {
	data, err := api.store.ConfirmScheduledTask(c.Param("id"))
	respond(c, data, err)
}

func (api API) dishes(c *gin.Context) {
	data, err := api.store.Dishes()
	respond(c, data, err)
}

func (api API) createDish(c *gin.Context) {
	var req domain.Dish
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		badRequest(c, "name required")
		return
	}
	if req.Icon == "" {
		req.Icon = "🍽️"
	}
	if req.Meal == "" {
		req.Meal = "通用"
	}
	req.Enabled = true
	data, err := api.store.AddDish(req)
	respond(c, data, err)
}

func (api API) deleteDish(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.store.DeleteDish(c.Param("id")))
}

func (api API) updateDishEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if !bindJSON(c, &req) {
		return
	}
	data, err := api.store.UpdateDishEnabled(c.Param("id"), req.Enabled)
	respond(c, data, err)
}

func (api API) orders(c *gin.Context) {
	data, err := api.store.Orders()
	respond(c, data, err)
}

func (api API) createOrder(c *gin.Context) {
	var req domain.Order
	if !bindJSON(c, &req) {
		return
	}
	if req.Meal == "" {
		req.Meal = "午餐"
	}
	if req.Picker == "" {
		req.Picker = "双方选的"
	}
	if len(req.Dishes) == 0 {
		badRequest(c, "dishes required")
		return
	}
	data, err := api.store.AddOrder(req)
	respond(c, data, err)
}

func (api API) goals(c *gin.Context) {
	data, err := api.store.Goals()
	respond(c, data, err)
}

func (api API) createGoal(c *gin.Context) {
	var req domain.Goal
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Period == "" {
		req.Period = "月目标"
	}
	req.Status = "active"
	data, err := api.store.AddGoal(req)
	respond(c, data, err)
}

func (api API) updateGoalValue(c *gin.Context) {
	var req struct {
		CurrentValue int `json:"currentValue"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.CurrentValue < 0 {
		badRequest(c, "currentValue must be >= 0")
		return
	}
	data, err := api.store.UpdateGoalValue(c.Param("id"), req.CurrentValue)
	respond(c, data, err)
}

func (api API) updateGoalStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.Status != "active" && req.Status != "done" {
		badRequest(c, "invalid status")
		return
	}
	data, err := api.store.UpdateGoalStatus(c.Param("id"), req.Status)
	respond(c, data, err)
}

func (api API) deleteGoal(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.store.DeleteGoal(c.Param("id")))
}

func respond(c *gin.Context, data any, err error) {
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, data)
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func fail(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
		message = "resource not found"
	}
	if errors.Is(err, store.ErrInvalidPairCode) {
		status = http.StatusBadRequest
		message = "分享码输入错误"
	}
	if errors.Is(err, store.ErrPairCodeExpired) {
		status = http.StatusBadRequest
		message = "分享码已过期，请让对方重新生成"
	}
	if errors.Is(err, store.ErrAlreadyPaired) {
		status = http.StatusBadRequest
		message = "您已绑定情侣关系，如需更换请先解绑"
	}
	c.JSON(status, gin.H{"code": status, "message": message})
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": message})
}

func absoluteURL(c *gin.Context, path string) string {
	host := c.Request.Host
	if host == "" {
		return path
	}
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + host + path
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		badRequest(c, "invalid json")
		return false
	}
	return true
}

func defaultUser(userID string) string {
	if userID == "" {
		return "u1"
	}
	return userID
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
