package services

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCNGAuctionService_ParseLotPage(t *testing.T) {
	svc := NewCNGAuctionService(nil)
	lot, err := svc.parseLotPage(cngLotFixture())
	if err != nil {
		t.Fatalf("parseLotPage returned error: %v", err)
	}

	if lot.SourceLotID != "4-LOTID" {
		t.Fatalf("SourceLotID = %q, want 4-LOTID", lot.SourceLotID)
	}
	if lot.SourceSaleID != "4-SALEID" {
		t.Fatalf("SourceSaleID = %q, want 4-SALEID", lot.SourceSaleID)
	}
	if lot.LotNumber != 43 {
		t.Fatalf("LotNumber = %d, want 43", lot.LotNumber)
	}
	if lot.URL != "https://auctions.cngcoins.com/lots/view/4-LOTID/test-lot" {
		t.Fatalf("URL = %q", lot.URL)
	}
	if lot.ImageURL != "https://images.example/43_1.jpg" {
		t.Fatalf("ImageURL = %q", lot.ImageURL)
	}
	if lot.Estimate == nil || *lot.Estimate != 100 {
		t.Fatalf("Estimate = %v, want 100", lot.Estimate)
	}
	if lot.CurrentBid == nil || *lot.CurrentBid != 60 {
		t.Fatalf("CurrentBid = %v, want 60", lot.CurrentBid)
	}
	if lot.Currency != "USD" {
		t.Fatalf("Currency = %q, want USD", lot.Currency)
	}
	if lot.SaleName != "Electronic Auction 612" {
		t.Fatalf("SaleName = %q", lot.SaleName)
	}
	if lot.Description == "" || strings.Contains(lot.Description, "<b>") {
		t.Fatalf("Description was not cleaned: %q", lot.Description)
	}
}

func TestCNGAuctionService_ParseWatchlist(t *testing.T) {
	svc := NewCNGAuctionService(nil)
	lots := svc.ParseWatchlist(cngWatchlistFixture())

	if len(lots) != 2 {
		t.Fatalf("ParseWatchlist returned %d lots, want 2", len(lots))
	}
	if lots[0].SourceLotID != "4-LOT1" || lots[1].SourceLotID != "4-LOT2" {
		t.Fatalf("unexpected source lot IDs: %#v", lots)
	}
	if lots[1].Currency != "USD" {
		t.Fatalf("second lot currency = %q, want USD fallback", lots[1].Currency)
	}
}

