package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func newTestPushoverService(t *testing.T, captured *url.Values) (*PushoverService, func()) {
	t.Helper()

	settingsSvc, _ := newTestSettingsService(t)
	if err := settingsSvc.SetSetting(SettingPushoverAppToken, "app-token"); err != nil {
		t.Fatalf("failed to set pushover token: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		*captured = r.PostForm
		w.WriteHeader(http.StatusOK)
	}))

	svc := NewPushoverService(settingsSvc, NewLogger(10))
	svc.httpClient = server.Client()
	svc.apiURL = server.URL

	return svc, server.Close
}

func TestPushoverServiceSendMessage_CoinOfDayUsesHTMLAndAbsoluteCoinLink(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := buildCoinOfDayPushoverMessage("Coin of the Day", 42, "<Rare & Coin>", "Summary with <script>bad</script>", "https://coins.example.com/")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if got := captured.Get("url"); got != "https://coins.example.com/coin/42" {
		t.Fatalf("url form field = %q, want absolute coin URL", got)
	}

	body := captured.Get("message")
	if !strings.Contains(body, `<b>&lt;Rare &amp; Coin&gt;</b>`) {
		t.Fatalf("message did not escape and bold coin name: %q", body)
	}
	if !strings.Contains(body, `<a href="https://coins.example.com/coin/42">Open coin</a>`) {
		t.Fatalf("message did not include coin link: %q", body)
	}
	if strings.Contains(body, "<script>") {
		t.Fatalf("message contains unescaped script tag: %q", body)
	}
}

func TestPushoverServiceSendMessage_CoinOfDayOmitsBrokenRelativeCoinLinkWhenUnconfigured(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := buildCoinOfDayPushoverMessage("Coin of the Day", 42, "Rare Coin", "Summary", "")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if _, ok := captured["url"]; ok {
		t.Fatalf("url form field should be omitted when public app URL is unconfigured")
	}

	body := captured.Get("message")
	if !strings.Contains(body, `<b>Rare Coin</b>`) {
		t.Fatalf("message did not keep HTML formatting: %q", body)
	}
	if strings.Contains(body, `href="/coin/42"`) || strings.Contains(body, `/coin/42`) {
		t.Fatalf("message contains broken relative coin link: %q", body)
	}
}

func TestPushoverServiceSendNotification_RemainsPlain(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	if err := svc.SendNotification("user-key", "Ancient Coins", "Pushover notifications are working!", ""); err != nil {
		t.Fatalf("SendNotification() error = %v", err)
	}

	if got := captured.Get("html"); got != "" {
		t.Fatalf("html form field = %q, want empty for plain notification", got)
	}
	if got := captured.Get("message"); got != "Pushover notifications are working!" {
		t.Fatalf("message form field = %q, want plain text", got)
	}
	if _, ok := captured["url"]; ok {
		t.Fatalf("url form field should be omitted when no reference URL is provided")
	}
}

func TestPushoverServiceSendMessage_SetsURLTitleWhenURLPresent(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := PushoverMessage{
		UserKey:  "user-key",
		Title:    "Auction Price Alert",
		Message:  "Julia Domna AR Denarius\nCNG - Keystone 17 (Lot 95)",
		URL:      "https://cngcoins.com/lot/95",
		URLTitle: "View auction lot",
	}

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("url"); got != "https://cngcoins.com/lot/95" {
		t.Fatalf("url form field = %q, want lot url", got)
	}
	if got := captured.Get("url_title"); got != "View auction lot" {
		t.Fatalf("url_title form field = %q, want %q", got, "View auction lot")
	}
}

func TestPushoverServiceSendMessage_OmitsURLTitleWhenURLBlank(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := PushoverMessage{
		UserKey:  "user-key",
		Title:    "Auction Bid Reminder",
		Message:  "Untitled lot\nCNG - Keystone 17 (Lot 95)",
		URLTitle: "View auction lot",
	}

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if _, ok := captured["url"]; ok {
		t.Fatalf("url form field should be omitted when URL is blank")
	}
	if _, ok := captured["url_title"]; ok {
		t.Fatalf("url_title form field should be omitted when URL is blank")
	}
}

func newTrackedLot(id uint, title string, lotNumber int, status models.AuctionLotStatus) models.AuctionLot {
	return models.AuctionLot{
		ID:           id,
		Title:        title,
		AuctionHouse: "CNG",
		SaleName:     "Keystone 17",
		LotNumber:    lotNumber,
		Status:       status,
	}
}

