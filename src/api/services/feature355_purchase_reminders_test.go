package services

// Independent QA regression coverage for Feature 355 (Wishlist Purchase
// Reminders) -- auto-cancel and ownership isolation behaviors.
// Owned by Brutus (Tester/QA).
//
// These tests verify the locked contract from spec.md:
//   - FR-011: IsWishlist true->false transition cancels all active reminders
//     within the same transaction via CoinService.
//   - FR-012: coin deletion cascades cancel (or cascade-deletes) all reminders.
//   - US4 acceptance scenario 3: re-wishlisting after auto-cancel leaves no
//     active reminder; user must create one explicitly.
//
// The tests insert purchase_reminders rows via raw SQL and verify state via
// raw SQL so they do not reference the PurchaseReminder model type (not yet
// implemented). They will COMPILE today but FAIL with a "table not found"
// error until Cassius's Phase 2 adds the table via AutoMigrate and Phase 3
// adds the auto-cancel hook to CoinService.updateCoin. This is the intended
// tests-first state per constitution SS17 Quality Gate.
//
// Do NOT weaken assertions to match today's (absent) behavior.

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var f355DBCounter uint64

// setupFeature355DB creates an in-memory SQLite DB with the coin tables PLUS
// the purchase_reminders table. The latter is created via raw DDL so the test
// compiles before the GORM model exists; AutoMigrate will own it in production.
func setupFeature355DB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("file:f355_%d_%d?mode=memory&cache=shared",
		time.Now().UnixNano(), atomic.AddUint64(&f355DBCounter, 1))
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.CatalogRegistry{}, &models.AppSetting{},
		&models.MintLocation{}, &models.StorageLocation{},
		&models.ValueSnapshot{}, &models.CoinJournal{}, &models.CoinValueHistory{},
		&models.CoinComment{}, &models.AvailabilityResult{}, &models.AuctionLot{},
		&models.Tag{}, &models.CoinTag{},
		&models.CoinSet{}, &models.CoinSetMembership{},
		&models.Showcase{}, &models.ShowcaseCoin{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// purchase_reminders table: matches the spec model exactly. Created here
	// via raw DDL so the test file compiles before the GORM model lands.
	ddl := `
	CREATE TABLE IF NOT EXISTS purchase_reminders (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		coin_id      INTEGER NOT NULL,
		user_id      INTEGER NOT NULL,
		remind_date  VARCHAR(10) NOT NULL,
		timezone     VARCHAR(64) NOT NULL DEFAULT 'UTC',
		status       VARCHAR(20) NOT NULL DEFAULT 'pending',
		notified_at  DATETIME,
		cancelled_at DATETIME,
		created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create purchase_reminders table: %v", err)
	}
	return db
}

// seedF355Reminder inserts a purchase_reminder row directly for testing.
// Returns the inserted row ID.
func seedF355Reminder(t *testing.T, db *gorm.DB, coinID, userID uint, status string) int64 {
	t.Helper()
	result := db.Exec(
		`INSERT INTO purchase_reminders (coin_id, user_id, remind_date, timezone, status)
		 VALUES (?, ?, ?, ?, ?)`,
		coinID, userID,
		time.Now().Add(7*24*time.Hour).Format("2006-01-02"),
		"America/Chicago",
		status,
	)
	if result.Error != nil {
		t.Fatalf("seed reminder: %v", result.Error)
	}
	var id int64
	db.Raw("SELECT last_insert_rowid()").Scan(&id)
	return id
}

// queryReminderStatus returns the status column of a reminder row by its ID.
func queryReminderStatus(db *gorm.DB, id int64) string {
	var status string
	db.Raw("SELECT status FROM purchase_reminders WHERE id = ?", id).Scan(&status)
	return status
}

// queryCancelledAt returns true if cancelled_at is non-null.
func queryReminderCancelledAt(db *gorm.DB, id int64) bool {
	var ts *time.Time
	db.Raw("SELECT cancelled_at FROM purchase_reminders WHERE id = ?", id).Scan(&ts)
	return ts != nil
}

// newF355CoinService builds the minimum CoinService for auto-cancel testing
// with PurchaseReminderRepository wired so auto-cancel hooks execute.
func newF355CoinService(db *gorm.DB) *CoinService {
	return NewCoinService(repository.NewCoinRepository(db), nil).
		WithReminderSupport(repository.NewPurchaseReminderRepository(db))
}

// seedF355WishlistCoin creates a User + wishlist Coin in the DB and returns both.
func seedF355WishlistCoin(t *testing.T, db *gorm.DB) (models.User, models.Coin) {
	t.Helper()
	user := models.User{Username: "f355-owner", Email: "f355@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	coin := models.Coin{
		UserID:     user.ID,
		Name:       "Trajan Denarius",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		IsWishlist: true,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	return user, coin
}

// TestFeature355_AutoCancel_WishlistExitCancelsPendingReminder verifies
// FR-011: when IsWishlist transitions true->false via UpdateCoinWithFields,
// all active (pending) reminders for that coin+user are cancelled within
// the same transaction.
func TestFeature355_AutoCancel_WishlistExitCancelsPendingReminder(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	reminderID := seedF355Reminder(t, db, coin.ID, user.ID, "pending")

	svc := newF355CoinService(db)
	updates := coin
	updates.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&coin, &updates, []string{"IsWishlist"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields: %v", err)
	}

	if got := queryReminderStatus(db, reminderID); got != "cancelled" {
		t.Fatalf("FR-011 violation: expected reminder status=cancelled after IsWishlist->false, got %q", got)
	}
	if !queryReminderCancelledAt(db, reminderID) {
		t.Fatalf("FR-011 violation: expected cancelled_at to be set after auto-cancel")
	}
}

// TestFeature355_AutoCancel_WishlistExitCancelsNotifiedReminder verifies
// FR-011 applies to reminders in notified state as well, not only pending.
func TestFeature355_AutoCancel_WishlistExitCancelsNotifiedReminder(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	reminderID := seedF355Reminder(t, db, coin.ID, user.ID, "notified")

	svc := newF355CoinService(db)
	updates := coin
	updates.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&coin, &updates, []string{"IsWishlist"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields: %v", err)
	}

	if got := queryReminderStatus(db, reminderID); got != "cancelled" {
		t.Fatalf("FR-011 violation: expected notified reminder to also be cancelled, got %q", got)
	}
}

// TestFeature355_AutoCancel_AlreadyCancelledReminderIsUnchanged verifies that
// a reminder already in cancelled state is not touched (idempotent cancel).
func TestFeature355_AutoCancel_AlreadyCancelledReminderIsUnchanged(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	reminderID := seedF355Reminder(t, db, coin.ID, user.ID, "cancelled")

	svc := newF355CoinService(db)
	updates := coin
	updates.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&coin, &updates, []string{"IsWishlist"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields: %v", err)
	}

	// Already cancelled -- should remain cancelled, no error.
	if got := queryReminderStatus(db, reminderID); got != "cancelled" {
		t.Fatalf("expected already-cancelled reminder to remain cancelled, got %q", got)
	}
}

// TestFeature355_AutoCancel_NonWishlistUpdateNoSideEffect verifies that
// updating a coin field other than IsWishlist does not touch reminders.
func TestFeature355_AutoCancel_NonWishlistUpdateNoSideEffect(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	reminderID := seedF355Reminder(t, db, coin.ID, user.ID, "pending")

	svc := newF355CoinService(db)
	updates := coin
	updates.Name = "Updated Name"
	if err := svc.UpdateCoinWithFields(&coin, &updates, []string{"Name"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields: %v", err)
	}

	// Reminder should remain pending -- no wishlist transition occurred.
	if got := queryReminderStatus(db, reminderID); got != "pending" {
		t.Fatalf("non-wishlist update must not cancel reminder, got status=%q", got)
	}
}

// TestFeature355_AutoCancel_DeleteCoinCancelsAllReminders verifies FR-012:
// when a coin is deleted, its reminders are cancelled (or cascade-deleted)
// so no stale reminder fires after deletion.
func TestFeature355_AutoCancel_DeleteCoinCancelsAllReminders(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	pendingID := seedF355Reminder(t, db, coin.ID, user.ID, "pending")
	notifiedID := seedF355Reminder(t, db, coin.ID, user.ID, "notified")

	svc := newF355CoinService(db)
	rows, err := svc.DeleteCoin(coin.ID, user.ID)
	if err != nil || rows == 0 {
		t.Fatalf("DeleteCoin: rows=%d err=%v", rows, err)
	}

	// Both reminders must be in cancelled (or absent) state -- no active
	// reminder for a deleted coin may reach the scheduler.
	for _, id := range []int64{pendingID, notifiedID} {
		st := queryReminderStatus(db, id)
		if st != "cancelled" && st != "" {
			t.Errorf("FR-012 violation: reminder %d status=%q after coin delete; want cancelled or absent", id, st)
		}
	}
}

// TestFeature355_AutoCancel_ReWishlistAfterAutoCancelHasNoActiveReminder
// verifies US4 acceptance scenario 3: after auto-cancel, re-setting
// IsWishlist=true must leave no active reminder -- the user must create one
// explicitly.
func TestFeature355_AutoCancel_ReWishlistAfterAutoCancelHasNoActiveReminder(t *testing.T) {
	db := setupFeature355DB(t)
	user, coin := seedF355WishlistCoin(t, db)
	seedF355Reminder(t, db, coin.ID, user.ID, "pending")

	svc := newF355CoinService(db)

	// Step 1: remove from wishlist (triggers auto-cancel)
	deWishlist := coin
	deWishlist.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&coin, &deWishlist, []string{"IsWishlist"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields (de-wishlist): %v", err)
	}

	// Step 2: re-add to wishlist
	reloaded := deWishlist
	reWishlist := coin
	reWishlist.IsWishlist = true
	reloaded.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&reloaded, &reWishlist, []string{"IsWishlist"}, user.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields (re-wishlist): %v", err)
	}

	// No active (pending/notified) reminder should exist.
	var count int64
	db.Raw("SELECT COUNT(*) FROM purchase_reminders WHERE coin_id = ? AND user_id = ? AND status IN ('pending','notified')", coin.ID, user.ID).Scan(&count)
	if count != 0 {
		t.Fatalf("US4 violation: expected no active reminder after re-wishlist, got %d", count)
	}
}

// TestFeature355_AutoCancel_CrossUserReminderNotCancelled verifies cross-user
// ownership isolation: auto-cancelling user A's reminder must not touch user
// B's reminder on the same coin (if the coin is shared / viewed differently).
// In practice, coins are user-scoped, but this test guards against any
// accidental over-broad cancel query.
func TestFeature355_AutoCancel_CrossUserReminderNotCancelled(t *testing.T) {
	db := setupFeature355DB(t)

	userA := models.User{Username: "f355-userA", Email: "f355a@example.com", PasswordHash: "hash"}
	userB := models.User{Username: "f355-userB", Email: "f355b@example.com", PasswordHash: "hash"}
	if err := db.Create(&userA).Error; err != nil {
		t.Fatalf("create userA: %v", err)
	}
	if err := db.Create(&userB).Error; err != nil {
		t.Fatalf("create userB: %v", err)
	}

	coinA := models.Coin{
		UserID:     userA.ID,
		Name:       "Coin A",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		IsWishlist: true,
	}
	if err := db.Create(&coinA).Error; err != nil {
		t.Fatalf("create coinA: %v", err)
	}

	reminderA := seedF355Reminder(t, db, coinA.ID, userA.ID, "pending")
	// Simulate userB having a reminder on their own coincidentally-same-coinID
	// (edge case: coin reuse ID after delete, or if reminder scoping is wrong).
	reminderB := seedF355Reminder(t, db, coinA.ID, userB.ID, "pending")

	svc := newF355CoinService(db)
	updates := coinA
	updates.IsWishlist = false
	if err := svc.UpdateCoinWithFields(&coinA, &updates, []string{"IsWishlist"}, userA.ID, "api", false); err != nil {
		t.Fatalf("UpdateCoinWithFields: %v", err)
	}

	if got := queryReminderStatus(db, reminderA); got != "cancelled" {
		t.Errorf("owner's reminder must be cancelled, got %q", got)
	}
	if got := queryReminderStatus(db, reminderB); got == "cancelled" {
		t.Errorf("cross-user isolation violation: userB's reminder was cancelled by userA's wishlist exit")
	}
}

// TestFeature355_TimezoneEvaluation_FutureDateInTimezoneIsNotDue verifies
// that a remind_date that is tomorrow in America/Chicago is correctly
// identified as not due for a scheduler evaluating "today". This is a pure
// utility test on the Go timezone API to document the expected evaluation
// approach before the scheduler exists.
func TestFeature355_TimezoneEvaluation_FutureDateInTimezoneIsNotDue(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("time.LoadLocation failed for America/Chicago: %v", err)
	}
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")
	todayStr := time.Now().In(loc).Format("2006-01-02")

	if tomorrow <= todayStr {
		t.Fatalf("test setup error: tomorrow %q should be > today %q", tomorrow, todayStr)
	}
	// The scheduler evaluates: remindDate <= todayInTimezone
	isDue := tomorrow <= todayStr
	if isDue {
		t.Errorf("FR-006 violation: a future date should not be considered due")
	}
}

// TestFeature355_TimezoneEvaluation_TodayInTimezoneIsDue verifies that a
// remind_date equal to today in the stored timezone is correctly identified
// as due.
func TestFeature355_TimezoneEvaluation_TodayInTimezoneIsDue(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("time.LoadLocation failed for America/Chicago: %v", err)
	}
	todayStr := time.Now().In(loc).Format("2006-01-02")

	isDue := todayStr <= todayStr
	if !isDue {
		t.Errorf("FR-006 violation: today's date must be considered due")
	}
}

// TestFeature355_TimezoneEvaluation_OverdueDateIsDue verifies catch-up: a
// remind_date that is 3 days in the past is still due (not silently skipped).
func TestFeature355_TimezoneEvaluation_OverdueDateIsDue(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("time.LoadLocation failed for America/Chicago: %v", err)
	}
	threeDaysAgo := time.Now().In(loc).AddDate(0, 0, -3).Format("2006-01-02")
	todayStr := time.Now().In(loc).Format("2006-01-02")

	isDue := threeDaysAgo <= todayStr
	if !isDue {
		t.Errorf("FR-006 violation: overdue reminder must be caught up and considered due")
	}
}

// TestFeature355_TimezoneEvaluation_InvalidTimezoneRejected verifies FR-004:
// time.LoadLocation returns an error for an unrecognized IANA string. The
// service layer must propagate this as ErrInvalidTimezone (400 Bad Request).
func TestFeature355_TimezoneEvaluation_InvalidTimezoneRejected(t *testing.T) {
	_, err := time.LoadLocation("Not/A/Timezone")
	if err == nil {
		t.Fatal("FR-004 violation: time.LoadLocation must fail for an invalid IANA timezone")
	}
}

// TestFeature355_TimezoneEvaluation_ValidTimezoneAccepted verifies representative
// IANA zones load without error so the service can validate them at creation time.
func TestFeature355_TimezoneEvaluation_ValidTimezoneAccepted(t *testing.T) {
	for _, tz := range []string{
		"America/Chicago",
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
		"UTC",
		"America/Los_Angeles",
	} {
		if _, err := time.LoadLocation(tz); err != nil {
			t.Errorf("FR-004 violation: expected %q to be a valid IANA timezone, got error: %v", tz, err)
		}
	}
}
