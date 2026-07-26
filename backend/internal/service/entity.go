package service

import "couple-mini/backend/internal/model"

type LoginRequest struct {
	UserID   string `json:"userId"`
	Code     string `json:"code"`
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type GeneratePairCodeRequest struct {
	UserID string `json:"userId"`
}

type ConfirmPairRequest struct {
	UserID   string `json:"userId"`
	Code     string `json:"code"`
	LoveDate string `json:"loveDate"`
}

type UpdateLoveDateRequest struct {
	LoveDate string `json:"loveDate"`
}

type UpdateMomentLikedRequest struct {
	Liked bool `json:"liked"`
}

type UpdateDishEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type UpdateGoalValueRequest struct {
	CurrentValue int `json:"currentValue"`
}

type UpdateGoalStatusRequest struct {
	Status string `json:"status"`
}

type UserProfileRequest = model.User
type CreateMomentRequest = model.Moment
type CreateTaskRequest = model.Task
type CreateScheduledTaskRequest = model.ScheduledTask
type CreateDishRequest = model.Dish
type CreateOrderRequest = model.Order
type CreateGoalRequest = model.Goal
