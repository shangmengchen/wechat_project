package service

import (
	"strings"

	"couple-mini/backend/internal/model"
)

type Repository interface {
	Login(userID, openid, nickname, avatar string) (model.User, error)
	UserByID(userID string) (model.User, error)
	SyncState(userID string) (model.SyncState, error)
	CoupleForUser(userID string) (model.Couple, error)
	GeneratePairCode(userID string) (model.Couple, error)
	PairByCode(userID, code, loveDate string) (model.Couple, error)
	UpdateLoveDate(userID, loveDate string) (model.Couple, error)
	UpdateUserProfile(currentUserID string, user model.User) (model.User, error)
	Unpair(currentUserID string) (model.UnpairResult, error)
	Dashboard(userID string) (model.DashboardPayload, error)
	Moments(userID string) ([]model.Moment, error)
	AddMoment(userID string, moment model.Moment) (model.Moment, error)
	DeleteMoment(userID, id string) error
	UpdateMomentLiked(userID, id string, liked bool) (model.Moment, error)
	Tasks(userID string) ([]model.Task, error)
	AddTask(userID string, task model.Task) (model.Task, error)
	DeleteTask(userID, id string) error
	UpdateTaskStatus(userID, id string, status model.TaskStatus) (model.Task, error)
	ScheduledTasks(userID string) ([]model.ScheduledTask, error)
	AddScheduledTask(userID string, task model.ScheduledTask) (model.ScheduledTask, error)
	DeleteScheduledTask(userID, id string) error
	ConfirmScheduledTask(userID, id string) (model.ScheduledTask, error)
	Dishes(userID string) ([]model.Dish, error)
	AddDish(userID string, dish model.Dish) (model.Dish, error)
	DeleteDish(userID, id string) error
	UpdateDishEnabled(userID, id string, enabled bool) (model.Dish, error)
	Orders(userID string) ([]model.Order, error)
	AddOrder(userID string, order model.Order) (model.Order, error)
	Goals(userID string) ([]model.Goal, error)
	AddGoal(userID string, goal model.Goal) (model.Goal, error)
	UpdateGoalValue(userID, id string, currentValue int) (model.Goal, error)
	UpdateGoalStatus(userID, id, status string) (model.Goal, error)
	DeleteGoal(userID, id string) error
	AddNotice(notice model.Notice) (model.Notice, error)
	UnreadNotices(userID string) ([]model.Notice, error)
	MarkNoticesRead(userID string, categories []string) error
	AdminOverview() (model.AdminOverview, error)
	AdminRecentUsers(limit int) ([]model.AdminUserSummary, error)
	AdminRecentCouples(limit int) ([]model.AdminCoupleSummary, error)
	AdminUnpairCouple(coupleID string) (model.UnpairResult, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(req *LoginRequest) (model.User, error) {
	return s.repo.Login(req.UserID, req.OpenID, req.Nickname, req.Avatar)
}

func (s *Service) UserByID(userID string) (model.User, error) {
	return s.repo.UserByID(userID)
}

func (s *Service) SyncState(userID string) (model.SyncState, error) {
	return s.repo.SyncState(userID)
}

func (s *Service) CoupleForUser(userID string) (model.Couple, error) {
	return s.repo.CoupleForUser(userID)
}

func (s *Service) GeneratePairCode(userID string) (model.Couple, error) {
	return s.repo.GeneratePairCode(userID)
}

func (s *Service) ConfirmPair(userID string, req *ConfirmPairRequest) (model.Couple, error) {
	return s.repo.PairByCode(userID, req.Code, req.LoveDate)
}

func (s *Service) UpdateLoveDate(userID string, req *UpdateLoveDateRequest) (model.Couple, error) {
	return s.repo.UpdateLoveDate(userID, req.LoveDate)
}

func (s *Service) UpdateUserProfile(userID string, req *UserProfileRequest) (model.User, error) {
	return s.repo.UpdateUserProfile(userID, *req)
}

func (s *Service) Unpair(userID string) (model.UnpairResult, error) {
	return s.repo.Unpair(userID)
}

func (s *Service) Dashboard(userID string) (model.DashboardPayload, error) {
	return s.repo.Dashboard(userID)
}

func (s *Service) Moments(userID string) ([]model.Moment, error) {
	return s.repo.Moments(userID)
}

func (s *Service) AddMoment(userID string, req *CreateMomentRequest) (model.Moment, error) {
	return s.repo.AddMoment(userID, *req)
}

func (s *Service) DeleteMoment(userID, id string) error {
	return s.repo.DeleteMoment(userID, id)
}

func (s *Service) UpdateMomentLiked(userID, id string, req *UpdateMomentLikedRequest) (model.Moment, error) {
	return s.repo.UpdateMomentLiked(userID, id, req.Liked)
}

func (s *Service) Tasks(userID string) ([]model.Task, error) {
	return s.repo.Tasks(userID)
}

func (s *Service) AddTask(userID string, req *CreateTaskRequest) (model.Task, error) {
	return s.repo.AddTask(userID, *req)
}

func (s *Service) DeleteTask(userID, id string) error {
	return s.repo.DeleteTask(userID, id)
}

func (s *Service) UpdateTaskStatus(userID, id string, status model.TaskStatus) (model.Task, error) {
	return s.repo.UpdateTaskStatus(userID, id, status)
}

func (s *Service) ScheduledTasks(userID string) ([]model.ScheduledTask, error) {
	return s.repo.ScheduledTasks(userID)
}

func (s *Service) AddScheduledTask(userID string, req *CreateScheduledTaskRequest) (model.ScheduledTask, error) {
	return s.repo.AddScheduledTask(userID, *req)
}

func (s *Service) DeleteScheduledTask(userID, id string) error {
	return s.repo.DeleteScheduledTask(userID, id)
}

func (s *Service) ConfirmScheduledTask(userID, id string) (model.ScheduledTask, error) {
	return s.repo.ConfirmScheduledTask(userID, id)
}

func (s *Service) Dishes(userID string) ([]model.Dish, error) {
	return s.repo.Dishes(userID)
}

func (s *Service) AddDish(userID string, req *CreateDishRequest) (model.Dish, error) {
	return s.repo.AddDish(userID, *req)
}

func (s *Service) DeleteDish(userID, id string) error {
	return s.repo.DeleteDish(userID, id)
}

func (s *Service) UpdateDishEnabled(userID, id string, req *UpdateDishEnabledRequest) (model.Dish, error) {
	return s.repo.UpdateDishEnabled(userID, id, req.Enabled)
}

func (s *Service) Orders(userID string) ([]model.Order, error) {
	return s.repo.Orders(userID)
}

func (s *Service) AddOrder(userID string, req *CreateOrderRequest) (model.Order, error) {
	return s.repo.AddOrder(userID, *req)
}

func (s *Service) Goals(userID string) ([]model.Goal, error) {
	return s.repo.Goals(userID)
}

func (s *Service) AddGoal(userID string, req *CreateGoalRequest) (model.Goal, error) {
	return s.repo.AddGoal(userID, *req)
}

func (s *Service) UpdateGoalValue(userID, id string, req *UpdateGoalValueRequest) (model.Goal, error) {
	return s.repo.UpdateGoalValue(userID, id, req.CurrentValue)
}

func (s *Service) UpdateGoalStatus(userID, id string, req *UpdateGoalStatusRequest) (model.Goal, error) {
	return s.repo.UpdateGoalStatus(userID, id, req.Status)
}

func (s *Service) DeleteGoal(userID, id string) error {
	return s.repo.DeleteGoal(userID, id)
}

func (s *Service) CreatePartnerNotice(userID string, req *CreateNoticeRequest) (model.Notice, error) {
	userID = strings.TrimSpace(userID)
	couple, err := s.repo.CoupleForUser(userID)
	if err != nil {
		return model.Notice{}, err
	}
	recipientID := couple.UserAID
	if recipientID == userID {
		recipientID = couple.UserBID
	}
	if strings.TrimSpace(recipientID) == "" || recipientID == userID {
		return model.Notice{}, nil
	}
	return s.repo.AddNotice(model.Notice{
		CoupleID:    couple.ID,
		RecipientID: recipientID,
		InitiatorID: userID,
		Category:    strings.TrimSpace(req.Category),
		Title:       strings.TrimSpace(req.Title),
		Content:     strings.TrimSpace(req.Content),
		Target:      strings.TrimSpace(req.Target),
	})
}

func (s *Service) UnreadNotices(userID string) ([]model.Notice, error) {
	return s.repo.UnreadNotices(userID)
}

func (s *Service) MarkNoticesRead(userID string, req *MarkNoticesReadRequest) error {
	categories := make([]string, 0, len(req.Categories))
	seen := map[string]struct{}{}
	for _, category := range req.Categories {
		category = strings.TrimSpace(category)
		if category == "" {
			continue
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	return s.repo.MarkNoticesRead(userID, categories)
}
