package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"couple-mini/backend/internal/domain"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidPairCode = errors.New("invalid pair code")
	ErrPairCodeExpired = errors.New("pair code expired")
	ErrAlreadyPaired   = errors.New("already paired")
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) EnsureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(32) PRIMARY KEY,
			openid VARCHAR(128) NOT NULL UNIQUE,
			nickname VARCHAR(64) NOT NULL,
			avatar VARCHAR(255) NOT NULL DEFAULT '',
			birthday VARCHAR(32) NOT NULL DEFAULT '',
			wxid VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS couples (
			id VARCHAR(32) PRIMARY KEY,
			user_a_id VARCHAR(32) NOT NULL,
			user_b_id VARCHAR(32) NOT NULL,
			love_date VARCHAR(32) NOT NULL,
			pair_code VARCHAR(16) NOT NULL DEFAULT '',
			code_expire_at DATETIME NULL,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS moments (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			author_id VARCHAR(32) NOT NULL DEFAULT '',
			author VARCHAR(64) NOT NULL,
			avatar VARCHAR(32) NOT NULL,
			time_label VARCHAR(64) NOT NULL,
			tag VARCHAR(64) NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			image VARCHAR(500) NOT NULL DEFAULT '',
			liked TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			title VARCHAR(120) NOT NULL,
			owner VARCHAR(64) NOT NULL,
			type VARCHAR(32) NOT NULL,
			tag VARCHAR(64) NOT NULL DEFAULT '',
			due VARCHAR(32) NOT NULL DEFAULT '',
			reward VARCHAR(64) NOT NULL DEFAULT '',
			status VARCHAR(16) NOT NULL,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS scheduled_tasks (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			title VARCHAR(120) NOT NULL,
			cycle VARCHAR(64) NOT NULL,
			assignee VARCHAR(64) NOT NULL,
			time_text VARCHAR(32) NOT NULL,
			next_text VARCHAR(64) NOT NULL,
			pending TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS dishes (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			icon VARCHAR(32) NOT NULL,
			name VARCHAR(80) NOT NULL,
			meal VARCHAR(32) NOT NULL,
			enabled TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS orders (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			date_text VARCHAR(32) NOT NULL,
			meal VARCHAR(32) NOT NULL,
			picker VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS order_dishes (
			order_id VARCHAR(32) NOT NULL,
			dish_name VARCHAR(80) NOT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			PRIMARY KEY(order_id, sort_order)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS goals (
			id VARCHAR(32) PRIMARY KEY,
			couple_id VARCHAR(32) NOT NULL,
			title VARCHAR(120) NOT NULL,
			period VARCHAR(32) NOT NULL,
			target_value INT NOT NULL DEFAULT 100,
			current_value INT NOT NULL DEFAULT 0,
			start_date VARCHAR(32) NOT NULL DEFAULT '',
			target_date VARCHAR(32) NOT NULL DEFAULT '',
			progress INT NOT NULL DEFAULT 0,
			remain_days INT NOT NULL DEFAULT 0,
			status VARCHAR(16) NOT NULL,
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := s.ensureColumn(ctx, "tasks", "tag", "VARCHAR(64) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{"target_value", "INT NOT NULL DEFAULT 100"},
		{"current_value", "INT NOT NULL DEFAULT 0"},
		{"start_date", "VARCHAR(32) NOT NULL DEFAULT ''"},
		{"target_date", "VARCHAR(32) NOT NULL DEFAULT ''"},
	} {
		if err := s.ensureColumn(ctx, "goals", column.name, column.definition); err != nil {
			return err
		}
	}
	return s.seed(ctx)
}

func (s *MySQLStore) Login(openid, nickname string) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRow(`SELECT id, openid, nickname, avatar, birthday, wxid, created_at FROM users WHERE openid = ?`, openid).
		Scan(&user.ID, &user.OpenID, &user.Nickname, &user.Avatar, &user.Birthday, &user.WxID, &user.CreatedAt)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, err
	}

	user = domain.User{ID: newID("u"), OpenID: openid, Nickname: nickname, CreatedAt: time.Now()}
	_, err = s.db.Exec(`INSERT INTO users (id, openid, nickname, created_at) VALUES (?, ?, ?, ?)`, user.ID, user.OpenID, user.Nickname, user.CreatedAt)
	return user, err
}

func (s *MySQLStore) GeneratePairCode(userID string) (domain.Couple, error) {
	code, err := s.uniquePairCode()
	if err != nil {
		return domain.Couple{}, err
	}
	expireAt := time.Now().Add(24 * time.Hour)

	var pendingID string
	err = s.db.QueryRow(`SELECT id FROM couples WHERE user_a_id = ? AND user_b_id = '' LIMIT 1`, userID).Scan(&pendingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Couple{}, err
	}
	if pendingID != "" {
		result, err := s.db.Exec(`UPDATE couples SET pair_code = ?, code_expire_at = ? WHERE id = ?`, code, expireAt, pendingID)
		if err != nil {
			return domain.Couple{}, err
		}
		if err := requireAffected(result); err != nil {
			return domain.Couple{}, err
		}
		return s.coupleByID(pendingID)
	}

	paired, err := s.count(`SELECT COUNT(*) FROM couples WHERE (user_a_id = ? OR user_b_id = ?) AND user_b_id <> ''`, userID, userID)
	if err != nil {
		return domain.Couple{}, err
	}
	if paired > 0 {
		return domain.Couple{}, ErrAlreadyPaired
	}

	couple := domain.Couple{
		ID:           newID("c"),
		UserAID:      userID,
		UserBID:      "",
		LoveDate:     time.Now().Format("2006-01-02"),
		PairCode:     code,
		CodeExpireAt: expireAt,
		CreatedAt:    time.Now(),
	}
	_, err = s.db.Exec(`INSERT INTO couples (id, user_a_id, user_b_id, love_date, pair_code, code_expire_at, created_at) VALUES (?, ?, '', ?, ?, ?, ?)`,
		couple.ID, couple.UserAID, couple.LoveDate, couple.PairCode, couple.CodeExpireAt, couple.CreatedAt)
	if err != nil {
		return domain.Couple{}, err
	}
	return couple, nil
}

func (s *MySQLStore) PairByCode(userID, code, loveDate string) (domain.Couple, error) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return domain.Couple{}, ErrInvalidPairCode
	}
	if loveDate == "" {
		loveDate = time.Now().Format("2006-01-02")
	}
	if _, err := parseISODate(loveDate); err != nil {
		return domain.Couple{}, err
	}

	paired, err := s.count(`SELECT COUNT(*) FROM couples WHERE (user_a_id = ? OR user_b_id = ?) AND user_b_id <> ''`, userID, userID)
	if err != nil {
		return domain.Couple{}, err
	}
	if paired > 0 {
		return domain.Couple{}, ErrAlreadyPaired
	}

	var couple domain.Couple
	var expire sql.NullTime
	err = s.db.QueryRow(`SELECT id, user_a_id, user_b_id, love_date, pair_code, code_expire_at, created_at FROM couples WHERE pair_code = ? LIMIT 1`, code).
		Scan(&couple.ID, &couple.UserAID, &couple.UserBID, &couple.LoveDate, &couple.PairCode, &expire, &couple.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Couple{}, ErrInvalidPairCode
	}
	if err != nil {
		return domain.Couple{}, err
	}
	if couple.UserAID == userID {
		return domain.Couple{}, ErrInvalidPairCode
	}
	if couple.UserBID != "" {
		return domain.Couple{}, ErrAlreadyPaired
	}
	if !expire.Valid || !expire.Time.After(time.Now()) {
		return domain.Couple{}, ErrPairCodeExpired
	}

	result, err := s.db.Exec(`UPDATE couples SET user_b_id = ?, love_date = ?, pair_code = '', code_expire_at = NULL WHERE id = ? AND user_b_id = '' AND pair_code = ?`,
		userID, loveDate, couple.ID, code)
	if err != nil {
		return domain.Couple{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Couple{}, err
	}
	return s.coupleByID(couple.ID)
}

func (s *MySQLStore) UpdateLoveDate(loveDate string) (domain.Couple, error) {
	if _, err := parseISODate(loveDate); err != nil {
		return domain.Couple{}, err
	}
	result, err := s.db.Exec(`UPDATE couples SET love_date = ? WHERE id = 'c1'`, loveDate)
	if err != nil {
		return domain.Couple{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Couple{}, err
	}
	return s.couple()
}

func (s *MySQLStore) UpdateUserProfile(user domain.User) (domain.User, error) {
	if birthday, ok := parseBirthday(user.Birthday); ok {
		user.Birthday = formatMonthDay(birthday)
	}
	result, err := s.db.Exec(`UPDATE users SET nickname = ?, avatar = ?, birthday = ?, wxid = ? WHERE id = ?`,
		user.Nickname, user.Avatar, user.Birthday, user.WxID, user.ID)
	if err != nil {
		return domain.User{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.User{}, err
	}
	return s.user(user.ID)
}

func (s *MySQLStore) Dashboard(userID string) (domain.DashboardPayload, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	me, err := s.user(couple.UserAID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	partner, err := s.user(couple.UserBID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}

	momentCount, _ := s.count(`SELECT COUNT(*) FROM moments WHERE couple_id = ?`, couple.ID)
	openTaskCount, _ := s.count(`SELECT COUNT(*) FROM tasks WHERE couple_id = ? AND status <> 'done'`, couple.ID)
	activeGoalCount, _ := s.count(`SELECT COUNT(*) FROM goals WHERE couple_id = ? AND status = 'active'`, couple.ID)
	goalProgress, _ := s.average(`SELECT AVG(progress) FROM goals WHERE couple_id = ? AND status = 'active'`, couple.ID)

	return domain.DashboardPayload{
		Users: map[string]domain.UserLite{
			"me":      lite(me, "小"),
			"partner": lite(partner, "杰"),
		},
		Dashboard: domain.Dashboard{
			LoveDays:      loveDays(couple.LoveDate),
			Since:         couple.LoveDate,
			Anniversaries: anniversaries(couple, me, partner),
			Stats: []domain.Stat{
				{Icon: "📷", Value: momentCount, Label: "纪念动态"},
				{Icon: "📋", Value: openTaskCount, Label: "待办任务"},
				{Icon: "🎯", Value: activeGoalCount, Label: "进行目标"},
			},
			ActiveGoals:  activeGoalCount,
			GoalProgress: goalProgress,
			PendingTasks: openTaskCount,
		},
	}, nil
}

func (s *MySQLStore) Moments() ([]domain.Moment, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, author_id, author, avatar, time_label, tag, content, image, liked, created_at FROM moments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Moment
	for rows.Next() {
		var item domain.Moment
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.AuthorID, &item.Author, &item.Avatar, &item.TimeLabel, &item.Tag, &item.Content, &item.Image, &item.Liked, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddMoment(moment domain.Moment) (domain.Moment, error) {
	moment.ID = newID("m")
	moment.CoupleID = defaultCouple(moment.CoupleID)
	moment.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO moments (id, couple_id, author_id, author, avatar, time_label, tag, content, image, liked, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		moment.ID, moment.CoupleID, moment.AuthorID, moment.Author, moment.Avatar, moment.TimeLabel, moment.Tag, moment.Content, moment.Image, moment.Liked, moment.CreatedAt)
	return moment, err
}

func (s *MySQLStore) DeleteMoment(id string) error {
	return deleteByID(s.db, "moments", id)
}

func (s *MySQLStore) UpdateMomentLiked(id string, liked bool) (domain.Moment, error) {
	result, err := s.db.Exec(`UPDATE moments SET liked = ? WHERE id = ?`, liked, id)
	if err != nil {
		return domain.Moment{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Moment{}, err
	}
	return s.moment(id)
}

func (s *MySQLStore) Tasks() ([]domain.Task, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, title, owner, type, tag, due, reward, status FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Task
	for rows.Next() {
		var item domain.Task
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Title, &item.Owner, &item.Type, &item.Tag, &item.Due, &item.Reward, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddTask(task domain.Task) (domain.Task, error) {
	task.ID = newID("t")
	task.CoupleID = defaultCouple(task.CoupleID)
	task.Status = domain.TaskTodo
	_, err := s.db.Exec(`INSERT INTO tasks (id, couple_id, title, owner, type, tag, due, reward, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.CoupleID, task.Title, task.Owner, task.Type, task.Tag, task.Due, task.Reward, task.Status, time.Now())
	return task, err
}

func (s *MySQLStore) DeleteTask(id string) error {
	return deleteByID(s.db, "tasks", id)
}

func (s *MySQLStore) UpdateTaskStatus(id string, status domain.TaskStatus) (domain.Task, error) {
	result, err := s.db.Exec(`UPDATE tasks SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return domain.Task{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Task{}, err
	}
	var task domain.Task
	err = s.db.QueryRow(`SELECT id, couple_id, title, owner, type, tag, due, reward, status FROM tasks WHERE id = ?`, id).
		Scan(&task.ID, &task.CoupleID, &task.Title, &task.Owner, &task.Type, &task.Tag, &task.Due, &task.Reward, &task.Status)
	return task, err
}

func (s *MySQLStore) ScheduledTasks() ([]domain.ScheduledTask, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, title, cycle, assignee, time_text, next_text, pending FROM scheduled_tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ScheduledTask
	for rows.Next() {
		var item domain.ScheduledTask
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Title, &item.Cycle, &item.Assignee, &item.Time, &item.Next, &item.Pending); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddScheduledTask(task domain.ScheduledTask) (domain.ScheduledTask, error) {
	task.ID = newID("s")
	task.CoupleID = defaultCouple(task.CoupleID)
	task.Pending = true
	_, err := s.db.Exec(`INSERT INTO scheduled_tasks (id, couple_id, title, cycle, assignee, time_text, next_text, pending, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.CoupleID, task.Title, task.Cycle, task.Assignee, task.Time, task.Next, task.Pending, time.Now())
	return task, err
}

func (s *MySQLStore) DeleteScheduledTask(id string) error {
	return deleteByID(s.db, "scheduled_tasks", id)
}

func (s *MySQLStore) ConfirmScheduledTask(id string) (domain.ScheduledTask, error) {
	result, err := s.db.Exec(`UPDATE scheduled_tasks SET pending = 0 WHERE id = ?`, id)
	if err != nil {
		return domain.ScheduledTask{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.ScheduledTask{}, err
	}
	var task domain.ScheduledTask
	err = s.db.QueryRow(`SELECT id, couple_id, title, cycle, assignee, time_text, next_text, pending FROM scheduled_tasks WHERE id = ?`, id).
		Scan(&task.ID, &task.CoupleID, &task.Title, &task.Cycle, &task.Assignee, &task.Time, &task.Next, &task.Pending)
	return task, err
}

func (s *MySQLStore) Dishes() ([]domain.Dish, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, icon, name, meal, enabled FROM dishes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Dish
	for rows.Next() {
		var item domain.Dish
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Icon, &item.Name, &item.Meal, &item.Enabled); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddDish(dish domain.Dish) (domain.Dish, error) {
	dish.ID = newID("d")
	dish.CoupleID = defaultCouple(dish.CoupleID)
	_, err := s.db.Exec(`INSERT INTO dishes (id, couple_id, icon, name, meal, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		dish.ID, dish.CoupleID, dish.Icon, dish.Name, dish.Meal, dish.Enabled, time.Now())
	return dish, err
}

func (s *MySQLStore) DeleteDish(id string) error {
	return deleteByID(s.db, "dishes", id)
}

func (s *MySQLStore) UpdateDishEnabled(id string, enabled bool) (domain.Dish, error) {
	result, err := s.db.Exec(`UPDATE dishes SET enabled = ? WHERE id = ?`, enabled, id)
	if err != nil {
		return domain.Dish{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Dish{}, err
	}
	var dish domain.Dish
	err = s.db.QueryRow(`SELECT id, couple_id, icon, name, meal, enabled FROM dishes WHERE id = ?`, id).
		Scan(&dish.ID, &dish.CoupleID, &dish.Icon, &dish.Name, &dish.Meal, &dish.Enabled)
	return dish, err
}

func (s *MySQLStore) Orders() ([]domain.Order, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, date_text, meal, picker FROM orders ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Order
	for rows.Next() {
		var item domain.Order
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Date, &item.Meal, &item.Picker); err != nil {
			return nil, err
		}
		item.Dishes, err = s.orderDishes(item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddOrder(order domain.Order) (domain.Order, error) {
	order.ID = newID("o")
	order.CoupleID = defaultCouple(order.CoupleID)
	if order.Date == "" {
		order.Date = formatMonthDay(time.Now())
	}
	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO orders (id, couple_id, date_text, meal, picker, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		order.ID, order.CoupleID, order.Date, order.Meal, order.Picker, now); err != nil {
		return domain.Order{}, err
	}
	for i, name := range order.Dishes {
		if _, err := tx.Exec(`INSERT INTO order_dishes (order_id, dish_name, sort_order) VALUES (?, ?, ?)`, order.ID, name, i); err != nil {
			return domain.Order{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (s *MySQLStore) Goals() ([]domain.Goal, error) {
	rows, err := s.db.Query(`SELECT id, couple_id, title, period, target_value, current_value, start_date, target_date, progress, remain_days, status FROM goals ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Goal
	for rows.Next() {
		var item domain.Goal
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Title, &item.Period, &item.TargetValue, &item.CurrentValue, &item.StartDate, &item.TargetDate, &item.Progress, &item.RemainDays, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, calculateGoal(item))
	}
	return items, rows.Err()
}

func (s *MySQLStore) AddGoal(goal domain.Goal) (domain.Goal, error) {
	goal.ID = newID("g")
	goal.CoupleID = defaultCouple(goal.CoupleID)
	if goal.TargetValue <= 0 {
		goal.TargetValue = 100
	}
	if strings.TrimSpace(goal.StartDate) == "" {
		goal.StartDate = time.Now().Format("2006-01-02")
	}
	if strings.TrimSpace(goal.TargetDate) == "" {
		goal.TargetDate = time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	}
	goal = calculateGoal(goal)
	if goal.Progress >= 100 {
		goal.Status = "done"
	}
	_, err := s.db.Exec(`INSERT INTO goals (id, couple_id, title, period, target_value, current_value, start_date, target_date, progress, remain_days, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		goal.ID, goal.CoupleID, goal.Title, goal.Period, goal.TargetValue, goal.CurrentValue, goal.StartDate, goal.TargetDate, goal.Progress, goal.RemainDays, goal.Status, time.Now())
	return goal, err
}

func (s *MySQLStore) UpdateGoalValue(id string, currentValue int) (domain.Goal, error) {
	goal, err := s.goal(id)
	if err != nil {
		return domain.Goal{}, err
	}
	goal.CurrentValue = currentValue
	goal = calculateGoal(goal)
	if goal.Progress >= 100 {
		goal.Status = "done"
	}
	result, err := s.db.Exec(`UPDATE goals SET current_value = ?, progress = ?, remain_days = ?, status = ? WHERE id = ?`, goal.CurrentValue, goal.Progress, goal.RemainDays, goal.Status, id)
	if err != nil {
		return domain.Goal{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Goal{}, err
	}
	return goal, nil
}

func (s *MySQLStore) UpdateGoalStatus(id, status string) (domain.Goal, error) {
	goal, err := s.goal(id)
	if err != nil {
		return domain.Goal{}, err
	}
	goal.Status = status
	if status == "done" {
		goal.CurrentValue = goal.TargetValue
	}
	goal = calculateGoal(goal)
	result, err := s.db.Exec(`UPDATE goals SET status = ?, current_value = ?, progress = ?, remain_days = ? WHERE id = ?`, goal.Status, goal.CurrentValue, goal.Progress, goal.RemainDays, id)
	if err != nil {
		return domain.Goal{}, err
	}
	if err := requireAffected(result); err != nil {
		return domain.Goal{}, err
	}
	return goal, nil
}

func (s *MySQLStore) DeleteGoal(id string) error {
	return deleteByID(s.db, "goals", id)
}

func (s *MySQLStore) seed(ctx context.Context) error {
	total, err := s.countContext(ctx, `SELECT COUNT(*) FROM users`)
	if err != nil || total > 0 {
		return err
	}
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	execs := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, openid, nickname, birthday, wxid, created_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"u1", "demo-openid-u1", "小雨", "8月31日", "xiaoyu_2024", now}},
		{`INSERT INTO users (id, openid, nickname, birthday, wxid, created_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"u2", "demo-openid-u2", "阿杰", "9月18日", "ajie_2024", now}},
		{`INSERT INTO couples (id, user_a_id, user_b_id, love_date, created_at) VALUES (?, ?, ?, ?, ?)`, []any{"c1", "u1", "u2", "2023-07-28", now}},
		{`INSERT INTO moments (id, couple_id, author_id, author, avatar, time_label, tag, content, image, liked, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"m1", "c1", "u1", "小雨", "小", "2小时前", "约会日记", "今天去了北海公园划船，阳光超好，幸福感满满～", "https://images.unsplash.com/photo-1507525428034-b723cf961d3e?w=900&auto=format&fit=crop", true, now.Add(-2 * time.Hour)}},
		{`INSERT INTO moments (id, couple_id, author_id, author, avatar, time_label, tag, content, image, liked, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"m2", "c1", "u2", "阿杰", "杰", "昨天 20:30", "生活小事", "给小雨做了她最爱的番茄炒蛋，说我做得比她妈妈还好吃哈哈！", "", false, now.AddDate(0, 0, -1)}},
	}
	for _, item := range execs {
		if _, err := tx.ExecContext(ctx, item.query, item.args...); err != nil {
			return err
		}
	}

	for _, task := range []domain.Task{
		{ID: "t1", Title: "帮小雨买她想要的无线耳机", Owner: "阿杰", Type: "一次性", Due: "7月25日", Reward: "亲亲", Status: domain.TaskTodo},
		{ID: "t2", Title: "一起做一顿丰盛的晚餐", Owner: "双方", Type: "一次性", Due: "7月22日", Reward: "拥抱", Status: domain.TaskTodo},
		{ID: "t3", Title: "整理衣柜", Owner: "小雨", Type: "每月", Status: domain.TaskTodo},
		{ID: "t4", Title: "陪小雨晨跑30分钟", Owner: "阿杰", Type: "每日", Reward: "夸夸", Status: domain.TaskReview},
		{ID: "t5", Title: "预约牙医复查", Owner: "小雨", Type: "一次性", Status: domain.TaskDone},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tasks (id, couple_id, title, owner, type, tag, due, reward, status, created_at) VALUES (?, 'c1', ?, ?, ?, ?, ?, ?, ?, ?)`, task.ID, task.Title, task.Owner, task.Type, task.Tag, task.Due, task.Reward, task.Status, now); err != nil {
			return err
		}
	}

	for _, task := range []domain.ScheduledTask{
		{ID: "s1", Title: "倒猫砂", Cycle: "每2天", Assignee: "轮流", Time: "20:00", Next: "今天 20:00", Pending: true},
		{ID: "s2", Title: "浇阳台的花", Cycle: "每周三", Assignee: "小雨", Time: "19:30", Next: "周三 19:30", Pending: false},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO scheduled_tasks (id, couple_id, title, cycle, assignee, time_text, next_text, pending, created_at) VALUES (?, 'c1', ?, ?, ?, ?, ?, ?, ?)`, task.ID, task.Title, task.Cycle, task.Assignee, task.Time, task.Next, task.Pending, now); err != nil {
			return err
		}
	}

	for _, dish := range []domain.Dish{
		{ID: "d1", Icon: "🍳", Name: "番茄炒蛋", Meal: "通用", Enabled: true},
		{ID: "d2", Icon: "🥣", Name: "皮蛋瘦肉粥", Meal: "早餐", Enabled: true},
		{ID: "d3", Icon: "🥩", Name: "红烧肉", Meal: "晚餐", Enabled: true},
		{ID: "d4", Icon: "🥦", Name: "蒜蓉西兰花", Meal: "通用", Enabled: true},
		{ID: "d5", Icon: "🥪", Name: "三明治", Meal: "早餐", Enabled: false},
		{ID: "d6", Icon: "🍲", Name: "麻婆豆腐", Meal: "午餐", Enabled: true},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO dishes (id, couple_id, icon, name, meal, enabled, created_at) VALUES (?, 'c1', ?, ?, ?, ?, ?)`, dish.ID, dish.Icon, dish.Name, dish.Meal, dish.Enabled, now); err != nil {
			return err
		}
	}

	orders := []domain.Order{
		{ID: "o1", Date: "7月18日", Meal: "晚餐", Picker: "阿杰选的", Dishes: []string{"番茄炒蛋", "蒜蓉西兰花"}},
		{ID: "o2", Date: "7月18日", Meal: "午餐", Picker: "小雨选的", Dishes: []string{"麻婆豆腐"}},
		{ID: "o3", Date: "7月17日", Meal: "晚餐", Picker: "阿杰选的", Dishes: []string{"红烧肉", "番茄炒蛋"}},
	}
	for _, order := range orders {
		if _, err := tx.ExecContext(ctx, `INSERT INTO orders (id, couple_id, date_text, meal, picker, created_at) VALUES (?, 'c1', ?, ?, ?, ?)`, order.ID, order.Date, order.Meal, order.Picker, now); err != nil {
			return err
		}
		for i, name := range order.Dishes {
			if _, err := tx.ExecContext(ctx, `INSERT INTO order_dishes (order_id, dish_name, sort_order) VALUES (?, ?, ?)`, order.ID, name, i); err != nil {
				return err
			}
		}
	}

	for _, goal := range []domain.Goal{
		{ID: "g1", Title: "一起存下旅行基金", Period: "月目标", Progress: 68, RemainDays: 12, Status: "active"},
		{ID: "g2", Title: "每周一起运动 3 次", Period: "周目标", Progress: 42, RemainDays: 4, Status: "active"},
		{ID: "g3", Title: "读完一本关系沟通书", Period: "季度目标", Progress: 100, RemainDays: 0, Status: "done"},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO goals (id, couple_id, title, period, progress, remain_days, status, created_at) VALUES (?, 'c1', ?, ?, ?, ?, ?, ?)`, goal.ID, goal.Title, goal.Period, goal.Progress, goal.RemainDays, goal.Status, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) couple() (domain.Couple, error) {
	return s.coupleByID("c1")
}

func (s *MySQLStore) coupleForUser(userID string) (domain.Couple, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return s.couple()
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM couples WHERE (user_a_id = ? OR user_b_id = ?) AND user_b_id <> '' ORDER BY created_at DESC LIMIT 1`, userID, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Couple{}, ErrNotFound
	}
	if err != nil {
		return domain.Couple{}, err
	}
	return s.coupleByID(id)
}

func (s *MySQLStore) coupleByID(id string) (domain.Couple, error) {
	var couple domain.Couple
	var expire sql.NullTime
	err := s.db.QueryRow(`SELECT id, user_a_id, user_b_id, love_date, pair_code, code_expire_at, created_at FROM couples WHERE id = ?`, id).
		Scan(&couple.ID, &couple.UserAID, &couple.UserBID, &couple.LoveDate, &couple.PairCode, &expire, &couple.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Couple{}, ErrNotFound
	}
	if expire.Valid {
		couple.CodeExpireAt = expire.Time
	}
	return couple, err
}

func (s *MySQLStore) user(id string) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRow(`SELECT id, openid, nickname, avatar, birthday, wxid, created_at FROM users WHERE id = ?`, id).
		Scan(&user.ID, &user.OpenID, &user.Nickname, &user.Avatar, &user.Birthday, &user.WxID, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *MySQLStore) moment(id string) (domain.Moment, error) {
	var item domain.Moment
	err := s.db.QueryRow(`SELECT id, couple_id, author_id, author, avatar, time_label, tag, content, image, liked, created_at FROM moments WHERE id = ?`, id).
		Scan(&item.ID, &item.CoupleID, &item.AuthorID, &item.Author, &item.Avatar, &item.TimeLabel, &item.Tag, &item.Content, &item.Image, &item.Liked, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Moment{}, ErrNotFound
	}
	return item, err
}

func (s *MySQLStore) goal(id string) (domain.Goal, error) {
	var item domain.Goal
	err := s.db.QueryRow(`SELECT id, couple_id, title, period, target_value, current_value, start_date, target_date, progress, remain_days, status FROM goals WHERE id = ?`, id).
		Scan(&item.ID, &item.CoupleID, &item.Title, &item.Period, &item.TargetValue, &item.CurrentValue, &item.StartDate, &item.TargetDate, &item.Progress, &item.RemainDays, &item.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Goal{}, ErrNotFound
	}
	return calculateGoal(item), err
}

func (s *MySQLStore) orderDishes(orderID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT dish_name FROM order_dishes WHERE order_id = ? ORDER BY sort_order`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	return items, rows.Err()
}

func (s *MySQLStore) count(query string, args ...any) (int, error) {
	var total int
	err := s.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (s *MySQLStore) average(query string, args ...any) (int, error) {
	var average sql.NullFloat64
	err := s.db.QueryRow(query, args...).Scan(&average)
	if !average.Valid {
		return 0, err
	}
	return int(math.Round(average.Float64)), err
}

func (s *MySQLStore) countContext(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *MySQLStore) ensureColumn(ctx context.Context, table, column, definition string) error {
	if !isKnownMigrationColumn(table, column) {
		return fmt.Errorf("unsupported migration column %s.%s", table, column)
	}
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column).Scan(&total)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func isKnownMigrationColumn(table, column string) bool {
	switch table + "." + column {
	case "tasks.tag", "goals.target_value", "goals.current_value", "goals.start_date", "goals.target_date":
		return true
	default:
		return false
	}
}

func deleteByID(db *sql.DB, table string, id string) error {
	if !isKnownTable(table) {
		return fmt.Errorf("unknown table %s", table)
	}
	result, err := db.Exec(`DELETE FROM `+table+` WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func requireAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func isKnownTable(table string) bool {
	switch table {
	case "moments", "tasks", "scheduled_tasks", "dishes", "goals":
		return true
	default:
		return false
	}
}

func defaultCouple(coupleID string) string {
	if strings.TrimSpace(coupleID) == "" {
		return "c1"
	}
	return coupleID
}

func newID(prefix string) string {
	return fmt.Sprintf("%s%d%d", prefix, time.Now().UnixNano(), rand.Intn(1000))
}

func (s *MySQLStore) uniquePairCode() (string, error) {
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("%06d", rand.Intn(900000)+100000)
		total, err := s.count(`SELECT COUNT(*) FROM couples WHERE pair_code = ? AND code_expire_at > ?`, code, time.Now())
		if err != nil {
			return "", err
		}
		if total == 0 {
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique pair code")
}

func lite(user domain.User, avatarText string) domain.UserLite {
	return domain.UserLite{
		ID:         user.ID,
		Name:       user.Nickname,
		AvatarText: avatarText,
		Birthday:   user.Birthday,
		WxID:       user.WxID,
	}
}

func loveDays(loveDate string) int {
	start, err := time.ParseInLocation("2006-01-02", loveDate, time.Local)
	if err != nil {
		return 0
	}
	days := int(time.Since(start).Hours()/24) + 1
	if days < 0 {
		return 0
	}
	return days
}

func calculateGoal(goal domain.Goal) domain.Goal {
	if goal.TargetValue <= 0 {
		goal.TargetValue = 100
	}
	if goal.Status == "done" {
		goal.CurrentValue = goal.TargetValue
		goal.Progress = 100
		goal.TimeProgress = 100
		goal.RemainDays = 0
		return goal
	}
	goal.Progress = clampPercent(int(math.Round(float64(goal.CurrentValue) * 100 / float64(goal.TargetValue))))

	start, startErr := parseISODate(goal.StartDate)
	target, targetErr := parseISODate(goal.TargetDate)
	today := todayDate()
	if startErr == nil && targetErr == nil {
		totalDays := int(target.Sub(start).Hours() / 24)
		if totalDays <= 0 {
			goal.TimeProgress = 100
		} else {
			passedDays := int(today.Sub(start).Hours() / 24)
			goal.TimeProgress = clampPercent(int(math.Round(float64(passedDays) * 100 / float64(totalDays))))
		}
		remain := int(target.Sub(today).Hours() / 24)
		if remain < 0 {
			remain = 0
		}
		goal.RemainDays = remain
	}
	if goal.Progress >= 100 {
		goal.Progress = 100
		goal.Status = "done"
		goal.RemainDays = 0
	}
	if goal.Status == "" {
		goal.Status = "active"
	}
	return goal
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func todayDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}

func anniversaries(couple domain.Couple, me domain.User, partner domain.User) []domain.Anniversary {
	items := []domain.Anniversary{}
	if start, err := parseISODate(couple.LoveDate); err == nil {
		items = append(items, domain.Anniversary{
			Icon:  "💖",
			Title: "恋爱纪念日",
			Days:  daysUntilMonthDay(start.Month(), start.Day()),
			Date:  formatMonthDay(start),
			Tone:  "pink",
		})
	}
	if item, ok := birthdayAnniversary(me, "purple"); ok {
		items = append(items, item)
	}
	if item, ok := birthdayAnniversary(partner, "blue"); ok {
		items = append(items, item)
	}
	items = append(items, domain.Anniversary{
		Icon:  "🌹",
		Title: "七夕节",
		Days:  daysUntilMonthDay(time.August, 10),
		Date:  "8月10日",
		Tone:  "yellow",
	})
	return items
}

func birthdayAnniversary(user domain.User, tone string) (domain.Anniversary, bool) {
	date, ok := parseBirthday(user.Birthday)
	if !ok || strings.TrimSpace(user.Nickname) == "" {
		return domain.Anniversary{}, false
	}
	return domain.Anniversary{
		Icon:  "🎂",
		Title: user.Nickname + "生日",
		Days:  daysUntilMonthDay(date.Month(), date.Day()),
		Date:  formatMonthDay(date),
		Tone:  tone,
	}, true
}

func parseISODate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, time.Local)
}

func parseBirthday(value string) (time.Time, bool) {
	if date, err := parseISODate(value); err == nil {
		return date, true
	}
	date, err := time.ParseInLocation("1月2日", value, time.Local)
	return date, err == nil
}

func daysUntilMonthDay(month time.Month, day int) int {
	now := time.Now()
	next := time.Date(now.Year(), month, day, 0, 0, 0, 0, time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if next.Before(today) {
		next = next.AddDate(1, 0, 0)
	}
	return int(next.Sub(today).Hours() / 24)
}

func formatMonthDay(date time.Time) string {
	return fmt.Sprintf("%d月%d日", date.Month(), date.Day())
}
