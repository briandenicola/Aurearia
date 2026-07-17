package services

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupWishlistSearchAlertSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.WishlistSearchAlert{}, &models.AlertRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestWishlistSearchAlertScheduler(t *testing.T, db *gorm.DB) (*WishlistSearchAlertScheduler, *repository.WishlistSearchAlertRepository) {
	t.Helper()
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	alertRepo := repository.NewWishlistSearchAlertRepository(db)
	alertSvc := NewWishlistSearchAlertService(alertRepo)
	scheduler := NewWishlistSearchAlertScheduler(alertSvc, alertRepo, settingsSvc, NewLogger(100))
	return scheduler, alertRepo
}

// TestWishlistSearchAlertScheduler_TimeUntilNextRun verifies the daily anchor
// calculation used to schedule sweeps.
func TestWishlistSearchAlertScheduler_TimeUntilNextRun(t *testing.T) {
	db := setupWishlistSearchAlertSchedulerDB(t)
	s, _ := newTestWishlistSearchAlertScheduler(t, db)

	future := time.Now().Add(3 * time.Hour)
	if err := s.settingsSvc.SetSetting(SettingWishlistSearchAlertsCheckStartTime, future.Format("15:04")); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait < 2*time.Hour+55*time.Minute || wait > 3*time.Hour+5*time.Minute {
		t.Errorf("expected ~3h wait, got %v", wait)
	}
}

// TestWishlistSearchAlertScheduler_RunCycleDisabledByDefault verifies that a
// sweep does nothing unless explicitly enabled via settings.
func TestWishlistSearchAlertScheduler_RunCycleDisabledByDefault(t *testing.T) {
	db := setupWishlistSearchAlertSchedulerDB(t)
	s, alertRepo := newTestWishlistSearchAlertScheduler(t, db)

	eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
	alert := &models.WishlistSearchAlert{UserID: 1, Name: "weekly", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &eightDaysAgo}
	if err := alertRepo.CreateAlert(alert); err != nil {
		t.Fatalf("create alert: %v", err)
	}

	s.runCycle()

	var count int64
	db.Model(&models.AlertRun{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no runs queued while disabled, got %d", count)
	}
}

// TestWishlistSearchAlertScheduler_RunCycleQueuesDueAlerts verifies that,
// once enabled, the scheduler queues a scheduled run for each due alert and
// leaves ones that aren't due alone. This is the core regression test for
// issue #483 (cadence was stored but never acted on).
func TestWishlistSearchAlertScheduler_RunCycleQueuesDueAlerts(t *testing.T) {
	db := setupWishlistSearchAlertSchedulerDB(t)
	s, alertRepo := newTestWishlistSearchAlertScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingWishlistSearchAlertsCheckEnabled, "true"); err != nil {
		t.Fatalf("failed to enable scheduler: %v", err)
	}

	eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
	twoDaysAgo := time.Now().Add(-2 * 24 * time.Hour)

	due := &models.WishlistSearchAlert{UserID: 1, Name: "weekly, overdue", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &eightDaysAgo}
	notDue := &models.WishlistSearchAlert{UserID: 1, Name: "weekly, not due", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &twoDaysAgo}
	manual := &models.WishlistSearchAlert{UserID: 1, Name: "manual", Cadence: models.WishlistAlertCadenceManual, IsActive: true, LastRunAt: &eightDaysAgo}
	for _, a := range []*models.WishlistSearchAlert{due, notDue, manual} {
		if err := alertRepo.CreateAlert(a); err != nil {
			t.Fatalf("create alert %q: %v", a.Name, err)
		}
	}

	s.runCycle()

	var runs []models.AlertRun
	if err := db.Find(&runs).Error; err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected exactly 1 queued run, got %d", len(runs))
	}
	if runs[0].AlertID != due.ID {
		t.Fatalf("expected run queued for the overdue alert %d, got alert %d", due.ID, runs[0].AlertID)
	}
	if runs[0].TriggerType != models.AlertRunTriggerScheduled {
		t.Fatalf("expected scheduled trigger type, got %q", runs[0].TriggerType)
	}
	if runs[0].Status != models.AlertRunStatusQueued {
		t.Fatalf("expected queued status, got %q", runs[0].Status)
	}
}
