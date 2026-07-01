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
)

var ErrCNGAuthenticationRequired = errors.New("cng authentication required")

const (
	cngBase      = "https://auctions.cngcoins.com"
	cngUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/138.0.0.0 Safari/537.36"
)

var (
	cngLoginURL     = cngBase + "/login"
	cngWatchlistURL = cngBase + "/watched-lots"
	cngRefreshMeURL = cngBase + "/ajax/refresh-me"
	cngLotPathRe    = regexp.MustCompile(`^/lots/view/([^/]+)(?:/|$)`)
)

// CNGAuctionService handles HTTP interactions with auctions.cngcoins.com.
type CNGAuctionService struct {
	logger *Logger
}

// NewCNGAuctionService creates a new CNGAuctionService.
func NewCNGAuctionService(logger *Logger) *CNGAuctionService {
	return &CNGAuctionService{logger: logger}
}

// Login authenticates with CNG and returns a cookie-jar-enabled client.
func (s *CNGAuctionService) Login(username, password string) (*http.Client, error) {
	s.debug("Attempting login to CNG")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar}

	req, err := http.NewRequest("GET", cngLoginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create login page request: %w", err)
	}
	req.Header.Set("User-Agent", cngUserAgent)
	if resp, err := client.Do(req); err != nil {
		return nil, fmt.Errorf("login page request failed: %w", err)
	} else {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("login page returned HTTP %d", resp.StatusCode)
		}
	}

	form := url.Values{
		"username": {username},
		"password": {password},
		"Login":    {"Login"},
	}
	req, err = http.NewRequest("POST", cngLoginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", cngUserAgent)
	req.Header.Set("Origin", cngBase)
	req.Header.Set("Referer", cngLoginURL)

	resp, err := client.Do(req)
	if err != nil {
		s.error("CNG login HTTP request failed: %v", err)
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("login returned HTTP %d", resp.StatusCode)
	}
	if err := s.verifyAuthentication(client); err != nil {
		s.warn("CNG authentication verification failed: %v", err)
		return nil, fmt.Errorf("login failed: %w", err)
	}

	s.info("CNG login and authentication verified")
	return client, nil
}

func (s *CNGAuctionService) verifyAuthentication(client *http.Client) error {
	req, err := http.NewRequest("GET", cngRefreshMeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create refresh-me request: %w", err)
	}
	req.Header.Set("User-Agent", cngUserAgent)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh-me request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh-me returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read refresh-me body: %w", err)
	}
	if strings.TrimSpace(string(body)) == "null" {
		return ErrCNGAuthenticationRequired
	}
	return nil
}

// FetchWatchlist retrieves the authenticated user's watched lots HTML.
func (s *CNGAuctionService) FetchWatchlist(client *http.Client) (string, error) {
	return s.fetchWatchlistPage(client, 1)
}

// FetchWatchlistLots retrieves every available CNG watched-lots page and parses the lots.
func (s *CNGAuctionService) FetchWatchlistLots(client *http.Client) ([]WatchlistLot, error) {
	firstPage, err := s.fetchWatchlistPage(client, 1)
	if err != nil {
		return nil, err
	}

	lots, info, err := s.parseLotsPage(firstPage)
	if err != nil {
		return nil, err
	}
	totalPages := info.totalPages()
	for page := 2; page <= totalPages; page++ {
		rawPage, err := s.fetchWatchlistPage(client, page)
		if err != nil {
			return nil, err
		}
		pageLots, _, err := s.parseLotsPage(rawPage)
		if err != nil {
			return nil, err
		}
		lots = append(lots, pageLots...)
	}
	s.info("Fetched %d CNG watched lots across %d page(s)", len(lots), totalPages)
	return lots, nil
}

