package services

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionEndingSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.AuctionEndingRun{}, &models.AuctionLot{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuctionEndingScheduler(t *testing.T, db *gorm.DB) *AuctionEndingScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	auctionEndingRepo := repository.NewAuctionEndingRepository(db)
	auctionLotRepo := repository.NewAuctionLotRepository(db)
	return NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, nil, nil, nil, settingsSvc, NewLogger(100))
}

func TestAuctionEndingTimeUntilNextRun_UsesLastCompletedRun(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "120"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-60 * time.Minute)
	run := &models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "success",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait < 59*time.Minute || wait > 61*time.Minute {
		t.Fatalf("expected ~60m wait, got %v", wait)
	}
}

func TestAuctionEndingTimeUntilNextRun_Overdue(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "60"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-2 * time.Hour)
	run := &models.AuctionEndingRun{
		TriggerType: "scheduled",
		Status:      "error",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait != 0 {
		t.Fatalf("expected immediate run (0), got %v", wait)
	}
}

func TestAuctionEndingTimeUntilNextRun_IgnoresManualRuns(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	future := time.Now().Add(2 * time.Hour)
	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckStartTime, future.Format("15:04")); err != nil {
		t.Fatalf("failed to set start time: %v", err)
	}
	if err := s.settingsSvc.SetSetting(SettingAuctionEndingCheckInterval, "1440"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-30 * time.Minute)
	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   completedAt.Add(-2 * time.Minute),
		CompletedAt: &completedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("failed to seed run: %v", err)
	}

	wait := s.timeUntilNextRun()
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Fatalf("expected ~2h wait when only manual runs exist, got %v", wait)
	}
}

func TestAuctionEndingNotifyUserSendsEndingSoonAlert(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate users: %v", err)
	}

	user := models.User{
		Username:        "bidder",
		Email:           "bidder@example.com",
		PasswordHash:    "hash",
		PushoverEnabled: true,
		PushoverUserKey: "user-key",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	scheduler := NewAuctionEndingScheduler(
		nil,
		repository.NewAuctionEndingRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		settingsSvc,
		NewLogger(100),
	)

	bidOne := 125.5
	bidTwo := 300.0
	sent := scheduler.notifyUser(user.ID, []models.AuctionLot{
		{AuctionHouse: "The Coin Cabinet", SaleName: "Ancients Auction 35", LotNumber: 30, CurrentBid: &bidOne, Currency: "GBP"},
		{AuctionHouse: "Classical Numismatic Group", SaleName: "Keystone 17", LotNumber: 95, CurrentBid: &bidTwo, Currency: "USD"},
	})
	if !sent {
		t.Fatal("notifyUser returned false")
	}

	if got := captured.Get("title"); got != "Auctions Ending Soon" {
		t.Fatalf("title = %q, want Auctions Ending Soon", got)
	}
	message := captured.Get("message")
	for _, want := range []string{
		"2 auction(s) you are bidding on end within 24 hours:",
		"The Coin Cabinet - Ancients Auction 35 (Lot 30)",
		"Classical Numismatic Group - Keystone 17 (Lot 95)",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("message %q missing %q", message, want)
		}
	}
}

