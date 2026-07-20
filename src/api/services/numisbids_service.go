package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var ErrNumisBidsAuthenticationRequired = errors.New("numisbids authentication required")

const numisbidsUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) " +
	"Chrome/131.0.0.0 Safari/537.36"

// numisbidsBase, like numisbidsLoginURL/numisbidsWatchlistURL below, is a var rather than
// a const so tests can point lot-link resolution at a local test server instead of the
// real site.
var (
	numisbidsBase         = "https://www.numisbids.com"
	numisbidsLoginURL     = numisbidsBase + "/registration/login.php"
	numisbidsWatchlistURL = numisbidsBase + "/watchlist"
)

var (
	// lot URL path pattern: /sale/{saleID}/lot/{lotNumber}
	lotLinkRe = regexp.MustCompile(`^/sale/(\d+)/lot/(\d+)`)

	// watchlist page structural regexes (derived from verified real markup, 2026-07)
	//
	// Browse div: <div class="browse {saleID} watch{watchlistID}" ...>
	// One browse div per watched lot; the class encodes both the sale and the watchlist entry ID.
	browseDivRe = regexp.MustCompile(`<div\s+class="browse\s+(\d+)\s+watch(\d+)"`)

	// Sale group header: <div class="togglewatch" id="{saleID}">...<b>{name}</b>...({date})...</div>
	togglewatchRe = regexp.MustCompile(`(?is)<div\s+class="togglewatch"\s+id="(\d+)">(.*?)</div>`)

	// Sale name inside a togglewatch header.
	watchlistHeaderNameRe = regexp.MustCompile(`(?is)<b>(.*?)</b>`)

	// Date in parentheses inside a togglewatch header, e.g. "(3 Aug 2026)" or "(20-21 Apr 2026)".
	saleDateParenRe = regexp.MustCompile(`\((\d{1,2}(?:-\d{1,2})?\s+\w+\s+\d{4})\)`)

	// Summary span href — extracts the lot link from <span class="summary"><a href="...">.
	summaryHrefRe = regexp.MustCompile(`(?i)<span\s+class="summary">\s*<a\s+href="([^"]+)"`)

	// Summary span anchor text — lot title (plain text, no nested tags expected).
	summaryTextRe = regexp.MustCompile(`(?i)<span\s+class="summary">\s*<a\s+[^>]*>([^<]+)</a>`)

	// Price field: matches both legacy "Estimate: 100 AUD" and current
	// "Starting price: <span class="rateclick" ...>40 EUR</span>" layouts.
	priceFieldRe = regexp.MustCompile(`(?i)(?:Estimate|Starting\s+price):\s*(?:<[^>]+>)?\s*([\d,]+(?:\.\d+)?)\s*(USD|EUR|GBP|CHF|AUD|CAD)`)

	// Image src attribute (used in watchlist cards and lot-page scraping).
	imgSrcRe = regexp.MustCompile(`<img[^>]*src="([^"]*)"`)

	// og:image meta tag (lot-page detail scraping).
	ogImageRe = regexp.MustCompile(`<meta\s+property="og:image"\s+content="([^"]+)"`)

	// Numeric value + currency code (used by parseCurrencyValue).
	currencyValRe = regexp.MustCompile(`([\d,]+(?:\.\d+)?)\s*(USD|EUR|GBP|CHF|AUD|CAD)`)
)

// WatchlistLot represents a single lot parsed from a NumisBids watchlist page.
type WatchlistLot struct {
	URL          string   `json:"url"`
	SourceLotID  string   `json:"sourceLotId"`
	SourceSaleID string   `json:"sourceSaleId"`
	SaleID       string   `json:"saleId"`
	LotNumber    int      `json:"lotNumber"`
	Title        string   `json:"title"`
	ImageURL     string   `json:"imageUrl"`
	Estimate     *float64 `json:"estimate"`
	CurrentBid   *float64 `json:"currentBid"`
	MaxBid       *float64 `json:"maxBid"`
	Currency     string   `json:"currency"`
	AuctionHouse string   `json:"auctionHouse"`
	SaleName     string   `json:"saleName"`
	SaleDate     string   `json:"saleDate"`
	Description  string   `json:"description"`

	// ProviderStatus, SoldPrice, and WinningCustomerRowID are populated by providers that
	// expose a definitive closed-lot outcome (currently CNG only) and are used by the
	// watchlist sync service to auto-detect won/lost instead of requiring a manual override.
	ProviderStatus       string   `json:"providerStatus,omitempty"`
	SoldPrice            *float64 `json:"soldPrice,omitempty"`
	WinningCustomerRowID string   `json:"-"`
}