func TestPushoverServiceSendMessage_NewlyTrackedLotsUseHTMLAndAbsoluteAppLinks(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	lots := []models.AuctionLot{
		newTrackedLot(11, "Julia Domna AR Denarius", 95, models.AuctionStatusWatching),
		newTrackedLot(12, "<Trajan> & Sons AV Aureus", 96, models.AuctionStatusBidding),
	}
	message := buildAuctionLotsTrackedPushoverMessage(lots, "https://coins.example.com/")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if got := captured.Get("title"); got != "2 New Auction Lots Tracked" {
		t.Fatalf("title form field = %q, want batched title", got)
	}
	// A batch has no single lot to open, so the action link lands on the auctions list.
	if got := captured.Get("url"); got != "https://coins.example.com/auctions" {
		t.Fatalf("url form field = %q, want auctions list URL", got)
	}

	body := captured.Get("message")
	if !strings.Contains(body, "<b>Julia Domna AR Denarius</b>") {
		t.Errorf("message did not bold the coin name: %q", body)
	}
	if !strings.Contains(body, "CNG - Keystone 17 (Lot 95) — Watching") {
		t.Errorf("message did not carry auction, lot number and tracking state: %q", body)
	}
	if !strings.Contains(body, "CNG - Keystone 17 (Lot 96) — Bidding") {
		t.Errorf("message did not mark the bid-on lot as bidding: %q", body)
	}
	if !strings.Contains(body, `<a href="https://coins.example.com/auctions?lot=11">View lot in Aurearia</a>`) {
		t.Errorf("message did not deep-link the first lot: %q", body)
	}
	if !strings.Contains(body, `<a href="https://coins.example.com/auctions?lot=12">View lot in Aurearia</a>`) {
		t.Errorf("message did not deep-link the second lot: %q", body)
	}
	// Lot titles are scraped from third-party auction sites, so they must never reach the
	// HTML body unescaped.
	if strings.Contains(body, "<Trajan>") {
		t.Errorf("message contains unescaped scraped markup: %q", body)
	}
	if !strings.Contains(body, "&lt;Trajan&gt; &amp; Sons AV Aureus") {
		t.Errorf("message did not escape the scraped lot title: %q", body)
	}
}

// NumisBids watchlist rows have no auction house, only a sale name; the notification names
// the provider rather than falling back to a bare "Auction".
func TestBuildAuctionLotsTrackedPushoverMessage_NamesProviderWhenAuctionHouseMissing(t *testing.T) {
	lot := newTrackedLot(3, "KELTEN, GALLIA Aedui, Quinar", 1, models.AuctionStatusWatching)
	lot.AuctionHouse = ""
	lot.Source = models.AuctionSourceNumisBids
	lot.SaleName = "VIA GmbH E-Auction 28"

	message := buildAuctionLotsTrackedPushoverMessage([]models.AuctionLot{lot}, "https://coins.example.com")

	if !strings.Contains(message.Message, "NumisBids - VIA GmbH E-Auction 28 (Lot 1)") {
		t.Errorf("message did not name the provider as the auction: %q", message.Message)
	}
}

func TestPushoverServiceSendMessage_SingleNewlyTrackedLotLinksStraightToTheLot(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := buildAuctionLotsTrackedPushoverMessage(
		[]models.AuctionLot{newTrackedLot(7, "Nero AR Denarius", 12, models.AuctionStatusBidding)},
		"https://coins.example.com",
	)
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("title"); got != "New Auction Lot Tracked" {
		t.Fatalf("title form field = %q, want single-lot title", got)
	}
	if got := captured.Get("url"); got != "https://coins.example.com/auctions?lot=7" {
		t.Fatalf("url form field = %q, want the lot's own app URL", got)
	}
	if got := captured.Get("url_title"); got != "View auction lot" {
		t.Fatalf("url_title form field = %q, want %q", got, "View auction lot")
	}
}