// TestAuctionEndingRunNowWithTrigger_Enqueues verifies RunNowWithTrigger returns a queued run immediately.
func TestAuctionEndingRunNowWithTrigger_Enqueues(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	triggerUserID := uint(42)
	run, err := s.RunNowWithTrigger(&triggerUserID)
	if err != nil {
		t.Fatalf("RunNowWithTrigger returned error: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.ID == 0 {
		t.Error("expected run.ID to be assigned")
	}
	// Must return immediately with queued or running status (goroutine may have claimed it).
	if run.Status != "queued" && run.Status != "running" {
		t.Errorf("expected status queued or running, got %q", run.Status)
	}
	if run.TriggerType != "manual" {
		t.Errorf("expected trigger_type=manual, got %q", run.TriggerType)
	}
	if run.TriggerUserID == nil || *run.TriggerUserID != triggerUserID {
		t.Errorf("expected trigger_user_id=%d, got %v", triggerUserID, run.TriggerUserID)
	}
}

// TestAuctionEndingRunNowWithTrigger_DedupQueued verifies a second call reuses the active run.
func TestAuctionEndingRunNowWithTrigger_DedupQueued(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	auctionEndingRepo := repository.NewAuctionEndingRepository(db)

	// Seed an active queued run.
	active := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "queued",
		StartedAt:   time.Now(),
	}
	if err := auctionEndingRepo.CreateRun(active); err != nil {
		t.Fatalf("failed to seed active run: %v", err)
	}

	// Second call should reuse the existing run.
	run, err := s.RunNowWithTrigger(nil)
	if err != nil {
		t.Fatalf("RunNowWithTrigger returned error: %v", err)
	}
	if run.ID != active.ID {
		t.Errorf("expected dedup to return active run ID=%d, got %d", active.ID, run.ID)
	}

	// Verify no new run was created.
	_, total, _ := auctionEndingRepo.ListRuns(1, 100)
	if total != 1 {
		t.Errorf("expected exactly 1 run in DB, got %d", total)
	}
}

// TestAuctionEndingRunNowWithTrigger_DedupRunning verifies a running run also prevents a new enqueue.
func TestAuctionEndingRunNowWithTrigger_DedupRunning(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	auctionEndingRepo := repository.NewAuctionEndingRepository(db)

	// Seed an active running run.
	active := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	if err := auctionEndingRepo.CreateRun(active); err != nil {
		t.Fatalf("failed to seed active run: %v", err)
	}

	run, err := s.RunNowWithTrigger(nil)
	if err != nil {
		t.Fatalf("RunNowWithTrigger returned error: %v", err)
	}
	if run.ID != active.ID {
		t.Errorf("expected dedup to return running run ID=%d, got %d", active.ID, run.ID)
	}

	_, total, _ := auctionEndingRepo.ListRuns(1, 100)
	if total != 1 {
		t.Errorf("expected exactly 1 run in DB after dedup, got %d", total)
	}
}

// TestAuctionEndingStaleRunRecovery verifies recoverStaleRuns marks old in-flight runs as error.
func TestAuctionEndingStaleRunRecovery(t *testing.T) {
	db := setupAuctionEndingSchedulerDB(t)
	s := newTestAuctionEndingScheduler(t, db)

	auctionEndingRepo := repository.NewAuctionEndingRepository(db)

	// Seed stale queued and running runs older than auctionEndingStaleTimeout.
	staleTime := time.Now().Add(-2 * time.Hour)
	for _, status := range []string{"queued", "running"} {
		run := &models.AuctionEndingRun{
			TriggerType: "manual",
			Status:      status,
			StartedAt:   staleTime,
		}
		if err := auctionEndingRepo.CreateRun(run); err != nil {
			t.Fatalf("failed to seed stale %s run: %v", status, err)
		}
	}

	// Seed a fresh run that should NOT be recovered.
	fresh := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "queued",
		StartedAt:   time.Now(),
	}
	if err := auctionEndingRepo.CreateRun(fresh); err != nil {
		t.Fatalf("failed to seed fresh run: %v", err)
	}

	s.recoverStaleRuns()

	// Stale runs should now be error.
	runs, _, _ := auctionEndingRepo.ListRuns(1, 100)
	for _, r := range runs {
		if r.ID == fresh.ID {
			if r.Status != "queued" {
				t.Errorf("expected fresh run to remain queued, got %q", r.Status)
			}
		} else {
			if r.Status != "error" {
				t.Errorf("expected stale run #%d to be error, got %q", r.ID, r.Status)
			}
			if r.CompletedAt == nil {
				t.Errorf("expected stale run #%d to have CompletedAt set", r.ID)
			}
		}
	}
}
