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
)

var ErrCNGAuthenticationRequired = errors.New("cng authentication required")

const (
	cngHost      = "auctions.cngcoins.com"
	cngUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/138.0.0.0 Safari/537.36"
)

var (
	cngBase         = "https://auctions.cngcoins.com"
	cngLoginURL     = cngBase + "/login"
	cngWatchlistURL = cngBase + "/watched-lots"
	cngRefreshMeURL = cngBase + "/ajax/refresh-me"
	cngLotPathRe    = regexp.MustCompile(`^/lots/view/([^/]+)(?:/|$)`)
	cngLotSafePathRe = regexp.MustCompile(`^/lots/view/[A-Za-z0-9._~-]+(?:/[A-Za-z0-9._~%-]+)?/?$`)
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

	client, err := newScraperClient()
	if err != nil {
		return nil, err
	}

	req, err := newScraperRequest(http.MethodGet, cngLoginURL, nil, cngDefaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("failed to create login page request: %w", err)
	}
	if _, err := doScraperRequest(client, req, "login page"); err != nil {
		return nil, err
	}

	form := url.Values{
		"username": {username},
		"password": {password},
		"Login":    {"Login"},
	}
	req, err = newScraperFormRequest(cngLoginURL, form, cngLoginHeaders())
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	if _, err := doScraperRequest(client, req, "login", http.StatusOK, http.StatusFound); err != nil {
		s.error("CNG login HTTP request failed: %v", err)
		return nil, err
	}
	if err := s.verifyAuthentication(client); err != nil {
		s.warn("CNG authentication verification failed: %v", err)
		return nil, fmt.Errorf("login failed: %w", err)
	}

	s.info("CNG login and authentication verified")
	return client, nil
}

func (s *CNGAuctionService) verifyAuthentication(client *http.Client) error {
	_, err := s.fetchCustomerProfile(client)
	return err
}

// CurrentCustomerRowID returns the authenticated user's own CNG customer row_id. This is
// compared against a closed lot's winning bidder to auto-detect won vs. lost during sync.
func (s *CNGAuctionService) CurrentCustomerRowID(client *http.Client) (string, error) {
	profile, err := s.fetchCustomerProfile(client)
	if err != nil {
		return "", err
	}
	if profile.RowID == "" {
		return "", fmt.Errorf("cng refresh-me response missing row_id")
	}
	return profile.RowID, nil
}

type cngCustomerProfile struct {
	RowID string `json:"row_id"`
}

