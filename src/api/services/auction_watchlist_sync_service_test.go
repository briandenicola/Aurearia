package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionWatchlistSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	// NotifyAuctionLotsTracked fires Pushover on a background goroutine; pin the pool to one
	// connection so it can't land on a second, unmigrated ":memory:" instance.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.AuctionEvent{}, &models.Notification{}, &models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// newSyncNotificationService builds the notification service the sync service notifies
// through. Pushover is left unconfigured: these tests assert on the in-app notification the
// sync creates synchronously, which is also the record a user without Pushover relies on (F027).
func newSyncNotificationService(db *gorm.DB) *NotificationService {
	logger := NewLogger(10)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	return NewNotificationService(
		repository.NewNotificationRepository(db),
		nil,
		repository.NewUserRepository(db),
		NewPushoverService(settingsSvc, logger),
		logger,
	)
}

// cngSyncFixture models a small watched-lots response covering, in one page, a still-active
// lot the user is bidding on, a closed lot the user won, a closed lot the user lost, and a
// closed lot the user was only watching (never bid on CNG) — this is the exact shape needed
// to exercise auto won/lost/passed detection end-to-end through syncCNG.
func cngSyncFixture(ourCustomerRowID, otherCustomerRowID string) string {
	return `<!doctype html><html><script>
viewVars = {
  "currentRouteName":"watched-lots-index",
  "lots":{
    "query_info":{"total_num_results":4,"page_size":50},
    "result_page":[
      {
        "row_id":"4-ACTIVE","lot_number":1,"title":"Active Lot","starting_price":"60.00",
        "status":"active","_detail_url":"/lots/view/4-ACTIVE/active-lot",
        "timed_auction_bid":{"amount":"90.00","registration":{"customer":{"row_id":"` + ourCustomerRowID + `"}}},
        "absentee_bid":{"max_bid":"200.00"},
        "auction":{"row_id":"4-SALE","title":"Sale","currency_code":"USD","effective_end_time":"2027-01-01T00:00:00Z"}
      },
      {
        "row_id":"4-WON","lot_number":2,"title":"Won Lot","starting_price":"100.00",
        "status":"sold","sold_price":"500.00","_detail_url":"/lots/view/4-WON/won-lot",
        "timed_auction_bid":{"amount":"500.00","registration":{"customer":{"row_id":"` + ourCustomerRowID + `"}}},
        "absentee_bid":{"max_bid":"600.00"},
        "auction":{"row_id":"4-SALE","title":"Sale","currency_code":"USD","effective_end_time":"2026-01-01T00:00:00Z"}
      },
      {
        "row_id":"4-LOST","lot_number":3,"title":"Lost Lot","starting_price":"80.00",
        "status":"sold","sold_price":"400.00","_detail_url":"/lots/view/4-LOST/lost-lot",
        "timed_auction_bid":{"amount":"400.00","registration":{"customer":{"row_id":"` + otherCustomerRowID + `"}}},
        "absentee_bid":{"max_bid":"350.00"},
        "auction":{"row_id":"4-SALE","title":"Sale","currency_code":"USD","effective_end_time":"2026-01-01T00:00:00Z"}
      },
      {
        "row_id":"4-WATCHEDONLY","lot_number":4,"title":"Watched Only Lot","starting_price":"50.00",
        "status":"sold","sold_price":"120.00","_detail_url":"/lots/view/4-WATCHEDONLY/watched-only-lot",
        "timed_auction_bid":{"amount":"120.00","registration":{"customer":{"row_id":"` + otherCustomerRowID + `"}}},
        "auction":{"row_id":"4-SALE","title":"Sale","currency_code":"USD","effective_end_time":"2026-01-01T00:00:00Z"}
      }
    ]
  }
};
</script></html>`
}

