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
)

type Repo struct {
	store *store.MySQLStore
}

func New(db *gorm.DB) *Repo {
	return &Repo{store: store.NewMySQLStore(db)}
}

func (r *Repo) EnsureSchema(ctx context.Context) error {
	return r.store.EnsureSchema(ctx)
}

func (r *Repo) Login(openid, nickname string) (model.User, error) {
	return r.store.Login(openid, nickname)
}

func (r *Repo) GeneratePairCode(userID string) (model.Couple, error) {
	return r.store.GeneratePairCode(userID)
}

func (r *Repo) PairByCode(userID, code, loveDate string) (model.Couple, error) {
	return r.store.PairByCode(userID, code, loveDate)
}

func (r *Repo) UpdateLoveDate(loveDate string) (model.Couple, error) {
	return r.store.UpdateLoveDate(loveDate)
}

func (r *Repo) UpdateUserProfile(user model.User) (model.User, error) {
	return r.store.UpdateUserProfile(user)
}

func (r *Repo) Dashboard(userID string) (model.DashboardPayload, error) {
	return r.store.Dashboard(userID)
}

func (r *Repo) Moments() ([]model.Moment, error) {
	return r.store.Moments()
}

func (r *Repo) AddMoment(moment model.Moment) (model.Moment, error) {
	return r.store.AddMoment(moment)
}

func (r *Repo) DeleteMoment(id string) error {
	return r.store.DeleteMoment(id)
}

func (r *Repo) UpdateMomentLiked(id string, liked bool) (model.Moment, error) {
	return r.store.UpdateMomentLiked(id, liked)
}

func (r *Repo) Tasks() ([]model.Task, error) {
	return r.store.Tasks()
}

func (r *Repo) AddTask(task model.Task) (model.Task, error) {
	return r.store.AddTask(task)
}

func (r *Repo) DeleteTask(id string) error {
	return r.store.DeleteTask(id)
}

func (r *Repo) UpdateTaskStatus(id string, status model.TaskStatus) (model.Task, error) {
	return r.store.UpdateTaskStatus(id, status)
}

func (r *Repo) ScheduledTasks() ([]model.ScheduledTask, error) {
	return r.store.ScheduledTasks()
}

func (r *Repo) AddScheduledTask(task model.ScheduledTask) (model.ScheduledTask, error) {
	return r.store.AddScheduledTask(task)
}

func (r *Repo) DeleteScheduledTask(id string) error {
	return r.store.DeleteScheduledTask(id)
}

func (r *Repo) ConfirmScheduledTask(id string) (model.ScheduledTask, error) {
	return r.store.ConfirmScheduledTask(id)
}

func (r *Repo) Dishes() ([]model.Dish, error) {
	return r.store.Dishes()
}

func (r *Repo) AddDish(dish model.Dish) (model.Dish, error) {
	return r.store.AddDish(dish)
}

func (r *Repo) DeleteDish(id string) error {
	return r.store.DeleteDish(id)
}

func (r *Repo) UpdateDishEnabled(id string, enabled bool) (model.Dish, error) {
	return r.store.UpdateDishEnabled(id, enabled)
}

func (r *Repo) Orders() ([]model.Order, error) {
	return r.store.Orders()
}

func (r *Repo) AddOrder(order model.Order) (model.Order, error) {
	return r.store.AddOrder(order)
}

func (r *Repo) Goals() ([]model.Goal, error) {
	return r.store.Goals()
}

func (r *Repo) AddGoal(goal model.Goal) (model.Goal, error) {
	return r.store.AddGoal(goal)
}

func (r *Repo) UpdateGoalValue(id string, currentValue int) (model.Goal, error) {
	return r.store.UpdateGoalValue(id, currentValue)
}

func (r *Repo) UpdateGoalStatus(id, status string) (model.Goal, error) {
	return r.store.UpdateGoalStatus(id, status)
}

func (r *Repo) DeleteGoal(id string) error {
	return r.store.DeleteGoal(id)
}
