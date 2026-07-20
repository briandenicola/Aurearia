package services

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ── HTTP / encoding guard ─────────────────────────────────────────────────────

// TestNumisBidsDefaultHeadersExcludesBrotli ensures the default request headers
// restrict Accept-Encoding to gzip/deflate and never include br (Brotli).
// Rationale: NumisBids is served through Cloudflare. When a Chrome user-agent is
// present, Cloudflare returns content-encoding: br even if the client only asked
// for gzip — verified from a real /watchlist response (2026-07). Go's net/http has
// no native Brotli decompressor, so accepting br would silently produce garbled
// HTML. Explicit exclusion of br is the safe default; Brotli support can be added
// if the dependency is approved.
func TestNumisBidsDefaultHeadersExcludesBrotli(t *testing.T) {
	headers := numisbidsDefaultHeaders()
	ae := strings.ToLower(headers["Accept-Encoding"])
	if strings.Contains(ae, "br") {
		t.Fatalf("Accept-Encoding = %q: must not include \"br\" — Go cannot decompress Brotli and Cloudflare sends it based on the Chrome user-agent", ae)
	}
	if !strings.Contains(ae, "gzip") {
		t.Fatalf("Accept-Encoding = %q: should include gzip so responses are still compressed in transit", ae)
	}
}

// ── ParseWatchlist tests (real markup shape, 2026-07) ─────────────────────────

// numisbidsLotsFixture returns minimal watchlist HTML matching the current
// NumisBids markup (browse divs + togglewatch headers). Used by tests that need
// a realistic but controlled HTML fragment.
func numisbidsLotsFixture() string {
	return `
<div class="togglewatch" id="10749">Test Auction House Sale 12 (20 Apr 2026)</div>
<div class="browse 10749 watch9900001" style="height: 360px;">
  <img src="//images.numisbids.com/sales/hosted/status/10749/image10003.jpg">
  <span class="lot"><a href="/sale/10749/lot/10003">Lot 10003</a></span>
  <span class="estimate">Starting price: <span class="rateclick" data-eur="80" data-usd="86">80 EUR</span></span>
  <span class="summary"><a href="/sale/10749/lot/10003">GREEK EASTERN EUROPE, Imitations of Alexander III</a></span>
  <a href="/sales/hosted/watchlist_ajax.php?lid=9900001&amp;remove=1">Remove</a>
</div>`
}

func TestParseWatchlistExtractsBrowseDivLot(t *testing.T) {
	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(numisbidsLotsFixture())
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
	if lot.ImageURL != "https://images.numisbids.com/sales/hosted/status/10749/image10003.jpg" {
		t.Fatalf("ImageURL = %q, want protocol-normalized image URL", lot.ImageURL)
	}
}

func TestParseWatchlistExtractsTitleFromSummarySpan(t *testing.T) {
	// The lot span contains only "Lot 1" (the number label); the summary span
	// contains the actual coin description. Verify we extract from summary.
	html := `
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="lot"><a href="/sale/10918/lot/1">Lot 1</a></span>
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}
	if lots[0].Title != "KELTEN, GALLIA Aedui, Quinar" {
		t.Fatalf("Title = %q, want coin description from summary span (not lot number)", lots[0].Title)
	}
}

func TestParseWatchlistExtractsSourceLotID(t *testing.T) {
	// SourceLotID is the watchlist entry ID from the browse div class (watch{id}).
	html := `
<div class="togglewatch" id="10918">
  &nbsp;<span class="arrow 10918">▼</span>
  &nbsp;<b>VIA GmbH E-Auction 28</b>
  &nbsp;&nbsp;(3 Aug 2026)<br>
  <div class="sharewatchlist">
    <span class="sendwatchlist" data-saleid="10918">
      <a class="button gray watchlist">Share this Watch List</a>
    </span>
  </div>
</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}
	if lots[0].SourceLotID != "12163211" {
		t.Fatalf("SourceLotID = %q, want 12163211 (watchlist entry ID from browse class)", lots[0].SourceLotID)
	}
}

