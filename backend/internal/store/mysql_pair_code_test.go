package store

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGeneratePairCodeReusesActiveCodeWithinTwentyMinutes(t *testing.T) {
	db := openPairCodeTestDB(t)
	now := time.Now().Add(-5 * time.Minute).Round(time.Second)
	expireAt := now.Add(20 * time.Minute)
	seedPendingCouple(t, db, coupleModel{
		ID:           "c-pending",
		UserAID:      "u1",
		UserBID:      "",
		LoveDate:     "2026-07-23",
		PairCode:     "123456",
		CodeExpireAt: &expireAt,
		CreatedAt:    now,
	})

	store := NewMySQLStore(db)

	got, err := store.GeneratePairCode("u1")
	if err != nil {
		t.Fatalf("GeneratePairCode returned error: %v", err)
	}

	if got.PairCode != "123456" {
		t.Fatalf("expected existing pair code to be reused, got %q", got.PairCode)
	}
	if !got.CodeExpireAt.Equal(expireAt) {
		t.Fatalf("expected expire time %v, got %v", expireAt, got.CodeExpireAt)
	}
}

func TestGeneratePairCodeRefreshesExpiredCodeAfterTwentyMinutes(t *testing.T) {
	db := openPairCodeTestDB(t)
	now := time.Now().Add(-30 * time.Minute).Round(time.Second)
	expireAt := now.Add(20 * time.Minute)
	seedPendingCouple(t, db, coupleModel{
		ID:           "c-expired",
		UserAID:      "u1",
		UserBID:      "",
		LoveDate:     "2026-07-23",
		PairCode:     "123456",
		CodeExpireAt: &expireAt,
		CreatedAt:    now,
	})

	store := NewMySQLStore(db)

	got, err := store.GeneratePairCode("u1")
	if err != nil {
		t.Fatalf("GeneratePairCode returned error: %v", err)
	}

	if got.PairCode == "" {
		t.Fatal("expected regenerated pair code to be non-empty")
	}
	if !got.CodeExpireAt.After(expireAt) {
		t.Fatalf("expected expire time to be refreshed beyond %v, got %v", expireAt, got.CodeExpireAt)
	}
	remaining := time.Until(got.CodeExpireAt)
	if remaining < 19*time.Minute || remaining > 20*time.Minute+5*time.Second {
		t.Fatalf("expected new code to expire in about 20 minutes, remaining=%v", remaining)
	}
}

func openPairCodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&coupleModel{}); err != nil {
		t.Fatalf("migrate couple model: %v", err)
	}
	return db
}

func seedPendingCouple(t *testing.T, db *gorm.DB, couple coupleModel) {
	t.Helper()
	if err := db.Create(&couple).Error; err != nil {
		t.Fatalf("seed pending couple: %v", err)
	}
}
