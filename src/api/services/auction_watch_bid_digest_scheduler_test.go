package services

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionWatchBidDigestSchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.User{}, &models.AuctionLot{}, &models.AuctionWatchBidDigestRun{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuctionWatchBidDigestScheduler(t *testing.T, db *gorm.DB) *AuctionWatchBidDigestScheduler {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)
	runRepo := repository.NewAuctionWatchBidDigestRepository(db)
	return NewAuctionWatchBidDigestScheduler(nil, runRepo, nil, nil, nil, settingsSvc, NewLogger(100))
}

func TestAuctionWatchBidDigestTimeUntilNextRun_UsesLastCompletedRun(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
	s := newTestAuctionWatchBidDigestScheduler(t, db)

	if err := s.settingsSvc.SetSetting(SettingAuctionWatchBidDigestInterval, "120"); err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	completedAt := time.Now().Add(-60 * time.Minute)
	run := &models.AuctionWatchBidDigestRun{
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

func TestAuctionWatchBidDigestNotifyUserIncludesCurrentHighBids(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
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

	scheduler := NewAuctionWatchBidDigestScheduler(
		nil,
		repository.NewAuctionWatchBidDigestRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(100),
	)

	bidOne := 125.5
	bidTwo := 300.0
	previousTwo := 250.0
	sent := scheduler.notifyUser(user.ID, []models.AuctionLot{
		{Title: "Denarius of Trajan", AuctionHouse: "The Coin Cabinet", SaleName: "Ancients Auction 35", LotNumber: 30, CurrentBid: &bidOne, Currency: "GBP"},
		{Title: "Keystone Tetradrachm", AuctionHouse: "Classical Numismatic Group", SaleName: "Keystone 17", LotNumber: 95, CurrentBid: &bidTwo, LastDigestBid: &previousTwo, Currency: "USD", SourceURL: "https://cngcoins.com/lot/95"},
	})
	if !sent {
		t.Fatal("notifyUser returned false")
	}

	if got := captured.Get("title"); got != "Auction Watch Bid Digest" {
		t.Fatalf("title = %q, want Auction Watch Bid Digest", got)
	}
	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1 — the digest body carries markup", got)
	}
	message := captured.Get("message")
	want := "2 watched auction lot(s):\n\n" +
		"<i>The Coin Cabinet - Ancients Auction 35</i>\n" +
		"<b>Denarius of Trajan</b> (Lot 30)\n" +
		"- Current high bid: 125.50 GBP\n\n" +
		"<i>Classical Numismatic Group - Keystone 17</i>\n" +
		"<b>Keystone Tetradrachm</b> (<a href=\"https://cngcoins.com/lot/95\">Lot 95</a>)\n" +
		"- Current high bid: 300.00 USD (up from 250.00)"
	if message != want {
		t.Fatalf("message = %q, want %q", message, want)
	}
}

func TestFormatAuctionBidHandlesMissingBid(t *testing.T) {
	if got := formatAuctionBid(nil, "USD"); got != "current high bid unavailable" {
		t.Fatalf("formatAuctionBid(nil) = %q", got)
	}
}

func TestBuildAuctionWatchBidDigestMessageBlankTitleFallsBackToUntitledLot(t *testing.T) {
	bid := 42.0
	message, _ := buildAuctionWatchBidDigestMessage([]models.AuctionLot{
		{Title: "   ", AuctionHouse: "NumisBids", SaleName: "Sale 12", LotNumber: 7, CurrentBid: &bid, Currency: "EUR"},
	})
	want := "1 watched auction lot(s):\n\n" +
		"<i>NumisBids - Sale 12</i>\n" +
		"<b>Untitled lot</b> (Lot 7)\n" +
		"- Current high bid: 42.00 EUR"
	if message.Message != want {
		t.Fatalf("message = %q, want %q", message.Message, want)
	}
}

func TestBuildAuctionWatchBidDigestMessageCurrentBidUnavailable(t *testing.T) {
	message, _ := buildAuctionWatchBidDigestMessage([]models.AuctionLot{
		{Title: "Athenian Owl Tetradrachm", AuctionHouse: "NumisBids", SaleName: "Sale 12", LotNumber: 7, CurrentBid: nil, Currency: "EUR"},
	})
	want := "1 watched auction lot(s):\n\n" +
		"<i>NumisBids - Sale 12</i>\n" +
		"<b>Athenian Owl Tetradrachm</b> (Lot 7)\n" +
		"- Current high bid: unavailable"
	if message.Message != want {
		t.Fatalf("message = %q, want %q", message.Message, want)
	}
}

func TestBuildAuctionWatchBidDigestMessageStaysWithinPushoverLimitAndNotesOmittedLots(t *testing.T) {
	bid := 99.0
	longTitle := strings.Repeat("Extremely Long Ancient Coin Lot Title ", 5)
	lots := make([]models.AuctionLot, 0, 40)
	for i := 0; i < 40; i++ {
		lots = append(lots, models.AuctionLot{
			Title:        longTitle,
			AuctionHouse: "Classical Numismatic Group",
			SaleName:     "Electronic Auction 616",
			LotNumber:    i + 1,
			CurrentBid:   &bid,
			Currency:     "USD",
		})
	}

	built, reported := buildAuctionWatchBidDigestMessage(lots)
	message := built.Message
	included := len(reported)

	if len(message) > pushoverMessageLimit {
		t.Fatalf("message length = %d, want <= %d", len(message), pushoverMessageLimit)
	}
	if !strings.Contains(message, "more lot(s) omitted") {
		t.Fatalf("message %q missing omitted-lots note", message)
	}
	if strings.Contains(message, "Lot 40") {
		t.Fatalf("message unexpectedly includes every lot: %q", message)
	}
	if included == 0 || included >= len(lots) {
		t.Fatalf("included = %d, want a trimmed count between 1 and %d", included, len(lots)-1)
	}
	if !strings.Contains(message, fmt.Sprintf("(Lot %d)", included)) {
		t.Fatalf("message %q does not name lot %d, the last one included", message, included)
	}
	if strings.Contains(message, fmt.Sprintf("(Lot %d)", included+1)) {
		t.Fatalf("message %q names lot %d, which was reported as omitted", message, included+1)
	}
}

func TestFormatAuctionDigestBidComparesWithPreviousDigest(t *testing.T) {
	bid := 80.0
	previousLower := 75.0
	previousHigher := 95.0
	previousSame := 80.0

	tests := []struct {
		name string
		lot  models.AuctionLot
		want string
	}{
		{
			name: "first digest for the lot has nothing to compare against",
			lot:  models.AuctionLot{CurrentBid: &bid, Currency: "USD"},
			want: "80.00 USD",
		},
		{
			name: "bid rose since the last digest",
			lot:  models.AuctionLot{CurrentBid: &bid, LastDigestBid: &previousLower, Currency: "USD"},
			want: "80.00 USD (up from 75.00)",
		},
		{
			name: "bid fell since the last digest",
			lot:  models.AuctionLot{CurrentBid: &bid, LastDigestBid: &previousHigher, Currency: "USD"},
			want: "80.00 USD (down from 95.00)",
		},
		{
			name: "bid unchanged since the last digest",
			lot:  models.AuctionLot{CurrentBid: &bid, LastDigestBid: &previousSame, Currency: "USD"},
			want: "80.00 USD (no change)",
		},
		{
			name: "blank currency falls back to USD",
			lot:  models.AuctionLot{CurrentBid: &bid, LastDigestBid: &previousLower},
			want: "80.00 USD (up from 75.00)",
		},
		{
			name: "no current bid cannot be compared",
			lot:  models.AuctionLot{CurrentBid: nil, LastDigestBid: &previousLower, Currency: "USD"},
			want: "unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatAuctionDigestBid(test.lot); got != test.want {
				t.Fatalf("formatAuctionDigestBid() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAuctionLotShortTitleKeepsTheIdentifyingClause(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "catalog description drops to its leading clause",
			title: "PAMPHYLIA, Aspendos. Circa 380/75-330/25 BC. AR Stater (20mm, 10.85 g, 2h). VF.",
			want:  "PAMPHYLIA, Aspendos",
		},
		{
			name:  "single-sentence title keeps everything but its full stop",
			title: "Athenian Owl Tetradrachm.",
			want:  "Athenian Owl Tetradrachm",
		},
		{
			name:  "title with no sentence break is capped",
			title: strings.Repeat("A", 80),
			want:  strings.Repeat("A", 57) + "...",
		},
		{
			name:  "blank title still falls back to the untitled placeholder",
			title: "   ",
			want:  "Untitled lot",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := auctionLotShortTitle(models.AuctionLot{Title: test.title})
			if got != test.want {
				t.Fatalf("auctionLotShortTitle() = %q, want %q", got, test.want)
			}
			if len([]rune(got)) > auctionLotShortTitleLimit {
				t.Fatalf("auctionLotShortTitle() = %q, longer than the %d-rune limit", got, auctionLotShortTitleLimit)
			}
		})
	}
}

// seedAuctionWatchBidDigestUser creates a Pushover-enabled user for digest delivery tests.
func seedAuctionWatchBidDigestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
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
	return user
}

