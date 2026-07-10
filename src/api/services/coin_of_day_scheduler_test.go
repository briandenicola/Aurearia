package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCoinOfDaySchedulerDB(t *testing.T, migrateUsers bool) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:cotd-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	modelsToMigrate := []interface{}{&models.CoinOfDayRun{}, &models.FeaturedCoin{}, &models.Coin{}}
	if migrateUsers {
		modelsToMigrate = append(modelsToMigrate, &models.User{})
	}
	if err := db.AutoMigrate(modelsToMigrate...); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

func newCoinOfDaySchedulerForTest(db *gorm.DB) *CoinOfDayScheduler {
	logger := NewLogger(10)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	return NewCoinOfDayScheduler(
		repository.NewFeaturedCoinRepository(db),
		repository.NewCoinOfDayRunRepository(db),
		repository.NewUserRepository(db),
		repository.NewCoinRepository(db),
		nil,
		settingsSvc,
		logger,
	)
}

func TestCoinOfDaySchedulerRunNowQueuesRun(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	scheduler := newCoinOfDaySchedulerForTest(db)
	triggerUserID := uint(42)
	run, err := scheduler.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		t.Fatalf("RunNowWithTrigger: %v", err)
	}
	if run.Status != models.CoinOfDayRunStatusQueued {
		t.Fatalf("expected queued run, got %s", run.Status)
	}
	var stored models.CoinOfDayRun
	if err := db.First(&stored, run.ID).Error; err != nil {
		t.Fatalf("load queued run: %v", err)
	}
	if stored.Status != models.CoinOfDayRunStatusQueued {
		t.Fatalf("expected stored status queued, got %s", stored.Status)
	}
}

func TestCoinOfDaySchedulerProcessRunCompletesWithoutUsers(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	scheduler := newCoinOfDaySchedulerForTest(db)
	run, err := scheduler.RunNowWithTrigger(nil)
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if err := scheduler.ProcessRun(run.ID); err != nil {
		t.Fatalf("ProcessRun: %v", err)
	}
	completed, err := scheduler.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if completed.Status != models.CoinOfDayRunStatusCompleted {
		t.Fatalf("expected completed status, got %s", completed.Status)
	}
	if completed.Picked != 0 || completed.Skipped != 0 || completed.Errors != 0 {
		t.Fatalf("unexpected counters: %+v", completed)
	}
}

func TestCoinOfDaySchedulerProcessRunFailsWhenUserQueryBreaks(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	scheduler := newCoinOfDaySchedulerForTest(db)
	run, err := scheduler.RunNowWithTrigger(nil)
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if err := db.Migrator().DropTable(&models.User{}); err != nil {
		t.Fatalf("drop users table: %v", err)
	}
	if err := scheduler.ProcessRun(run.ID); err != nil {
		t.Fatalf("ProcessRun: %v", err)
	}
	failed, err := scheduler.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if failed.Status != models.CoinOfDayRunStatusFailed {
		t.Fatalf("expected failed status, got %s", failed.Status)
	}
	if failed.ErrorMessage == "" {
		t.Fatalf("expected sanitized error message")
	}
}

func TestCoinOfDaySchedulerPreservesDailyIdempotency(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	scheduler := newCoinOfDaySchedulerForTest(db)
	user := models.User{
		Username:         "cotd-user",
		Email:            "cotd@example.com",
		PasswordHash:     "hash",
		CoinOfDayEnabled: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	coin := models.Coin{
		UserID:     user.ID,
		Name:       "Coin",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		IsWishlist: false,
		IsSold:     false,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	if err := db.Create(&models.FeaturedCoin{
		UserID:     user.ID,
		CoinID:     coin.ID,
		Summary:    "already featured",
		FeaturedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed featured coin: %v", err)
	}
	run, err := scheduler.RunNowWithTrigger(nil)
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if err := scheduler.ProcessRun(run.ID); err != nil {
		t.Fatalf("ProcessRun: %v", err)
	}
	completed, err := scheduler.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if completed.Picked != 0 || completed.Skipped != 1 {
		t.Fatalf("expected idempotent skip counters, got picked=%d skipped=%d", completed.Picked, completed.Skipped)
	}
	var count int64
	if err := db.Model(&models.FeaturedCoin{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count featured records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected no duplicate featured records, got %d", count)
	}
}

func TestCoinOfDaySchedulerStartWorkersRecoversStaleRun(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	scheduler := newCoinOfDaySchedulerForTest(db)
	stale := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerScheduled,
		Status:      models.CoinOfDayRunStatusRunning,
		StartedAt:   time.Now().Add(-2 * time.Hour),
	}
	if err := db.Create(stale).Error; err != nil {
		t.Fatalf("seed stale run: %v", err)
	}
	scheduler.StartWorkers(1)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := scheduler.GetRun(stale.ID)
		if err != nil {
			t.Fatalf("GetRun: %v", err)
		}
		if run.Status == models.CoinOfDayRunStatusCompleted {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	run, err := scheduler.GetRun(stale.ID)
	if err != nil {
		t.Fatalf("GetRun after timeout: %v", err)
	}
	t.Fatalf("expected stale run to be recovered and completed, got status=%s", run.Status)
}
