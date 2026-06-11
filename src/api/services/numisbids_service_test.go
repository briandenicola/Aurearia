package services

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestParseWatchlistAcceptsCurrentAbsoluteLotLinks(t *testing.T) {
	html := `
		<section>
			<h2>Watched Lots in Current Auctions</h2>
			<div class="lot">
				<a href="https://www.numisbids.com/sale/10749/lot/10003">
					GREEK EASTERN EUROPE, Imitations of Alexander III of Macedon.
					1st century BC. Silver Drachm (3.55g).
				</a>
				<img src="//images.numisbids.com/sales/hosted/status/10749/image10003.jpg">
				<span>Estimate: 100 AUD</span>
			</div>
		</section>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}

	lot := lots[0]
	if lot.URL != "https://www.numisbids.com/sale/10749/lot/10003" {
		t.Fatalf("URL = %q, want canonical NumisBids lot URL", lot.URL)
	}
	if lot.SaleID != "10749" {
		t.Fatalf("SaleID = %q, want 10749", lot.SaleID)
	}
	if lot.LotNumber != 10003 {
		t.Fatalf("LotNumber = %d, want 10003", lot.LotNumber)
	}
	if lot.Title == "" {
		t.Fatal("Title is empty")
	}
	if lot.ImageURL != "https://images.numisbids.com/sales/hosted/status/10749/image10003.jpg" {
		t.Fatalf("ImageURL = %q, want protocol-normalized image URL", lot.ImageURL)
	}
	if lot.Estimate == nil || *lot.Estimate != 100 {
		t.Fatalf("Estimate = %v, want 100", lot.Estimate)
	}
	if lot.Currency != "AUD" {
		t.Fatalf("Currency = %q, want AUD", lot.Currency)
	}
}

func TestParseWatchlistAcceptsLegacyLotLinks(t *testing.T) {
	html := `
		<a href='/n.php?p=lot&sid=7996&lot=10003'>Status International Auction 406, Lot 10003</a>
		<span>Estimate: 150 USD</span>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}
	if lots[0].URL != "https://www.numisbids.com/n.php?p=lot&sid=7996&lot=10003" {
		t.Fatalf("URL = %q, want preserved legacy NumisBids lot URL", lots[0].URL)
	}
	if lots[0].SaleID != "7996" {
		t.Fatalf("SaleID = %q, want 7996", lots[0].SaleID)
	}
	if lots[0].LotNumber != 10003 {
		t.Fatalf("LotNumber = %d, want 10003", lots[0].LotNumber)
	}
	if lots[0].Title != "Status International Auction 406, Lot 10003" {
		t.Fatalf("Title = %q, want legacy link title", lots[0].Title)
	}
}

func TestParseWatchlistIgnoresNonNumisBidsAbsoluteLinks(t *testing.T) {
	html := `<a href="https://example.com/sale/10749/lot/10003">External Lot</a>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 0 {
		t.Fatalf("ParseWatchlist returned %d lots, want 0", len(lots))
	}
}

func TestWatchlistDiagnosticsDetectsLoginPrompt(t *testing.T) {
	html := `
		<div class="heading"><b>My Watch List</b></div>
		<p>Already have items on your Watch List?</p>
		<a href="" class="loginreload">Login</a>`

	logger := NewLogger(100)
	diagnostics := NewNumisBidsService(logger).WatchlistDiagnostics(html)
	if !diagnostics.HasLoginPrompt {
		t.Fatal("HasLoginPrompt = false, want true")
	}
	if !diagnostics.HasWatchlistText {
		t.Fatal("HasWatchlistText = false, want true")
	}
	if diagnostics.CandidateLinkCount != 0 {
		t.Fatalf("CandidateLinkCount = %d, want 0", diagnostics.CandidateLinkCount)
	}
}

func TestFetchWatchlistRejectsLoginPromptPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`
			<div class="heading"><b>My Watch List</b></div>
			<p>Already have items on your Watch List?</p>
			<a href="" class="loginreload">Login</a>`))
	}))
	defer server.Close()

	client := server.Client()
	originalWatchlistURL := numisbidsWatchlistURL
	numisbidsWatchlistURL = server.URL
	defer func() { numisbidsWatchlistURL = originalWatchlistURL }()

	logger := NewLogger(100)
	_, err := NewNumisBidsService(logger).FetchWatchlist(client)
	if !errors.Is(err, ErrNumisBidsAuthenticationRequired) {
		t.Fatalf("FetchWatchlist error = %v, want ErrNumisBidsAuthenticationRequired", err)
	}
}

func TestLoginRequiresSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/registration/login.php" {
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			_, _ = w.Write([]byte(`{"status":"error1"}`))
			return
		}
		t.Fatalf("unexpected request path %s", r.URL.Path)
	}))
	defer server.Close()

	originalLoginURL := numisbidsLoginURL
	numisbidsLoginURL = server.URL + "/registration/login.php"
	defer func() { numisbidsLoginURL = originalLoginURL }()

	logger := NewLogger(100)
	_, err := NewNumisBidsService(logger).Login("user@example.com", "password")
	if err == nil {
		t.Fatal("Login returned nil error for non-success NumisBids status")
	}
}

func TestLoginPostsCurrentNumisBidsPayloadAndVerifiesWatchlist(t *testing.T) {
	var loginFormEmail string
	var loginFormPassword string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registration/login.php":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm failed: %v", err)
			}
			loginFormEmail = r.Form.Get("email")
			loginFormPassword = r.Form.Get("password")
			if r.Form.Get("login") != "" {
				t.Fatalf("unexpected legacy login form field = %q", r.Form.Get("login"))
			}
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			_, _ = w.Write([]byte(`{"status":"success"}`))
		case "/watchlist":
			_, _ = w.Write([]byte(`<div class="heading"><b>My Watch List</b></div><a href="/sale/10749/lot/10003">Lot 10003</a>`))
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	originalLoginURL := numisbidsLoginURL
	originalWatchlistURL := numisbidsWatchlistURL
	numisbidsLoginURL = server.URL + "/registration/login.php"
	numisbidsWatchlistURL = server.URL + "/watchlist"
	defer func() {
		numisbidsLoginURL = originalLoginURL
		numisbidsWatchlistURL = originalWatchlistURL
	}()

	logger := NewLogger(100)
	client, err := NewNumisBidsService(logger).Login("user@example.com", "password")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if client == nil {
		t.Fatal("Login returned nil client")
	}
	if loginFormEmail != "user@example.com" {
		t.Fatalf("posted email = %q, want user@example.com", loginFormEmail)
	}
	if loginFormPassword != "password" {
		t.Fatalf("posted password = %q, want password", loginFormPassword)
	}
}

func TestVerifyAuthenticationRejectsWatchlistLoginPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`
			<div class="heading"><b>My Watch List</b></div>
			<p>Already have items on your Watch List?</p>
			<a href="" class="loginreload">Login</a>`))
	}))
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New failed: %v", err)
	}
	client := server.Client()
	client.Jar = jar

	originalWatchlistURL := numisbidsWatchlistURL
	numisbidsWatchlistURL = server.URL
	defer func() { numisbidsWatchlistURL = originalWatchlistURL }()

	logger := NewLogger(100)
	err = NewNumisBidsService(logger).verifyAuthentication(client)
	if err == nil {
		t.Fatal("verifyAuthentication returned nil for login prompt page")
	}
}