func TestParseWatchlistExtractsStartingPrice(t *testing.T) {
	// Current watchlist: "Starting price: <span class="rateclick">40 EUR</span>"
	// Legacy format: "Estimate: 100 AUD" (plain text). Both must be parsed.
	htmlStartingPrice := `
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="estimate">Starting price: <span class="rateclick" data-eur="40" data-usd="43">40 EUR</span></span>
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>`

	htmlLegacyEstimate := `
<div class="togglewatch" id="10749">Test Sale (20 Apr 2026)</div>
<div class="browse 10749 watch9900001" style="height: 360px;">
  <span>Estimate: 100 AUD</span>
  <span class="summary"><a href="/sale/10749/lot/1">GREEK Coin</a></span>
</div>`

	logger := NewLogger(100)
	svc := NewNumisBidsService(logger)

	lotsNew := svc.ParseWatchlist(htmlStartingPrice)
	if len(lotsNew) != 1 {
		t.Fatalf("starting price: ParseWatchlist returned %d lots, want 1", len(lotsNew))
	}
	if lotsNew[0].Estimate == nil || *lotsNew[0].Estimate != 40 {
		t.Fatalf("starting price: Estimate = %v, want 40", lotsNew[0].Estimate)
	}
	if lotsNew[0].Currency != "EUR" {
		t.Fatalf("starting price: Currency = %q, want EUR", lotsNew[0].Currency)
	}

	lotsLegacy := svc.ParseWatchlist(htmlLegacyEstimate)
	if len(lotsLegacy) != 1 {
		t.Fatalf("legacy estimate: ParseWatchlist returned %d lots, want 1", len(lotsLegacy))
	}
	if lotsLegacy[0].Estimate == nil || *lotsLegacy[0].Estimate != 100 {
		t.Fatalf("legacy estimate: Estimate = %v, want 100", lotsLegacy[0].Estimate)
	}
	if lotsLegacy[0].Currency != "AUD" {
		t.Fatalf("legacy estimate: Currency = %q, want AUD", lotsLegacy[0].Currency)
	}
}

func TestParseWatchlistExtractsSaleNameAndDate(t *testing.T) {
	// Sale name and date from togglewatch header; no lot-page scrape required.
	html := `
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}
	if lots[0].SaleName != "VIA GmbH E-Auction 28" {
		t.Fatalf("SaleName = %q, want VIA GmbH E-Auction 28", lots[0].SaleName)
	}
	if lots[0].SaleDate != "3 Aug 2026" {
		t.Fatalf("SaleDate = %q, want 3 Aug 2026", lots[0].SaleDate)
	}
}

func TestParseWatchlistMultipleLotsOneSale(t *testing.T) {
	// Two lots in the same sale must each appear exactly once — not doubled.
	// The old link-based parser produced two entries per lot because each lot
	// has two identical hrefs (lot span + summary span).
	html := `
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="lot"><a href="/sale/10918/lot/1">Lot 1</a></span>
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN, GALLIA Aedui, Quinar</a></span>
</div>
<div class="browse 10918 watch12163299" style="height: 360px;">
  <span class="lot"><a href="/sale/10918/lot/2">Lot 2</a></span>
  <span class="summary"><a href="/sale/10918/lot/2">ROME, Julius Caesar, Denarius</a></span>
</div>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 2 {
		t.Fatalf("ParseWatchlist returned %d lots, want exactly 2 (one per browse div)", len(lots))
	}
	if lots[0].LotNumber != 1 {
		t.Fatalf("lots[0].LotNumber = %d, want 1", lots[0].LotNumber)
	}
	if lots[1].LotNumber != 2 {
		t.Fatalf("lots[1].LotNumber = %d, want 2", lots[1].LotNumber)
	}
}

