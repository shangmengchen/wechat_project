package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/pkg/wechat"
	"couple-mini/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestValidImageContentRejectsExtensionMismatch(t *testing.T) {
	fileHeader := multipartHeader(t, "image.png", []byte("not an image"))

	if validImageContent(fileHeader, ".png") {
		t.Fatal("expected non-image content to be rejected")
	}
}

func TestValidImageContentAcceptsPNG(t *testing.T) {
	fileHeader := multipartHeader(t, "image.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
	})

	if !validImageContent(fileHeader, ".png") {
		t.Fatal("expected PNG content to be accepted")
	}
}

func TestLoginIgnoresClientOpenIDAndUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configs.InitGlobalConfig()

	repo := &loginCaptureRepo{}
	api := New(service.New(repo), fakeWeChatClient{openid: "server-openid"}, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"userId":"victim-user",
		"openid":"attacker-openid",
		"code":"valid-code",
		"nickname":"Tester"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	api.Login(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastUserID != "" {
		t.Fatalf("expected client userId to be cleared, got %q", repo.lastUserID)
	}
	if repo.lastOpenID != "server-openid" {
		t.Fatalf("expected server openid, got %q", repo.lastOpenID)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			User model.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || body.Data.User.OpenID != "server-openid" {
		t.Fatalf("unexpected response body: %+v", body)
	}
}

func TestLoginRequiresCode2SessionEvenWithClientOpenID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configs.InitGlobalConfig()

	repo := &loginCaptureRepo{}
	api := New(service.New(repo), fakeWeChatClient{err: context.Canceled}, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"userId":"victim-user",
		"openid":"attacker-openid",
		"nickname":"Tester"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	api.Login(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected code2session failure, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.loginCalls != 0 {
		t.Fatalf("expected repository login not to be called, got %d", repo.loginCalls)
	}
}

func TestLoginRejectsUnconfiguredWeChatAuthInRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configs.InitGlobalConfig()

	repo := &loginCaptureRepo{}
	api := New(service.New(repo), nil, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"code":"dev-code"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	api.Login(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected unconfigured wechat auth to fail in release, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.loginCalls != 0 {
		t.Fatalf("expected repository login not to be called, got %d", repo.loginCalls)
	}
}

func multipartHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(1024)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	t.Cleanup(func() {
		_ = form.RemoveAll()
	})
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected one file, got %d", len(files))
	}
	return files[0]
}

type fakeWeChatClient struct {
	openid string
	err    error
}

func (f fakeWeChatClient) Enabled() bool { return true }

func (f fakeWeChatClient) Code2Session(context.Context, string) (wechat.Session, error) {
	if f.err != nil {
		return wechat.Session{}, f.err
	}
	return wechat.Session{OpenID: f.openid}, nil
}

func (f fakeWeChatClient) SendSubscribeMessage(context.Context, wechat.SubscribeMessage) error {
	return nil
}

type loginCaptureRepo struct {
	loginCalls int
	lastUserID string
	lastOpenID string
}

func (r *loginCaptureRepo) Login(userID, openid, nickname, avatar string) (model.User, error) {
	r.loginCalls++
	r.lastUserID = userID
	r.lastOpenID = openid
	return model.User{ID: "server-user", OpenID: openid, Nickname: nickname, Avatar: avatar}, nil
}

func (r *loginCaptureRepo) UserByID(string) (model.User, error)           { return model.User{}, nil }
func (r *loginCaptureRepo) SyncState(string) (model.SyncState, error)     { return model.SyncState{}, nil }
func (r *loginCaptureRepo) CoupleForUser(string) (model.Couple, error)    { return model.Couple{}, nil }
func (r *loginCaptureRepo) GeneratePairCode(string) (model.Couple, error) { return model.Couple{}, nil }
func (r *loginCaptureRepo) PairByCode(string, string, string) (model.Couple, error) {
	return model.Couple{}, nil
}
func (r *loginCaptureRepo) UpdateLoveDate(string, string) (model.Couple, error) {
	return model.Couple{}, nil
}
func (r *loginCaptureRepo) UpdateUserProfile(string, model.User) (model.User, error) {
	return model.User{}, nil
}
func (r *loginCaptureRepo) Unpair(string) (model.UnpairResult, error) {
	return model.UnpairResult{}, nil
}
func (r *loginCaptureRepo) Dashboard(string) (model.DashboardPayload, error) {
	return model.DashboardPayload{}, nil
}
func (r *loginCaptureRepo) Moments(string) ([]model.Moment, error) { return nil, nil }
func (r *loginCaptureRepo) AddMoment(string, model.Moment) (model.Moment, error) {
	return model.Moment{}, nil
}
func (r *loginCaptureRepo) DeleteMoment(string, string) error { return nil }
func (r *loginCaptureRepo) UpdateMomentLiked(string, string, bool) (model.Moment, error) {
	return model.Moment{}, nil
}
func (r *loginCaptureRepo) Tasks(string) ([]model.Task, error)             { return nil, nil }
func (r *loginCaptureRepo) AddTask(string, model.Task) (model.Task, error) { return model.Task{}, nil }
func (r *loginCaptureRepo) DeleteTask(string, string) error                { return nil }
func (r *loginCaptureRepo) UpdateTaskStatus(string, string, model.TaskStatus) (model.Task, error) {
	return model.Task{}, nil
}
func (r *loginCaptureRepo) ScheduledTasks(string) ([]model.ScheduledTask, error) { return nil, nil }
func (r *loginCaptureRepo) AddScheduledTask(string, model.ScheduledTask) (model.ScheduledTask, error) {
	return model.ScheduledTask{}, nil
}
func (r *loginCaptureRepo) DeleteScheduledTask(string, string) error { return nil }
func (r *loginCaptureRepo) ConfirmScheduledTask(string, string) (model.ScheduledTask, error) {
	return model.ScheduledTask{}, nil
}
func (r *loginCaptureRepo) Dishes(string) ([]model.Dish, error)            { return nil, nil }
func (r *loginCaptureRepo) AddDish(string, model.Dish) (model.Dish, error) { return model.Dish{}, nil }
func (r *loginCaptureRepo) DeleteDish(string, string) error                { return nil }
func (r *loginCaptureRepo) UpdateDishEnabled(string, string, bool) (model.Dish, error) {
	return model.Dish{}, nil
}
func (r *loginCaptureRepo) Orders(string) ([]model.Order, error) { return nil, nil }
func (r *loginCaptureRepo) AddOrder(string, model.Order) (model.Order, error) {
	return model.Order{}, nil
}
func (r *loginCaptureRepo) Goals(string) ([]model.Goal, error)             { return nil, nil }
func (r *loginCaptureRepo) AddGoal(string, model.Goal) (model.Goal, error) { return model.Goal{}, nil }
func (r *loginCaptureRepo) UpdateGoalValue(string, string, int) (model.Goal, error) {
	return model.Goal{}, nil
}
func (r *loginCaptureRepo) UpdateGoalStatus(string, string, string) (model.Goal, error) {
	return model.Goal{}, nil
}
func (r *loginCaptureRepo) DeleteGoal(string, string) error              { return nil }
func (r *loginCaptureRepo) AddNotice(model.Notice) (model.Notice, error) { return model.Notice{}, nil }
func (r *loginCaptureRepo) UnreadNotices(string) ([]model.Notice, error) { return nil, nil }
func (r *loginCaptureRepo) MarkNoticesRead(string, []string) error       { return nil }
func (r *loginCaptureRepo) AdminOverview() (model.AdminOverview, error) {
	return model.AdminOverview{}, nil
}
func (r *loginCaptureRepo) AdminRecentUsers(int) ([]model.AdminUserSummary, error) { return nil, nil }
func (r *loginCaptureRepo) AdminRecentCouples(int) ([]model.AdminCoupleSummary, error) {
	return nil, nil
}
func (r *loginCaptureRepo) AdminUnpairCouple(string) (model.UnpairResult, error) {
	return model.UnpairResult{}, nil
}
