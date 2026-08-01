package store

import (
	"strings"
	"time"

	"couple-mini/backend/internal/domain"

	"gorm.io/gorm"
)

func (s *MySQLStore) SyncState(userID string) (domain.SyncState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.SyncState{}, ErrUnauthorized
	}
	couple, err := s.coupleForUser(userID)
	if err != nil {
		if err == ErrNotFound {
			return domain.SyncState{Paired: false, Version: 0}, nil
		}
		return domain.SyncState{}, err
	}
	var row coupleModel
	if err := s.db.Where("id = ?", couple.ID).Take(&row).Error; err != nil {
		return domain.SyncState{}, err
	}
	return domain.SyncState{
		Paired:    row.UserBID != "",
		CoupleID:  row.ID,
		Version:   row.Version,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *MySQLStore) CoupleForUser(userID string) (domain.Couple, error) {
	return s.coupleForUser(userID)
}

func (s *MySQLStore) UserByID(userID string) (domain.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.User{}, ErrUnauthorized
	}
	return s.user(userID)
}

func (s *MySQLStore) AddNotice(notice domain.Notice) (domain.Notice, error) {
	now := time.Now()
	row := noticeModel{
		ID:          firstNonEmpty(strings.TrimSpace(notice.ID), newID("n")),
		CoupleID:    strings.TrimSpace(notice.CoupleID),
		RecipientID: strings.TrimSpace(notice.RecipientID),
		InitiatorID: strings.TrimSpace(notice.InitiatorID),
		Category:    strings.TrimSpace(notice.Category),
		Title:       strings.TrimSpace(notice.Title),
		Content:     strings.TrimSpace(notice.Content),
		Target:      strings.TrimSpace(notice.Target),
		ReadAt:      notice.ReadAt,
		CreatedAt:   now,
	}
	if row.CoupleID == "" || row.RecipientID == "" || row.Category == "" || row.Title == "" {
		return domain.Notice{}, ErrNotFound
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.Notice{}, err
	}
	return toNotice(row), nil
}

func (s *MySQLStore) UnreadNoticesForUser(userID string) ([]domain.Notice, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrUnauthorized
	}
	var rows []noticeModel
	if err := s.db.Where("recipient_id = ? AND read_at IS NULL", userID).Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Notice, 0, len(rows))
	for _, row := range rows {
		items = append(items, toNotice(row))
	}
	return items, nil
}

func (s *MySQLStore) MarkNoticesReadForUser(userID string, categories []string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrUnauthorized
	}
	now := time.Now()
	tx := s.db.Model(&noticeModel{}).Where("recipient_id = ? AND read_at IS NULL", userID)
	if len(categories) > 0 {
		tx = tx.Where("category IN ?", categories)
	}
	return tx.Update("read_at", &now).Error
}

func (s *MySQLStore) UpdateLoveDateForUser(userID, loveDate string) (domain.Couple, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Couple{}, err
	}
	if _, err := parseISODate(loveDate); err != nil {
		return domain.Couple{}, err
	}
	now := time.Now()
	result := s.db.Model(&coupleModel{}).Where("id = ?", couple.ID).Updates(map[string]any{
		"love_date":  loveDate,
		"updated_at": now,
		"version":    now.UnixNano(),
	})
	if result.Error != nil {
		return domain.Couple{}, result.Error
	}
	if err := requireAffected(result.RowsAffected); err != nil {
		return domain.Couple{}, err
	}
	return s.coupleByID(couple.ID)
}

func (s *MySQLStore) UpdateUserProfileForUser(currentUserID string, user domain.User) (domain.User, error) {
	if strings.TrimSpace(currentUserID) == "" || strings.TrimSpace(currentUserID) != strings.TrimSpace(user.ID) {
		return domain.User{}, ErrUnauthorized
	}
	var updated domain.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if birthday, ok := parseBirthday(user.Birthday); ok {
			user.Birthday = formatMonthDay(birthday)
		}
		result := tx.Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]any{
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"birthday": user.Birthday,
			"wxid":     user.WxID,
		})
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		if couple, err := s.coupleForUser(currentUserID); err == nil {
			if err := s.touchCouple(tx, couple.ID); err != nil {
				return err
			}
		}
		var row userModel
		if err := tx.Where("id = ?", user.ID).Take(&row).Error; err != nil {
			return err
		}
		updated = toUser(row)
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}
	return updated, nil
}

