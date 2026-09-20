package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type auctionSyncTransport func(*http.Request) (*http.Response, error)

func (f auctionSyncTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestAuctionSyncManualScheduledParity(t *testing.T) {
	for _, source := range []models.AuctionSource{models.AuctionSourceCNG, models.AuctionSourceNumisBids} {
		t.Run(string(source), func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			sqlDB.SetMaxOpenConns(1)
			defer sqlDB.Close()
			if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.AuctionEvent{}, &models.QuickAccessPin{}, &models.Notification{}, &models.AppSetting{}); err != nil {
				t.Fatal(err)
			}
			lotRepo := repository.NewAuctionLotRepository(db)
			userRepo := repository.NewUserRepository(db)
			quick := services.NewQuickAccessService(repository.NewQuickAccessRepository(db), nil)
			logger := services.NewLogger(10)
			notifier := services.NewNotificationService(repository.NewNotificationRepository(db), nil, userRepo,
				services.NewPushoverService(services.NewSettingsService(repository.NewSettingsRepository(db)), logger), logger)
			nb := services.NewNumisBidsService(nil)
			cng := services.NewCNGAuctionService(nil)
			sync := services.NewAuctionWatchlistSyncService(lotRepo, userRepo, nb, cng, nil, logger).WithQuickAccessSupport(quick).WithNotifications(notifier)
			handler := NewAuctionLotHandler(lotRepo, services.NewAuctionLotService(lotRepo, nil), userRepo, nb, cng, logger).WithWatchlistSync(sync)
			closed := false
			previous := http.DefaultTransport
			http.DefaultTransport = auctionSyncTransport(func(req *http.Request) (*http.Response, error) {
				var body string
				switch req.URL.Host + req.URL.Path {
				case "auctions.cngcoins.com/login":
					body = `<form></form>`
				case "auctions.cngcoins.com/ajax/refresh-me":
					body = `{"row_id":"OUR"}`
				case "auctions.cngcoins.com/watched-lots":
					status := "active"
					if closed {
						status = "sold"
					}
					body = fmt.Sprintf(`<script>viewVars = {"currentRouteName":"watched-lots-index","lots":{"query_info":{"total_num_results":1,"page_size":50},"result_page":[{"row_id":"LOT","lot_number":1,"title":"Fixture","status":"%s","_detail_url":"/lots/view/LOT/fixture","absentee_bid":{"max_bid":"200"},"timed_auction_bid":{"amount":"100","registration":{"customer":{"row_id":"OTHER"}}},"auction":{"row_id":"SALE","title":"Sale","currency_code":"USD","effective_end_time":"2099-01-01T00:00:00Z"}}]}};</script>`, status)
				case "www.numisbids.com/registration/login.php":
					body = `{"status":"success"}`
				case "www.numisbids.com/watchlist":
					date := "1 Jan 2099"
					if closed {
						date = "1 Jan 2020"
					}
					body = fmt.Sprintf(`<div class="heading"><b>My Watch List</b></div><div class="togglewatch" id="123">Sale (%s)</div><div class="browse 123 watch456"><span class="lot"><a href="/sale/123/lot/1">Lot 1</a></span><span class="summary"><a href="/sale/123/lot/1">Fixture</a></span></div>`, date)
				default:
					return nil, fmt.Errorf("unexpected provider/per-lot request: %s", req.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Set-Cookie": {"PHPSESSID=fixture; Path=/"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})
			defer func() { http.DefaultTransport = previous }()
			users := []models.User{
				{Username: "manual", Email: "manual@example.test"},
				{Username: "scheduled", Email: "scheduled@example.test"},
			}
			for i := range users {
				if source == models.AuctionSourceCNG {
					users[i].CNGUsername, users[i].CNGPassword = "fixture", "fixture"
				} else {
					users[i].NumisBidsUsername, users[i].NumisBidsPassword = "fixture", "fixture"
				}
				if err := db.Create(&users[i]).Error; err != nil {
					t.Fatal(err)
				}
			}
			for _, terminal := range []bool{false, true} {
				closed = terminal
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest(http.MethodPost, "/auctions/sync?source="+string(source), nil)
				c.Set("userId", users[0].ID)
				handler.SyncWatchlist(c)
				if w.Code != http.StatusOK {
					t.Fatalf("manual: %d %s", w.Code, w.Body.String())
				}
				var response struct {
					Synced int                 `json:"synced"`
					Lots   []models.AuctionLot `json:"lots"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Synced != 1 || len(response.Lots) != 1 {
					t.Fatalf("unexpected response: %s", w.Body.String())
				}
				if n, err := sync.SyncUser(&users[1]); err != nil || n != 1 {
					t.Fatalf("scheduled: %d %v", n, err)
				}
				stored, _, err := lotRepo.List(users[1].ID, repository.AuctionLotListFilters{Limit: 50, Page: 1})
				if err != nil || len(stored) != 1 {
					t.Fatalf("stored: %v %v", stored, err)
				}
				manual, scheduled := response.Lots[0], stored[0]
				if manual.EventID == nil || scheduled.EventID == nil {
					t.Fatal("calendar links missing")
				}
				if !terminal && source == models.AuctionSourceCNG && !manual.IsOutbid {
					t.Fatal("manual sync lost outbid state")
				}
				if terminal {
					want := models.AuctionStatusLost
					if source == models.AuctionSourceNumisBids {
						want = models.AuctionStatusPassed
					}
					if manual.Status != want {
						t.Fatalf("terminal outcome: %s", manual.Status)
					}
				}
				normalize := func(lot *models.AuctionLot) {
					lot.ID, lot.UserID = 0, 0
					lot.EventID, lot.Event = nil, nil
					lot.CreatedAt, lot.UpdatedAt = time.Time{}, time.Time{}
				}
				normalize(&manual)
				normalize(&scheduled)
				if !reflect.DeepEqual(manual, scheduled) {
					t.Fatalf("manual/scheduled differ:\n%+v\n%+v", manual, scheduled)
				}
				for i, lot := range []models.AuctionLot{response.Lots[0], stored[0]} {
					if !terminal {
						if _, _, err := quick.Pin(users[i].ID, models.QuickAccessTargetAuctionLot, lot.ID); err != nil {
							t.Fatal(err)
						}
					} else {
						var count int64
						if err := db.Model(&models.QuickAccessPin{}).Where("user_id = ?", users[i].ID).Count(&count).Error; err != nil {
							t.Fatal(err)
						}
						if count != 0 {
							t.Fatal("terminal pin retained")
						}
					}
				}
				var notifications int64
				if err := db.Model(&models.Notification{}).Where("user_id = ?", users[0].ID).Count(&notifications).Error; err != nil {
					t.Fatal(err)
				}
				if notifications != 0 {
					t.Fatal("manual sync sent notifications")
				}
			}
		})
	}
}
