package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var ErrNumisBidsAuthenticationRequired = errors.New("numisbids authentication required")

const (
	numisbidsBase      = "https://www.numisbids.com"
	numisbidsUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/131.0.0.0 Safari/537.36"
)

var (
	numisbidsLoginURL     = numisbidsBase + "/registration/login.php"
	numisbidsWatchlistURL = numisbidsBase + "/watchlist"
)

var (
	lotLinkRe     = regexp.MustCompile(`^/sale/(\d+)/lot/(\d+)`)
	lotHrefRe     = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	imgSrcRe      = regexp.MustCompile(`<img[^>]*src="([^"]*)"`)
	ogImageRe     = regexp.MustCompile(`<meta\s+property="og:image"\s+content="([^"]+)"`)
	estimateRe    = regexp.MustCompile(`Estimate:\s*([\d,]+(?:\.\d+)?\s*\w+)`)
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
	Currency     string   `json:"currency"`
	AuctionHouse string   `json:"auctionHouse"`
	SaleName     string   `json:"saleName"`
	SaleDate     string   `json:"saleDate"`
	Description  string   `json:"description"`
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

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{Jar: jar}

	form := url.Values{
		"email":    {username},
		"password": {password},
	}

	req, err := http.NewRequest("POST", numisbidsLoginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", numisbidsUserAgent)
	req.Header.Set("Referer", numisbidsBase+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		s.error("Login HTTP request failed: %v", err)
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	s.debug("Login response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("login returned HTTP %d", resp.StatusCode)
	}

	// Read body to check the JSON result returned by NumisBids' AJAX login.
	body, _ := io.ReadAll(resp.Body)
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
	cookies := jar.Cookies(parsedURL)
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
	req, err := http.NewRequest("GET", numisbidsWatchlistURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create verification request: %w", err)
	}
	req.Header.Set("User-Agent", numisbidsUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verification request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read verification response: %w", err)
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
func (s *NumisBidsService) FetchWatchlist(client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", numisbidsWatchlistURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create watchlist request: %w", err)
	}
	req.Header.Set("User-Agent", numisbidsUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("watchlist request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("watchlist returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read watchlist body: %w", err)
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
	req, err := http.NewRequest("GET", lotURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", numisbidsUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lot page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
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

// ParseWatchlist extracts lot data from NumisBids watchlist HTML.
// Mirrors the Python scrape_numisbids_watchlist logic.
func (s *NumisBidsService) ParseWatchlist(rawHTML string) []WatchlistLot {
	s.debug("Parsing watchlist HTML (%d bytes)", len(rawHTML))

	// Find all lot link positions, then extract the block between each pair.
	// Go's regexp engine (RE2) doesn't support lookaheads, so we split manually.
	matches := findWatchlistLotLinks(rawHTML)

	s.debug("Found %d lot links in watchlist HTML", len(matches))

	var lots []WatchlistLot
	for i, match := range matches {
		start := match.start
		end := len(rawHTML)
		if i+1 < len(matches) {
			end = matches[i+1].start
		}
		block := rawHTML[start:end]

		lot := WatchlistLot{
			URL:          match.url,
			SourceSaleID: match.saleID,
			SaleID:       match.saleID,
			LotNumber:    match.lotNumber,
			Currency:     "USD",
		}

		// Image URL
		if imgMatch := imgSrcRe.FindStringSubmatch(block); imgMatch != nil {
			imgURL := imgMatch[1]
			if strings.HasPrefix(imgURL, "//") {
				imgURL = "https:" + imgURL
			}
			lot.ImageURL = imgURL
		}

		// Title: extract only the text inside the lot anchor tag
		lot.Title = extractLotTitle(block, match.href)
		if len(lot.Title) > 200 {
			lot.Title = lot.Title[:200]
		}

		// Estimate
		if estMatch := estimateRe.FindStringSubmatch(block); estMatch != nil {
			val, cur := parseCurrencyValue(estMatch[1])
			lot.Estimate = val
			if cur != "" {
				lot.Currency = cur
			}
		}

		s.trace("Parsed lot %d: saleID=%s lotNumber=%d", i+1, lot.SaleID, lot.LotNumber)
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
		CandidateLinkCount: len(findWatchlistLotLinks(rawHTML)),
		HasLoginPrompt:     isNumisBidsLoginPrompt(rawHTML),
		HasWatchlistText:   strings.Contains(strings.ToLower(rawHTML), "watch list"),
	}
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

type watchlistLotLink struct {
	start     int
	href      string
	url       string
	saleID    string
	lotNumber int
}

func findWatchlistLotLinks(rawHTML string) []watchlistLotLink {
	hrefMatches := lotHrefRe.FindAllStringSubmatchIndex(rawHTML, -1)
	links := make([]watchlistLotLink, 0, len(hrefMatches))
	for _, hrefMatch := range hrefMatches {
		if len(hrefMatch) < 4 {
			continue
		}
		href := rawHTML[hrefMatch[2]:hrefMatch[3]]
		urlValue, saleID, lotNumber, ok := parseNumisBidsLotHref(href)
		if !ok {
			continue
		}
		start := hrefMatch[0]
		if anchorStart := strings.LastIndex(strings.ToLower(rawHTML[:hrefMatch[0]]), "<a"); anchorStart >= 0 {
			start = anchorStart
		}
		links = append(links, watchlistLotLink{
			start:     start,
			href:      href,
			url:       urlValue,
			saleID:    saleID,
			lotNumber: lotNumber,
		})
	}
	return links
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

// extractLotTitle extracts the text content of the anchor tag that links to the lot.
// It walks the HTML tokens looking for <a href="...lotPath...">, then collects
// text until the closing </a> tag.
func extractLotTitle(block, lotPath string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(block))
	inLotLink := false
	var result strings.Builder

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		switch tt {
		case html.StartTagToken:
			t := tokenizer.Token()
			if t.Data == "a" && !inLotLink {
				for _, attr := range t.Attr {
					if attr.Key == "href" && attr.Val == lotPath {
						inLotLink = true
						break
					}
				}
			}
		case html.EndTagToken:
			if inLotLink && tokenizer.Token().Data == "a" {
				goto done
			}
		case html.TextToken:
			if inLotLink {
				result.WriteString(tokenizer.Token().Data)
			}
		}
	}

done:
	text := strings.Join(strings.Fields(result.String()), " ")
	return strings.TrimSpace(text)
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