func TestBuildAuctionLotsTrackedPushoverMessage_OmitsLinksWhenPublicAppURLUnconfigured(t *testing.T) {
	message := buildAuctionLotsTrackedPushoverMessage(
		[]models.AuctionLot{newTrackedLot(7, "Nero AR Denarius", 12, models.AuctionStatusWatching)},
		"",
	)

	if message.URL != "" {
		t.Errorf("URL = %q, want empty when no public app URL is configured", message.URL)
	}
	if strings.Contains(message.Message, "href") || strings.Contains(message.Message, "/auctions") {
		t.Errorf("message contains a broken relative link: %q", message.Message)
	}
	if !strings.Contains(message.Message, "<b>Nero AR Denarius</b>") {
		t.Errorf("message dropped the lot details along with the link: %q", message.Message)
	}
}

// A watchlist can grow by dozens of lots at once (e.g. the first sync after a user watches a
// whole sale). Pushover rejects any message over pushoverMessageLimit outright, so the batch
// must trim itself rather than lose the notification entirely.
func TestBuildAuctionLotsTrackedPushoverMessage_TrimsOversizedBatches(t *testing.T) {
	lots := make([]models.AuctionLot, 0, 60)
	for i := 1; i <= 60; i++ {
		lots = append(lots, newTrackedLot(uint(i), fmt.Sprintf("Roman Imperial Denarius %d", i), i, models.AuctionStatusWatching))
	}

	message := buildAuctionLotsTrackedPushoverMessage(lots, "https://coins.example.com")

	if len(message.Message) > pushoverMessageLimit {
		t.Fatalf("message length = %d, want <= %d", len(message.Message), pushoverMessageLimit)
	}
	if !strings.Contains(message.Message, "more lot(s) omitted") {
		t.Errorf("trimmed message does not say lots were omitted: %q", message.Message)
	}
	if got := message.Title; got != "60 New Auction Lots Tracked" {
		t.Errorf("title = %q, want the full new-lot count even when the body is trimmed", got)
	}
}

func TestPushoverServiceSendMessage_BiddingLotsUseHTMLAndCarryBidState(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	currentBid := 90.0
	maxBid := 200.0
	lot := newTrackedLot(11, "Julia Domna AR Denarius", 95, models.AuctionStatusBidding)
	lot.CurrentBid = &currentBid
	lot.MaxBid = &maxBid

	message := buildAuctionLotsBiddingPushoverMessage([]models.AuctionLot{lot}, "https://coins.example.com")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if got := captured.Get("title"); got != "Now Bidding on an Auction Lot" {
		t.Fatalf("title form field = %q, want single-lot bidding title", got)
	}
	if got := captured.Get("url"); got != "https://coins.example.com/auctions?lot=11" {
		t.Fatalf("url form field = %q, want the lot's own app URL", got)
	}

	body := captured.Get("message")
	if !strings.Contains(body, "<b>Julia Domna AR Denarius</b>") {
		t.Errorf("message did not bold the coin name: %q", body)
	}
	if !strings.Contains(body, "CNG - Keystone 17 (Lot 95)") {
		t.Errorf("message did not carry the auction and lot number: %q", body)
	}
	if !strings.Contains(body, "current high bid 90.00 USD · your max bid 200.00 USD") {
		t.Errorf("message did not carry the bid state: %q", body)
	}
	if !strings.Contains(body, `<a href="https://coins.example.com/auctions?lot=11">View lot in Aurearia</a>`) {
		t.Errorf("message did not deep-link the lot: %q", body)
	}
}

func TestBuildAuctionLotsBiddingPushoverMessage_BatchesAndEscapes(t *testing.T) {
	first := newTrackedLot(11, "Julia Domna AR Denarius", 95, models.AuctionStatusBidding)
	second := newTrackedLot(12, "<Trajan> & Sons AV Aureus", 96, models.AuctionStatusBidding)

	message := buildAuctionLotsBiddingPushoverMessage([]models.AuctionLot{first, second}, "https://coins.example.com")

	if message.Title != "Now Bidding on 2 Auction Lots" {
		t.Errorf("title = %q, want batched bidding title", message.Title)
	}
	// A batch has no single lot to open, so the action link lands on the auctions list.
	if message.URL != "https://coins.example.com/auctions" {
		t.Errorf("URL = %q, want auctions list URL", message.URL)
	}
	if strings.Contains(message.Message, "<Trajan>") {
		t.Errorf("message contains unescaped scraped markup: %q", message.Message)
	}
	// No provider bid data at all still reads sensibly, via the shared bid formatter.
	if !strings.Contains(message.Message, "current high bid unavailable") {
		t.Errorf("message did not fall back for a missing bid: %q", message.Message)
	}
}

