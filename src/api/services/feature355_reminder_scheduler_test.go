package services

// Independent QA regression coverage for Feature 355 (Wishlist Purchase
// Reminders) -- ReminderScheduler runCycle behavioral tests.
// Owned by Brutus (Tester/QA).
//
// Build tag: feature355
// These tests reference ReminderScheduler, PurchaseReminderRepository, and
// PurchaseReminder -- types that do not yet exist. They will compile and run
// only after Cassius completes Phase 2 (model/repo) and Phase 5 (scheduler).
// Remove the build tag once implementation lands and verify all tests pass.
//
// Contract being tested (spec.md FR-006, FR-007, FR-008, FR-009):
//   - runCycle fires notification + sets status=notified for due pending reminders.
//   - runCycle skips future reminders (no notification, no status change).
//   - runCycle catches up overdue reminders (past dates still notified).
//   - Re-running the cycle on a notified reminder does not create a duplicate.
//   - Disabled setting (ReminderCheckEnabled=false) skips the entire cycle.
//   - Pushover failure is logged but does not block notification or status transition.
//   - Timezone boundary: reminder due "today" in America/Chicago is not fired
//     if the server evaluates it as "yesterday" in UTC (wrong TZ usage).

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupReminderSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.PurchaseReminder{},
		&models.Notification{},
		&models.AppSetting{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestReminderScheduler(t *testing.T, db *gorm.DB) *ReminderScheduler {
	t.Helper()
	logger := NewLogger(100)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	notifSvc := NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
	reminderRepo := repository.NewPurchaseReminderRepository(db)
	return NewReminderScheduler(reminderRepo, notifSvc, settingsSvc, logger)
}

func seedUserAndWishlistCoin(t *testing.T, db *gorm.DB) (models.User, models.Coin) {
	t.Helper()
	user := models.User{Username: "rem-user", Email: "rem@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{
		UserID:     user.ID,
		Name:       "Reminder Denarius",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		IsWishlist: true,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	return user, coin
}

// TestFeature355_ReminderScheduler_DueReminderCreatesNotificationAndTransitions
// FR-006/FR-007/FR-009: a pending reminder whose remindDate <= today in its
// timezone must produce exactly one notification and transition to notified.
func TestFeature355_ReminderScheduler_DueReminderCreatesNotificationAndTransitions(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var updated models.PurchaseReminder
	db.First(&updated, reminder.ID)
	if updated.Status != "notified" {
		t.Fatalf("FR-009 violation: expected status=notified after runCycle, got %q", updated.Status)
	}
	if updated.NotifiedAt == nil {
		t.Fatal("FR-009 violation: expected notifiedAt to be set after runCycle")
	}

	var count int64
	db.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", user.ID, NotificationTypePurchaseReminder).
		Count(&count)
	if count != 1 {
		t.Fatalf("FR-007 violation: expected exactly 1 notification, got %d", count)
	}
}

// TestFeature355_ReminderScheduler_FutureReminderIsSkipped
// FR-006: a reminder whose remindDate is in the future must not fire.
func TestFeature355_ReminderScheduler_FutureReminderIsSkipped(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var updated models.PurchaseReminder
	db.First(&updated, reminder.ID)
	if updated.Status != "pending" {
		t.Fatalf("FR-006 violation: future reminder status changed to %q, want pending", updated.Status)
	}
	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("FR-006 violation: expected no notification for a future reminder, got %d", count)
	}
}

// TestFeature355_ReminderScheduler_OverdueReminderIsCaughtUp
// FR-006: overdue reminders (past dates) must be notified -- not silently skipped.
func TestFeature355_ReminderScheduler_OverdueReminderIsCaughtUp(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var updated models.PurchaseReminder
	db.First(&updated, reminder.ID)
	if updated.Status != "notified" {
		t.Fatalf("FR-006 violation: overdue reminder was not caught up, status=%q", updated.Status)
	}
}

// TestFeature355_ReminderScheduler_NotifiedReminderProducesNoDuplicate
// FR-009: re-running the cycle on an already-notified reminder must not
// create a second notification. The status=notified guard is the durable
// idempotency gate across restarts.
func TestFeature355_ReminderScheduler_NotifiedReminderProducesNoDuplicate(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)
	notifiedAt := time.Now()
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "notified",
		NotifiedAt: &notifiedAt,
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()
	scheduler.runCycle() // second run should be a no-op

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("FR-009 violation: expected no notifications for a already-notified reminder, got %d", count)
	}
}

// TestFeature355_ReminderScheduler_DisabledSettingSkipsCycle verifies that
// when ReminderCheckEnabled=false, no reminders are processed and no
// notifications are created.
func TestFeature355_ReminderScheduler_DisabledSettingSkipsCycle(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "false")

	user, coin := seedUserAndWishlistCoin(t, db)
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var updated models.PurchaseReminder
	db.First(&updated, reminder.ID)
	if updated.Status != "pending" {
		t.Fatalf("expected status=pending when scheduler is disabled, got %q", updated.Status)
	}
	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected no notifications when scheduler is disabled, got %d", count)
	}
}

