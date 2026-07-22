package service

import "couple-mini/backend/internal/model"

type Repository interface {
	Login(openid, nickname string) (model.User, error)
	GeneratePairCode(userID string) (model.Couple, error)
	PairByCode(userID, code, loveDate string) (model.Couple, error)
	UpdateLoveDate(loveDate string) (model.Couple, error)
	UpdateUserProfile(user model.User) (model.User, error)
	Dashboard(userID string) (model.DashboardPayload, error)
	Moments() ([]model.Moment, error)
	AddMoment(moment model.Moment) (model.Moment, error)
	DeleteMoment(id string) error
	UpdateMomentLiked(id string, liked bool) (model.Moment, error)
	Tasks() ([]model.Task, error)
	AddTask(task model.Task) (model.Task, error)
	DeleteTask(id string) error
	UpdateTaskStatus(id string, status model.TaskStatus) (model.Task, error)
	ScheduledTasks() ([]model.ScheduledTask, error)
	AddScheduledTask(task model.ScheduledTask) (model.ScheduledTask, error)
	DeleteScheduledTask(id string) error
	ConfirmScheduledTask(id string) (model.ScheduledTask, error)
	Dishes() ([]model.Dish, error)
	AddDish(dish model.Dish) (model.Dish, error)
	DeleteDish(id string) error
	UpdateDishEnabled(id string, enabled bool) (model.Dish, error)
	Orders() ([]model.Order, error)
	AddOrder(order model.Order) (model.Order, error)
	Goals() ([]model.Goal, error)
	AddGoal(goal model.Goal) (model.Goal, error)
	UpdateGoalValue(id string, currentValue int) (model.Goal, error)
	UpdateGoalStatus(id, status string) (model.Goal, error)
	DeleteGoal(id string) error
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(req *LoginRequest) (model.User, string, error) {
	user, err := s.repo.Login(req.OpenID, req.Nickname)
	if err != nil {
		return model.User{}, "", err
	}
	return user, "demo-token-" + user.ID, nil
}

func (s *Service) GeneratePairCode(req *GeneratePairCodeRequest) (model.Couple, error) {
	return s.repo.GeneratePairCode(req.UserID)
}

func (s *Service) ConfirmPair(req *ConfirmPairRequest) (model.Couple, error) {
	return s.repo.PairByCode(req.UserID, req.Code, req.LoveDate)
}

func (s *Service) UpdateLoveDate(req *UpdateLoveDateRequest) (model.Couple, error) {
	return s.repo.UpdateLoveDate(req.LoveDate)
}

func (s *Service) UpdateUserProfile(req *UserProfileRequest) (model.User, error) {
	return s.repo.UpdateUserProfile(*req)
}

func (s *Service) Dashboard(userID string) (model.DashboardPayload, error) {
	return s.repo.Dashboard(userID)
}

func (s *Service) Moments() ([]model.Moment, error) {
	return s.repo.Moments()
}

func (s *Service) AddMoment(req *CreateMomentRequest) (model.Moment, error) {
	return s.repo.AddMoment(*req)
}

func (s *Service) DeleteMoment(id string) error {
	return s.repo.DeleteMoment(id)
}

func (s *Service) UpdateMomentLiked(id string, req *UpdateMomentLikedRequest) (model.Moment, error) {
	return s.repo.UpdateMomentLiked(id, req.Liked)
}

func (s *Service) Tasks() ([]model.Task, error) {
	return s.repo.Tasks()
}

func (s *Service) AddTask(req *CreateTaskRequest) (model.Task, error) {
	return s.repo.AddTask(*req)
}

func (s *Service) DeleteTask(id string) error {
	return s.repo.DeleteTask(id)
}

func (s *Service) UpdateTaskStatus(id string, status model.TaskStatus) (model.Task, error) {
	return s.repo.UpdateTaskStatus(id, status)
}

func (s *Service) ScheduledTasks() ([]model.ScheduledTask, error) {
	return s.repo.ScheduledTasks()
}

func (s *Service) AddScheduledTask(req *CreateScheduledTaskRequest) (model.ScheduledTask, error) {
	return s.repo.AddScheduledTask(*req)
}

func (s *Service) DeleteScheduledTask(id string) error {
	return s.repo.DeleteScheduledTask(id)
}

func (s *Service) ConfirmScheduledTask(id string) (model.ScheduledTask, error) {
	return s.repo.ConfirmScheduledTask(id)
}

func (s *Service) Dishes() ([]model.Dish, error) {
	return s.repo.Dishes()
}

func (s *Service) AddDish(req *CreateDishRequest) (model.Dish, error) {
	return s.repo.AddDish(*req)
}

func (s *Service) DeleteDish(id string) error {
	return s.repo.DeleteDish(id)
}

func (s *Service) UpdateDishEnabled(id string, req *UpdateDishEnabledRequest) (model.Dish, error) {
	return s.repo.UpdateDishEnabled(id, req.Enabled)
}

func (s *Service) Orders() ([]model.Order, error) {
	return s.repo.Orders()
}

func (s *Service) AddOrder(req *CreateOrderRequest) (model.Order, error) {
	return s.repo.AddOrder(*req)
}

func (s *Service) Goals() ([]model.Goal, error) {
	return s.repo.Goals()
}

func (s *Service) AddGoal(req *CreateGoalRequest) (model.Goal, error) {
	return s.repo.AddGoal(*req)
}

func (s *Service) UpdateGoalValue(id string, req *UpdateGoalValueRequest) (model.Goal, error) {
	return s.repo.UpdateGoalValue(id, req.CurrentValue)
}

func (s *Service) UpdateGoalStatus(id string, req *UpdateGoalStatusRequest) (model.Goal, error) {
	return s.repo.UpdateGoalStatus(id, req.Status)
}

func (s *Service) DeleteGoal(id string) error {
	return s.repo.DeleteGoal(id)
}