// NumisBidsService handles HTTP interactions with numisbids.com.
type NumisBidsService struct {
	logger *Logger
}

// NewNumisBidsService creates a new NumisBidsService.
func NewNumisBidsService(logger *Logger) *NumisBidsService {
	return &NumisBidsService{logger: logger}
}

// Login authenticates with NumisBids and returns a cookie-jar-enabled client.
func (s *NumisBidsService) Login(username, password string) (*http.Client, error) {
	s.debug("Attempting login to NumisBids")

	client, err := newScraperClient()
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"email":    {username},
		"password": {password},
	}

	req, err := newScraperFormRequest(numisbidsLoginURL, form, numisbidsLoginHeaders())
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	body, err := doScraperRequest(client, req, "login", http.StatusOK, http.StatusFound)
	if err != nil {
		s.error("Login HTTP request failed: %v", err)
		return nil, err
	}

	// Read body to check the JSON result returned by NumisBids' AJAX login.
	bodyStr := string(body)

	s.trace("Login response body length: %d bytes", len(bodyStr))

	var loginResponse struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &loginResponse); err != nil {
		s.warn("Login failed: unexpected response format")
		return nil, fmt.Errorf("unexpected login response")
	}
	if !strings.EqualFold(loginResponse.Status, "success") {
		s.warn("Login failed: NumisBids returned status %q", loginResponse.Status)
		return nil, fmt.Errorf("login returned status %q", loginResponse.Status)
	}

	// Verify session cookie was set by checking the login host for PHPSESSID or similar.
	parsedURL, _ := url.Parse(numisbidsLoginURL)
	cookies := client.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		s.warn("No session cookie received after login")
		return nil, fmt.Errorf("no session cookie received — login may have failed")
	}

	s.debug("Login successful, received %d cookie(s)", len(cookies))

	// Verify authentication by requesting a protected page
	if err := s.verifyAuthentication(client); err != nil {
		s.error("Authentication verification failed: %v", err)
		return nil, fmt.Errorf("login succeeded but authentication verification failed: %w", err)
	}

	s.info("Login and authentication verified")
	return client, nil
}

// verifyAuthentication checks that the client is actually authenticated by fetching
// the watchlist page and checking for login indicators.
func (s *NumisBidsService) verifyAuthentication(client *http.Client) error {
	req, err := newScraperRequest(http.MethodGet, numisbidsWatchlistURL, nil, numisbidsDefaultHeaders())
	if err != nil {
		return fmt.Errorf("failed to create verification request: %w", err)
	}

	body, err := doScraperRequest(client, req, "verification")
	if err != nil {
		return err
	}

	bodyStr := strings.ToLower(string(body))

	// Check for login form indicators (unauthenticated)
	if isNumisBidsLoginPrompt(bodyStr) ||
		strings.Contains(bodyStr, `name="email"`) ||
		strings.Contains(bodyStr, `name="password"`) ||
		strings.Contains(bodyStr, "login to your account") {
		s.debug("Verification page contains login form — not authenticated")
		return fmt.Errorf("not authenticated: watchlist page returned login form")
	}

	s.debug("Authentication verified — no login form detected")
	return nil
}

// FetchWatchlist retrieves the authenticated user's watchlist HTML.
//
// Transport note: the NumisBids server sends content-encoding: br (Brotli) to
// browser clients (verified from HAR, 2026-07). The Go scraper uses the default
// http.Transport, which advertises only Accept-Encoding: gzip — Cloudflare
// responds with gzip for this client, which Go decompresses transparently.
// If the scraper transport is ever changed to advertise br, a Brotli decompressor
// must be added: doScraperRequest would return undecoded bytes, ParseWatchlist
// would silently produce 0 lots, and no error would be raised.
//
// Cache note: the server sends cache-control: no-store, no-cache, must-revalidate,
// so every call hits the origin; do not assume cached responses.
func (s *NumisBidsService) FetchWatchlist(client *http.Client) (string, error) {
	req, err := newScraperRequest(http.MethodGet, numisbidsWatchlistURL, nil, numisbidsDefaultHeaders())
	if err != nil {
		return "", fmt.Errorf("failed to create watchlist request: %w", err)
	}

	body, err := doScraperRequest(client, req, "watchlist")
	if err != nil {
		return "", err
	}
	bodyStr := string(body)
	if isNumisBidsLoginPrompt(bodyStr) {
		return "", ErrNumisBidsAuthenticationRequired
	}

	return bodyStr, nil
}