func (s *MySQLStore) UnpairForUser(userID string) (domain.UnpairResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.UnpairResult{}, ErrUnauthorized
	}
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.UnpairResult{}, err
	}
	if err := s.unpairCouple(couple.ID); err != nil {
		return domain.UnpairResult{}, err
	}
	return domain.UnpairResult{Couple: couple, InitiatorID: userID}, nil
}

func (s *MySQLStore) DashboardForUser(userID string) (domain.DashboardPayload, error) {
	payload, err := s.Dashboard(userID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	meID := couple.UserAID
	partnerID := couple.UserBID
	if strings.TrimSpace(userID) == strings.TrimSpace(couple.UserBID) {
		meID, partnerID = couple.UserBID, couple.UserAID
	}
	me, err := s.user(meID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	partner, err := s.user(partnerID)
	if err != nil {
		return domain.DashboardPayload{}, err
	}
	payload.Users = map[string]domain.UserLite{
		"me":      liteUser(me),
		"partner": liteUser(partner),
	}
	return payload, nil
}

func (s *MySQLStore) MomentsForUser(userID string) ([]domain.Moment, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []momentModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Moment, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMoment(row))
	}
	return items, nil
}

func (s *MySQLStore) AddMomentForUser(userID string, moment domain.Moment) (domain.Moment, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Moment{}, err
	}
	author, err := s.user(userID)
	if err != nil {
		return domain.Moment{}, err
	}
	row := momentModel{
		ID:        newID("m"),
		CoupleID:  couple.ID,
		AuthorID:  userID,
		Author:    firstNonEmpty(strings.TrimSpace(moment.Author), strings.TrimSpace(author.Nickname)),
		Avatar:    firstNonEmpty(strings.TrimSpace(moment.Avatar), strings.TrimSpace(author.Avatar), avatarText(author.Nickname)),
		TimeLabel: firstNonEmpty(strings.TrimSpace(moment.TimeLabel), "just now"),
		Tag:       moment.Tag,
		Content:   moment.Content,
		Image:     moment.Image,
		Liked:     moment.Liked,
		CreatedAt: time.Now(),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Moment{}, err
	}
	return toMoment(row), nil
}

func (s *MySQLStore) DeleteMomentForUser(userID, id string) error {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&momentModel{}, "id = ? AND couple_id = ?", id, couple.ID)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	})
}

func (s *MySQLStore) UpdateMomentLikedForUser(userID, id string, liked bool) (domain.Moment, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Moment{}, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&momentModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Update("liked", liked)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Moment{}, err
	}
	return s.moment(id)
}

func (s *MySQLStore) TasksForUser(userID string) ([]domain.Task, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []taskModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTask(row))
	}
	return items, nil
}

func (s *MySQLStore) AddTaskForUser(userID string, task domain.Task) (domain.Task, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Task{}, err
	}
	row := taskModel{
		ID:        newID("t"),
		CoupleID:  couple.ID,
		Title:     task.Title,
		Owner:     task.Owner,
		Type:      task.Type,
		Tag:       task.Tag,
		Due:       task.Due,
		Reward:    task.Reward,
		Status:    domain.TaskTodo,
		CreatedAt: time.Now(),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Task{}, err
	}
	return toTask(row), nil
}

func (s *MySQLStore) DeleteTaskForUser(userID, id string) error {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&taskModel{}, "id = ? AND couple_id = ?", id, couple.ID)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	})
}

