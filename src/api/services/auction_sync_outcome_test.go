package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func TestAuctionWatchlistSyncService_CNGUnknownOutcomeRecovers(t *testing.T) {
	for _, scenario := range []string{"customer unavailable", "winner missing", "status missing", "active past date", "unknown status"} {
		t.Run(scenario, func(t *testing.T) {
			var recovered atomic.Bool
			var profileCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/", "/login":
					w.Write([]byte(`viewVars = {"me":null};`))
				case "/ajax/refresh-me":
					if profileCalls.Add(1)%2 == 0 && scenario == "customer unavailable" && !recovered.Load() {
						http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
						return
					}
					w.Write([]byte(`{"row_id":"4-OURCUSTOMER"}`))
				case "/watched-lots":
					fixture := cngSyncFixture("4-OURCUSTOMER", "4-OTHER")
					if !recovered.Load() {
						switch scenario {
						case "winner missing":
							fixture = strings.ReplaceAll(fixture, `"customer":{"row_id":"4-OURCUSTOMER"}`, `"customer":{}`)
						case "status missing":
							fixture = strings.ReplaceAll(fixture, `"status":"sold"`, `"status":""`)
						case "active past date":
							fixture = strings.ReplaceAll(fixture, `"status":"sold"`, `"status":"active"`)
						case "unknown status":
							fixture = strings.ReplaceAll(fixture, `"status":"sold"`, `"status":"pending"`)
						}
					}
					w.Write([]byte(fixture))
				default:
					t.Errorf("unexpected per-lot request: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			restore := overrideCNGURLs(server.URL)
			defer restore()
			db := setupAuctionWatchlistSyncDB(t)
			repo := repository.NewAuctionLotRepository(db)
			svc := NewAuctionWatchlistSyncService(repo, repository.NewUserRepository(db), nil, NewCNGAuctionService(nil), nil, nil)
			user := &models.User{ID: 42, CNGUsername: "fixture", CNGPassword: "fixture"}
			for _, want := range []models.AuctionLotStatus{models.AuctionStatusBidding, models.AuctionStatusWon} {
				if _, err := svc.SyncUser(user); err != nil {
					t.Fatal(err)
				}
				lot, err := repo.GetBySourceURL(models.AuctionSourceCNG, server.URL+"/lots/view/4-WON/won-lot", user.ID)
				if err != nil {
					t.Fatal(err)
				}
				if lot.Status != want {
					t.Fatalf("status = %s, want %s (recovered=%t)", lot.Status, want, recovered.Load())
				}
				recovered.Store(true)
			}
		})
	}
}