// LotPageDetails holds fields extracted from a NumisBids lot detail page.
type LotPageDetails struct {
	ImageURL     string
	AuctionHouse string
	SaleName     string
	SaleDate     string // raw date text, e.g. "20-21 Apr 2026"
	LotNumber    int
	CurrentBid   *float64
	Currency     string
	Description  string
}

var (
	houseNameRe  = regexp.MustCompile(`<span class="name">(.*?)</span>`)
	saleNameRe   = regexp.MustCompile(`<span class="name">.*?</span>\s*(?:<br\s*/?>)\s*<b>(.*?)</b>`)
	saleDateRe   = regexp.MustCompile(`</b>\s*(?:&nbsp;)+\s*(\d{1,2}(?:-\d{1,2})?\s+\w+\s+\d{4})`)
	currentBidRe = regexp.MustCompile(`(?i)Current\s+bid:\s*([\d,]+(?:\.\d+)?\s*\w+)`)
	lotNumberRe  = regexp.MustCompile(`(?i)<div class="left">Lot\s+(\d+)`)
	// Matches the coin description div — the one after the watchnote div, containing the actual lot text
	descriptionRe = regexp.MustCompile(`(?s)<div class="description"><b>(.*?)</b>(.*?)</div>`)
)

// ScrapeLotImage fetches a NumisBids lot page and extracts the og:image URL.
func (s *NumisBidsService) ScrapeLotImage(lotURL string) (string, error) {
	details, err := s.ScrapeLotPage(lotURL)
	if err != nil {
		return "", err
	}
	if details.ImageURL == "" {
		return "", fmt.Errorf("no og:image found")
	}
	return details.ImageURL, nil
}

// ScrapeLotPage fetches a NumisBids lot page and extracts image, auction house,
// sale name, and current bid.
func (s *NumisBidsService) ScrapeLotPage(lotURL string) (*LotPageDetails, error) {
	req, err := newScraperRequest(http.MethodGet, lotURL, nil, numisbidsDefaultHeaders())
	if err != nil {
		return nil, err
	}

	body, err := doScraperRequest(http.DefaultClient, req, "lot page")
	if err != nil {
		return nil, err
	}

	html := string(body)
	details := &LotPageDetails{}

	// og:image
	if match := ogImageRe.FindStringSubmatch(html); match != nil {
		details.ImageURL = match[1]
	}

	// Auction house: <span class="name">...</span>
	if match := houseNameRe.FindStringSubmatch(html); match != nil {
		details.AuctionHouse = cleanHTML(match[1])
	}

	// Sale name: <b>...</b> after the house name span
	if match := saleNameRe.FindStringSubmatch(html); match != nil {
		details.SaleName = cleanHTML(match[1])
	}

	// Sale date: appears after </b>&nbsp;&nbsp;20-21 Apr 2026
	if match := saleDateRe.FindStringSubmatch(html); match != nil {
		details.SaleDate = strings.TrimSpace(match[1])
	}

	// Current bid
	if match := currentBidRe.FindStringSubmatch(html); match != nil {
		val, cur := parseCurrencyValue(match[1])
		details.CurrentBid = val
		if cur != "" {
			details.Currency = cur
		}
	}

	// Lot number from detail page: <div class="left">Lot 15<br>
	if match := lotNumberRe.FindStringSubmatch(html); match != nil {
		details.LotNumber, _ = strconv.Atoi(match[1])
	}

	// Description: find all <div class="description"><b>...</b>...</div> blocks,
	// use the last one (earlier matches are postbid/watchnote containers)
	if matches := descriptionRe.FindAllStringSubmatch(html, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		desc := cleanHTML(last[1] + last[2])
		desc = strings.TrimSpace(desc)
		if len(desc) > 2000 {
			desc = desc[:2000]
		}
		details.Description = desc
	}

	return details, nil
}

