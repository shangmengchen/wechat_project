package model

import "couple-mini/backend/internal/domain"

type User = domain.User
type Couple = domain.Couple
type UnpairResult = domain.UnpairResult
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
type Notice = domain.Notice
type MePayload = domain.MePayload
type DashboardPayload = domain.DashboardPayload
type UserLite = domain.UserLite
type SyncState = domain.SyncState
type AdminOverview = domain.AdminOverview
type AdminUserSummary = domain.AdminUserSummary
type AdminCoupleSummary = domain.AdminCoupleSummary
type AdminSystemPoint = domain.AdminSystemPoint
type AdminRuntime = domain.AdminRuntime
type AdminErrorLog = domain.AdminErrorLog
type AdminDashboard = domain.AdminDashboard

const (
	TaskTodo   = domain.TaskTodo
	TaskReview = domain.TaskReview
	TaskDone   = domain.TaskDone
)
