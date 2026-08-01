package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"couple-mini/backend/internal/domain"

	"gorm.io/gorm"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidPairCode = errors.New("invalid pair code")
	ErrPairCodeExpired = errors.New("pair code expired")
	ErrAlreadyPaired   = errors.New("already paired")
	ErrUnauthorized    = errors.New("unauthorized")
)

const pairCodeTTL = 20 * time.Minute

type MySQLStore struct {
	db     *gorm.DB
	pairMu sync.Mutex
}

func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) EnsureSchema(ctx context.Context, autoMigrate, autoSeed bool) error {
	if autoMigrate {
		tx := s.db.WithContext(ctx).Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
		if err := s.ensureCoupleMigrationCompatibility(ctx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(
			&userModel{},
			&coupleModel{},
			&momentModel{},
			&taskModel{},
			&scheduledTaskModel{},
			&dishModel{},
			&orderModel{},
			&orderDishModel{},
			&goalModel{},
			&noticeModel{},
		); err != nil {
			return err
		}
	}
	if !autoSeed {
		return nil
	}
	return s.seed(ctx)
}

func (s *MySQLStore) ensureCoupleMigrationCompatibility(ctx context.Context) error {
	tx := s.db.WithContext(ctx)
	if !tx.Migrator().HasTable(&coupleModel{}) || tx.Migrator().HasColumn(&coupleModel{}, "updated_at") {
		return nil
	}

	if err := tx.Exec("ALTER TABLE `couples` ADD COLUMN `updated_at` datetime(3) NULL").Error; err != nil {
		return err
	}
	if err := tx.Exec("UPDATE `couples` SET `updated_at` = NOW(3) WHERE `updated_at` IS NULL").Error; err != nil {
		return err
	}
	return tx.Exec("ALTER TABLE `couples` MODIFY COLUMN `updated_at` datetime(3) NOT NULL").Error
}

func (s *MySQLStore) Login(userID, openid, nickname, avatar string) (domain.User, error) {
	var user userModel
	if strings.TrimSpace(userID) != "" {
		err := s.db.Where("id = ?", userID).Take(&user).Error
		if err == nil {
			updates := map[string]any{}
			if strings.TrimSpace(openid) != "" && openid != user.OpenID {
				updates["openid"] = openid
			}
			if strings.TrimSpace(nickname) != "" && nickname != user.Nickname {
				updates["nickname"] = nickname
			}
			if strings.TrimSpace(avatar) != "" && avatar != user.Avatar {
				updates["avatar"] = avatar
			}
			if len(updates) == 0 {
				return toUser(user), nil
			}
			if err := s.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				return domain.User{}, err
			}
			return s.user(user.ID)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, err
		}
	}

	err := s.db.Where("openid = ?", openid).Take(&user).Error
	if err == nil {
		updates := map[string]any{}
		if strings.TrimSpace(nickname) != "" && nickname != user.Nickname {
			updates["nickname"] = nickname
		}
		if strings.TrimSpace(avatar) != "" && avatar != user.Avatar {
			updates["avatar"] = avatar
		}
		if len(updates) == 0 {
			return toUser(user), nil
		}
		if err := s.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return domain.User{}, err
		}
		return s.user(user.ID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, err
	}
	user = userModel{
		ID:        firstNonEmpty(strings.TrimSpace(userID), newID("u")),
		OpenID:    openid,
		Nickname:  nickname,
		Avatar:    avatar,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&user).Error; err != nil {
		return domain.User{}, err
	}
	return toUser(user), nil
}