// ParseWatchlist extracts lot data from NumisBids watchlist HTML using the
// verified real-page structure (browse divs + togglewatch headers, 2026-07).
func (s *NumisBidsService) ParseWatchlist(rawHTML string) []WatchlistLot {
	s.debug("Parsing watchlist HTML (%d bytes)", len(rawHTML))

	// Pass 1: build a saleID → {name, date} map from togglewatch section headers.
	saleGroups := extractWatchlistSaleGroups(rawHTML)

	// Pass 2: each <div class="browse {saleID} watch{watchlistID}"> is one watched lot.
	browseDivMatches := browseDivRe.FindAllStringSubmatchIndex(rawHTML, -1)
	s.debug("Found %d lot cards in watchlist HTML", len(browseDivMatches))

	var lots []WatchlistLot
	for i, match := range browseDivMatches {
		saleID := rawHTML[match[2]:match[3]]
		watchID := rawHTML[match[4]:match[5]]

		// Block for this lot: from its browse div start to the next browse div start.
		start := match[0]
		end := len(rawHTML)
		if i+1 < len(browseDivMatches) {
			end = browseDivMatches[i+1][0]
		}
		block := rawHTML[start:end]

		lot := WatchlistLot{
			SourceSaleID: saleID,
			SaleID:       saleID,
			SourceLotID:  watchID,
			Currency:     "USD",
		}

		// Sale name and date from the preceding togglewatch group header.
		if sg, ok := saleGroups[saleID]; ok {
			lot.SaleName = sg.Name
			lot.SaleDate = sg.Date
		}

		// Canonical lot URL and lot number from the summary span href.
		// The lot span (<span class="lot">) carries only the lot number label; the
		// summary span carries the full coin title and is the canonical link.
		if hrefMatch := summaryHrefRe.FindStringSubmatch(block); hrefMatch != nil {
			href := hrefMatch[1]
			urlVal, lotSaleID, lotNumber, ok := parseNumisBidsLotHref(href)
			if ok {
				lot.URL = urlVal
				if lotSaleID != "" {
					lot.SaleID = lotSaleID
					lot.SourceSaleID = lotSaleID
				}
				lot.LotNumber = lotNumber
			}
		}
		if lot.URL == "" {
			s.trace("Skipping browse lot watchID=%s: could not resolve a valid lot URL", watchID)
			continue
		}

		// Title from the summary anchor text.
		if textMatch := summaryTextRe.FindStringSubmatch(block); textMatch != nil {
			title := strings.TrimSpace(textMatch[1])
			if len(title) > 200 {
				title = title[:200]
			}
			lot.Title = title
		}

		// Image URL — protocol-normalize "//" URLs.
		if imgMatch := imgSrcRe.FindStringSubmatch(block); imgMatch != nil {
			imgURL := imgMatch[1]
			if strings.HasPrefix(imgURL, "//") {
				imgURL = "https:" + imgURL
			}
			lot.ImageURL = imgURL
		}

		// Price: handles both "Estimate: 100 AUD" and "Starting price: <span...>40 EUR</span>".
		if priceMatch := priceFieldRe.FindStringSubmatch(block); priceMatch != nil {
			numStr := strings.ReplaceAll(priceMatch[1], ",", "")
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				lot.Estimate = &val
			}
			if priceMatch[2] != "" {
				lot.Currency = priceMatch[2]
			}
		}

		s.trace("Parsed lot %d: saleID=%s watchID=%s lotNumber=%d", i+1, lot.SaleID, watchID, lot.LotNumber)
		lots = append(lots, lot)
	}

	s.info("Parsed %d lots from watchlist", len(lots))
	return lots
}

func (s *NumisBidsService) trace(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Trace("numisbids", format, args...)
	}
}

func (s *NumisBidsService) debug(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Debug("numisbids", format, args...)
	}
}

func (s *NumisBidsService) info(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info("numisbids", format, args...)
	}
}

func (s *NumisBidsService) warn(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn("numisbids", format, args...)
	}
}

func (s *NumisBidsService) error(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error("numisbids", format, args...)
	}
}

func (s *NumisBidsService) WatchlistDiagnostics(rawHTML string) WatchlistDiagnostics {
	return WatchlistDiagnostics{
		HTMLBytes:          len(rawHTML),
		CandidateLinkCount: len(browseDivRe.FindAllStringIndex(rawHTML, -1)),
		HasLoginPrompt:     isNumisBidsLoginPrompt(rawHTML),
		HasWatchlistText:   strings.Contains(strings.ToLower(rawHTML), "watch list"),
	}
}

func numisbidsDefaultHeaders() map[string]string {
	return map[string]string{
		"User-Agent": numisbidsUserAgent,
		// Explicitly exclude br (Brotli): Cloudflare infers br support from the
		// Chrome user-agent and sends content-encoding: br regardless of the
		// client's actual capabilities. Go has no native Brotli decompressor, so
		// we restrict to gzip/deflate and handle decompression in the scraper
		// transport layer (readScraperResponseBody).
		"Accept-Encoding": "gzip, deflate",
	}
}

func numisbidsLoginHeaders() map[string]string {
	headers := numisbidsDefaultHeaders()
	headers["Referer"] = numisbidsBase + "/"
	headers["X-Requested-With"] = "XMLHttpRequest"
	return headers
}

type WatchlistDiagnostics struct {
	HTMLBytes          int
	CandidateLinkCount int
	HasLoginPrompt     bool
	HasWatchlistText   bool
}