func (s *MySQLStore) UpdateTaskStatusForUser(userID, id string, status domain.TaskStatus) (domain.Task, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&taskModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Update("status", status)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Task{}, err
	}
	return s.task(id)
}

func (s *MySQLStore) ScheduledTasksForUser(userID string) ([]domain.ScheduledTask, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []scheduledTaskModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ScheduledTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, toScheduledTask(row))
	}
	return items, nil
}

func (s *MySQLStore) AddScheduledTaskForUser(userID string, task domain.ScheduledTask) (domain.ScheduledTask, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.ScheduledTask{}, err
	}
	row := scheduledTaskModel{
		ID:        newID("s"),
		CoupleID:  couple.ID,
		Title:     task.Title,
		Cycle:     task.Cycle,
		Assignee:  task.Assignee,
		TimeText:  task.Time,
		NextText:  task.Next,
		Pending:   true,
		CreatedAt: time.Now(),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.ScheduledTask{}, err
	}
	return toScheduledTask(row), nil
}

func (s *MySQLStore) DeleteScheduledTaskForUser(userID, id string) error {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&scheduledTaskModel{}, "id = ? AND couple_id = ?", id, couple.ID)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	})
}

func (s *MySQLStore) ConfirmScheduledTaskForUser(userID, id string) (domain.ScheduledTask, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.ScheduledTask{}, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&scheduledTaskModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Update("pending", false)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.ScheduledTask{}, err
	}
	return s.scheduledTask(id)
}

func (s *MySQLStore) DishesForUser(userID string) ([]domain.Dish, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []dishModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Dish, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDish(row))
	}
	return items, nil
}

func (s *MySQLStore) AddDishForUser(userID string, dish domain.Dish) (domain.Dish, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Dish{}, err
	}
	row := dishModel{
		ID:        newID("d"),
		CoupleID:  couple.ID,
		Icon:      dish.Icon,
		Name:      dish.Name,
		Meal:      dish.Meal,
		Enabled:   dish.Enabled,
		CreatedAt: time.Now(),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Dish{}, err
	}
	return toDish(row), nil
}

func (s *MySQLStore) DeleteDishForUser(userID, id string) error {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&dishModel{}, "id = ? AND couple_id = ?", id, couple.ID)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	})
}

func (s *MySQLStore) UpdateDishEnabledForUser(userID, id string, enabled bool) (domain.Dish, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Dish{}, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&dishModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Update("enabled", enabled)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Dish{}, err
	}
	return s.dish(id)
}

func (s *MySQLStore) OrdersForUser(userID string) ([]domain.Order, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []orderModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
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

func (s *MySQLStore) AddOrderForUser(userID string, order domain.Order) (domain.Order, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Order{}, err
	}
	item := orderModel{
		ID:        newID("o"),
		CoupleID:  couple.ID,
		DateText:  order.Date,
		Meal:      order.Meal,
		Picker:    order.Picker,
		CreatedAt: time.Now(),
	}
	if item.DateText == "" {
		item.DateText = formatMonthDay(time.Now())
	}
	if len(order.Dishes) == 0 {
		return domain.Order{}, ErrNotFound
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		rows := make([]orderDishModel, 0, len(order.Dishes))
		for i, name := range order.Dishes {
			rows = append(rows, orderDishModel{OrderID: item.ID, DishName: name, SortOrder: i})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Order{}, err
	}
	result := toOrder(item)
	result.Dishes = append([]string(nil), order.Dishes...)
	return result, nil
}

func (s *MySQLStore) GoalsForUser(userID string) ([]domain.Goal, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return nil, err
	}
	var rows []goalModel
	if err := s.db.Where("couple_id = ?", couple.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Goal, 0, len(rows))
	for _, row := range rows {
		items = append(items, calculateGoal(toGoal(row)))
	}
	return items, nil
}

func (s *MySQLStore) AddGoalForUser(userID string, goal domain.Goal) (domain.Goal, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Goal{}, err
	}
	goal.CoupleID = couple.ID
	row := buildGoalModel(goal)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	}); err != nil {
		return domain.Goal{}, err
	}
	return calculateGoal(toGoal(row)), nil
}

