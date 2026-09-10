package repository

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var quickAccessRepoTestCounter uint64

func TestQuickAccessRepositoryUniquenessOrderingAndScope(t *testing.T) {
	dsn := fmt.Sprintf("file:quick_access_repo_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&quickAccessRepoTestCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.QuickAccessPin{}); err != nil {
		t.Fatal(err)
	}
	repo := NewQuickAccessRepository(db)
	tied := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	first := models.QuickAccessPin{UserID: 1, TargetType: models.QuickAccessTargetCoin, TargetID: 1, PinnedAt: tied}
	if created, err := repo.CreateOrGet(&first); err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	duplicate := models.QuickAccessPin{UserID: 1, TargetType: models.QuickAccessTargetCoin, TargetID: 1, PinnedAt: tied.Add(time.Hour)}
	if created, err := repo.CreateOrGet(&duplicate); err != nil || created {
		t.Fatalf("duplicate: %v %v", created, err)
	}
	if !duplicate.PinnedAt.Equal(tied) {
		t.Fatalf("duplicate changed timestamp: %s", duplicate.PinnedAt)
	}
	second := models.QuickAccessPin{UserID: 1, TargetType: models.QuickAccessTargetCoinSet, TargetID: 2, PinnedAt: tied}
	foreign := models.QuickAccessPin{UserID: 2, TargetType: models.QuickAccessTargetCoin, TargetID: 1, PinnedAt: tied.Add(time.Hour)}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	pins, err := repo.List(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 2 || pins[0].ID != second.ID {
		t.Fatalf("unexpected scoped/tie ordering: %#v", pins)
	}
	if count, err := repo.CountByType(1, models.QuickAccessTargetCoin); err != nil || count != 1 {
		t.Fatalf("coin count=%d err=%v", count, err)
	}
	if err := repo.Delete(2, models.QuickAccessTargetCoin, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(1, models.QuickAccessTargetCoin, 1); err != nil {
		t.Fatalf("foreign delete removed owner pin: %v", err)
	}
	if err := repo.DeleteTargets(1, models.QuickAccessTargetCoin, []uint{1}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(1, models.QuickAccessTargetCoin, 1); !IsRecordNotFound(err) {
		t.Fatalf("target cleanup error=%v", err)
	}

	rollbackPin := models.QuickAccessPin{UserID: 1, TargetType: models.QuickAccessTargetCalendarEvent, TargetID: 9, PinnedAt: tied}
	sentinel := errors.New("force rollback")
	if err := repo.RunInTransaction(func(tx *Transaction) error {
		if _, err := repo.WithTransaction(tx).CreateOrGet(&rollbackPin); err != nil {
			return err
		}
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatalf("rollback error=%v", err)
	}
	if _, err := repo.Get(1, models.QuickAccessTargetCalendarEvent, 9); !IsRecordNotFound(err) {
		t.Fatalf("transaction did not roll back pin: %v", err)
	}
}

func TestQuickAccessPinCheckConstraintAndIndexes(t *testing.T) {
	dsn := fmt.Sprintf("file:quick_access_schema_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&quickAccessRepoTestCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.QuickAccessPin{}); err != nil {
		t.Fatal(err)
	}
	columns, err := db.Migrator().ColumnTypes(&models.QuickAccessPin{})
	if err != nil {
		t.Fatal(err)
	}
	gotColumns := make(map[string]bool, len(columns))
	for _, column := range columns {
		gotColumns[column.Name()] = true
	}
	for _, name := range []string{"id", "user_id", "target_type", "target_id", "pinned_at", "created_at"} {
		if !gotColumns[name] {
			t.Fatalf("missing column %s; got %#v", name, gotColumns)
		}
	}
	if err := db.Create(&models.QuickAccessPin{UserID: 1, TargetType: "bad", TargetID: 1, PinnedAt: time.Now().UTC()}).Error; err == nil {
		t.Fatal("expected target_type check constraint")
	}
	if err := db.Exec("INSERT INTO quick_access_pins (user_id, target_type, target_id, pinned_at, created_at) VALUES (?, ?, ?, NULL, ?)",
		1, models.QuickAccessTargetCoin, 1, time.Now().UTC()).Error; err == nil {
		t.Fatal("expected pinned_at not-null constraint")
	}
	utc := time.Now().UTC().Truncate(time.Microsecond)
	valid := models.QuickAccessPin{UserID: 1, TargetType: models.QuickAccessTargetCoin, TargetID: 2, PinnedAt: utc}
	if err := db.Create(&valid).Error; err != nil {
		t.Fatal(err)
	}
	var loaded models.QuickAccessPin
	if err := db.First(&loaded, valid.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !loaded.PinnedAt.Equal(utc) || loaded.PinnedAt.Location() != time.UTC {
		t.Fatalf("pinned_at=%s location=%v", loaded.PinnedAt, loaded.PinnedAt.Location())
	}
	for _, index := range []string{"idx_quick_access_owner_target", "idx_quick_access_owner_time", "idx_quick_access_target"} {
		if !db.Migrator().HasIndex(&models.QuickAccessPin{}, index) {
			t.Fatalf("missing index %s", index)
		}
	}
}