func TestAuctionWatchBidDigestRecordsReportedBidsAsTheNextBaseline(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
	user := seedAuctionWatchBidDigestUser(t, db)

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	auctionRepo := repository.NewAuctionLotRepository(db)
	scheduler := NewAuctionWatchBidDigestScheduler(
		auctionRepo,
		repository.NewAuctionWatchBidDigestRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(100),
	)

	openingBid := 75.0
	lot := models.AuctionLot{
		Title: "PAMPHYLIA, Aspendos. Circa 380/75-330/25 BC. AR Stater.", AuctionHouse: "Classical Numismatic Group",
		SaleName: "Electronic Auction 616", LotNumber: 337, CurrentBid: &openingBid, Currency: "USD", UserID: user.ID,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to seed lot: %v", err)
	}

	// First digest: no baseline yet, so the bid stands alone.
	if !scheduler.notifyUser(user.ID, []models.AuctionLot{lot}) {
		t.Fatal("first notifyUser returned false")
	}
	if got := captured.Get("message"); !strings.HasSuffix(got, "- Current high bid: 75.00 USD") {
		t.Fatalf("first digest = %q, want an uncompared 75.00 USD bid line", got)
	}

	// The delivered bid becomes the baseline the next digest compares against.
	var stored models.AuctionLot
	if err := db.First(&stored, lot.ID).Error; err != nil {
		t.Fatalf("failed to reload lot: %v", err)
	}
	if stored.LastDigestBid == nil || *stored.LastDigestBid != openingBid {
		t.Fatalf("LastDigestBid = %v, want %v", stored.LastDigestBid, openingBid)
	}
	if !stored.UpdatedAt.Equal(lot.UpdatedAt) {
		t.Fatalf("UpdatedAt moved from %v to %v; snapshotting a digest bid is not a lot change", lot.UpdatedAt, stored.UpdatedAt)
	}

	// Second digest, after sync advanced the bid, compares against what the first reported.
	raisedBid := 80.0
	if err := db.Model(&models.AuctionLot{}).Where("id = ?", lot.ID).Update("current_bid", raisedBid).Error; err != nil {
		t.Fatalf("failed to advance the current bid: %v", err)
	}
	if err := db.First(&stored, lot.ID).Error; err != nil {
		t.Fatalf("failed to reload lot: %v", err)
	}
	if !scheduler.notifyUser(user.ID, []models.AuctionLot{stored}) {
		t.Fatal("second notifyUser returned false")
	}
	if got := captured.Get("message"); !strings.Contains(got, "- Current high bid: 80.00 USD (up from 75.00)") {
		t.Fatalf("second digest = %q, want it to compare against the first digest's 75.00", got)
	}

	// A third digest with nothing new says so rather than repeating a stale comparison.
	if err := db.First(&stored, lot.ID).Error; err != nil {
		t.Fatalf("failed to reload lot: %v", err)
	}
	if !scheduler.notifyUser(user.ID, []models.AuctionLot{stored}) {
		t.Fatal("third notifyUser returned false")
	}
	if got := captured.Get("message"); !strings.Contains(got, "- Current high bid: 80.00 USD (no change)") {
		t.Fatalf("third digest = %q, want it to report no change", got)
	}
}