func TestCNGAuctionService_LoginAndFetchWatchlist(t *testing.T) {
	var loggedIn bool
	var watchedLotRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			if !loggedIn {
				w.Write([]byte(`viewVars = {"me":null};`))
				return
			}
			w.Write([]byte(`viewVars = {"me":{"row_id":"user"}};`))
			return
		case "/login":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form action="/login"><input name="username"><input name="password"></form>`))
				return
			}
			if r.Method == http.MethodPost {
				if err := r.ParseForm(); err != nil {
					t.Fatalf("ParseForm failed: %v", err)
				}
				if r.Form.Get("username") != "user@example.com" || r.Form.Get("password") != "secret" || r.Form.Get("Login") != "Login" {
					t.Fatalf("unexpected login form: %#v", r.Form)
				}
				loggedIn = true
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		case "/ajax/refresh-me":
			if !loggedIn {
				w.Write([]byte(`null`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"row_id":"user"}`))
			return
		case "/watched-lots":
			if !loggedIn {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			watchedLotRequests = append(watchedLotRequests, r.URL.RawQuery)
			if r.URL.Query().Get("page") == "2" {
				w.Write([]byte(cngWatchlistPageFixture(2, 3, 2, []string{"4-LOT3"})))
				return
			}
			w.Write([]byte(cngWatchlistPageFixture(1, 3, 2, []string{"4-LOT1", "4-LOT2"})))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	restore := overrideCNGURLs(server.URL)
	defer restore()

	svc := NewCNGAuctionService(nil)
	client, err := svc.Login("user@example.com", "secret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	raw, err := svc.FetchWatchlist(client)
	if err != nil {
		t.Fatalf("FetchWatchlist returned error: %v", err)
	}
	if got := svc.ParseWatchlist(raw); len(got) != 2 {
		t.Fatalf("parsed %d watched lots, want 2", len(got))
	}
	lots, err := svc.FetchWatchlistLots(client)
	if err != nil {
		t.Fatalf("FetchWatchlistLots returned error: %v", err)
	}
	if len(lots) != 3 {
		t.Fatalf("FetchWatchlistLots returned %d lots, want 3", len(lots))
	}
	if len(watchedLotRequests) != 3 || watchedLotRequests[2] != "page=2" {
		t.Fatalf("unexpected watched-lots requests: %#v", watchedLotRequests)
	}
}

func TestCNGAuctionService_LoginInvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<form action="/login"><input name="username"><input name="password"></form>`))
		case "/ajax/refresh-me":
			w.Write([]byte(`null`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	restore := overrideCNGURLs(server.URL)
	defer restore()

	svc := NewCNGAuctionService(nil)
	if _, err := svc.Login("bad@example.com", "wrong"); err == nil {
		t.Fatal("Login succeeded, want error")
	}
}

func TestCNGAuctionService_ScrapeLotRejectsNonCNGURL(t *testing.T) {
	svc := NewCNGAuctionService(nil)
	tests := []string{
		"http://auctions.cngcoins.com/lots/view/4-LOT/test",
		"https://example.com/lots/view/4-LOT/test",
		"https://localhost/lots/view/4-LOT/test",
		"https://127.0.0.1/lots/view/4-LOT/test",
		"https://auctions.cngcoins.com.evil.example/lots/view/4-LOT/test",
		"https://attacker@example.com@auctions.cngcoins.com/lots/view/4-LOT/test",
		"https://auctions.cngcoins.com:8443/lots/view/4-LOT/test",
		"https://auctions.cngcoins.com/lots/view/4-LOT/test?next=https://example.com",
		"https://auctions.cngcoins.com/lots/view/4-LOT/test#fragment",
		"https://auctions.cngcoins.com/auctions/4-SALE/test",
		"://bad",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := svc.ScrapeLot(rawURL); err == nil {
				t.Fatal("ScrapeLot succeeded, want URL validation error")
			}
		})
	}
}

func TestCanonicalCNGLotPath(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "basic lot path",
			raw:  "https://auctions.cngcoins.com/lots/view/4-LOT/test-lot",
			want: "/lots/view/4-LOT/test-lot",
		},
		{
			name: "standard https port",
			raw:  "https://auctions.cngcoins.com:443/lots/view/4-LOT/test-lot",
			want: "/lots/view/4-LOT/test-lot",
		},
		{
			name: "trailing slash",
			raw:  "https://auctions.cngcoins.com/lots/view/4-LOT/test-lot/",
			want: "/lots/view/4-LOT/test-lot/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalCNGLotPath(tt.raw)
			if err != nil {
				t.Fatalf("canonicalCNGLotPath returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("canonicalCNGLotPath = %q, want %q", got, tt.want)
			}
		})
	}
}

func overrideCNGURLs(base string) func() {
	oldLoginURL := cngLoginURL
	oldWatchlistURL := cngWatchlistURL
	oldRefreshMeURL := cngRefreshMeURL
	cngLoginURL = base + "/login"
	cngWatchlistURL = base + "/watched-lots"
	cngRefreshMeURL = base + "/ajax/refresh-me"
	return func() {
		cngLoginURL = oldLoginURL
		cngWatchlistURL = oldWatchlistURL
		cngRefreshMeURL = oldRefreshMeURL
	}
}

func cngLotFixture() string {
	return `<!doctype html><html><script>
viewVars = {
  "currentRouteName":"lot-detail-slug",
  "lot":{
    "row_id":"4-LOTID",
    "lot_number":43,
    "lot_number_extension":"",
    "title":"CARTHAGE. Half-Shekel. Good VF.",
    "description":"<b>CARTHAGE.</b> Second Punic War. Good VF.",
    "estimate_low":"100.00",
    "estimate_high":"150.00",
    "currency_code":"USD",
    "starting_price":"60.00",
    "sold_price":null,
    "status":"active",
    "_detail_url":"/lots/view/4-LOTID/test-lot",
    "cover_thumbnail":"",
    "images":[{"detail_url":"https://images.example/43_1.jpg","thumbnail_url":"https://images.example/thumb.jpg"}],
    "auction":{
      "row_id":"4-SALEID",
      "title":"Electronic Auction 612",
      "currency_code":"USD",
      "time_start":"2026-06-17T20:00:00Z",
      "effective_end_time":"2026-07-01T21:15:00Z"
    }
  }
};
</script></html>`
}

func cngWatchlistFixture() string {
	return cngWatchlistPageFixture(1, 2, 48, []string{"4-LOT1", "4-LOT2"})
}

func cngWatchlistPageFixture(page, total, pageSize int, lotIDs []string) string {
	lots := make([]string, 0, len(lotIDs))
	for index, lotID := range lotIDs {
		lotNumber := ((page - 1) * pageSize) + index + 1
		currency := `"currency_code":"USD",`
		if lotID == "4-LOT2" {
			currency = ""
		}
		lots = append(lots, `{
        "row_id":"`+lotID+`",
        "lot_number":`+strconv.Itoa(lotNumber)+`,
        "title":"Lot `+strconv.Itoa(lotNumber)+`",
        "truncated_description":"<b>Lot</b> description",
        "estimate_low":"100.00",
        `+currency+`
        "starting_price":"60.00",
        "_detail_url":"/lots/view/`+lotID+`/lot-`+strconv.Itoa(lotNumber)+`",
        "cover_thumbnail":"https://images.example/`+strconv.Itoa(lotNumber)+`.jpg",
        "auction":{"row_id":"4-SALEID","title":"Electronic Auction 612","currency_code":"USD","effective_end_time":"2026-07-01T21:15:00Z"}
      }`)
	}
	return `<!doctype html><html><script>
viewVars = {
  "currentRouteName":"watched-lots-index",
  "lots":{
    "query_info":{"total_num_results":` + strconv.Itoa(total) + `,"page_size":` + strconv.Itoa(pageSize) + `},
    "result_page":[
      ` + strings.Join(lots, ",") + `
    ]
  }
};
</script></html>`
}
