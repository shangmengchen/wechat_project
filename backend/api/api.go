package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type API struct {
	service *service.Service
}

func New(service *service.Service) *API {
	return &API{service: service}
}

func (api *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (api *API) Login(c *gin.Context) {
	req := &service.LoginRequest{}
	if !bindJSON(c, req) {
		return
	}
	if req.OpenID == "" {
		req.OpenID = "mock-openid-" + req.Code
	}
	if req.Nickname == "" {
		req.Nickname = "WeChat User"
	}
	user, token, err := api.service.Login(req)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"token": token, "user": user})
}

func (api *API) GeneratePairCode(c *gin.Context) {
	req := &service.GeneratePairCodeRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		badRequest(c, "userId required")
		return
	}
	couple, err := api.service.GeneratePairCode(req)
	respond(c, couple, err)
}

func (api *API) ConfirmPair(c *gin.Context) {
	req := &service.ConfirmPairRequest{}
	if !bindJSON(c, req) {
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if !regexp.MustCompile(`^\d{6}$`).MatchString(req.Code) {
		badRequest(c, "invalid pair code")
		return
	}
	couple, err := api.service.ConfirmPair(req)
	respond(c, couple, err)
}

func (api *API) UpdateLoveDate(c *gin.Context) {
	req := &service.UpdateLoveDateRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.LoveDate) == "" {
		badRequest(c, "loveDate required")
		return
	}
	couple, err := api.service.UpdateLoveDate(req)
	respond(c, couple, err)
}

func (api *API) UpdateUserProfile(c *gin.Context) {
	req := &service.UserProfileRequest{}
	if !bindJSON(c, req) {
		return
	}
	req.ID = c.Param("id")
	if strings.TrimSpace(req.Nickname) == "" {
		badRequest(c, "nickname required")
		return
	}
	user, err := api.service.UpdateUserProfile(req)
	respond(c, user, err)
}

func (api *API) Dashboard(c *gin.Context) {
	data, err := api.service.Dashboard(c.Query("userId"))
	respond(c, data, err)
}

func (api *API) UploadImage(c *gin.Context) {
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
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

func (api *API) Moments(c *gin.Context) {
	data, err := api.service.Moments()
	respond(c, data, err)
}

func (api *API) CreateMoment(c *gin.Context) {
	req := &service.CreateMomentRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		badRequest(c, "content required")
		return
	}
	if req.Author == "" {
		req.Author = "Xiao Yu"
	}
	if req.Avatar == "" {
		req.Avatar = "X"
	}
	req.TimeLabel = "just now"
	data, err := api.service.AddMoment(req)
	respond(c, data, err)
}

func (api *API) DeleteMoment(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteMoment(c.Param("id")))
}

func (api *API) UpdateMomentLiked(c *gin.Context) {
	req := &service.UpdateMomentLikedRequest{}
	if !bindJSON(c, req) {
		return
	}
	data, err := api.service.UpdateMomentLiked(c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) Tasks(c *gin.Context) {
	data, err := api.service.Tasks()
	respond(c, data, err)
}

func (api *API) CreateTask(c *gin.Context) {
	req := &service.CreateTaskRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Owner == "" {
		req.Owner = "both"
	}
	if req.Type == "" {
		req.Type = "one-time"
	}
	if strings.TrimSpace(req.Tag) == "" {
		req.Tag = "life"
	}
	data, err := api.service.AddTask(req)
	respond(c, data, err)
}

func (api *API) DeleteTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteTask(c.Param("id")))
}

func (api *API) TaskAction(status model.TaskStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := api.service.UpdateTaskStatus(c.Param("id"), status)
		respond(c, data, err)
	}
}

func (api *API) ScheduledTasks(c *gin.Context) {
	data, err := api.service.ScheduledTasks()
	respond(c, data, err)
}

func (api *API) CreateScheduledTask(c *gin.Context) {
	req := &service.CreateScheduledTaskRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Cycle == "" {
		req.Cycle = "every day"
	}
	if req.Assignee == "" {
		req.Assignee = "both"
	}
	if req.Time == "" {
		req.Time = "20:00"
	}
	if req.Next == "" {
		req.Next = "today " + req.Time
	}
	data, err := api.service.AddScheduledTask(req)
	respond(c, data, err)
}

func (api *API) DeleteScheduledTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteScheduledTask(c.Param("id")))
}

func (api *API) ConfirmScheduledTask(c *gin.Context) {
	data, err := api.service.ConfirmScheduledTask(c.Param("id"))
	respond(c, data, err)
}

func (api *API) Dishes(c *gin.Context) {
	data, err := api.service.Dishes()
	respond(c, data, err)
}

func (api *API) CreateDish(c *gin.Context) {
	req := &service.CreateDishRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		badRequest(c, "name required")
		return
	}
	if req.Icon == "" {
		req.Icon = "dish"
	}
	if req.Meal == "" {
		req.Meal = "any"
	}
	req.Enabled = true
	data, err := api.service.AddDish(req)
	respond(c, data, err)
}

func (api *API) DeleteDish(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteDish(c.Param("id")))
}

func (api *API) UpdateDishEnabled(c *gin.Context) {
	req := &service.UpdateDishEnabledRequest{}
	if !bindJSON(c, req) {
		return
	}
	data, err := api.service.UpdateDishEnabled(c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) Orders(c *gin.Context) {
	data, err := api.service.Orders()
	respond(c, data, err)
}

func (api *API) CreateOrder(c *gin.Context) {
	req := &service.CreateOrderRequest{}
	if !bindJSON(c, req) {
		return
	}
	if req.Meal == "" {
		req.Meal = "lunch"
	}
	if req.Picker == "" {
		req.Picker = "both"
	}
	if len(req.Dishes) == 0 {
		badRequest(c, "dishes required")
		return
	}
	data, err := api.service.AddOrder(req)
	respond(c, data, err)
}

func (api *API) Goals(c *gin.Context) {
	data, err := api.service.Goals()
	respond(c, data, err)
}

func (api *API) CreateGoal(c *gin.Context) {
	req := &service.CreateGoalRequest{}
	if !bindJSON(c, req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "title required")
		return
	}
	if req.Period == "" {
		req.Period = "month"
	}
	req.Status = "active"
	data, err := api.service.AddGoal(req)
	respond(c, data, err)
}

func (api *API) UpdateGoalValue(c *gin.Context) {
	req := &service.UpdateGoalValueRequest{}
	if !bindJSON(c, req) {
		return
	}
	if req.CurrentValue < 0 {
		badRequest(c, "currentValue must be >= 0")
		return
	}
	data, err := api.service.UpdateGoalValue(c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) UpdateGoalStatus(c *gin.Context) {
	req := &service.UpdateGoalStatusRequest{}
	if !bindJSON(c, req) {
		return
	}
	if req.Status != "active" && req.Status != "done" {
		badRequest(c, "invalid status")
		return
	}
	data, err := api.service.UpdateGoalStatus(c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) DeleteGoal(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteGoal(c.Param("id")))
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