func (s *MySQLStore) GeneratePairCode(userID string) (domain.Couple, error) {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()

	now := time.Now()
	var result domain.Couple

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var pending coupleModel
		err := tx.Where("user_a_id = ? AND user_b_id = ''", userID).Order("created_at DESC").Take(&pending).Error
		if err == nil {
			if pending.PairCode != "" && pending.CodeExpireAt != nil && pending.CodeExpireAt.After(now) {
				result = toCouple(pending)
				return nil
			}

			code, err := s.uniquePairCode(tx)
			if err != nil {
				return err
			}
			expireAt := now.Add(pairCodeTTL)
			updates := map[string]any{
				"pair_code":      code,
				"code_expire_at": expireAt,
				"version":        now.UnixNano(),
				"updated_at":     now,
			}
			update := tx.Model(&coupleModel{}).Where("id = ?", pending.ID).Updates(updates)
			if update.Error != nil {
				return update.Error
			}
			if err := requireAffected(update.RowsAffected); err != nil {
				return err
			}
			pending.PairCode = code
			pending.CodeExpireAt = &expireAt
			pending.Version = now.UnixNano()
			pending.UpdatedAt = now
			result = toCouple(pending)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		paired, err := s.countModelWithDB(tx, &coupleModel{}, "(user_a_id = ? OR user_b_id = ?) AND user_b_id <> ''", userID, userID)
		if err != nil {
			return err
		}
		if paired > 0 {
			return ErrAlreadyPaired
		}

		code, err := s.uniquePairCode(tx)
		if err != nil {
			return err
		}
		expireAt := now.Add(pairCodeTTL)

		couple := coupleModel{
			ID:           newID("c"),
			UserAID:      userID,
			UserBID:      "",
			LoveDate:     now.Format("2006-01-02"),
			PairCode:     code,
			CodeExpireAt: &expireAt,
			CreatedAt:    now,
			UpdatedAt:    now,
			Version:      now.UnixNano(),
		}
		if err := tx.Create(&couple).Error; err != nil {
			return err
		}
		result = toCouple(couple)
		return nil
	})
	if err != nil {
		return domain.Couple{}, err
	}
	return result, nil
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

	s.pairMu.Lock()
	defer s.pairMu.Unlock()

	var result domain.Couple
	err := s.db.Transaction(func(tx *gorm.DB) error {
		paired, err := s.countModelWithDB(tx, &coupleModel{}, "(user_a_id = ? OR user_b_id = ?) AND user_b_id <> ''", userID, userID)
		if err != nil {
			return err
		}
		if paired > 0 {
			return ErrAlreadyPaired
		}

		var couple coupleModel
		err = tx.Where("pair_code = ?", code).Take(&couple).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidPairCode
		}
		if err != nil {
			return err
		}
		if couple.UserAID == userID {
			return ErrInvalidPairCode
		}
		if couple.UserBID != "" {
			return ErrAlreadyPaired
		}
		if couple.CodeExpireAt == nil || !couple.CodeExpireAt.After(time.Now()) {
			return ErrPairCodeExpired
		}

		now := time.Now()
		updates := map[string]any{
			"user_b_id":      userID,
			"love_date":      loveDate,
			"pair_code":      "",
			"code_expire_at": nil,
			"version":        now.UnixNano(),
			"updated_at":     now,
		}
		update := tx.Model(&coupleModel{}).Where("id = ? AND user_b_id = '' AND pair_code = ?", couple.ID, code).Updates(updates)
		if update.Error != nil {
			return update.Error
		}
		if err := requireAffected(update.RowsAffected); err != nil {
			return err
		}
		couple.UserBID = userID
		couple.LoveDate = loveDate
		couple.PairCode = ""
		couple.CodeExpireAt = nil
		couple.Version = now.UnixNano()
		couple.UpdatedAt = now
		result = toCouple(couple)
		return nil
	})
	if err != nil {
		return domain.Couple{}, err
	}
	return result, nil
}

