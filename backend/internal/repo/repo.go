package repo

import (
	"context"

	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/store"

	"gorm.io/gorm"
)

var (
	ErrNotFound        = store.ErrNotFound
	ErrInvalidPairCode = store.ErrInvalidPairCode
	ErrPairCodeExpired = store.ErrPairCodeExpired
	ErrAlreadyPaired   = store.ErrAlreadyPaired
	ErrUnauthorized    = store.ErrUnauthorized
)

type Repo struct {
	store *store.MySQLStore
}

func New(db *gorm.DB) *Repo {
	return &Repo{store: store.NewMySQLStore(db)}
}

func (r *Repo) EnsureSchema(ctx context.Context, autoMigrate, autoSeed bool) error {
	return r.store.EnsureSchema(ctx, autoMigrate, autoSeed)
}

func (r *Repo) Login(userID, openid, nickname, avatar string) (model.User, error) {
	return r.store.Login(userID, openid, nickname, avatar)
}

func (r *Repo) SyncState(userID string) (model.SyncState, error) {
	return r.store.SyncState(userID)
}

func (r *Repo) CoupleForUser(userID string) (model.Couple, error) {
	return r.store.CoupleForUser(userID)
}

func (r *Repo) GeneratePairCode(userID string) (model.Couple, error) {
	return r.store.GeneratePairCode(userID)
}

func (r *Repo) PairByCode(userID, code, loveDate string) (model.Couple, error) {
	return r.store.PairByCode(userID, code, loveDate)
}

func (r *Repo) UpdateLoveDate(userID, loveDate string) (model.Couple, error) {
	return r.store.UpdateLoveDateForUser(userID, loveDate)
}

func (r *Repo) UpdateUserProfile(currentUserID string, user model.User) (model.User, error) {
	return r.store.UpdateUserProfileForUser(currentUserID, user)
}

func (r *Repo) Unpair(currentUserID string) (model.UnpairResult, error) {
	return r.store.UnpairForUser(currentUserID)
}

func (r *Repo) Dashboard(userID string) (model.DashboardPayload, error) {
	return r.store.DashboardForUser(userID)
}

func (r *Repo) Moments(userID string) ([]model.Moment, error) {
	return r.store.MomentsForUser(userID)
}

func (r *Repo) AddMoment(userID string, moment model.Moment) (model.Moment, error) {
	return r.store.AddMomentForUser(userID, moment)
}

func (r *Repo) DeleteMoment(userID, id string) error {
	return r.store.DeleteMomentForUser(userID, id)
}

func (r *Repo) UpdateMomentLiked(userID, id string, liked bool) (model.Moment, error) {
	return r.store.UpdateMomentLikedForUser(userID, id, liked)
}

func (r *Repo) Tasks(userID string) ([]model.Task, error) {
	return r.store.TasksForUser(userID)
}

func (r *Repo) AddTask(userID string, task model.Task) (model.Task, error) {
	return r.store.AddTaskForUser(userID, task)
}

func (r *Repo) DeleteTask(userID, id string) error {
	return r.store.DeleteTaskForUser(userID, id)
}

func (r *Repo) UpdateTaskStatus(userID, id string, status model.TaskStatus) (model.Task, error) {
	return r.store.UpdateTaskStatusForUser(userID, id, status)
}

func (r *Repo) ScheduledTasks(userID string) ([]model.ScheduledTask, error) {
	return r.store.ScheduledTasksForUser(userID)
}

func (r *Repo) AddScheduledTask(userID string, task model.ScheduledTask) (model.ScheduledTask, error) {
	return r.store.AddScheduledTaskForUser(userID, task)
}

func (r *Repo) DeleteScheduledTask(userID, id string) error {
	return r.store.DeleteScheduledTaskForUser(userID, id)
}

func (r *Repo) ConfirmScheduledTask(userID, id string) (model.ScheduledTask, error) {
	return r.store.ConfirmScheduledTaskForUser(userID, id)
}

func (r *Repo) Dishes(userID string) ([]model.Dish, error) {
	return r.store.DishesForUser(userID)
}

func (r *Repo) AddDish(userID string, dish model.Dish) (model.Dish, error) {
	return r.store.AddDishForUser(userID, dish)
}

func (r *Repo) DeleteDish(userID, id string) error {
	return r.store.DeleteDishForUser(userID, id)
}

func (r *Repo) UpdateDishEnabled(userID, id string, enabled bool) (model.Dish, error) {
	return r.store.UpdateDishEnabledForUser(userID, id, enabled)
}

func (r *Repo) Orders(userID string) ([]model.Order, error) {
	return r.store.OrdersForUser(userID)
}

func (r *Repo) AddOrder(userID string, order model.Order) (model.Order, error) {
	return r.store.AddOrderForUser(userID, order)
}

func (r *Repo) Goals(userID string) ([]model.Goal, error) {
	return r.store.GoalsForUser(userID)
}

func (r *Repo) AddGoal(userID string, goal model.Goal) (model.Goal, error) {
	return r.store.AddGoalForUser(userID, goal)
}

func (r *Repo) UpdateGoalValue(userID, id string, currentValue int) (model.Goal, error) {
	return r.store.UpdateGoalValueForUser(userID, id, currentValue)
}

func (r *Repo) UpdateGoalStatus(userID, id, status string) (model.Goal, error) {
	return r.store.UpdateGoalStatusForUser(userID, id, status)
}

func (r *Repo) DeleteGoal(userID, id string) error {
	return r.store.DeleteGoalForUser(userID, id)
}

func (r *Repo) AddNotice(notice model.Notice) (model.Notice, error) {
	return r.store.AddNotice(notice)
}

func (r *Repo) UnreadNotices(userID string) ([]model.Notice, error) {
	return r.store.UnreadNoticesForUser(userID)
}

func (r *Repo) MarkNoticesRead(userID string, categories []string) error {
	return r.store.MarkNoticesReadForUser(userID, categories)
}

func (r *Repo) AdminOverview() (model.AdminOverview, error) {
	return r.store.AdminOverview()
}

func (r *Repo) AdminRecentUsers(limit int) ([]model.AdminUserSummary, error) {
	return r.store.AdminRecentUsers(limit)
}

func (r *Repo) AdminRecentCouples(limit int) ([]model.AdminCoupleSummary, error) {
	return r.store.AdminRecentCouples(limit)
}

func (r *Repo) AdminUnpairCouple(coupleID string) (model.UnpairResult, error) {
	return r.store.AdminUnpairCouple(coupleID)
}