func (s *CNGAuctionService) fetchCustomerProfile(client *http.Client) (cngCustomerProfile, error) {
	req, err := newScraperRequest(http.MethodGet, cngRefreshMeURL, nil, cngRefreshHeaders())
	if err != nil {
		return cngCustomerProfile{}, fmt.Errorf("failed to create refresh-me request: %w", err)
	}

	body, err := doScraperRequest(client, req, "refresh-me")
	if err != nil {
		return cngCustomerProfile{}, err
	}
	if strings.TrimSpace(string(body)) == "null" {
		return cngCustomerProfile{}, ErrCNGAuthenticationRequired
	}
	var profile cngCustomerProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return cngCustomerProfile{}, fmt.Errorf("failed to parse refresh-me response: %w", err)
	}
	return profile, nil
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
	req, err := newScraperRequest(http.MethodGet, watchlistURL, nil, cngDefaultHeaders())
	if err != nil {
		return "", fmt.Errorf("failed to create watchlist request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("watchlist page %d request failed: %w", page, err)
	}

	body, err := readScraperResponseBody(resp, fmt.Sprintf("watchlist page %d", page), http.StatusOK, http.StatusFound, http.StatusUnauthorized)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusUnauthorized {
		return "", ErrCNGAuthenticationRequired
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

// ScrapeLot fetches a CNG lot page using the default (unauthenticated) HTTP client.
func (s *CNGAuctionService) ScrapeLot(lotURL string) (WatchlistLot, error) {
	return s.ScrapeLotWithClient(http.DefaultClient, lotURL)
}

// ScrapeLotWithClient fetches a CNG lot page using the provided HTTP client and extracts a
// source-aware lot summary. Pass the authenticated client from Login during watchlist sync
// to obtain user-specific data (current bid_amount and autobid) that are only returned on
// the individual lot page, not the watched-lots list.
func (s *CNGAuctionService) ScrapeLotWithClient(client *http.Client, lotURL string) (WatchlistLot, error) {
	lotPath, err := canonicalCNGLotPath(lotURL)
	if err != nil {
		return WatchlistLot{}, err
	}
	req, err := newScraperRequest(http.MethodGet, cngBase, nil, cngDefaultHeaders())
	if err != nil {
		return WatchlistLot{}, err
	}
	req.URL.Path = lotPath

	body, err := doScraperRequest(client, req, "lot page")
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
	RowID                string              `json:"row_id"`
	LotNumber            int                 `json:"lot_number"`
	LotNumberExtension   string              `json:"lot_number_extension"`
	Title                string              `json:"title"`
	Description          string              `json:"description"`
	TruncatedDescription string              `json:"truncated_description"`
	EstimateLow          string              `json:"estimate_low"`
	EstimateHigh         string              `json:"estimate_high"`
	CurrencyCode         string              `json:"currency_code"`
	StartingPrice        string              `json:"starting_price"`
	SoldPrice            string              `json:"sold_price"`
	Status               string              `json:"status"`
	DetailURL            string              `json:"_detail_url"`
	CoverThumbnail       string              `json:"cover_thumbnail"`
	Images               []cngImage          `json:"images"`
	Auction              cngAuction          `json:"auction"`
	ExtendedEndTime      string              `json:"extended_end_time"`
	TimedAuctionBid      *cngTimedAuctionBid `json:"timed_auction_bid"`
	AbsenteeBid          *cngAbsenteeBid     `json:"absentee_bid"`
}

// cngTimedAuctionBid is the current (or, once the lot closes, final) bid on a timed-auction
// lot. It is public data: present even for an unauthenticated request. The nested
// registration/customer identifies whose bid it is, which is how a closed lot's winner is
// determined — compare against the logged-in user's own customer row_id.
type cngTimedAuctionBid struct {
	Amount       string `json:"amount"`
	Registration struct {
		Customer struct {
			RowID string `json:"row_id"`
		} `json:"customer"`
	} `json:"registration"`
}

// cngAbsenteeBid is the authenticated user's own bid ceiling on a lot. Only populated when
// fetched with a logged-in client for a lot the user has actually placed a bid on.
type cngAbsenteeBid struct {
	MaxBid string `json:"max_bid"`
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

	// timed_auction_bid.amount is the current (or, once the lot closes, final) bid and is
	// public data; fall back to starting_price only when no bid has been placed yet.
	currentBidStr := lot.StartingPrice
	winningCustomerRowID := ""
	if lot.TimedAuctionBid != nil {
		if lot.TimedAuctionBid.Amount != "" {
			currentBidStr = lot.TimedAuctionBid.Amount
		}
		winningCustomerRowID = lot.TimedAuctionBid.Registration.Customer.RowID
	}
	currentBid, _ := parseCNGDecimal(currentBidStr)

	// absentee_bid.max_bid is the authenticated user's own bid ceiling; nil unless fetched
	// with a logged-in client for a lot the user has actually bid on.
	var maxBid *float64
	if lot.AbsenteeBid != nil {
		maxBid, _ = parseCNGDecimal(lot.AbsenteeBid.MaxBid)
	}

	soldPrice, _ := parseCNGDecimal(lot.SoldPrice)
	estimate, _ := parseCNGDecimal(firstNonEmpty(lot.EstimateLow, lot.EstimateHigh))
	// extended_end_time is this specific lot's own close time. auction.effective_end_time is
	// when the entire (often multi-hundred-lot) sale ends, which can be hours later for any
	// individual lot in a staggered timed auction.
	saleDate := firstNonEmpty(lot.ExtendedEndTime, lot.Auction.EffectiveEndTime, lot.Auction.TimeStart)
	description := cleanHTML(firstNonEmpty(lot.Description, lot.TruncatedDescription))
	if len(description) > 2000 {
		description = description[:2000]
	}

	return WatchlistLot{
		URL:                  normalizeCNGURL(lot.DetailURL),
		SourceLotID:          lot.RowID,
		SourceSaleID:         lot.Auction.RowID,
		SaleID:               lot.Auction.RowID,
		LotNumber:            lot.LotNumber,
		Title:                strings.TrimSpace(lot.Title),
		ImageURL:             imageURL,
		Estimate:             estimate,
		CurrentBid:           currentBid,
		MaxBid:               maxBid,
		Currency:             strings.ToUpper(currency),
		AuctionHouse:         "Classical Numismatic Group",
		SaleName:             strings.TrimSpace(lot.Auction.Title),
		SaleDate:             saleDate,
		Description:          description,
		ProviderStatus:       strings.ToLower(strings.TrimSpace(lot.Status)),
		SoldPrice:            soldPrice,
		WinningCustomerRowID: winningCustomerRowID,
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
	_, err := canonicalCNGLotPath(rawURL)
	return err
}

func canonicalCNGLotPath(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("invalid CNG lot URL: %w", err)
	}
	if parsed.Scheme != "https" || strings.ToLower(parsed.Hostname()) != cngHost || parsed.User != nil {
		return "", fmt.Errorf("CNG lot URL must be on https://auctions.cngcoins.com")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return "", fmt.Errorf("CNG lot URL must use the standard HTTPS port")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("CNG lot URL must not include query parameters or fragments")
	}
	path := parsed.EscapedPath()
	if !cngLotSafePathRe.MatchString(path) || parseCNGLotID(path) == "" {
		return "", fmt.Errorf("CNG lot URL must be a /lots/view/ URL")
	}
	return path, nil
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

func cngDefaultHeaders() map[string]string {
	return map[string]string{
		"User-Agent": cngUserAgent,
	}
}

func cngLoginHeaders() map[string]string {
	headers := cngDefaultHeaders()
	headers["Origin"] = cngBase
	headers["Referer"] = cngLoginURL
	return headers
}

func cngRefreshHeaders() map[string]string {
	headers := cngDefaultHeaders()
	headers["X-Requested-With"] = "XMLHttpRequest"
	headers["Accept"] = "application/json, text/plain, */*"
	return headers
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