func TestAuctionWatchBidDigestDoesNotBaselineLotsItNeverReported(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
	user := seedAuctionWatchBidDigestUser(t, db)

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	auctionRepo := repository.NewAuctionLotRepository(db)
	scheduler := NewAuctionWatchBidDigestScheduler(
		auctionRepo,
		repository.NewAuctionWatchBidDigestRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(100),
	)

	bid := 99.0
	longTitle := strings.Repeat("Extremely Long Ancient Coin Lot Title ", 5)
	lots := make([]models.AuctionLot, 0, 40)
	for i := 0; i < 40; i++ {
		lot := models.AuctionLot{
			Title: longTitle, AuctionHouse: "Classical Numismatic Group", SaleName: "Electronic Auction 616",
			LotNumber: i + 1, CurrentBid: &bid, Currency: "USD", UserID: user.ID,
		}
		if err := db.Create(&lot).Error; err != nil {
			t.Fatalf("failed to seed lot %d: %v", i+1, err)
		}
		lots = append(lots, lot)
	}

	if !scheduler.notifyUser(user.ID, lots) {
		t.Fatal("notifyUser returned false")
	}

	_, reported := buildAuctionWatchBidDigestMessage(lots)

	if len(reported) == 0 || len(reported) >= len(lots) {
		t.Fatalf("reported = %d lots, want the digest to have trimmed some", len(reported))
	}

	var baselined int64
	if err := db.Model(&models.AuctionLot{}).Where("last_digest_bid IS NOT NULL").Count(&baselined).Error; err != nil {
		t.Fatalf("failed to count baselined lots: %v", err)
	}
	if baselined != int64(len(reported)) {
		t.Fatalf("%d lots baselined, want %d — only lots the push actually named", baselined, len(reported))
	}
}