func newCNGSyncTestServer(t *testing.T, ourCustomerRowID, otherCustomerRowID string) *httptest.Server {
	t.Helper()
	var loggedIn bool
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// The POST /login handler below redirects here on success, same as the real site.
			w.Write([]byte(`viewVars = {"me":null};`))
		case "/login":
			if r.Method == http.MethodPost {
				loggedIn = true
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<form action="/login"><input name="username"><input name="password"></form>`))
		case "/ajax/refresh-me":
			if !loggedIn {
				w.Write([]byte(`null`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"row_id":"` + ourCustomerRowID + `"}`))
		case "/watched-lots":
			if !loggedIn {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			w.Write([]byte(cngSyncFixture(ourCustomerRowID, otherCustomerRowID)))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAuctionWatchlistSyncService_SyncCNGAutoDetectsWonLostAndPassed(t *testing.T) {
	server := newCNGSyncTestServer(t, "4-OURCUSTOMER", "4-OTHERCUSTOMER")
	defer server.Close()
	restore := overrideCNGURLs(server.URL)
	defer restore()

	db := setupAuctionWatchlistSyncDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	userRepo := repository.NewUserRepository(db)
	cngSvc := NewCNGAuctionService(nil)
	syncSvc := NewAuctionWatchlistSyncService(auctionRepo, userRepo, nil, cngSvc, nil, nil)

	user := &models.User{Username: "tester", Email: "tester@example.com", CNGUsername: "user@example.com", CNGPassword: "secret"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	synced, err := syncSvc.SyncUser(user)
	if err != nil {
		t.Fatalf("SyncUser returned error: %v", err)
	}
	if synced != 4 {
		t.Fatalf("synced = %d, want 4", synced)
	}

	assertStatus := func(sourceLotID string, wantStatus models.AuctionLotStatus, wantWinningBid *float64) {
		t.Helper()
		var lot models.AuctionLot
		if err := db.Where("source_lot_id = ?", sourceLotID).First(&lot).Error; err != nil {
			t.Fatalf("lot %s not found: %v", sourceLotID, err)
		}
		if lot.Status != wantStatus {
			t.Fatalf("lot %s status = %q, want %q", sourceLotID, lot.Status, wantStatus)
		}
		if wantWinningBid != nil {
			if lot.WinningBid == nil || *lot.WinningBid != *wantWinningBid {
				t.Fatalf("lot %s WinningBid = %v, want %v", sourceLotID, lot.WinningBid, *wantWinningBid)
			}
		}
	}

	assertStatus("4-ACTIVE", models.AuctionStatusBidding, nil)
	winningBid := 500.0
	assertStatus("4-WON", models.AuctionStatusWon, &winningBid)
	assertStatus("4-LOST", models.AuctionStatusLost, nil)
	assertStatus("4-WATCHEDONLY", models.AuctionStatusPassed, nil)
}

// TestAuctionWatchlistSyncService_SyncNumisBidsSetsAuctionEndTime guards against a
// regression of a bug found during the auction provider audit for issue #482:
// syncNumisBids parsed a sale date but never assigned it to AuctionLot.AuctionEndTime,
// which auction_alert_service.go's bidReminderDue() hard-requires — so bid reminders
// could never fire for any NumisBids lot, unconditionally. NumisBids only exposes a
// coarse sale-wide date (not a precise per-lot close time the way CNG's
// extended_end_time does), so this is a best-effort deadline, not a verified-precise
// one — see specs/_backlog/F021 for the still-open live-verification work.
func TestAuctionWatchlistSyncService_SyncNumisBidsSetsAuctionEndTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registration/login.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			w.Write([]byte(`{"status":"success"}`))
		case "/watchlist":
			// Use real browse-div markup so ParseWatchlist extracts the sale date
			// from the togglewatch header without a per-lot scrape.
			w.Write([]byte(`<div class="heading"><b>My Watch List</b></div>
<div class="togglewatch" id="10749">Test House Sale (20 Apr 2026)</div>
<div class="browse 10749 watch9900001" style="height: 360px;">
  <span class="lot"><a href="/sale/10749/lot/10003">Lot 10003</a></span>
  <span class="summary"><a href="/sale/10749/lot/10003">GREEK Coin</a></span>
</div>`))
		default:
			// Any unexpected request (e.g. a per-lot scrape) fails the test.
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalBase := numisbidsBase
	originalLoginURL := numisbidsLoginURL
	originalWatchlistURL := numisbidsWatchlistURL
	numisbidsBase = server.URL
	numisbidsLoginURL = server.URL + "/registration/login.php"
	numisbidsWatchlistURL = server.URL + "/watchlist"
	defer func() {
		numisbidsBase = originalBase
		numisbidsLoginURL = originalLoginURL
		numisbidsWatchlistURL = originalWatchlistURL
	}()

	db := setupAuctionWatchlistSyncDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	userRepo := repository.NewUserRepository(db)
	nbSvc := NewNumisBidsService(nil)
	syncSvc := NewAuctionWatchlistSyncService(auctionRepo, userRepo, nbSvc, nil, nil, nil)

	user := &models.User{Username: "tester", Email: "tester@example.com", NumisBidsUsername: "user@example.com", NumisBidsPassword: "secret"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if _, err := syncSvc.SyncUser(user); err != nil {
		t.Fatalf("SyncUser returned error: %v", err)
	}

	var lot models.AuctionLot
	if err := db.Where("source = ?", models.AuctionSourceNumisBids).First(&lot).Error; err != nil {
		t.Fatalf("lot not found: %v", err)
	}
	if lot.SaleDate == nil {
		t.Fatal("SaleDate was not set")
	}
	if lot.AuctionEndTime == nil {
		t.Fatal("AuctionEndTime was not set — bid reminders can never fire for this lot")
	}
	if !lot.AuctionEndTime.Equal(*lot.SaleDate) {
		t.Fatalf("AuctionEndTime = %v, want it to match SaleDate %v", lot.AuctionEndTime, lot.SaleDate)
	}
}

// TestAuctionWatchlistSyncService_SyncNumiBidsNoPerLotScrape verifies that syncNumisBids
// does not perform per-lot HTTP requests. The watchlist page carries everything needed
// (image, title, sale name/date, starting price, watchlist ID) without extra round-trips.
// A lot-page request would produce a 404 here, causing the test to fail.
func TestAuctionWatchlistSyncService_SyncNumiBidsNoPerLotScrape(t *testing.T) {
	var lotPageRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registration/login.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			w.Write([]byte(`{"status":"success"}`))
		case "/watchlist":
			w.Write([]byte(`<div class="heading"><b>My Watch List</b></div>
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <img src="//media.numisbids.com/sales/hosted/via/e28/thumb00001.jpg">
  <span class="lot"><a href="/sale/10918/lot/1">Lot 1</a></span>
  <span class="estimate">Starting price: <span class="rateclick">40 EUR</span></span>
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>`))
		default:
			lotPageRequests++
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalBase := numisbidsBase
	originalLoginURL := numisbidsLoginURL
	originalWatchlistURL := numisbidsWatchlistURL
	numisbidsBase = server.URL
	numisbidsLoginURL = server.URL + "/registration/login.php"
	numisbidsWatchlistURL = server.URL + "/watchlist"
	defer func() {
		numisbidsBase = originalBase
		numisbidsLoginURL = originalLoginURL
		numisbidsWatchlistURL = originalWatchlistURL
	}()

	db := setupAuctionWatchlistSyncDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	userRepo := repository.NewUserRepository(db)
	nbSvc := NewNumisBidsService(nil)
	syncSvc := NewAuctionWatchlistSyncService(auctionRepo, userRepo, nbSvc, nil, nil, nil)

	user := &models.User{Username: "tester", Email: "tester@example.com", NumisBidsUsername: "user@example.com", NumisBidsPassword: "secret"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	synced, err := syncSvc.SyncUser(user)
	if err != nil {
		t.Fatalf("SyncUser returned error: %v", err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
	if lotPageRequests > 0 {
		t.Fatalf("syncNumisBids made %d per-lot scrape request(s); want 0 — watchlist provides all needed data", lotPageRequests)
	}

	var lot models.AuctionLot
	if err := db.Where("source = ?", models.AuctionSourceNumisBids).First(&lot).Error; err != nil {
		t.Fatalf("lot not found: %v", err)
	}
	if lot.SourceLotID != "12163211" {
		t.Errorf("SourceLotID = %q, want 12163211", lot.SourceLotID)
	}
	if lot.SaleName != "VIA GmbH E-Auction 28" {
		t.Errorf("SaleName = %q, want VIA GmbH E-Auction 28", lot.SaleName)
	}
	if lot.Estimate == nil || *lot.Estimate != 40 {
		t.Errorf("Estimate = %v, want 40", lot.Estimate)
	}
	if lot.Currency != "EUR" {
		t.Errorf("Currency = %q, want EUR", lot.Currency)
	}
}

// TestAuctionWatchlistSyncService_NotifiesOnceForNewlyTrackedLots covers F031: a sync that
// starts tracking lots the user did not have before must produce exactly one batched
// notification naming every new lot, and a second sync over the same watchlist — where
// nothing is new — must produce none.
func TestAuctionWatchlistSyncService_NotifiesOnceForNewlyTrackedLots(t *testing.T) {
	server := newCNGSyncTestServer(t, "4-OURCUSTOMER", "4-OTHERCUSTOMER")
	defer server.Close()
	restore := overrideCNGURLs(server.URL)
	defer restore()

	db := setupAuctionWatchlistSyncDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	userRepo := repository.NewUserRepository(db)
	syncSvc := NewAuctionWatchlistSyncService(auctionRepo, userRepo, nil, NewCNGAuctionService(nil), nil, nil).
		WithNotifications(newSyncNotificationService(db))

	user := &models.User{Username: "tester", Email: "tester@example.com", CNGUsername: "user@example.com", CNGPassword: "secret"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if _, err := syncSvc.SyncUser(user); err != nil {
		t.Fatalf("SyncUser returned error: %v", err)
	}

	var notifications []models.Notification
	if err := db.Where("type = ?", NotificationTypeAuctionLotsTracked).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to load notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("got %d new-lot notifications, want exactly 1 batched notification", len(notifications))
	}

	notification := notifications[0]
	// The fixture holds four lots, but only the active one the user is bidding on is newly
	// *tracked*; the three that arrive already closed are not new tracking activity.
	if notification.Title != "New Auction Lot Tracked" {
		t.Errorf("title = %q, want single-lot title", notification.Title)
	}
	if !strings.Contains(notification.Message, "Active Lot") {
		t.Errorf("message does not name the new lot: %q", notification.Message)
	}
	if !strings.Contains(notification.Message, "Bidding") {
		t.Errorf("message does not say how the lot is tracked: %q", notification.Message)
	}
	for _, closedLot := range []string{"Won Lot", "Lost Lot", "Watched Only Lot"} {
		if strings.Contains(notification.Message, closedLot) {
			t.Errorf("message names already-closed lot %q: %q", closedLot, notification.Message)
		}
	}
	var newLot models.AuctionLot
	if err := db.Where("source_lot_id = ?", "4-ACTIVE").First(&newLot).Error; err != nil {
		t.Fatalf("new lot not found: %v", err)
	}
	if want := fmt.Sprintf("/auctions?lot=%d", newLot.ID); notification.ReferenceURL != want {
		t.Errorf("ReferenceURL = %q, want %q", notification.ReferenceURL, want)
	}

	// Re-syncing the same watchlist adds nothing, so it must stay silent.
	if _, err := syncSvc.SyncUser(user); err != nil {
		t.Fatalf("second SyncUser returned error: %v", err)
	}
	var afterResync int64
	if err := db.Model(&models.Notification{}).Where("type = ?", NotificationTypeAuctionLotsTracked).Count(&afterResync).Error; err != nil {
		t.Fatalf("failed to count notifications: %v", err)
	}
	if afterResync != 1 {
		t.Fatalf("got %d notifications after re-sync, want 1 — an unchanged watchlist must not re-notify", afterResync)
	}
}

// TestAuctionWatchlistSyncService_SyncsWithoutNotificationService guards the F026 rule that
// syncing never depends on notification wiring: with no notification service attached, the
// sync still refreshes lots instead of panicking or short-circuiting.
func TestAuctionWatchlistSyncService_SyncsWithoutNotificationService(t *testing.T) {
	server := newCNGSyncTestServer(t, "4-OURCUSTOMER", "4-OTHERCUSTOMER")
	defer server.Close()
	restore := overrideCNGURLs(server.URL)
	defer restore()

	db := setupAuctionWatchlistSyncDB(t)
	syncSvc := NewAuctionWatchlistSyncService(
		repository.NewAuctionLotRepository(db),
		repository.NewUserRepository(db),
		nil,
		NewCNGAuctionService(nil),
		nil,
		nil,
	)

	user := &models.User{Username: "tester", Email: "tester@example.com", CNGUsername: "user@example.com", CNGPassword: "secret"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	synced, err := syncSvc.SyncUser(user)
	if err != nil {
		t.Fatalf("SyncUser returned error: %v", err)
	}
	if synced != 4 {
		t.Fatalf("synced = %d, want 4", synced)
	}
}
