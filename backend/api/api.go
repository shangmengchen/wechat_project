package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/pkg/auth"
	"couple-mini/backend/internal/pkg/httpmw"
	"couple-mini/backend/internal/pkg/push"
	wechatcli "couple-mini/backend/internal/pkg/wechat"
	"couple-mini/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type API struct {
	service      *service.Service
	wechatClient *wechatcli.Client
	push         *push.Hub
}

func New(service *service.Service, wechatClient *wechatcli.Client, pushHub *push.Hub) *API {
	return &API{
		service:      service,
		wechatClient: wechatClient,
		push:         pushHub,
	}
}

func (api *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (api *API) Login(c *gin.Context) {
	req := &service.LoginRequest{}
	if !bindJSON(c, req) {
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Avatar = strings.TrimSpace(req.Avatar)

	if req.Nickname == "" {
		req.Nickname = "WeChat User"
	}

	if strings.TrimSpace(req.OpenID) == "" {
		session, err := api.exchangeOpenID(c.Request.Context(), req.Code)
		if err != nil {
			fail(c, err)
			return
		}
		req.OpenID = session.OpenID
	}

	user, err := api.service.Login(req)
	if err != nil {
		fail(c, err)
		return
	}

	cfg := configs.GetGlobalConfig()
	token, err := auth.SignToken(
		cfg.AuthConfig.TokenSecret,
		time.Duration(cfg.AuthConfig.TokenTTLHours)*time.Hour,
		user.ID,
		user.OpenID,
	)
	if err != nil {
		fail(c, err)
		return
	}

	ok(c, gin.H{"token": token, "user": user})
}

func (api *API) SyncState(c *gin.Context) {
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.SyncState(userID)
	respond(c, data, err)
}

func (api *API) GeneratePairCode(c *gin.Context) {
	userID := httpmw.GetCurrentUserID(c)
	couple, err := api.service.GeneratePairCode(userID)
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
	couple, err := api.service.ConfirmPair(httpmw.GetCurrentUserID(c), req)
	if err != nil {
		respond(c, couple, err)
		return
	}
	if api.push != nil {
		api.push.NotifyPairConfirmed(couple)
	}
	ok(c, couple)
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
	couple, err := api.service.UpdateLoveDate(httpmw.GetCurrentUserID(c), req)
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
	user, err := api.service.UpdateUserProfile(httpmw.GetCurrentUserID(c), req)
	respond(c, user, err)
}

func (api *API) Dashboard(c *gin.Context) {
	data, err := api.service.Dashboard(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.Moments(httpmw.GetCurrentUserID(c))
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
	req.TimeLabel = "just now"
	data, err := api.service.AddMoment(httpmw.GetCurrentUserID(c), req)
	respond(c, data, err)
}

func (api *API) DeleteMoment(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteMoment(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) UpdateMomentLiked(c *gin.Context) {
	req := &service.UpdateMomentLikedRequest{}
	if !bindJSON(c, req) {
		return
	}
	data, err := api.service.UpdateMomentLiked(httpmw.GetCurrentUserID(c), c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) Tasks(c *gin.Context) {
	data, err := api.service.Tasks(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.AddTask(httpmw.GetCurrentUserID(c), req)
	respond(c, data, err)
}

func (api *API) DeleteTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteTask(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) TaskAction(status model.TaskStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := api.service.UpdateTaskStatus(httpmw.GetCurrentUserID(c), c.Param("id"), status)
		respond(c, data, err)
	}
}

func (api *API) ScheduledTasks(c *gin.Context) {
	data, err := api.service.ScheduledTasks(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.AddScheduledTask(httpmw.GetCurrentUserID(c), req)
	respond(c, data, err)
}

func (api *API) DeleteScheduledTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteScheduledTask(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) ConfirmScheduledTask(c *gin.Context) {
	data, err := api.service.ConfirmScheduledTask(httpmw.GetCurrentUserID(c), c.Param("id"))
	respond(c, data, err)
}

func (api *API) Dishes(c *gin.Context) {
	data, err := api.service.Dishes(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.AddDish(httpmw.GetCurrentUserID(c), req)
	respond(c, data, err)
}

func (api *API) DeleteDish(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteDish(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) UpdateDishEnabled(c *gin.Context) {
	req := &service.UpdateDishEnabledRequest{}
	if !bindJSON(c, req) {
		return
	}
	data, err := api.service.UpdateDishEnabled(httpmw.GetCurrentUserID(c), c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) Orders(c *gin.Context) {
	data, err := api.service.Orders(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.AddOrder(httpmw.GetCurrentUserID(c), req)
	respond(c, data, err)
}

func (api *API) Goals(c *gin.Context) {
	data, err := api.service.Goals(httpmw.GetCurrentUserID(c))
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
	data, err := api.service.AddGoal(httpmw.GetCurrentUserID(c), req)
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
	data, err := api.service.UpdateGoalValue(httpmw.GetCurrentUserID(c), c.Param("id"), req)
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
	data, err := api.service.UpdateGoalStatus(httpmw.GetCurrentUserID(c), c.Param("id"), req)
	respond(c, data, err)
}

func (api *API) DeleteGoal(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteGoal(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) exchangeOpenID(ctx context.Context, code string) (wechatcli.Session, error) {
	if api.wechatClient != nil && api.wechatClient.Enabled() {
		return api.wechatClient.Code2Session(ctx, code)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return wechatcli.Session{}, fmt.Errorf("login code required")
	}
	return wechatcli.Session{OpenID: "mock-openid-" + code}, nil
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