func isNumisBidsLoginPrompt(rawHTML string) bool {
	normalized := strings.ToLower(rawHTML)
	return strings.Contains(normalized, "already have items on your watch list") &&
		strings.Contains(normalized, "loginreload")
}

type watchlistSaleGroup struct {
	Name string
	Date string // raw date text, e.g. "3 Aug 2026" or "20-21 Apr 2026"
}

// extractWatchlistSaleGroups builds a saleID → sale name/date map from togglewatch
// section headers found in the watchlist HTML. Used by ParseWatchlist to annotate lots
// with their auction's human-readable name and date without a per-lot HTTP scrape.
func extractWatchlistSaleGroups(rawHTML string) map[string]watchlistSaleGroup {
	groups := make(map[string]watchlistSaleGroup)
	matches := togglewatchRe.FindAllStringSubmatch(rawHTML, -1)
	for _, m := range matches {
		saleID := m[1]
		rawContent := m[2]
		plainContent := cleanHTML(rawContent)
		sg := watchlistSaleGroup{}
		if nameMatch := watchlistHeaderNameRe.FindStringSubmatch(rawContent); nameMatch != nil {
			sg.Name = cleanHTML(nameMatch[1])
		}
		if dateMatch := saleDateParenRe.FindStringSubmatch(plainContent); dateMatch != nil {
			sg.Date = dateMatch[1]
			if sg.Name == "" {
				// Sale name is everything before the date parenthetical.
				parenIdx := strings.LastIndex(plainContent, "("+sg.Date+")")
				if parenIdx > 0 {
					sg.Name = strings.TrimSpace(plainContent[:parenIdx])
				} else {
					sg.Name = plainContent
				}
			}
		}
		if sg.Name == "" {
			sg.Name = plainContent
		}
		groups[saleID] = sg
	}
	return groups
}

func parseNumisBidsLotHref(href string) (string, string, int, bool) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", "", 0, false
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return "", "", 0, false
	}
	if parsed.IsAbs() && !strings.EqualFold(parsed.Host, "www.numisbids.com") && !strings.EqualFold(parsed.Host, "numisbids.com") {
		return "", "", 0, false
	}

	path := parsed.Path
	if path == "" && !parsed.IsAbs() {
		path = href
	}
	if saleMatch := lotLinkRe.FindStringSubmatch(path); saleMatch != nil {
		lotNumber, err := strconv.Atoi(saleMatch[2])
		if err != nil {
			return "", "", 0, false
		}
		return numisbidsBase + saleMatch[0], saleMatch[1], lotNumber, true
	}

	if strings.EqualFold(path, "/n.php") || strings.EqualFold(path, "n.php") {
		query := parsed.Query()
		if query.Get("p") != "lot" {
			return "", "", 0, false
		}
		saleID := query.Get("sid")
		lotRaw := query.Get("lot")
		lotNumber, err := strconv.Atoi(lotRaw)
		if saleID == "" || err != nil {
			return "", "", 0, false
		}
		if parsed.IsAbs() {
			return parsed.String(), saleID, lotNumber, true
		}
		if strings.HasPrefix(parsed.String(), "/") {
			return numisbidsBase + parsed.String(), saleID, lotNumber, true
		}
		return numisbidsBase + "/" + parsed.String(), saleID, lotNumber, true
	}

	return "", "", 0, false
}

// parseCurrencyValue extracts a numeric value and currency code from a string
// like "150 USD" or "1,200.50 EUR".
func parseCurrencyValue(text string) (*float64, string) {
	match := currencyValRe.FindStringSubmatch(text)
	if match == nil {
		return nil, "USD"
	}
	numStr := strings.ReplaceAll(match[1], ",", "")
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, match[2]
	}
	return &val, match[2]
}

// cleanHTML strips HTML tags and normalizes whitespace.
func cleanHTML(s string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(s))
	var result strings.Builder
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			result.WriteString(tokenizer.Token().Data)
		}
	}
	// Normalize whitespace
	text := result.String()
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

// ParseSaleDate attempts to parse a NumisBids sale date string like
// "20-21 Apr 2026" or "5 May 2026" into a time.Time (using the last date if a range).
func ParseSaleDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// If it's a range like "20-21 Apr 2026", take the end date
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) < 2 {
		return nil
	}
	dayPart := parts[0]
	rest := parts[1] // "Apr 2026"

	// Handle range: take the last day number
	if idx := strings.LastIndex(dayPart, "-"); idx >= 0 {
		dayPart = dayPart[idx+1:]
	}

	dateStr := dayPart + " " + rest
	for _, layout := range []string{"2 Jan 2006", "2 January 2006"} {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return &t
		}
	}
	return nil
}
