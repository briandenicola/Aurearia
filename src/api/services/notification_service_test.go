package services

import (
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupNotificationServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	// NotifyAuctionPriceAlert/NotifyAuctionBidReminder fire Pushover on a background goroutine;
	// pin the pool to one connection so it can't land on a second, unmigrated ":memory:" instance.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.Notification{}, &models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestNotificationService(db *gorm.DB) *NotificationService {
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	logger := NewLogger(100)
	return NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
}

func TestNotifyAuctionPriceAlertCreatesNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	bid := 150.0
	lot := models.AuctionLot{
		UserID:       user.ID,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		CurrentBid:   &bid,
		Currency:     "USD",
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionPriceAlert(user.ID, lot, 100)

	var notifications []models.Notification
	if err := db.Where("user_id = ?", user.ID).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.Type != NotificationTypeAuctionPriceAlert {
		t.Errorf("type = %q, want %q", n.Type, NotificationTypeAuctionPriceAlert)
	}
	if n.ReferenceID != lot.ID {
		t.Errorf("referenceId = %d, want %d", n.ReferenceID, lot.ID)
	}
	if n.ReferenceURL != lot.SourceURL {
		t.Errorf("referenceUrl = %q, want %q", n.ReferenceURL, lot.SourceURL)
	}
}

func TestNotifyAuctionBidReminderCreatesNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lot := models.AuctionLot{
		UserID:       user.ID,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionBidReminder(user.ID, lot, 30)

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionBidReminder).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 bid reminder notification, got %d", len(notifications))
	}
}

func TestNotifyAuctionPriceAlertMessageLeadsWithTitleAndKeepsBidDetails(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	bid := 150.0
	lot := models.AuctionLot{
		UserID:       user.ID,
		Title:        "Julia Domna AR Denarius",
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		CurrentBid:   &bid,
		Currency:     "USD",
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionPriceAlert(user.ID, lot, 100)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionPriceAlert).First(&n).Error; err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}

	wantMessage := "Julia Domna AR Denarius (Lot 95)\n" +
		"CNG - Keystone 17\n" +
		"- Target: 100.00 USD\n" +
		"- Current high bid: 150.00 USD (50.00 over target)"
	if n.Message != wantMessage {
		t.Errorf("message = %q, want %q", n.Message, wantMessage)
	}
	if n.Title != "Auction Price Alert" {
		t.Errorf("title = %q, want %q", n.Title, "Auction Price Alert")
	}
	if n.ReferenceURL != lot.SourceURL {
		t.Errorf("referenceUrl = %q, want %q", n.ReferenceURL, lot.SourceURL)
	}
}

func TestNotifyAuctionPriceAlertBlankTitleFallsBackToUntitledLot(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lot := models.AuctionLot{
		UserID:       user.ID,
		Title:        "   ",
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		Currency:     "USD",
	}

	svc.NotifyAuctionPriceAlert(user.ID, lot, 100)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionPriceAlert).First(&n).Error; err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}

	if !strings.HasPrefix(n.Message, "Untitled lot (Lot 95)\n") {
		t.Errorf("message = %q, want it to start with fallback title line", n.Message)
	}
	if !strings.Contains(n.Message, "- Current high bid: unavailable") {
		t.Errorf("message = %q, want unavailable-bid fallback", n.Message)
	}
}

func TestNotifyAuctionBidReminderMessageLeadsWithTitleAndIncludesCurrentBid(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	bid := 42.5
	lot := models.AuctionLot{
		UserID:       user.ID,
		Title:        "Athens AR Tetradrachm",
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
		CurrentBid:   &bid,
		Currency:     "USD",
		SourceURL:    "https://cngcoins.com/lot/95",
	}

	svc.NotifyAuctionBidReminder(user.ID, lot, 30)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionBidReminder).First(&n).Error; err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}

	wantMessage := "Athens AR Tetradrachm\n" +
		"CNG - Keystone 17 (Lot 95)\n" +
		"Reminder: 30 minutes before close\n" +
		"Current bid: current high bid 42.50 USD"
	if n.Message != wantMessage {
		t.Errorf("message = %q, want %q", n.Message, wantMessage)
	}
	if n.Title != "Auction Bid Reminder" {
		t.Errorf("title = %q, want %q", n.Title, "Auction Bid Reminder")
	}
}