func TestAuctionWatchBidDigestDoesNotBaselineWhenTheSendFails(t *testing.T) {
	db := setupAuctionWatchBidDigestSchedulerDB(t)
	user := seedAuctionWatchBidDigestUser(t, db)

	var captured url.Values
	pushoverSvc, cleanup := newTestPushoverService(t, &captured)
	cleanup() // the Pushover endpoint is gone, so delivery fails

	scheduler := NewAuctionWatchBidDigestScheduler(
		repository.NewAuctionLotRepository(db),
		repository.NewAuctionWatchBidDigestRepository(db),
		repository.NewUserRepository(db),
		pushoverSvc,
		nil,
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(100),
	)

	bid := 75.0
	lot := models.AuctionLot{
		Title: "Athenian Owl Tetradrachm", AuctionHouse: "NumisBids", SaleName: "Sale 12",
		LotNumber: 7, CurrentBid: &bid, Currency: "USD", UserID: user.ID,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("failed to seed lot: %v", err)
	}

	if scheduler.notifyUser(user.ID, []models.AuctionLot{lot}) {
		t.Fatal("notifyUser returned true despite a failed send")
	}

	var stored models.AuctionLot
	if err := db.First(&stored, lot.ID).Error; err != nil {
		t.Fatalf("failed to reload lot: %v", err)
	}
	if stored.LastDigestBid != nil {
		t.Fatalf("LastDigestBid = %v, want nil — an undelivered digest sets no baseline", *stored.LastDigestBid)
	}
}

