package api

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
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

const maxImageUploadBytes = 5 << 20

type API struct {
	service      *service.Service
	wechatClient wechatSessionClient
	push         *push.Hub
}

type wechatSessionClient interface {
	Enabled() bool
	Code2Session(context.Context, string) (wechatcli.Session, error)
	SendSubscribeMessage(context.Context, wechatcli.SubscribeMessage) error
}

func New(service *service.Service, wechatClient wechatSessionClient, pushHub *push.Hub) *API {
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

	session, err := api.exchangeOpenID(c.Request.Context(), req.Code)
	if err != nil {
		fail(c, err)
		return
	}
	req.UserID = ""
	req.OpenID = strings.TrimSpace(session.OpenID)
	if req.OpenID == "" {
		fail(c, fmt.Errorf("wechat auth missing openid"))
		return
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

func (api *API) Unpair(c *gin.Context) {
	result, err := api.service.Unpair(httpmw.GetCurrentUserID(c))
	if err != nil {
		respond(c, result, err)
		return
	}
	if api.push != nil {
		api.push.NotifyPairUnbound(result)
	}
	ok(c, result)
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
	if file.Size <= 0 || file.Size > maxImageUploadBytes {
		badRequest(c, "image size must be between 1 byte and 5MB")
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
	if !validImageContent(file, ext) {
		badRequest(c, "invalid image content")
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

func validImageContent(fileHeader *multipart.FileHeader, ext string) bool {
	file, err := fileHeader.Open()
	if err != nil {
		return false
	}
	defer file.Close()

	var header [512]byte
	n, err := file.Read(header[:])
	if err != nil && err != io.EOF {
		return false
	}
	contentType := http.DetectContentType(header[:n])
	switch ext {
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg"
	case ".png":
		return contentType == "image/png"
	case ".gif":
		return contentType == "image/gif"
	case ".webp":
		return isWebP(header[:n])
	default:
		return false
	}
}

func isWebP(header []byte) bool {
	return len(header) >= 12 &&
		string(header[0:4]) == "RIFF" &&
		string(header[8:12]) == "WEBP"
}

func (api *API) SubscribeTemplates(c *gin.Context) {
	cfg := configs.GetGlobalConfig().WeChatConfig
	ok(c, gin.H{
		"notice":   strings.TrimSpace(cfg.SubscribeNoticeTemplateID),
		"schedule": strings.TrimSpace(cfg.SubscribeScheduleTemplateID),
	})
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddMoment(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "memory", "新的动态", previewText(data.Content, "对方发布了一条新动态"), "/pages/memory/memory")
	ok(c, data)
}

func (api *API) DeleteMoment(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteMoment(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) UpdateMomentLiked(c *gin.Context) {
	req := &service.UpdateMomentLikedRequest{}
	if !bindJSON(c, req) {
		return
	}
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.UpdateMomentLiked(userID, c.Param("id"), req)
	if err != nil {
		respond(c, data, err)
		return
	}
	if req.Liked {
		api.notifyPartner(userID, "memory", "动态有新互动", "对方喜欢了你的动态", "/pages/memory/memory")
	}
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddTask(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "todos", "新的待办任务", previewText(data.Title, "对方新建了一个待办任务"), "/pages/todos/todos")
	ok(c, data)
}

func (api *API) DeleteTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteTask(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) TaskAction(status model.TaskStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := httpmw.GetCurrentUserID(c)
		data, err := api.service.UpdateTaskStatus(userID, c.Param("id"), status)
		if err != nil {
			respond(c, data, err)
			return
		}
		api.notifyPartner(userID, "todos", taskNoticeTitle(status), taskNoticeContent(status, data.Title), "/pages/todos/todos")
		ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddScheduledTask(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "schedule", "新的定时任务", scheduledNoticeContent(data), "/pages/schedule/schedule")
	ok(c, data)
}

func (api *API) DeleteScheduledTask(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteScheduledTask(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) ConfirmScheduledTask(c *gin.Context) {
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.ConfirmScheduledTask(userID, c.Param("id"))
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "schedule", "定时任务已完成", previewText(data.Title, "对方完成了一个定时任务"), "/pages/schedule/schedule")
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddDish(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "order", "新增菜品", previewText(data.Name, "对方新增了一道菜"), "/pages/order/order")
	ok(c, data)
}

func (api *API) DeleteDish(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteDish(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) UpdateDishEnabled(c *gin.Context) {
	req := &service.UpdateDishEnabledRequest{}
	if !bindJSON(c, req) {
		return
	}
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.UpdateDishEnabled(userID, c.Param("id"), req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "order", "菜品状态更新", previewText(data.Name, "对方更新了菜品状态"), "/pages/order/order")
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddOrder(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "order", "今日点餐更新", orderNoticeContent(data), "/pages/order/order")
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.AddGoal(userID, req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "goals", "新的两人小目标", previewText(data.Title, "对方新建了一个小目标"), "/pages/goal/goals")
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.UpdateGoalValue(userID, c.Param("id"), req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "goals", "目标进度更新", goalNoticeContent(data), "/pages/goal/goals")
	ok(c, data)
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
	userID := httpmw.GetCurrentUserID(c)
	data, err := api.service.UpdateGoalStatus(userID, c.Param("id"), req)
	if err != nil {
		respond(c, data, err)
		return
	}
	api.notifyPartner(userID, "goals", "目标状态更新", goalNoticeContent(data), "/pages/goal/goals")
	ok(c, data)
}

func (api *API) DeleteGoal(c *gin.Context) {
	respond(c, gin.H{"deleted": true}, api.service.DeleteGoal(httpmw.GetCurrentUserID(c), c.Param("id")))
}

func (api *API) UnreadNotices(c *gin.Context) {
	items, err := api.service.UnreadNotices(httpmw.GetCurrentUserID(c))
	if err != nil {
		respond(c, nil, err)
		return
	}
	ok(c, gin.H{
		"items":  items,
		"counts": noticeCounts(items),
	})
}

func (api *API) MarkNoticesRead(c *gin.Context) {
	req := &service.MarkNoticesReadRequest{}
	if !bindJSON(c, req) {
		return
	}
	if err := api.service.MarkNoticesRead(httpmw.GetCurrentUserID(c), req); err != nil {
		respond(c, nil, err)
		return
	}
	ok(c, gin.H{"read": true})
}

func (api *API) exchangeOpenID(ctx context.Context, code string) (wechatcli.Session, error) {
	if api.wechatClient != nil && api.wechatClient.Enabled() {
		return api.wechatClient.Code2Session(ctx, code)
	}
	if strings.EqualFold(strings.TrimSpace(configs.GetGlobalConfig().AppConfig.RunMode), "release") {
		return wechatcli.Session{}, fmt.Errorf("wechat auth not configured")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return wechatcli.Session{}, fmt.Errorf("login code required")
	}
	return wechatcli.Session{OpenID: "mock-openid-" + code}, nil
}

func (api *API) notifyPartner(userID, category, title, content, target string) {
	if api == nil || api.service == nil {
		return
	}
	notice, err := api.service.CreatePartnerNotice(userID, &service.CreateNoticeRequest{
		Category: category,
		Title:    title,
		Content:  content,
		Target:   target,
	})
	if err != nil || strings.TrimSpace(notice.ID) == "" {
		return
	}
	if api.push != nil {
		api.push.NotifyNotice(notice)
	}
	go api.sendWeChatSubscribeNotice(notice)
}

func (api *API) sendWeChatSubscribeNotice(notice model.Notice) {
	if api == nil || api.wechatClient == nil || !api.wechatClient.Enabled() {
		return
	}
	cfg := configs.GetGlobalConfig().WeChatConfig
	templateID := subscribeTemplateID(cfg, notice.Category)
	if strings.TrimSpace(templateID) == "" {
		return
	}
	recipient, err := api.service.UserByID(notice.RecipientID)
	if err != nil || strings.TrimSpace(recipient.OpenID) == "" {
		return
	}
	titleKey := firstNonEmpty(strings.TrimSpace(cfg.SubscribeTitleKey), "thing1")
	contentKey := firstNonEmpty(strings.TrimSpace(cfg.SubscribeContentKey), "thing2")
	page := firstNonEmpty(strings.TrimSpace(notice.Target), strings.TrimSpace(cfg.SubscribePage), "pages/home/home")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = api.wechatClient.SendSubscribeMessage(ctx, wechatcli.SubscribeMessage{
		ToUser:     recipient.OpenID,
		TemplateID: templateID,
		Page:       page,
		Data: map[string]string{
			titleKey:   truncateRunes(notice.Title, 20),
			contentKey: truncateRunes(notice.Content, 20),
		},
	})
}

func subscribeTemplateID(cfg configs.WeChatConfig, category string) string {
	if normalizeNoticeCategory(category) == "schedule" && strings.TrimSpace(cfg.SubscribeScheduleTemplateID) != "" {
		return cfg.SubscribeScheduleTemplateID
	}
	return cfg.SubscribeNoticeTemplateID
}

func normalizeNoticeCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "schedule", "scheduledtask":
		return "schedule"
	default:
		return strings.ToLower(strings.TrimSpace(category))
	}
}

func noticeCounts(items []model.Notice) map[string]int {
	counts := map[string]int{}
	for _, item := range items {
		category := strings.TrimSpace(item.Category)
		if category == "" {
			continue
		}
		counts[category]++
	}
	return counts
}

func previewText(value, fallback string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return fallback
	}
	runes := []rune(text)
	if len(runes) > 32 {
		return string(runes[:32]) + "..."
	}
	return text
}

func truncateRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func scheduledNoticeContent(task model.ScheduledTask) string {
	title := previewText(task.Title, "对方新建了一个定时任务")
	if strings.TrimSpace(task.Time) == "" {
		return title
	}
	return fmt.Sprintf("%s，提醒时间 %s", title, task.Time)
}

func orderNoticeContent(order model.Order) string {
	if len(order.Dishes) == 0 {
		return "对方保存了今日点餐"
	}
	return previewText(strings.Join(order.Dishes, "、"), "对方保存了今日点餐")
}

func goalNoticeContent(goal model.Goal) string {
	return fmt.Sprintf("%s：%d%%", previewText(goal.Title, "两人小目标有更新"), goal.Progress)
}

func taskNoticeTitle(status model.TaskStatus) string {
	switch status {
	case model.TaskReview:
		return "待办等待确认"
	case model.TaskDone:
		return "待办已完成"
	default:
		return "待办被驳回"
	}
}

func taskNoticeContent(status model.TaskStatus, title string) string {
	text := previewText(title, "一个待办任务")
	switch status {
	case model.TaskReview:
		return text + " 已提交确认"
	case model.TaskDone:
		return text + " 已确认完成"
	default:
		return text + " 被驳回，需要重新处理"
	}
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