func (s *MySQLStore) UpdateLoveDate(loveDate string) (domain.Couple, error) {
	if _, err := parseISODate(loveDate); err != nil {
		return domain.Couple{}, err
	}
	result := s.db.Model(&coupleModel{}).Where("id = ?", "c1").Update("love_date", loveDate)
	if result.Error != nil {
		return domain.Couple{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Couple{}, err
	}
	return s.couple()
}

func (s *MySQLStore) UpdateUserProfile(user domain.User) (domain.User, error) {
	if birthday, ok := parseBirthday(user.Birthday); ok {
		user.Birthday = formatMonthDay(birthday)
	}
	result := s.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]any{
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"birthday": user.Birthday,
		"wxid":     user.WxID,
	})
	if result.Error != nil {
		return domain.User{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
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

	momentCount, _ := s.countModel(&momentModel{}, "couple_id = ?", couple.ID)
	openTaskCount, _ := s.countModel(&taskModel{}, "couple_id = ? AND status <> ?", couple.ID, domain.TaskDone)
	activeGoalCount, _ := s.countModel(&goalModel{}, "couple_id = ? AND status = ?", couple.ID, "active")
	goalProgress, _ := s.averageGoalProgress(couple.ID)

	return domain.DashboardPayload{
		Users: map[string]domain.UserLite{
			"me":      lite(me, "小"),
			"partner": lite(partner, "阿"),
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
	var rows []momentModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Moment, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMoment(row))
	}
	return items, nil
}

func (s *MySQLStore) AddMoment(moment domain.Moment) (domain.Moment, error) {
	row := momentModel{
		ID:        newID("m"),
		CoupleID:  defaultCouple(moment.CoupleID),
		AuthorID:  moment.AuthorID,
		Author:    moment.Author,
		Avatar:    moment.Avatar,
		TimeLabel: moment.TimeLabel,
		Tag:       moment.Tag,
		Content:   moment.Content,
		Image:     moment.Image,
		Liked:     moment.Liked,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.Moment{}, err
	}
	return toMoment(row), nil
}

func (s *MySQLStore) DeleteMoment(id string) error {
	result := s.db.Delete(&momentModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) UpdateMomentLiked(id string, liked bool) (domain.Moment, error) {
	result := s.db.Model(&momentModel{}).Where("id = ?", id).Update("liked", liked)
	if result.Error != nil {
		return domain.Moment{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Moment{}, err
	}
	return s.moment(id)
}

func (s *MySQLStore) Tasks() ([]domain.Task, error) {
	var rows []taskModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTask(row))
	}
	return items, nil
}

func (s *MySQLStore) AddTask(task domain.Task) (domain.Task, error) {
	row := taskModel{
		ID:        newID("t"),
		CoupleID:  defaultCouple(task.CoupleID),
		Title:     task.Title,
		Owner:     task.Owner,
		Type:      task.Type,
		Tag:       task.Tag,
		Due:       task.Due,
		Reward:    task.Reward,
		Status:    domain.TaskTodo,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.Task{}, err
	}
	return toTask(row), nil
}

func (s *MySQLStore) DeleteTask(id string) error {
	result := s.db.Delete(&taskModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) UpdateTaskStatus(id string, status domain.TaskStatus) (domain.Task, error) {
	result := s.db.Model(&taskModel{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return domain.Task{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Task{}, err
	}
	return s.task(id)
}

func (s *MySQLStore) ScheduledTasks() ([]domain.ScheduledTask, error) {
	var rows []scheduledTaskModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ScheduledTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, toScheduledTask(row))
	}
	return items, nil
}

func (s *MySQLStore) AddScheduledTask(task domain.ScheduledTask) (domain.ScheduledTask, error) {
	row := scheduledTaskModel{
		ID:        newID("s"),
		CoupleID:  defaultCouple(task.CoupleID),
		Title:     task.Title,
		Cycle:     task.Cycle,
		Assignee:  task.Assignee,
		TimeText:  task.Time,
		NextText:  task.Next,
		Pending:   true,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.ScheduledTask{}, err
	}
	return toScheduledTask(row), nil
}

func (s *MySQLStore) DeleteScheduledTask(id string) error {
	result := s.db.Delete(&scheduledTaskModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) ConfirmScheduledTask(id string) (domain.ScheduledTask, error) {
	result := s.db.Model(&scheduledTaskModel{}).Where("id = ?", id).Update("pending", false)
	if result.Error != nil {
		return domain.ScheduledTask{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.ScheduledTask{}, err
	}
	return s.scheduledTask(id)
}

func (s *MySQLStore) Dishes() ([]domain.Dish, error) {
	var rows []dishModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Dish, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDish(row))
	}
	return items, nil
}

func (s *MySQLStore) AddDish(dish domain.Dish) (domain.Dish, error) {
	row := dishModel{
		ID:        newID("d"),
		CoupleID:  defaultCouple(dish.CoupleID),
		Icon:      dish.Icon,
		Name:      dish.Name,
		Meal:      dish.Meal,
		Enabled:   dish.Enabled,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.Dish{}, err
	}
	return toDish(row), nil
}

func (s *MySQLStore) DeleteDish(id string) error {
	result := s.db.Delete(&dishModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) UpdateDishEnabled(id string, enabled bool) (domain.Dish, error) {
	result := s.db.Model(&dishModel{}).Where("id = ?", id).Update("enabled", enabled)
	if result.Error != nil {
		return domain.Dish{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Dish{}, err
	}
	return s.dish(id)
}

func (s *MySQLStore) Orders() ([]domain.Order, error) {
	var rows []orderModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		item := toOrder(row)
		dishes, err := s.orderDishes(row.ID)
		if err != nil {
			return nil, err
		}
		item.Dishes = dishes
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) AddOrder(order domain.Order) (domain.Order, error) {
	item := orderModel{
		ID:        newID("o"),
		CoupleID:  defaultCouple(order.CoupleID),
		DateText:  order.Date,
		Meal:      order.Meal,
		Picker:    order.Picker,
		CreatedAt: time.Now(),
	}
	if item.DateText == "" {
		item.DateText = formatMonthDay(time.Now())
	}
	if len(order.Dishes) == 0 {
		return domain.Order{}, errors.New("dishes required")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		rows := make([]orderDishModel, 0, len(order.Dishes))
		for i, name := range order.Dishes {
			rows = append(rows, orderDishModel{OrderID: item.ID, DishName: name, SortOrder: i})
		}
		return tx.Create(&rows).Error
	}); err != nil {
		return domain.Order{}, err
	}
	result := toOrder(item)
	result.Dishes = append([]string(nil), order.Dishes...)
	return result, nil
}

func (s *MySQLStore) Goals() ([]domain.Goal, error) {
	var rows []goalModel
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Goal, 0, len(rows))
	for _, row := range rows {
		items = append(items, calculateGoal(toGoal(row)))
	}
	return items, nil
}

func (s *MySQLStore) AddGoal(goal domain.Goal) (domain.Goal, error) {
	row := buildGoalModel(goal)
	if err := s.db.Create(&row).Error; err != nil {
		return domain.Goal{}, err
	}
	return calculateGoal(toGoal(row)), nil
}

func (s *MySQLStore) UpdateGoalValue(id string, currentValue int) (domain.Goal, error) {
	goal, err := s.goal(id)
	if err != nil {
		return domain.Goal{}, err
	}
	goal.CurrentValue = currentValue
	goal = calculateGoal(goal)
	result := s.db.Model(&goalModel{}).Where("id = ?", id).Updates(map[string]any{
		"current_value": goal.CurrentValue,
		"progress":      goal.Progress,
		"remain_days":   goal.RemainDays,
		"status":        goal.Status,
	})
	if result.Error != nil {
		return domain.Goal{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
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
	result := s.db.Model(&goalModel{}).Where("id = ?", id).Updates(map[string]any{
		"status":        goal.Status,
		"current_value": goal.CurrentValue,
		"progress":      goal.Progress,
		"remain_days":   goal.RemainDays,
	})
	if result.Error != nil {
		return domain.Goal{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Goal{}, err
	}
	return goal, nil
}

func (s *MySQLStore) DeleteGoal(id string) error {
	result := s.db.Delete(&goalModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) AdminOverview() (domain.AdminOverview, error) {
	now := time.Now()

	totalUsers, err := s.countModel(&userModel{}, "")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	pairedCouples, err := s.countModel(&coupleModel{}, "user_b_id <> ''")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	pendingCodes, err := s.countModel(&coupleModel{}, "pair_code <> '' AND code_expire_at > ?", now)
	if err != nil {
		return domain.AdminOverview{}, err
	}
	totalMoments, err := s.countModel(&momentModel{}, "")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	openTasks, err := s.countModel(&taskModel{}, "status <> ?", domain.TaskDone)
	if err != nil {
		return domain.AdminOverview{}, err
	}
	activeGoals, err := s.countModel(&goalModel{}, "status = ?", "active")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	totalOrders, err := s.countModel(&orderModel{}, "")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	enabledDishes, err := s.countModel(&dishModel{}, "enabled = ?", true)
	if err != nil {
		return domain.AdminOverview{}, err
	}
	scheduledTasks, err := s.countModel(&scheduledTaskModel{}, "")
	if err != nil {
		return domain.AdminOverview{}, err
	}
	newUsers24h, err := s.countModel(&userModel{}, "created_at >= ?", now.Add(-24*time.Hour))
	if err != nil {
		return domain.AdminOverview{}, err
	}
	newCouples7d, err := s.countModel(&coupleModel{}, "created_at >= ? AND user_b_id <> ''", now.Add(-7*24*time.Hour))
	if err != nil {
		return domain.AdminOverview{}, err
	}

	return domain.AdminOverview{
		TotalUsers:       totalUsers,
		PairedCouples:    pairedCouples,
		PendingPairCodes: pendingCodes,
		TotalMoments:     totalMoments,
		OpenTasks:        openTasks,
		ActiveGoals:      activeGoals,
		TotalOrders:      totalOrders,
		EnabledDishes:    enabledDishes,
		ScheduledTasks:   scheduledTasks,
		NewUsers24h:      newUsers24h,
		NewCouples7d:     newCouples7d,
	}, nil
}

func (s *MySQLStore) AdminRecentUsers(limit int) ([]domain.AdminUserSummary, error) {
	if limit <= 0 {
		limit = 8
	}
	var rows []userModel
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.AdminUserSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.AdminUserSummary{
			ID:        row.ID,
			Nickname:  row.Nickname,
			Avatar:    row.Avatar,
			WxID:      row.WxID,
			CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}

func (s *MySQLStore) AdminRecentCouples(limit int) ([]domain.AdminCoupleSummary, error) {
	if limit <= 0 {
		limit = 8
	}
	var rows []coupleModel
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.AdminCoupleSummary, 0, len(rows))
	for _, row := range rows {
		item := domain.AdminCoupleSummary{
			ID:        row.ID,
			UserAID:   row.UserAID,
			UserBID:   row.UserBID,
			LoveDate:  row.LoveDate,
			CreatedAt: row.CreatedAt,
		}
		if row.PairCode != "" {
			item.PairCode = row.PairCode
		}
		if row.CodeExpireAt != nil {
			item.CodeExpireAt = *row.CodeExpireAt
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) AdminUnpairCouple(coupleID string) (domain.UnpairResult, error) {
	coupleID = strings.TrimSpace(coupleID)
	if coupleID == "" {
		return domain.UnpairResult{}, ErrNotFound
	}
	couple, err := s.coupleByID(coupleID)
	if err != nil {
		return domain.UnpairResult{}, err
	}
	if strings.TrimSpace(couple.UserBID) == "" {
		return domain.UnpairResult{}, ErrNotFound
	}
	if err := s.unpairCouple(couple.ID); err != nil {
		return domain.UnpairResult{}, err
	}
	return domain.UnpairResult{Couple: couple, InitiatorID: "admin"}, nil
}

func (s *MySQLStore) seed(ctx context.Context) error {
	total, err := s.countModel(&userModel{}, "")
	if err != nil || total > 0 {
		return err
	}
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := []userModel{
			{ID: "u1", OpenID: "demo-openid-u1", Nickname: "Xiao Yu", Birthday: "2000-08-01", WxID: "xiaoyu_2024", CreatedAt: now},
			{ID: "u2", OpenID: "demo-openid-u2", Nickname: "A Jie", Birthday: "1999-09-08", WxID: "ajie_2024", CreatedAt: now},
		}
		if err := tx.Create(&users).Error; err != nil {
			return err
		}
		couples := []coupleModel{
			{ID: "c1", UserAID: "u1", UserBID: "u2", LoveDate: "2023-07-28", CreatedAt: now},
		}
		if err := tx.Create(&couples).Error; err != nil {
			return err
		}
		moments := []momentModel{
			{ID: "m1", CoupleID: "c1", AuthorID: "u1", Author: "Xiao Yu", Avatar: "小", TimeLabel: "2 hours ago", Tag: "date", Content: "Went to the park today.", Image: "https://images.unsplash.com/photo-1507525428034-b723cf961d3e?w=900&auto=format&fit=crop", Liked: true, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "m2", CoupleID: "c1", AuthorID: "u2", Author: "A Jie", Avatar: "阿", TimeLabel: "Yesterday 20:30", Tag: "life", Content: "Made dinner together.", Image: "", Liked: false, CreatedAt: now.AddDate(0, 0, -1)},
		}
		if err := tx.Create(&moments).Error; err != nil {
			return err
		}
		tasks := []taskModel{
			{ID: "t1", CoupleID: "c1", Title: "Buy headphones", Owner: "A Jie", Type: "one-time", Due: "07-25", Reward: "hug", Status: domain.TaskTodo, CreatedAt: now},
			{ID: "t2", CoupleID: "c1", Title: "Cook dinner", Owner: "both", Type: "one-time", Due: "07-22", Reward: "hug", Status: domain.TaskTodo, CreatedAt: now},
			{ID: "t3", CoupleID: "c1", Title: "Organize closet", Owner: "Xiao Yu", Type: "monthly", Status: domain.TaskTodo, CreatedAt: now},
			{ID: "t4", CoupleID: "c1", Title: "Run 30 min", Owner: "A Jie", Type: "daily", Reward: "praise", Status: domain.TaskReview, CreatedAt: now},
			{ID: "t5", CoupleID: "c1", Title: "Dental checkup", Owner: "Xiao Yu", Type: "one-time", Status: domain.TaskDone, CreatedAt: now},
		}
		if err := tx.Create(&tasks).Error; err != nil {
			return err
		}
		schedules := []scheduledTaskModel{
			{ID: "s1", CoupleID: "c1", Title: "Feed cat", Cycle: "every day", Assignee: "rotate", TimeText: "20:00", NextText: "today 20:00", Pending: true, CreatedAt: now},
			{ID: "s2", CoupleID: "c1", Title: "Water plants", Cycle: "weekly", Assignee: "Xiao Yu", TimeText: "19:30", NextText: "Wed 19:30", Pending: false, CreatedAt: now},
		}
		if err := tx.Create(&schedules).Error; err != nil {
			return err
		}
		dishes := []dishModel{
			{ID: "d1", CoupleID: "c1", Icon: "🍳", Name: "Egg fry", Meal: "any", Enabled: true, CreatedAt: now},
			{ID: "d2", CoupleID: "c1", Icon: "🥘", Name: "Claypot rice", Meal: "dinner", Enabled: true, CreatedAt: now},
			{ID: "d3", CoupleID: "c1", Icon: "🥗", Name: "Salad", Meal: "lunch", Enabled: false, CreatedAt: now},
		}
		if err := tx.Create(&dishes).Error; err != nil {
			return err
		}
		orders := []orderModel{
			{ID: "o1", CoupleID: "c1", DateText: "07-18", Meal: "dinner", Picker: "A Jie", CreatedAt: now},
			{ID: "o2", CoupleID: "c1", DateText: "07-18", Meal: "lunch", Picker: "Xiao Yu", CreatedAt: now},
		}
		if err := tx.Create(&orders).Error; err != nil {
			return err
		}
		orderDishes := []orderDishModel{
			{OrderID: "o1", DishName: "Egg fry", SortOrder: 0},
			{OrderID: "o1", DishName: "Claypot rice", SortOrder: 1},
			{OrderID: "o2", DishName: "Salad", SortOrder: 0},
		}
		if err := tx.Create(&orderDishes).Error; err != nil {
			return err
		}
		goals := []goalModel{
			{ID: "g1", CoupleID: "c1", Title: "Travel fund", Period: "month", Progress: 68, RemainDays: 12, Status: "active", CreatedAt: now},
			{ID: "g2", CoupleID: "c1", Title: "Workout 3x/week", Period: "week", Progress: 42, RemainDays: 4, Status: "active", CreatedAt: now},
			{ID: "g3", CoupleID: "c1", Title: "Read one book", Period: "quarter", Progress: 100, RemainDays: 0, Status: "done", CreatedAt: now},
		}
		return tx.Create(&goals).Error
	})
}

func (s *MySQLStore) couple() (domain.Couple, error) {
	return s.coupleByID("c1")
}

func (s *MySQLStore) coupleForUser(userID string) (domain.Couple, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return s.couple()
	}
	var couple coupleModel
	err := s.db.Where("(user_a_id = ? OR user_b_id = ?) AND user_b_id <> ''", userID, userID).Order("created_at DESC").Take(&couple).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Couple{}, ErrNotFound
	}
	if err != nil {
		return domain.Couple{}, err
	}
	return toCouple(couple), nil
}

func (s *MySQLStore) coupleByID(id string) (domain.Couple, error) {
	var couple coupleModel
	err := s.db.Where("id = ?", id).Take(&couple).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Couple{}, ErrNotFound
	}
	if err != nil {
		return domain.Couple{}, err
	}
	return toCouple(couple), nil
}

func (s *MySQLStore) user(id string) (domain.User, error) {
	var user userModel
	err := s.db.Where("id = ?", id).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return toUser(user), nil
}

func (s *MySQLStore) moment(id string) (domain.Moment, error) {
	var item momentModel
	err := s.db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Moment{}, ErrNotFound
	}
	if err != nil {
		return domain.Moment{}, err
	}
	return toMoment(item), nil
}

func (s *MySQLStore) task(id string) (domain.Task, error) {
	var item taskModel
	err := s.db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Task{}, ErrNotFound
	}
	if err != nil {
		return domain.Task{}, err
	}
	return toTask(item), nil
}

func (s *MySQLStore) scheduledTask(id string) (domain.ScheduledTask, error) {
	var item scheduledTaskModel
	err := s.db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ScheduledTask{}, ErrNotFound
	}
	if err != nil {
		return domain.ScheduledTask{}, err
	}
	return toScheduledTask(item), nil
}

