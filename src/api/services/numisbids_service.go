package services

import (
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

const (
	numisbidsBase    = "https://www.numisbids.com"
	numisbidsLoginURL = numisbidsBase + "/registration/login.php"
	numisbidsWatchlistURL = numisbidsBase + "/watchlist"
	numisbidsUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/131.0.0.0 Safari/537.36"
)

var (
	lotLinkRe    = regexp.MustCompile(`href="(/sale/(\d+)/lot/(\d+))"`)
	imgSrcRe     = regexp.MustCompile(`<img[^>]*src="([^"]*)"`)
	ogImageRe    = regexp.MustCompile(`<meta\s+property="og:image"\s+content="([^"]+)"`)
	estimateRe   = regexp.MustCompile(`Estimate:\s*([\d,]+(?:\.\d+)?\s*\w+)`)
	currencyValRe = regexp.MustCompile(`([\d,]+(?:\.\d+)?)\s*(USD|EUR|GBP|CHF)`)
)

// WatchlistLot represents a single lot parsed from a NumisBids watchlist page.
type WatchlistLot struct {
	URL          string   `json:"url"`
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
type NumisBidsService struct{}

// NewNumisBidsService creates a new NumisBidsService.
func NewNumisBidsService() *NumisBidsService {
	return &NumisBidsService{}
}

// Login authenticates with NumisBids and returns a cookie-jar-enabled client.
func (s *NumisBidsService) Login(username, password string) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{Jar: jar}

	form := url.Values{
		"email":    {username},
		"password": {password},
		"login":    {"Login"},
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
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("login returned HTTP %d", resp.StatusCode)
	}

	// Read body to check for login errors (AJAX form returns HTML fragment)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "Incorrect") || strings.Contains(bodyStr, "invalid") {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify session cookie was set by checking for a PHPSESSID or similar
	parsedURL, _ := url.Parse(numisbidsBase)
	if len(jar.Cookies(parsedURL)) == 0 {
		return nil, fmt.Errorf("no session cookie received — login may have failed")
	}

	return client, nil
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

	return string(body), nil
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
	// Find all lot link positions, then extract the block between each pair.
	// Go's regexp engine (RE2) doesn't support lookaheads, so we split manually.
	indices := lotLinkRe.FindAllStringIndex(rawHTML, -1)

	var lots []WatchlistLot
	for i, idx := range indices {
		start := idx[0]
		end := len(rawHTML)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		block := rawHTML[start:end]

		linkMatch := lotLinkRe.FindStringSubmatch(block)
		if linkMatch == nil {
			continue
		}

		lotNum, _ := strconv.Atoi(linkMatch[3])
		lot := WatchlistLot{
			URL:       numisbidsBase + linkMatch[1],
			SaleID:    linkMatch[2],
			LotNumber: lotNum,
			Currency:  "USD",
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
		lot.Title = extractLotTitle(block, linkMatch[1])
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

		lots = append(lots, lot)
	}

	return lots
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