func TestParseWatchlistAcceptsAbsoluteLotLinks(t *testing.T) {
	// Absolute numisbids.com URLs in summary spans must be accepted and preserved.
	html := `
<div class="togglewatch" id="10749">Test Sale (20 Apr 2026)</div>
<div class="browse 10749 watch9900001" style="height: 360px;">
  <img src="//images.numisbids.com/sales/hosted/status/10749/image10003.jpg">
  <span class="lot"><a href="https://www.numisbids.com/sale/10749/lot/10003">Lot 10003</a></span>
  <span class="estimate">Starting price: <span class="rateclick">100 AUD</span></span>
  <span class="summary"><a href="https://www.numisbids.com/sale/10749/lot/10003">GREEK EASTERN EUROPE, Imitations of Alexander III of Macedon.</a></span>
</div>`

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
	if lot.Estimate == nil || *lot.Estimate != 100 {
		t.Fatalf("Estimate = %v, want 100", lot.Estimate)
	}
	if lot.Currency != "AUD" {
		t.Fatalf("Currency = %q, want AUD", lot.Currency)
	}
}

func TestParseWatchlistAcceptsLegacyLotLinks(t *testing.T) {
	// Legacy /n.php?p=lot&sid=...&lot=... URLs still appear in saved watchlists.
	html := `
<div class="togglewatch" id="7996">Status International Auction 406 (1 May 2023)</div>
<div class="browse 7996 watch8800001" style="height: 360px;">
  <span class="lot"><a href="/n.php?p=lot&sid=7996&lot=10003">Lot 10003</a></span>
  <span class="estimate">Starting price: <span class="rateclick">150 USD</span></span>
  <span class="summary"><a href="/n.php?p=lot&sid=7996&lot=10003">Status International Auction 406, Lot 10003</a></span>
</div>`

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
}

func TestParseWatchlistIgnoresNonNumisBidsAbsoluteLinks(t *testing.T) {
	// A browse div whose summary points to a non-numisbids host must be skipped.
	html := `
<div class="togglewatch" id="10749">Test Sale (20 Apr 2026)</div>
<div class="browse 10749 watch9900001" style="height: 360px;">
  <span class="summary"><a href="https://example.com/sale/10749/lot/10003">External Lot</a></span>
</div>`

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(html)
	if len(lots) != 0 {
		t.Fatalf("ParseWatchlist returned %d lots, want 0 (non-numisbids URL must be ignored)", len(lots))
	}
}

// TestParseWatchlistRealMarkupFixture tests against the sanitized fixture file
// in testdata/numisbids_watchlist.html, which mirrors the real page structure
// verified from a live account (2026-07). Update this fixture (and re-verify)
// if the real site changes its markup.
func TestParseWatchlistRealMarkupFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/numisbids_watchlist.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	rawHTML := string(data)

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(rawHTML)

	if len(lots) != 2 {
		t.Fatalf("ParseWatchlist returned %d lots, want 2", len(lots))
	}

	l1 := lots[0]
	if l1.URL != "https://www.numisbids.com/sale/10918/lot/1" {
		t.Errorf("lots[0].URL = %q, want /sale/10918/lot/1", l1.URL)
	}
	if l1.SaleID != "10918" {
		t.Errorf("lots[0].SaleID = %q, want 10918", l1.SaleID)
	}
	if l1.LotNumber != 1 {
		t.Errorf("lots[0].LotNumber = %d, want 1", l1.LotNumber)
	}
	if l1.SourceLotID != "12163211" {
		t.Errorf("lots[0].SourceLotID = %q, want 12163211", l1.SourceLotID)
	}
	if l1.Title != "KELTEN, GALLIA Aedui, Quinar" {
		t.Errorf("lots[0].Title = %q, want coin description from summary span", l1.Title)
	}
	if l1.SaleName != "VIA GmbH E-Auction 28" {
		t.Errorf("lots[0].SaleName = %q, want VIA GmbH E-Auction 28", l1.SaleName)
	}
	if l1.SaleDate != "3 Aug 2026" {
		t.Errorf("lots[0].SaleDate = %q, want 3 Aug 2026", l1.SaleDate)
	}
	if l1.Estimate == nil || *l1.Estimate != 40 {
		t.Errorf("lots[0].Estimate = %v, want 40", l1.Estimate)
	}
	if l1.Currency != "EUR" {
		t.Errorf("lots[0].Currency = %q, want EUR", l1.Currency)
	}
	if l1.ImageURL != "https://media.numisbids.com/sales/hosted/via/e28/thumb00001.jpg" {
		t.Errorf("lots[0].ImageURL = %q, want protocol-normalized image URL", l1.ImageURL)
	}

	l2 := lots[1]
	if l2.LotNumber != 2 {
		t.Errorf("lots[1].LotNumber = %d, want 2", l2.LotNumber)
	}
	if l2.SourceLotID != "12163299" {
		t.Errorf("lots[1].SourceLotID = %q, want 12163299", l2.SourceLotID)
	}
	if l2.Title != "ROME, Julius Caesar, Denarius" {
		t.Errorf("lots[1].Title = %q", l2.Title)
	}
}