func (s *MySQLStore) UpdateGoalValueForUser(userID, id string, currentValue int) (domain.Goal, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Goal{}, err
	}
	var updated domain.Goal
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var row goalModel
		err := tx.Where("id = ? AND couple_id = ?", id, couple.ID).Take(&row).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNotFound
			}
			return err
		}
		goal := calculateGoal(toGoal(row))
		goal.CurrentValue = currentValue
		goal = calculateGoal(goal)
		result := tx.Model(&goalModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Updates(map[string]any{
			"current_value": goal.CurrentValue,
			"progress":      goal.Progress,
			"remain_days":   goal.RemainDays,
			"status":        goal.Status,
		})
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		if err := s.touchCouple(tx, couple.ID); err != nil {
			return err
		}
		updated = goal
		return nil
	}); err != nil {
		return domain.Goal{}, err
	}
	return updated, nil
}

func (s *MySQLStore) UpdateGoalStatusForUser(userID, id, status string) (domain.Goal, error) {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return domain.Goal{}, err
	}
	var updated domain.Goal
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var row goalModel
		err := tx.Where("id = ? AND couple_id = ?", id, couple.ID).Take(&row).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNotFound
			}
			return err
		}
		goal := calculateGoal(toGoal(row))
		goal.Status = status
		if status == "done" {
			goal.CurrentValue = goal.TargetValue
		}
		goal = calculateGoal(goal)
		result := tx.Model(&goalModel{}).Where("id = ? AND couple_id = ?", id, couple.ID).Updates(map[string]any{
			"status":        goal.Status,
			"current_value": goal.CurrentValue,
			"progress":      goal.Progress,
			"remain_days":   goal.RemainDays,
		})
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		if err := s.touchCouple(tx, couple.ID); err != nil {
			return err
		}
		updated = goal
		return nil
	}); err != nil {
		return domain.Goal{}, err
	}
	return updated, nil
}

func (s *MySQLStore) DeleteGoalForUser(userID, id string) error {
	couple, err := s.coupleForUser(userID)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&goalModel{}, "id = ? AND couple_id = ?", id, couple.ID)
		if result.Error != nil {
			return result.Error
		}
		if err := requireAffected(result.RowsAffected); err != nil {
			return err
		}
		return s.touchCouple(tx, couple.ID)
	})
}

func (s *MySQLStore) touchCouple(tx *gorm.DB, coupleID string) error {
	now := time.Now()
	return tx.Model(&coupleModel{}).Where("id = ?", coupleID).Updates(map[string]any{
		"updated_at": now,
		"version":    now.UnixNano(),
	}).Error
}

func (s *MySQLStore) unpairCouple(coupleID string) error {
	now := time.Now()
	result := s.db.Model(&coupleModel{}).
		Where("id = ? AND user_b_id <> ''", strings.TrimSpace(coupleID)).
		Updates(map[string]any{
			"user_b_id":      "",
			"pair_code":      "",
			"code_expire_at": nil,
			"updated_at":     now,
			"version":        now.UnixNano(),
		})
	if result.Error != nil {
		return result.Error
	}
	return requireAffected(result.RowsAffected)
}

func (s *MySQLStore) belongsToCouple(model any, id, coupleID string) bool {
	var count int64
	if err := s.db.Model(model).Where("id = ? AND couple_id = ?", id, coupleID).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func liteUser(user domain.User) domain.UserLite {
	return domain.UserLite{
		ID:         user.ID,
		Name:       user.Nickname,
		Avatar:     user.Avatar,
		AvatarText: avatarText(user.Nickname),
		Birthday:   user.Birthday,
		WxID:       user.WxID,
	}
}

func avatarText(name string) string {
	text := strings.TrimSpace(name)
	if text == "" {
		return "?"
	}
	runes := []rune(text)
	return string(runes[0])
}