func TestNotifyAuctionBidReminderBlankTitleFallsBackToUntitledLot(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lot := models.AuctionLot{
		UserID:       user.ID,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    95,
	}

	svc.NotifyAuctionBidReminder(user.ID, lot, 15)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionBidReminder).First(&n).Error; err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}

	if !strings.HasPrefix(n.Message, "Untitled lot\n") {
		t.Errorf("message = %q, want it to start with fallback title line", n.Message)
	}
}

func TestNotifyAuctionEndingSoonCreatesSingleConsolidatedNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	lots := []models.AuctionLot{
		{AuctionHouse: "The Coin Cabinet", SaleName: "Ancients Auction 35", LotNumber: 30},
		{AuctionHouse: "Classical Numismatic Group", SaleName: "Keystone 17", LotNumber: 95},
	}

	svc.NotifyAuctionEndingSoon(user.ID, lots)

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAuctionEndingSoon).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 consolidated notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.Title != "Auctions Ending Soon" {
		t.Errorf("title = %q, want %q", n.Title, "Auctions Ending Soon")
	}
	for _, want := range []string{"The Coin Cabinet - Ancients Auction 35 (Lot 30)", "Classical Numismatic Group - Keystone 17 (Lot 95)"} {
		if !strings.Contains(n.Message, want) {
			t.Errorf("message %q missing %q", n.Message, want)
		}
	}
}

func TestNotifyAuctionEndingSoonNoLotsCreatesNoNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "bidder", Email: "bidder@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	svc.NotifyAuctionEndingSoon(user.ID, nil)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 notifications, got %d", count)
	}
}

// --- Feature 353 (T023): NotifyAvailabilityRunTerminal / NotifyAdminCycleChildFailure ---

func TestNotifyAvailabilityRunTerminal_CreatesOneNotificationWithGenericSuccessMessage(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "owner1", Email: "owner1@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	run := &models.AvailabilityRun{
		ID:           42,
		UserID:       user.ID,
		Status:       models.AvailabilityRunStatusCompleted,
		CoinsChecked: 2,
		Available:    2,
		Unavailable:  0,
		Unknown:      0,
	}

	svc.NotifyAvailabilityRunTerminal(user.ID, run, nil)

	var notifications []models.Notification
	if err := db.Where("user_id = ?", user.ID).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.Type != NotificationTypeAvailabilityRun {
		t.Errorf("type = %q, want %q", n.Type, NotificationTypeAvailabilityRun)
	}
	if n.ReferenceID != run.ID {
		t.Errorf("referenceId = %d, want %d", n.ReferenceID, run.ID)
	}
	if strings.Contains(n.Message, "http://") || strings.Contains(n.Message, "https://") {
		t.Errorf("message must not contain a URL, got %q", n.Message)
	}
	if strings.Contains(n.Title, "http://") || strings.Contains(n.Title, "https://") {
		t.Errorf("title must not contain a URL, got %q", n.Title)
	}
}

func TestNotifyAvailabilityRunTerminal_FailureMessageIsGenericNoURLOrErrorText(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "owner2", Email: "owner2@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	run := &models.AvailabilityRun{
		ID:          7,
		UserID:      user.ID,
		Status:      models.AvailabilityRunStatusFailed,
		FailMessage: models.GenericAvailabilityFailureMessage,
	}

	svc.NotifyAvailabilityRunTerminal(user.ID, run, nil)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).First(&n).Error; err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}
	if n.Message != models.GenericAvailabilityFailureMessage {
		t.Errorf("message = %q, want the single generic failure message %q", n.Message, models.GenericAvailabilityFailureMessage)
	}
	for _, forbidden := range []string{"http://", "https://", "SELECT", "panic", "?"} {
		if strings.Contains(n.Message, forbidden) {
			t.Errorf("failure message leaked internal detail (%q found): %q", forbidden, n.Message)
		}
	}
}

