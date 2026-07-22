package model

import "couple-mini/backend/internal/domain"

type User = domain.User
type Couple = domain.Couple
type Anniversary = domain.Anniversary
type Stat = domain.Stat
type Dashboard = domain.Dashboard
type Moment = domain.Moment
type TaskStatus = domain.TaskStatus
type Task = domain.Task
type ScheduledTask = domain.ScheduledTask
type Dish = domain.Dish
type Order = domain.Order
type Goal = domain.Goal
type MePayload = domain.MePayload
type DashboardPayload = domain.DashboardPayload
type UserLite = domain.UserLite

const (
	TaskTodo   = domain.TaskTodo
	TaskReview = domain.TaskReview
	TaskDone   = domain.TaskDone
)