// TestFeature355_ReminderScheduler_TimezoneBoundaryDueInLocalNotInUTC verifies
// D3: scheduler evaluates remindDate in the STORED timezone, not UTC.
// A reminder for "today" in America/Chicago must fire when the scheduler runs
// in that timezone context, even if it is "yesterday" in UTC (before midnight UTC).
func TestFeature355_ReminderScheduler_TimezoneBoundaryDueInLocalNotInUTC(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)

	// Use the local clock to compute "today" in America/Chicago explicitly.
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("time.LoadLocation: %v", err)
	}
	todayInChicago := time.Now().In(loc).Format("2006-01-02")

	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: todayInChicago,
		Timezone:   "America/Chicago",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var updated models.PurchaseReminder
	db.First(&updated, reminder.ID)
	if updated.Status != "notified" {
		t.Fatalf("D3 violation: a reminder due today in America/Chicago was not fired, status=%q", updated.Status)
	}
}

// TestFeature355_ReminderScheduler_NotificationContractFields verifies FR-007:
// the notification created must have type=purchase_reminder,
// referenceId=reminder.ID, and referenceUrl=/coin/{coinId}.
func TestFeature355_ReminderScheduler_NotificationContractFields(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckEnabled, "true")

	user, coin := seedUserAndWishlistCoin(t, db)
	reminder := models.PurchaseReminder{
		CoinID:     coin.ID,
		UserID:     user.ID,
		RemindDate: time.Now().Format("2006-01-02"),
		Timezone:   "UTC",
		Status:     "pending",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}

	scheduler.runCycle()

	var notif models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypePurchaseReminder).First(&notif).Error; err != nil {
		t.Fatalf("no purchase_reminder notification found: %v", err)
	}
	if notif.Type != NotificationTypePurchaseReminder {
		t.Errorf("FR-007 violation: notification type=%q, want %q", notif.Type, NotificationTypePurchaseReminder)
	}
	if notif.ReferenceID != reminder.ID {
		t.Errorf("FR-007 violation: referenceId=%d, want %d (reminder.ID)", notif.ReferenceID, reminder.ID)
	}
	want := fmt.Sprintf("/coin/%d", coin.ID)
	if notif.ReferenceURL != want {
		t.Errorf("FR-007/D10 violation: referenceUrl=%q, want %q", notif.ReferenceURL, want)
	}
	if notif.Title == "" {
		t.Error("FR-007 violation: notification title must not be empty")
	}
	if notif.Message == "" {
		t.Error("FR-007 violation: notification message must not be empty")
	}
	if !strings.Contains(notif.Message, coin.Name) {
		t.Errorf("FR-007 violation: notification message %q must contain coin name %q", notif.Message, coin.Name)
	}
}

// TestFeature355_ReminderScheduler_TimeUntilNextRun_LaterToday verifies the
// daily anchor calculation matches the ReminderCheckStartTime setting.
func TestFeature355_ReminderScheduler_TimeUntilNextRun_LaterToday(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)

	future := time.Now().Add(2 * time.Hour)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckStartTime, future.Format("15:04"))

	wait := scheduler.timeUntilNextRun()
	if wait < 1*time.Hour+55*time.Minute || wait > 2*time.Hour+5*time.Minute {
		t.Errorf("expected ~2h wait for a later-today anchor, got %v", wait)
	}
}

// TestFeature355_ReminderScheduler_TimeUntilNextRun_RollsOverToTomorrow verifies
// that when today's anchor has already passed, the scheduler waits ~24h.
func TestFeature355_ReminderScheduler_TimeUntilNextRun_RollsOverToTomorrow(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)

	past := time.Now().Add(-1 * time.Hour)
	_ = scheduler.settingsSvc.SetSetting(SettingReminderCheckStartTime, past.Format("15:04"))

	wait := scheduler.timeUntilNextRun()
	if wait < 22*time.Hour+55*time.Minute || wait > 23*time.Hour+5*time.Minute {
		t.Errorf("expected ~23h wait when today's anchor has passed, got %v", wait)
	}
}

// TestFeature355_ReminderScheduler_DefaultStartTimeIsEightAM verifies
// FR-015: SettingReminderCheckStartTime defaults to "08:00".
func TestFeature355_ReminderScheduler_DefaultStartTimeIsEightAM(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)

	h, m := scheduler.getStartTime()
	if h != 8 || m != 0 {
		t.Errorf("FR-015 violation: default start time = %d:%02d, want 08:00", h, m)
	}
}

// TestFeature355_ReminderScheduler_GetStatus_NameIsReminderCheck verifies
// FR-014/D7: GetStatus().Name must match the expected display name so the
// admin schedules panel shows it correctly.
func TestFeature355_ReminderScheduler_GetStatus_NameIsReminderCheck(t *testing.T) {
	db := setupReminderSchedulerDB(t)
	scheduler := newTestReminderScheduler(t, db)

	status := scheduler.GetStatus()
	if status.Name != "Reminder Check" {
		t.Errorf("FR-014 violation: GetStatus().Name=%q, want %q", status.Name, "Reminder Check")
	}
}