func TestNotifyAvailabilityRunTerminal_ZeroURLsStillNotifiesOnce(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "owner3", Email: "owner3@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	run := &models.AvailabilityRun{
		ID:           9,
		UserID:       user.ID,
		Status:       models.AvailabilityRunStatusCompleted,
		CoinsChecked: 0,
	}

	svc.NotifyAvailabilityRunTerminal(user.ID, run, nil)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 notification for a zero-URL run, got %d", count)
	}
}

func TestNotifyAvailabilityRunTerminal_NoDedupeAcrossMultipleRuns(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	user := models.User{Username: "owner4", Email: "owner4@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	for i := 0; i < 3; i++ {
		run := &models.AvailabilityRun{
			ID:           uint(100 + i),
			UserID:       user.ID,
			Status:       models.AvailabilityRunStatusCompleted,
			CoinsChecked: 1,
			Available:    1,
		}
		svc.NotifyAvailabilityRunTerminal(user.ID, run, nil)
	}

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&count)
	if count != 3 {
		t.Fatalf("expected 3 separate notifications (no daily/content dedupe), got %d", count)
	}
}

func TestNotifyAdminCycleChildFailure_MessageContainsOwnerAndCycleButNoURL(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	admin := models.User{Username: "adminuser", Email: "admin@example.com", PasswordHash: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	svc.NotifyAdminCycleChildFailure(admin.ID, "affected-owner", 55)

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", admin.ID, NotificationTypeAvailabilityRun).First(&n).Error; err != nil {
		t.Fatalf("failed to query admin failure notification: %v", err)
	}
	if !strings.Contains(n.Message, "affected-owner") {
		t.Errorf("message should mention the affected owner's username, got %q", n.Message)
	}
	if !strings.Contains(n.Message, "55") {
		t.Errorf("message should mention the cycle ID, got %q", n.Message)
	}
	if strings.Contains(n.Message, "http://") || strings.Contains(n.Message, "https://") {
		t.Errorf("admin failure message must not contain a URL, got %q", n.Message)
	}
}

func TestNotifyAdminCycleChildFailure_NoOpWhenAdminIDZero(t *testing.T) {
	db := setupNotificationServiceDB(t)
	svc := newTestNotificationService(db)

	svc.NotifyAdminCycleChildFailure(0, "someone", 1)

	var count int64
	db.Model(&models.Notification{}).Where("type = ?", NotificationTypeAvailabilityRun).Count(&count)
	if count != 0 {
		t.Fatalf("expected no notification created for adminID == 0, got %d", count)
	}
}

// --- Feature 353 (T023/SC-007): Pushover isolation — a failing Pushover client must never
// affect the in-app notification or the caller. ---

func TestNotifyAvailabilityRunTerminal_PushoverFailureDoesNotBlockInAppNotification(t *testing.T) {
	db := setupNotificationServiceDB(t)
	if err := db.AutoMigrate(&models.AvailabilityRun{}); err != nil {
		t.Fatalf("failed to migrate AvailabilityRun: %v", err)
	}
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	logger := NewLogger(100)
	// Leave the Pushover app token unset so SendMessage deterministically fails with
	// ErrPushoverNotConfigured (no network call needed) — proves the fire-and-forget
	// Pushover failure path never blocks the in-app notification (FR-011, SC-008).
	pushoverSvc := NewPushoverService(settingsSvc, logger)
	svc := NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		pushoverSvc,
		logger,
	)

	user := models.User{
		Username: "pushoverfail", Email: "pushoverfail@example.com", PasswordHash: "hash",
		PushoverEnabled: true, PushoverUserKey: "fake-user-key",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	run := &models.AvailabilityRun{ID: 1, UserID: user.ID, Status: models.AvailabilityRunStatusCompleted, CoinsChecked: 1, Available: 1}
	svc.NotifyAvailabilityRunTerminal(user.ID, run, nil)

	// Give the fire-and-forget Pushover goroutine a moment to fail internally.
	time.Sleep(50 * time.Millisecond)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&count)
	if count != 1 {
		t.Fatalf("expected in-app notification to persist despite Pushover failure, got count=%d", count)
	}
}

