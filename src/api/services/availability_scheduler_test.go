package services

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAvailSchedulerDB creates an in-memory SQLite DB with required tables.
func setupAvailSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.AvailabilityRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// newTestAvailabilityScheduler builds a minimal scheduler for timing tests.
func newTestAvailabilityScheduler(t *testing.T, db *gorm.DB) *AvailabilityScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	availRepo := repository.NewAvailabilityRepository(db)
	return NewAvailabilityScheduler(nil, nil, availRepo, settingsSvc, NewLogger(100))
}

// TestTimeUntilNextRun_NoHistory verifies anchor-based scheduling when no run
// history exists.
func TestTimeUntilNextRun_NoHistory(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Configure start time to 2 hours in the future, interval = 1440 min.
	settingsSvc := s.settingsSvc
	future := time.Now().Add(2 * time.Hour)
	startTime := future.Format("15:04")
	if err := settingsSvc.SetSetting(SettingWishlistCheckStartTime, startTime); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	wait := s.timeUntilNextRun()

	// Should wait approximately 2 hours (within a few seconds of tolerance).
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Errorf("expected ~2h wait, got %v", wait)
	}
}

// TestTimeUntilNextRun_UsesLastRun verifies that the interval is measured from
// the most recent completed scheduled run, not recalculated from today's anchor.
func TestTimeUntilNextRun_UsesLastRun(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 1440 minutes.
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed a completed scheduled run that happened 60 minutes ago.
	sixtyMinsAgo := time.Now().Add(-60 * time.Minute)
	completedAt := time.Now().Add(-59 * time.Minute)
	user := models.User{Username: "testuser"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "scheduled",
		StartedAt:   sixtyMinsAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	// With a last run 60 minutes ago and 1440-minute interval, the next run
	// should be in approximately 1380 minutes (~23 hours).
	expectedMin := 1379 * time.Minute
	expectedMax := 1381 * time.Minute
	if wait < expectedMin || wait > expectedMax {
		t.Errorf("expected wait ~1380m, got %v", wait)
	}
}

// TestTimeUntilNextRun_Overdue verifies that when the last run is further back
// than the configured interval, the scheduler returns 0 to run immediately.
func TestTimeUntilNextRun_Overdue(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 60 minutes.
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "60"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed a run that completed 120 minutes ago — clearly overdue.
	twoHoursAgo := time.Now().Add(-120 * time.Minute)
	completedAt := time.Now().Add(-119 * time.Minute)
	user := models.User{Username: "testuser2"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "scheduled",
		StartedAt:   twoHoursAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	if wait != 0 {
		t.Errorf("expected 0 (immediate) for overdue run, got %v", wait)
	}
}

// TestTimeUntilNextRun_IgnoresManualRuns verifies that manual runs are not
// counted as the scheduling anchor.
func TestTimeUntilNextRun_IgnoresManualRuns(t *testing.T) {
	db := setupAvailSchedulerDB(t)
	s := newTestAvailabilityScheduler(t, db)

	// Set interval to 1440 minutes, start time 2 hours in the future.
	future := time.Now().Add(2 * time.Hour)
	startTime := future.Format("15:04")
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckStartTime, startTime); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := s.settingsSvc.SetSetting(SettingWishlistCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	// Seed only a MANUAL run (not scheduled) 5 minutes ago.
	fiveMinsAgo := time.Now().Add(-5 * time.Minute)
	completedAt := time.Now().Add(-4 * time.Minute)
	user := models.User{Username: "testuser3"}
	db.Create(&user)
	run := &models.AvailabilityRun{
		UserID:      user.ID,
		TriggerType: "manual", // not scheduled
		StartedAt:   fiveMinsAgo,
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()

	// Should still wait ~2h (falls back to anchor-based calculation because no
	// scheduled run exists).
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Errorf("expected ~2h wait (ignoring manual run), got %v", wait)
	}
}