func (s *CNGAuctionService) fetchWatchlistPage(client *http.Client, page int) (string, error) {
	watchlistURL, err := cngWatchlistPageURL(page)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", watchlistURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create watchlist request: %w", err)
	}
	req.Header.Set("User-Agent", cngUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("watchlist page %d request failed: %w", page, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusUnauthorized {
		return "", ErrCNGAuthenticationRequired
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("watchlist page %d returned HTTP %d", page, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read watchlist body: %w", err)
	}
	bodyStr := string(body)
	if isCNGLoginPrompt(bodyStr) {
		return "", ErrCNGAuthenticationRequired
	}
	return bodyStr, nil
}

// ScrapeLotPage fetches a CNG lot page and extracts lot details.
func (s *CNGAuctionService) ScrapeLotPage(lotURL string) (*LotPageDetails, error) {
	lot, err := s.ScrapeLot(lotURL)
	if err != nil {
		return nil, err
	}
	return cngWatchlistLotToDetails(lot), nil
}

// ScrapeLot fetches a CNG lot page and extracts a source-aware lot summary.
func (s *CNGAuctionService) ScrapeLot(lotURL string) (WatchlistLot, error) {
	if err := validateCNGLotURL(lotURL); err != nil {
		return WatchlistLot{}, err
	}
	req, err := http.NewRequest("GET", lotURL, nil)
	if err != nil {
		return WatchlistLot{}, err
	}
	req.Header.Set("User-Agent", cngUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return WatchlistLot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return WatchlistLot{}, fmt.Errorf("lot page returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WatchlistLot{}, err
	}
	lot, err := s.parseLotPage(string(body))
	if err != nil {
		return WatchlistLot{}, err
	}
	return lot, nil
}

// ParseWatchlist extracts watched lots from CNG's viewVars.lots.result_page payload.
func (s *CNGAuctionService) ParseWatchlist(rawHTML string) []WatchlistLot {
	lots, _, err := s.parseLotsPage(rawHTML)
	if err != nil {
		s.warn("Failed to parse CNG watchlist: %v", err)
		return nil
	}
	s.info("Parsed %d lots from CNG watchlist", len(lots))
	return lots
}

func (s *CNGAuctionService) parseLotPage(rawHTML string) (WatchlistLot, error) {
	var root cngViewVars
	if err := parseCNGViewVars(rawHTML, &root); err != nil {
		return WatchlistLot{}, err
	}
	if root.Lot == nil {
		return WatchlistLot{}, fmt.Errorf("cng lot page missing viewVars.lot")
	}
	return cngLotToWatchlistLot(*root.Lot), nil
}

func (s *CNGAuctionService) parseLotsPage(rawHTML string) ([]WatchlistLot, cngQueryInfo, error) {
	var root cngViewVars
	if err := parseCNGViewVars(rawHTML, &root); err != nil {
		return nil, cngQueryInfo{}, err
	}
	lots := make([]WatchlistLot, 0, len(root.Lots.ResultPage))
	for _, lot := range root.Lots.ResultPage {
		parsed := cngLotToWatchlistLot(lot)
		if parsed.URL == "" || parsed.LotNumber == 0 {
			continue
		}
		lots = append(lots, parsed)
	}
	return lots, root.Lots.QueryInfo, nil
}

func parseCNGViewVars(rawHTML string, target interface{}) error {
	idx := strings.Index(rawHTML, "viewVars = ")
	if idx < 0 {
		return fmt.Errorf("cng page missing viewVars")
	}
	start := strings.Index(rawHTML[idx:], "{")
	if start < 0 {
		return fmt.Errorf("cng page missing viewVars object")
	}
	start += idx
	end, err := findJSONObjectEnd(rawHTML, start)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(rawHTML[start:end+1]), target)
}

func findJSONObjectEnd(raw string, start int) (int, error) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(raw); i++ {
		ch := raw[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("cng viewVars object is incomplete")
}

type cngViewVars struct {
	Lot  *cngLot `json:"lot"`
	Lots struct {
		ResultPage []cngLot     `json:"result_page"`
		QueryInfo  cngQueryInfo `json:"query_info"`
	} `json:"lots"`
}

type cngQueryInfo struct {
	TotalNumResults int `json:"total_num_results"`
	PageSize        int `json:"page_size"`
}

func (q cngQueryInfo) totalPages() int {
	if q.TotalNumResults <= 0 || q.PageSize <= 0 {
		return 1
	}
	pages := q.TotalNumResults / q.PageSize
	if q.TotalNumResults%q.PageSize != 0 {
		pages++
	}
	if pages < 1 {
		return 1
	}
	return pages
}

type cngLot struct {
	RowID                string     `json:"row_id"`
	LotNumber            int        `json:"lot_number"`
	LotNumberExtension   string     `json:"lot_number_extension"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	TruncatedDescription string     `json:"truncated_description"`
	EstimateLow          string     `json:"estimate_low"`
	EstimateHigh         string     `json:"estimate_high"`
	CurrencyCode         string     `json:"currency_code"`
	StartingPrice        string     `json:"starting_price"`
	SoldPrice            string     `json:"sold_price"`
	Status               string     `json:"status"`
	DetailURL            string     `json:"_detail_url"`
	CoverThumbnail       string     `json:"cover_thumbnail"`
	Images               []cngImage `json:"images"`
	Auction              cngAuction `json:"auction"`
}

type cngImage struct {
	DetailURL    string `json:"detail_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type cngAuction struct {
	RowID            string `json:"row_id"`
	Title            string `json:"title"`
	CurrencyCode     string `json:"currency_code"`
	TimeStart        string `json:"time_start"`
	EffectiveEndTime string `json:"effective_end_time"`
}

func cngLotToWatchlistLot(lot cngLot) WatchlistLot {
	currency := firstNonEmpty(lot.CurrencyCode, lot.Auction.CurrencyCode, "USD")
	imageURL := lot.CoverThumbnail
	if imageURL == "" && len(lot.Images) > 0 {
		imageURL = firstNonEmpty(lot.Images[0].DetailURL, lot.Images[0].ThumbnailURL)
	}
	currentBid, _ := parseCNGDecimal(lot.StartingPrice)
	estimate, _ := parseCNGDecimal(firstNonEmpty(lot.EstimateLow, lot.EstimateHigh))
	saleDate := firstNonEmpty(lot.Auction.EffectiveEndTime, lot.Auction.TimeStart)
	description := cleanHTML(firstNonEmpty(lot.Description, lot.TruncatedDescription))
	if len(description) > 2000 {
		description = description[:2000]
	}

	return WatchlistLot{
		URL:          normalizeCNGURL(lot.DetailURL),
		SourceLotID:  lot.RowID,
		SourceSaleID: lot.Auction.RowID,
		SaleID:       lot.Auction.RowID,
		LotNumber:    lot.LotNumber,
		Title:        strings.TrimSpace(lot.Title),
		ImageURL:     imageURL,
		Estimate:     estimate,
		CurrentBid:   currentBid,
		Currency:     strings.ToUpper(currency),
		AuctionHouse: "Classical Numismatic Group",
		SaleName:     strings.TrimSpace(lot.Auction.Title),
		SaleDate:     saleDate,
		Description:  description,
	}
}

func cngWatchlistLotToDetails(lot WatchlistLot) *LotPageDetails {
	return &LotPageDetails{
		ImageURL:     lot.ImageURL,
		AuctionHouse: lot.AuctionHouse,
		SaleName:     lot.SaleName,
		SaleDate:     lot.SaleDate,
		LotNumber:    lot.LotNumber,
		CurrentBid:   lot.CurrentBid,
		Currency:     lot.Currency,
		Description:  lot.Description,
	}
}

func normalizeCNGURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.IsAbs() {
		return parsed.String()
	}
	if strings.HasPrefix(raw, "/") {
		return cngBase + raw
	}
	return cngBase + "/" + raw
}

func cngWatchlistPageURL(page int) (string, error) {
	parsed, err := url.Parse(cngWatchlistURL)
	if err != nil {
		return "", fmt.Errorf("invalid CNG watchlist URL: %w", err)
	}
	if page > 1 {
		query := parsed.Query()
		query.Set("page", strconv.Itoa(page))
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func parseCNGLotID(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if match := cngLotPathRe.FindStringSubmatch(parsed.Path); match != nil {
		return match[1]
	}
	return ""
}

func validateCNGLotURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("invalid CNG lot URL: %w", err)
	}
	if parsed.Scheme != "https" || strings.ToLower(parsed.Hostname()) != "auctions.cngcoins.com" {
		return fmt.Errorf("CNG lot URL must be on https://auctions.cngcoins.com")
	}
	if parseCNGLotID(rawURL) == "" {
		return fmt.Errorf("CNG lot URL must be a /lots/view/ URL")
	}
	return nil
}

func parseCNGDecimal(raw string) (*float64, string) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if raw == "" {
		return nil, ""
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, ""
	}
	return &val, ""
}

func ParseCNGDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
}

func isCNGLoginPrompt(rawHTML string) bool {
	normalized := strings.ToLower(rawHTML)
	return strings.Contains(normalized, `action="/login"`) &&
		strings.Contains(normalized, `name="username"`) &&
		strings.Contains(normalized, `name="password"`)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *CNGAuctionService) trace(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Trace("cng", format, args...)
	}
}

func (s *CNGAuctionService) debug(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Debug("cng", format, args...)
	}
}

func (s *CNGAuctionService) info(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info("cng", format, args...)
	}
}

func (s *CNGAuctionService) warn(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn("cng", format, args...)
	}
}

func (s *CNGAuctionService) error(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error("cng", format, args...)
	}
}