func (s *MySQLStore) dish(id string) (domain.Dish, error) {
	var item dishModel
	err := s.db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Dish{}, ErrNotFound
	}
	if err != nil {
		return domain.Dish{}, err
	}
	return toDish(item), nil
}

func (s *MySQLStore) goal(id string) (domain.Goal, error) {
	var item goalModel
	err := s.db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Goal{}, ErrNotFound
	}
	if err != nil {
		return domain.Goal{}, err
	}
	return calculateGoal(toGoal(item)), nil
}

func (s *MySQLStore) orderDishes(orderID string) ([]string, error) {
	var rows []orderDishModel
	if err := s.db.Where("order_id = ?", orderID).Order("sort_order").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]string, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.DishName)
	}
	return items, nil
}

func (s *MySQLStore) countModel(model any, where string, args ...any) (int, error) {
	return s.countModelWithDB(s.db, model, where, args...)
}

func (s *MySQLStore) countModelWithDB(tx *gorm.DB, model any, where string, args ...any) (int, error) {
	var total int64
	query := tx.Model(model)
	if where != "" {
		query = query.Where(where, args...)
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return int(total), nil
}

func (s *MySQLStore) countCouples(where string, args ...any) (int, error) {
	return s.countModel(&coupleModel{}, where, args...)
}

func (s *MySQLStore) averageGoalProgress(coupleID string) (int, error) {
	var avg sql.NullFloat64
	if err := s.db.Model(&goalModel{}).Select("AVG(progress)").Where("couple_id = ? AND status = ?", coupleID, "active").Scan(&avg).Error; err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return int(math.Round(avg.Float64)), nil
}

func (s *MySQLStore) uniquePairCode(tx *gorm.DB) (string, error) {
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("%06d", rand.Intn(900000)+100000)
		total, err := s.countModelWithDB(tx, &coupleModel{}, "pair_code = ? AND code_expire_at > ?", code, time.Now())
		if err != nil {
			return "", err
		}
		if total == 0 {
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique pair code")
}

func buildGoalModel(goal domain.Goal) goalModel {
	now := time.Now()
	row := goalModel{
		ID:           firstNonEmpty(strings.TrimSpace(goal.ID), newID("g")),
		CoupleID:     defaultCouple(goal.CoupleID),
		Title:        goal.Title,
		Period:       goal.Period,
		TargetValue:  goal.TargetValue,
		CurrentValue: goal.CurrentValue,
		StartDate:    goal.StartDate,
		TargetDate:   goal.TargetDate,
		Progress:     goal.Progress,
		RemainDays:   goal.RemainDays,
		Status:       goal.Status,
		CreatedAt:    now,
	}
	if row.TargetValue <= 0 {
		row.TargetValue = 100
	}
	if strings.TrimSpace(row.StartDate) == "" {
		row.StartDate = now.Format("2006-01-02")
	}
	if strings.TrimSpace(row.TargetDate) == "" {
		row.TargetDate = now.AddDate(0, 0, 30).Format("2006-01-02")
	}
	goal = calculateGoal(toGoal(row))
	row = fromGoal(goal)
	row.CreatedAt = now
	return row
}

func requireAffected(rows int64) error {
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func toUser(row userModel) domain.User {
	return domain.User{
		ID:        row.ID,
		OpenID:    row.OpenID,
		Nickname:  row.Nickname,
		Avatar:    row.Avatar,
		Birthday:  row.Birthday,
		WxID:      row.WxID,
		CreatedAt: row.CreatedAt,
	}
}

func toCouple(row coupleModel) domain.Couple {
	couple := domain.Couple{
		ID:        row.ID,
		UserAID:   row.UserAID,
		UserBID:   row.UserBID,
		LoveDate:  row.LoveDate,
		CreatedAt: row.CreatedAt,
	}
	if row.PairCode != "" {
		couple.PairCode = row.PairCode
	}
	if row.CodeExpireAt != nil {
		couple.CodeExpireAt = *row.CodeExpireAt
	}
	return couple
}

func toMoment(row momentModel) domain.Moment {
	return domain.Moment{
		ID:        row.ID,
		CoupleID:  row.CoupleID,
		AuthorID:  row.AuthorID,
		Author:    row.Author,
		Avatar:    row.Avatar,
		TimeLabel: row.TimeLabel,
		Tag:       row.Tag,
		Content:   row.Content,
		Image:     row.Image,
		Liked:     row.Liked,
		CreatedAt: row.CreatedAt,
	}
}

func toTask(row taskModel) domain.Task {
	return domain.Task{
		ID:       row.ID,
		CoupleID: row.CoupleID,
		Title:    row.Title,
		Owner:    row.Owner,
		Type:     row.Type,
		Tag:      row.Tag,
		Due:      row.Due,
		Reward:   row.Reward,
		Status:   row.Status,
	}
}

func toScheduledTask(row scheduledTaskModel) domain.ScheduledTask {
	return domain.ScheduledTask{
		ID:       row.ID,
		CoupleID: row.CoupleID,
		Title:    row.Title,
		Cycle:    row.Cycle,
		Assignee: row.Assignee,
		Time:     row.TimeText,
		Next:     row.NextText,
		Pending:  row.Pending,
	}
}

func toDish(row dishModel) domain.Dish {
	return domain.Dish{
		ID:       row.ID,
		CoupleID: row.CoupleID,
		Icon:     row.Icon,
		Name:     row.Name,
		Meal:     row.Meal,
		Enabled:  row.Enabled,
	}
}

func toOrder(row orderModel) domain.Order {
	return domain.Order{
		ID:       row.ID,
		CoupleID: row.CoupleID,
		Date:     row.DateText,
		Meal:     row.Meal,
		Picker:   row.Picker,
	}
}

func toGoal(row goalModel) domain.Goal {
	return domain.Goal{
		ID:           row.ID,
		CoupleID:     row.CoupleID,
		Title:        row.Title,
		Period:       row.Period,
		TargetValue:  row.TargetValue,
		CurrentValue: row.CurrentValue,
		StartDate:    row.StartDate,
		TargetDate:   row.TargetDate,
		Progress:     row.Progress,
		RemainDays:   row.RemainDays,
		Status:       row.Status,
	}
}

func toNotice(row noticeModel) domain.Notice {
	return domain.Notice{
		ID:          row.ID,
		CoupleID:    row.CoupleID,
		RecipientID: row.RecipientID,
		InitiatorID: row.InitiatorID,
		Category:    row.Category,
		Title:       row.Title,
		Content:     row.Content,
		Target:      row.Target,
		ReadAt:      row.ReadAt,
		CreatedAt:   row.CreatedAt,
	}
}

func fromGoal(goal domain.Goal) goalModel {
	return goalModel{
		ID:           goal.ID,
		CoupleID:     goal.CoupleID,
		Title:        goal.Title,
		Period:       goal.Period,
		TargetValue:  goal.TargetValue,
		CurrentValue: goal.CurrentValue,
		StartDate:    goal.StartDate,
		TargetDate:   goal.TargetDate,
		Progress:     goal.Progress,
		RemainDays:   goal.RemainDays,
		Status:       goal.Status,
	}
}

func lite(user domain.User, avatarText string) domain.UserLite {
	return domain.UserLite{
		ID:         user.ID,
		Name:       user.Nickname,
		Avatar:     user.Avatar,
		AvatarText: avatarText,
		Birthday:   user.Birthday,
		WxID:       user.WxID,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

type userModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:32"`
	OpenID    string    `gorm:"column:openid;size:128;not null;uniqueIndex"`
	Nickname  string    `gorm:"column:nickname;size:64;not null"`
	Avatar    string    `gorm:"column:avatar;size:255;not null;default:''"`
	Birthday  string    `gorm:"column:birthday;size:32;not null;default:''"`
	WxID      string    `gorm:"column:wxid;size:64;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (userModel) TableName() string { return "users" }

type coupleModel struct {
	ID           string     `gorm:"column:id;primaryKey;size:32"`
	UserAID      string     `gorm:"column:user_a_id;size:32;not null"`
	UserBID      string     `gorm:"column:user_b_id;size:32;not null"`
	LoveDate     string     `gorm:"column:love_date;size:32;not null"`
	PairCode     string     `gorm:"column:pair_code;size:16;not null;default:''"`
	CodeExpireAt *time.Time `gorm:"column:code_expire_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null"`
	Version      int64      `gorm:"column:version;not null;default:0"`
}

func (coupleModel) TableName() string { return "couples" }

type momentModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:32"`
	CoupleID  string    `gorm:"column:couple_id;size:32;not null"`
	AuthorID  string    `gorm:"column:author_id;size:32;not null;default:''"`
	Author    string    `gorm:"column:author;size:64;not null"`
	Avatar    string    `gorm:"column:avatar;size:32;not null"`
	TimeLabel string    `gorm:"column:time_label;size:64;not null"`
	Tag       string    `gorm:"column:tag;size:64;not null;default:''"`
	Content   string    `gorm:"column:content;type:text;not null"`
	Image     string    `gorm:"column:image;size:500;not null;default:''"`
	Liked     bool      `gorm:"column:liked;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (momentModel) TableName() string { return "moments" }

type taskModel struct {
	ID        string            `gorm:"column:id;primaryKey;size:32"`
	CoupleID  string            `gorm:"column:couple_id;size:32;not null"`
	Title     string            `gorm:"column:title;size:120;not null"`
	Owner     string            `gorm:"column:owner;size:64;not null"`
	Type      string            `gorm:"column:type;size:32;not null"`
	Tag       string            `gorm:"column:tag;size:64;not null;default:''"`
	Due       string            `gorm:"column:due;size:32;not null;default:''"`
	Reward    string            `gorm:"column:reward;size:64;not null;default:''"`
	Status    domain.TaskStatus `gorm:"column:status;size:16;not null"`
	CreatedAt time.Time         `gorm:"column:created_at;not null"`
}

func (taskModel) TableName() string { return "tasks" }

type scheduledTaskModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:32"`
	CoupleID  string    `gorm:"column:couple_id;size:32;not null"`
	Title     string    `gorm:"column:title;size:120;not null"`
	Cycle     string    `gorm:"column:cycle;size:64;not null"`
	Assignee  string    `gorm:"column:assignee;size:64;not null"`
	TimeText  string    `gorm:"column:time_text;size:32;not null"`
	NextText  string    `gorm:"column:next_text;size:64;not null"`
	Pending   bool      `gorm:"column:pending;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (scheduledTaskModel) TableName() string { return "scheduled_tasks" }

type dishModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:32"`
	CoupleID  string    `gorm:"column:couple_id;size:32;not null"`
	Icon      string    `gorm:"column:icon;size:32;not null"`
	Name      string    `gorm:"column:name;size:80;not null"`
	Meal      string    `gorm:"column:meal;size:32;not null"`
	Enabled   bool      `gorm:"column:enabled;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (dishModel) TableName() string { return "dishes" }

type orderModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:32"`
	CoupleID  string    `gorm:"column:couple_id;size:32;not null"`
	DateText  string    `gorm:"column:date_text;size:32;not null"`
	Meal      string    `gorm:"column:meal;size:32;not null"`
	Picker    string    `gorm:"column:picker;size:64;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (orderModel) TableName() string { return "orders" }

type orderDishModel struct {
	OrderID   string `gorm:"column:order_id;primaryKey;size:32"`
	DishName  string `gorm:"column:dish_name;primaryKey;size:80"`
	SortOrder int    `gorm:"column:sort_order;primaryKey"`
}

func (orderDishModel) TableName() string { return "order_dishes" }

type goalModel struct {
	ID           string    `gorm:"column:id;primaryKey;size:32"`
	CoupleID     string    `gorm:"column:couple_id;size:32;not null"`
	Title        string    `gorm:"column:title;size:120;not null"`
	Period       string    `gorm:"column:period;size:32;not null"`
	TargetValue  int       `gorm:"column:target_value;not null;default:100"`
	CurrentValue int       `gorm:"column:current_value;not null;default:0"`
	StartDate    string    `gorm:"column:start_date;size:32;not null;default:''"`
	TargetDate   string    `gorm:"column:target_date;size:32;not null;default:''"`
	Progress     int       `gorm:"column:progress;not null;default:0"`
	RemainDays   int       `gorm:"column:remain_days;not null;default:0"`
	Status       string    `gorm:"column:status;size:16;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (goalModel) TableName() string { return "goals" }

type noticeModel struct {
	ID          string     `gorm:"column:id;primaryKey;size:32"`
	CoupleID    string     `gorm:"column:couple_id;size:32;not null;index"`
	RecipientID string     `gorm:"column:recipient_id;size:32;not null;index"`
	InitiatorID string     `gorm:"column:initiator_id;size:32;not null;default:''"`
	Category    string     `gorm:"column:category;size:32;not null;index"`
	Title       string     `gorm:"column:title;size:120;not null"`
	Content     string     `gorm:"column:content;size:500;not null;default:''"`
	Target      string     `gorm:"column:target;size:120;not null;default:''"`
	ReadAt      *time.Time `gorm:"column:read_at;index"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;index"`
}

func (noticeModel) TableName() string { return "notices" }