func TestBuildAuctionLotsOutbidPushoverMessage(t *testing.T) {
	closesAt := time.Now().Add(50 * time.Hour)
	currentBid := 90.0
	maxBid := 200.0
	lots := []models.AuctionLot{{
		ID:             7,
		Title:          "PERSIA, Achaemenid Empire. temp. Darios I to Xerxes I. Circa 505-480 BC. AR Siglos.",
		AuctionHouse:   "Classical Numismatic Group",
		SaleName:       "Keystone Auction 18",
		LotNumber:      24,
		CurrentBid:     &currentBid,
		MaxBid:         &maxBid,
		Currency:       "USD",
		AuctionEndTime: &closesAt,
		SourceURL:      "https://auctions.cngcoins.com/lots/view/4-ACTIVE",
	}}

	message := buildAuctionLotsOutbidPushoverMessage(lots, "https://coins.example.com")

	if !message.HTML {
		t.Fatal("outbid push is not flagged as HTML")
	}
	if message.Title != "Outbid on an Auction Lot" {
		t.Fatalf("title = %q, want the single-lot outbid title", message.Title)
	}
	if !strings.Contains(message.Message, `<b>PERSIA, Achaemenid Empire</b> (<a href="https://auctions.cngcoins.com/lots/view/4-ACTIVE">Lot 24</a>)`) {
		t.Fatalf("message does not lead with a linked lot headline: %q", message.Message)
	}
	if !strings.Contains(message.Message, "current high bid 90.00 USD · your max bid 200.00 USD") {
		t.Fatalf("message does not carry the bid state: %q", message.Message)
	}
	if !strings.Contains(message.Message, "Closes in 2d 1h") {
		t.Fatalf("message does not say how long is left: %q", message.Message)
	}
	if message.URL != "https://coins.example.com/auctions?lot=7" || message.URLTitle != "View auction lot" {
		t.Fatalf("action link = %q / %q, want the deep link to the lot", message.URL, message.URLTitle)
	}
}

func TestBuildAuctionLotsOutbidPushoverMessageEscapesScrapedTitles(t *testing.T) {
	lots := []models.AuctionLot{{
		ID: 7, Title: "<script>alert(1)</script> Siglos", AuctionHouse: "CNG & Co", SaleName: "Keystone 18", LotNumber: 24,
	}}

	message := buildAuctionLotsOutbidPushoverMessage(lots, "")

	if strings.Contains(message.Message, "<script>") {
		t.Fatalf("scraped title reached the HTML body unescaped: %q", message.Message)
	}
	if !strings.Contains(message.Message, "CNG &amp; Co - Keystone 18") {
		t.Fatalf("sale label not escaped: %q", message.Message)
	}
	// No public app URL configured: the body still renders, just without the app link.
	if strings.Contains(message.Message, "View lot in Aurearia") {
		t.Fatalf("app link emitted without a public app URL: %q", message.Message)
	}
}

func TestAuctionLotClosesIn(t *testing.T) {
	inTwoDays := time.Now().Add(50 * time.Hour)
	inHours := time.Now().Add(3*time.Hour + 30*time.Minute)
	inMinutes := time.Now().Add(20 * time.Minute)
	past := time.Now().Add(-time.Hour)
	saleDate := time.Now().Add(26 * time.Hour)

	tests := []struct {
		name string
		lot  models.AuctionLot
		want string
	}{
		// Remaining time truncates rather than rounds, so 50h reads as 2d 1h — the same
		// convention the on-screen countdown uses.
		{name: "days out", lot: models.AuctionLot{AuctionEndTime: &inTwoDays}, want: "Closes in 2d 1h"},
		{name: "hours out", lot: models.AuctionLot{AuctionEndTime: &inHours}, want: "Closes in 3h 29m"},
		{name: "minutes out", lot: models.AuctionLot{AuctionEndTime: &inMinutes}, want: "Closes in 19m"},
		{name: "already closed", lot: models.AuctionLot{AuctionEndTime: &past}, want: "Closed"},
		{name: "falls back to the sale date", lot: models.AuctionLot{SaleDate: &saleDate}, want: "Closes in 1d 1h"},
		{name: "no close time at all", lot: models.AuctionLot{}, want: "Closing time unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := auctionLotClosesIn(test.lot); got != test.want {
				t.Fatalf("auctionLotClosesIn() = %q, want %q", got, test.want)
			}
		})
	}
}
