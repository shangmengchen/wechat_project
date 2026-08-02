package store

import (
	"fmt"
	"testing"
	"time"

	"couple-mini/backend/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncStateAndTaskMutationBumpsCoupleVersion(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)

	before, err := store.SyncState("u1")
	if err != nil {
		t.Fatalf("SyncState before mutation: %v", err)
	}
	if !before.Paired {
		t.Fatalf("expected paired state, got %+v", before)
	}

	_, err = store.AddTaskForUser("u1", domainTaskFixture("Buy fruit"))
	if err != nil {
		t.Fatalf("AddTaskForUser: %v", err)
	}

	after, err := store.SyncState("u2")
	if err != nil {
		t.Fatalf("SyncState after mutation: %v", err)
	}
	if after.Version <= before.Version {
		t.Fatalf("expected version to increase, before=%d after=%d", before.Version, after.Version)
	}
}

func TestUpdateUserProfileForUserRejectsOtherUser(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)

	_, err := store.UpdateUserProfileForUser("u2", domainUserFixture("u1", "Changed"))
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLoginDoesNotRebindExistingUserByClientUserID(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)
	user, err := store.Login("u1", "attacker-openid", "Attacker", "X")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID == "u1" {
		t.Fatalf("expected new server-owned user, got existing user: %+v", user)
	}
	if user.OpenID != "attacker-openid" {
		t.Fatalf("expected attacker openid on a new user, got %+v", user)
	}

	protected, err := store.user("u1")
	if err != nil {
		t.Fatalf("fetch protected user: %v", err)
	}
	if protected.OpenID != "openid-u1" || protected.Nickname != "User One" {
		t.Fatalf("existing user was modified: %+v", protected)
	}
}

func TestLoginUsesExistingUserOnlyWhenOpenIDMatches(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)
	user, err := store.Login("u1", "openid-u1", "Renamed", "Z")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID != "u1" || user.OpenID != "openid-u1" || user.Nickname != "Renamed" || user.Avatar != "Z" {
		t.Fatalf("expected matched user to update profile only, got %+v", user)
	}
}

func TestLoginGeneratesServerUserIDForNewOpenID(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)
	user, err := store.Login("client-chosen-id", "new-openid", "New User", "N")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID == "client-chosen-id" {
		t.Fatalf("expected server-generated user ID, got %+v", user)
	}
	if user.OpenID != "new-openid" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestUnpairForUserClearsPairForBothUsers(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)
	result, err := store.UnpairForUser("u2")
	if err != nil {
		t.Fatalf("UnpairForUser: %v", err)
	}
	if result.Couple.UserAID != "u1" || result.Couple.UserBID != "u2" || result.InitiatorID != "u2" {
		t.Fatalf("unexpected unpair result: %+v", result)
	}

	for _, userID := range []string{"u1", "u2"} {
		state, err := store.SyncState(userID)
		if err != nil {
			t.Fatalf("SyncState(%s): %v", userID, err)
		}
		if state.Paired {
			t.Fatalf("expected %s to be unpaired, got %+v", userID, state)
		}
	}
}

func TestAdminUnpairCoupleClearsPair(t *testing.T) {
	db := openScopedTestDB(t)
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.Local)
	seedScopedFixture(t, db, now)

	store := NewMySQLStore(db)
	result, err := store.AdminUnpairCouple("c1")
	if err != nil {
		t.Fatalf("AdminUnpairCouple: %v", err)
	}
	if result.InitiatorID != "admin" {
		t.Fatalf("expected admin initiator, got %+v", result)
	}

	state, err := store.SyncState("u1")
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if state.Paired {
		t.Fatalf("expected unpaired state, got %+v", state)
	}
}

func openScopedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&userModel{},
		&coupleModel{},
		&taskModel{},
	); err != nil {
		t.Fatalf("auto migrate scoped test db: %v", err)
	}
	return db
}

func seedScopedFixture(t *testing.T, db *gorm.DB, now time.Time) {
	t.Helper()
	users := []userModel{
		{ID: "u1", OpenID: "openid-u1", Nickname: "User One", Avatar: "A", CreatedAt: now},
		{ID: "u2", OpenID: "openid-u2", Nickname: "User Two", Avatar: "B", CreatedAt: now},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	couple := coupleModel{
		ID:        "c1",
		UserAID:   "u1",
		UserBID:   "u2",
		LoveDate:  "2026-07-25",
		CreatedAt: now,
		UpdatedAt: now,
		Version:   now.UnixNano(),
	}
	if err := db.Create(&couple).Error; err != nil {
		t.Fatalf("seed couple: %v", err)
	}
}

func domainTaskFixture(title string) domain.Task {
	return domain.Task{
		Title:  title,
		Owner:  "both",
		Type:   "one-time",
		Tag:    "life",
		Status: domain.TaskTodo,
	}
}

func domainUserFixture(id, nickname string) domain.User {
	return domain.User{
		ID:       id,
		Nickname: nickname,
		Avatar:   "X",
	}
}