func TestFormatAuctionTargetBidStatesTheGapToTheTarget(t *testing.T) {
	over := 475.0
	under := 180.0
	exact := 200.0

	tests := []struct {
		name string
		lot  models.AuctionLot
		want string
	}{
		{
			name: "bid above the target names the overshoot",
			lot:  models.AuctionLot{CurrentBid: &over, Currency: "USD"},
			want: "475.00 USD (275.00 over target)",
		},
		{
			name: "bid below the target names the gap",
			lot:  models.AuctionLot{CurrentBid: &under, Currency: "USD"},
			want: "180.00 USD (20.00 under target)",
		},
		{
			name: "bid exactly on the target",
			lot:  models.AuctionLot{CurrentBid: &exact, Currency: "USD"},
			want: "200.00 USD (at target)",
		},
		{
			name: "blank currency falls back to USD",
			lot:  models.AuctionLot{CurrentBid: &over},
			want: "475.00 USD (275.00 over target)",
		},
		{
			name: "no current bid has nothing to compare",
			lot:  models.AuctionLot{CurrentBid: nil, Currency: "USD"},
			want: "unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatAuctionTargetBid(test.lot, 200); got != test.want {
				t.Fatalf("formatAuctionTargetBid() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildAuctionPriceAlertPushoverMessageLinksTheLotAndEscapesScrapedText(t *testing.T) {
	bid := 475.0
	lot := models.AuctionLot{
		Title:        "Vespasian. AD 69-79. AR Denarius (16.5mm, 3.22 g, 6h). Rome mint. Near VF.",
		AuctionHouse: "Classical Numismatic Group",
		SaleName:     "Electronic Auction 616",
		LotNumber:    684,
		CurrentBid:   &bid,
		Currency:     "USD",
		SourceURL:    "https://auctions.cngcoins.com/lots/view/4-5000000",
	}

	message := buildAuctionPriceAlertPushoverMessage(lot, 200)

	want := "<b>Vespasian</b> (<a href=\"https://auctions.cngcoins.com/lots/view/4-5000000\">Lot 684</a>)\n" +
		"Classical Numismatic Group - Electronic Auction 616\n" +
		"- Target: 200.00 USD\n" +
		"- Current high bid: 475.00 USD (275.00 over target)"
	if message.Message != want {
		t.Fatalf("message = %q, want %q", message.Message, want)
	}
	if !message.HTML {
		t.Fatal("price alert push is not flagged as HTML")
	}
	if message.Title != "Auction Price Alert" {
		t.Fatalf("title = %q, want Auction Price Alert", message.Title)
	}
	if message.URL != lot.SourceURL || message.URLTitle != "View auction lot" {
		t.Fatalf("url = %q / %q, want the lot URL and its action label", message.URL, message.URLTitle)
	}

	// A scraped sale name carrying markup must not reach the HTML body unescaped.
	lot.SaleName = "Sale <b>616</b> & friends"
	if escaped := buildAuctionPriceAlertPushoverMessage(lot, 200); !strings.Contains(escaped.Message, "Sale &lt;b&gt;616&lt;/b&gt; &amp; friends") {
		t.Fatalf("message = %q, want the scraped sale name escaped", escaped.Message)
	}
}

func TestBuildAuctionPriceAlertPushoverMessageOmitsAnUnusableLotURL(t *testing.T) {
	bid := 475.0
	lot := models.AuctionLot{
		Title: "Vespasian AR Denarius", AuctionHouse: "CNG", SaleName: "Electronic Auction 616",
		LotNumber: 684, CurrentBid: &bid, Currency: "USD", SourceURL: "javascript:alert(1)",
	}

	message := buildAuctionPriceAlertPushoverMessage(lot, 200)

	if message.URL != "" {
		t.Fatalf("url = %q, want it dropped — Pushover rejects a malformed url outright", message.URL)
	}
	if strings.Contains(message.Message, "<a href") {
		t.Fatalf("message = %q, want the lot number left unlinked", message.Message)
	}
}