// ── WatchlistDiagnostics tests ────────────────────────────────────────────────

// TestParseWatchlist_NoProviderOutcomeFieldsForOpenLot is the core #490 regression:
// it proves ParseWatchlist never invents MaxBid, ProviderStatus, or SoldPrice from
// watchlist HTML that carries no closed-lot outcome signals. NumisBids does not expose
// a winner identity, final price, or auction lifecycle status on its watchlist page, so
// open lots must always require a manual status override — they cannot be auto-resolved
// the way CNG lots are via syncCNG. This guards against accidentally copying CNG's
// auto-detection logic into the NumisBids path.
func TestParseWatchlist_NoProviderOutcomeFieldsForOpenLot(t *testing.T) {
	data, err := os.ReadFile("testdata/numisbids_watchlist.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	logger := NewLogger(100)
	lots := NewNumisBidsService(logger).ParseWatchlist(string(data))
	if len(lots) == 0 {
		t.Fatal("ParseWatchlist returned 0 lots — fixture may be broken")
	}

	for i, lot := range lots {
		if lot.MaxBid != nil {
			t.Errorf("lots[%d].MaxBid = %v, want nil — NumisBids watchlist does not expose user max-bid", i, lot.MaxBid)
		}
		if lot.ProviderStatus != "" {
			t.Errorf("lots[%d].ProviderStatus = %q, want empty — NumisBids has no hosted closed-lot outcome signal", i, lot.ProviderStatus)
		}
		if lot.SoldPrice != nil {
			t.Errorf("lots[%d].SoldPrice = %v, want nil — NumisBids watchlist does not report sold prices", i, lot.SoldPrice)
		}
		if lot.WinningCustomerRowID != "" {
			t.Errorf("lots[%d].WinningCustomerRowID = %q, want empty", i, lot.WinningCustomerRowID)
		}
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
		t.Fatalf("CandidateLinkCount = %d, want 0 (no browse divs)", diagnostics.CandidateLinkCount)
	}
}

func TestWatchlistDiagnosticsCandidateLinkCountUsesBrowseDivs(t *testing.T) {
	// CandidateLinkCount must count browse divs, not raw lot links (two per lot).
	html := `
<div class="togglewatch" id="10918">VIA GmbH E-Auction 28 (3 Aug 2026)</div>
<div class="browse 10918 watch12163211" style="height: 360px;">
  <span class="lot"><a href="/sale/10918/lot/1">Lot 1</a></span>
  <span class="summary"><a href="/sale/10918/lot/1">KELTEN</a></span>
</div>
<div class="browse 10918 watch12163299" style="height: 360px;">
  <span class="lot"><a href="/sale/10918/lot/2">Lot 2</a></span>
  <span class="summary"><a href="/sale/10918/lot/2">ROME</a></span>
</div>`

	logger := NewLogger(100)
	d := NewNumisBidsService(logger).WatchlistDiagnostics(html)
	if d.CandidateLinkCount != 2 {
		t.Fatalf("CandidateLinkCount = %d, want 2 (browse divs, not raw links)", d.CandidateLinkCount)
	}
}

// ── Login / FetchWatchlist tests ──────────────────────────────────────────────

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