func TestAuctionWatchBidDigestHeadlineLinksTheLotNumber(t *testing.T) {
	tests := []struct {
		name string
		lot  models.AuctionLot
		want string
	}{
		{
			name: "source URL becomes the lot number's link",
			lot:  models.AuctionLot{Title: "Athenian Owl", LotNumber: 337, SourceURL: "https://cngcoins.com/lot/337"},
			want: "<b>Athenian Owl</b> (<a href=\"https://cngcoins.com/lot/337\">Lot 337</a>)",
		},
		{
			name: "lots with only a NumisBids URL still link",
			lot:  models.AuctionLot{Title: "Athenian Owl", LotNumber: 337, NumisBidsURL: "https://www.numisbids.com/n.php?p=lot&sid=1&lot=337"},
			want: "<b>Athenian Owl</b> (<a href=\"https://www.numisbids.com/n.php?p=lot&amp;sid=1&amp;lot=337\">Lot 337</a>)",
		},
		{
			name: "no provider URL leaves the lot number as plain text",
			lot:  models.AuctionLot{Title: "Athenian Owl", LotNumber: 337},
			want: "<b>Athenian Owl</b> (Lot 337)",
		},
		{
			name: "a non-http scheme is never rendered as a link",
			lot:  models.AuctionLot{Title: "Athenian Owl", LotNumber: 337, SourceURL: "javascript:alert(1)"},
			want: "<b>Athenian Owl</b> (Lot 337)",
		},
		{
			name: "a relative URL is never rendered as a link",
			lot:  models.AuctionLot{Title: "Athenian Owl", LotNumber: 337, SourceURL: "/lot/337"},
			want: "<b>Athenian Owl</b> (Lot 337)",
		},
		{
			name: "a scraped title carrying markup is escaped",
			lot:  models.AuctionLot{Title: "<script>alert(1)</script> Owl", LotNumber: 337, SourceURL: "https://cngcoins.com/lot/337"},
			want: "<b>&lt;script&gt;alert(1)&lt;/script&gt; Owl</b> (<a href=\"https://cngcoins.com/lot/337\">Lot 337</a>)",
		},
		{
			name: "a lot with no lot number has nothing to link",
			lot:  models.AuctionLot{Title: "Athenian Owl", SourceURL: "https://cngcoins.com/lot/337"},
			want: "<b>Athenian Owl</b>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := auctionWatchBidDigestHeadline(test.lot); got != test.want {
				t.Fatalf("auctionWatchBidDigestHeadline() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildAuctionWatchBidDigestMessageEscapesScrapedSaleNames(t *testing.T) {
	bid := 42.0
	message, _ := buildAuctionWatchBidDigestMessage([]models.AuctionLot{
		{Title: "Athenian Owl", AuctionHouse: "Ancients & Co", SaleName: "Sale <b>12</b>", LotNumber: 7, CurrentBid: &bid, Currency: "EUR"},
	})
	if !message.HTML {
		t.Fatal("digest message is not flagged as HTML")
	}
	if !strings.Contains(message.Message, "Ancients &amp; Co - Sale &lt;b&gt;12&lt;/b&gt;") {
		t.Fatalf("message = %q, want the scraped sale label escaped", message.Message)
	}
}

func TestBuildAuctionWatchBidDigestMessageGroupsInterleavedSalesUnderOneHeading(t *testing.T) {
	bid := 50.0
	lots := []models.AuctionLot{
		{Title: "Owl", AuctionHouse: "CNG", SaleName: "Electronic Auction 616", LotNumber: 1, CurrentBid: &bid, Currency: "USD"},
		{Title: "Stater", AuctionHouse: "The Coin Cabinet", SaleName: "Ancients 35", LotNumber: 2, CurrentBid: &bid, Currency: "USD"},
		{Title: "Denarius", AuctionHouse: "CNG", SaleName: "Electronic Auction 616", LotNumber: 3, CurrentBid: &bid, Currency: "USD"},
	}

	message, reported := buildAuctionWatchBidDigestMessage(lots)

	want := "3 watched auction lot(s):\n\n" +
		"<i>CNG - Electronic Auction 616</i>\n" +
		"<b>Owl</b> (Lot 1)\n" +
		"- Current high bid: 50.00 USD\n\n" +
		"<b>Denarius</b> (Lot 3)\n" +
		"- Current high bid: 50.00 USD\n\n" +
		"<i>The Coin Cabinet - Ancients 35</i>\n" +
		"<b>Stater</b> (Lot 2)\n" +
		"- Current high bid: 50.00 USD"
	if message.Message != want {
		t.Fatalf("message = %q, want %q", message.Message, want)
	}

	// The reported lots follow the order the digest named them, which is what the caller
	// snapshots — not the order they were passed in.
	if len(reported) != 3 {
		t.Fatalf("reported %d lots, want 3", len(reported))
	}
	wantLotNumbers := []int{1, 3, 2}
	for i, lot := range reported {
		if lot.LotNumber != wantLotNumbers[i] {
			t.Fatalf("reported[%d] = Lot %d, want Lot %d", i, lot.LotNumber, wantLotNumbers[i])
		}
	}
}
