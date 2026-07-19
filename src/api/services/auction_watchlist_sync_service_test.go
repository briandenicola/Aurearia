package services

import (
	"net/http"
	"net/http/httptest"
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
	if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.AuctionEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
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
