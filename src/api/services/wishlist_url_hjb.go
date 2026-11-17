package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Harlan J. Berk (hjbltd.com) is an AngularJS app: the coin is identified only
// by the "#!/inventory/item-detail/{type}/{id}" fragment, which browsers never
// send to the server, so every listing returns the same empty shell. The app
// itself loads the coin from a public JSON endpoint, which this adapter reads.

const (
	hjbHost         = "www.hjbltd.com"
	hjbSiteName     = "Harlan J. Berk, Ltd."
	hjbDetailAPIFmt = "https://" + hjbHost + "/api/cms/inventory_lot_detail/%s/%s"
)

var (
	hjbItemDetailPattern = regexp.MustCompile(`^!?/inventory/item-detail/([a-z-]+)/(\d{1,12})(?:[/?].*)?$`)
	// Mirrors translateInvType in the dealer's own scripts.
	hjbInventoryGroups = map[string]string{
		"us-coins":        "us",
		"ancient-coins":   "cc",
		"world-coins":     "wd",
		"us-currency":     "ap",
		"antiquities":     "aq",
		"books":           "bk",
		"paintings":       "pt",
		"maps-and-prints": "mp",
	}
)

type hjbListing struct {
	pageURL  string
	group    string
	itemType string
	itemID   string
}

// parseHJBListingURL recognizes an HJB item-detail URL, with or without the
// optional "www." host prefix.
func parseHJBListingURL(parsed *url.URL) (hjbListing, bool) {
	host := strings.ToLower(parsed.Hostname())
	if host != hjbHost && host != "hjbltd.com" {
		return hjbListing{}, false
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return hjbListing{}, false
	}
	match := hjbItemDetailPattern.FindStringSubmatch(parsed.Fragment)
	if match == nil {
		return hjbListing{}, false
	}
	group, ok := hjbInventoryGroups[match[1]]
	if !ok {
		return hjbListing{}, false
	}
	return hjbListing{
		pageURL:  fmt.Sprintf("https://%s/#!/inventory/item-detail/%s/%s", hjbHost, match[1], match[2]),
		group:    group,
		itemType: match[1],
		itemID:   match[2],
	}, true
}

type hjbDetailResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Data    *hjbDetailFields `json:"data"`
}

type hjbDetailFields struct {
	WebTitle       string `json:"WebTitle"`
	Category       string `json:"Category"`
	Era            string `json:"Era"`
	Metal          string `json:"Metal"`
	Grade          string `json:"Grade"`
	IsCertified    string `json:"isCertified"`
	Estimate       string `json:"Estimate"`
	PriceOnRequest string `json:"PriceOnRequest"`
	Quantity       string `json:"Quantity"`
	BBSTitle       string `json:"BBSTitle"`
	PointOfSale    string `json:"PointOfSale"`
	LotNumber      string `json:"LotNumber"`
	CloseTime      string `json:"CloseTime"`
	IsCurrentBBS   string `json:"isCurrentBBS"`
	Description    string `json:"Description"`
	Lot            []struct {
		ImageUrls []string `json:"ImageUrls"`
	} `json:"Lot"`
}

func (s *WishlistURLService) fetchHJBListing(ctx context.Context, listing hjbListing) (string, string, WishlistURLPageMetadata, string, error) {
	apiURL := fmt.Sprintf(hjbDetailAPIFmt, url.PathEscape(listing.group), url.PathEscape(listing.itemID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: %v", ErrWishlistURLInvalid, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", wishlistURLUserAgent)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")

	response, err := s.client.Do(request)
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: %v", ErrWishlistURLFetchFailed, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: HTTP %d", ErrWishlistURLFetchFailed, response.StatusCode)
	}
	if response.ContentLength > wishlistURLMaxResponseBytes {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: response exceeds size limit", ErrWishlistURLFetchFailed)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: unsupported content type", ErrWishlistURLFetchFailed)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, wishlistURLMaxResponseBytes+1))
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: read response: %v", ErrWishlistURLFetchFailed, err)
	}
	if len(body) > wishlistURLMaxResponseBytes {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: response exceeds size limit", ErrWishlistURLFetchFailed)
	}

	var detail hjbDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: parse listing data: %v", ErrWishlistURLFetchFailed, err)
	}
	if detail.Status != "ok" || detail.Data == nil || strings.TrimSpace(detail.Data.WebTitle) == "" {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: listing not found", ErrWishlistURLFetchFailed)
	}

	title, text, metadata := hjbListingContent(detail.Data)
	metadata.ImageURL = resolvePublicWishlistImage(listing.pageURL, metadata.ImageURL)
	return title, text, metadata, listing.pageURL, nil
}

// hjbListingContent renders the dealer's fields as labeled lines so the
// extractor can quote them verbatim as evidence.
func hjbListingContent(data *hjbDetailFields) (string, string, WishlistURLPageMetadata) {
	title := strings.TrimSpace(data.WebTitle)
	lines := []string{title}
	add := func(label, value string) {
		value = strings.Join(strings.Fields(value), " ")
		if value != "" {
			lines = append(lines, label+": "+value)
		}
	}
	add("Category", data.Category)
	add("Era", data.Era)
	add("Metal", data.Metal)
	add("Grade", data.Grade)
	if data.IsCertified == "1" {
		lines = append(lines, "Certified: yes")
	}
	if data.PriceOnRequest == "1" {
		lines = append(lines, "Price: on request")
	} else if price, err := strconv.ParseFloat(strings.TrimSpace(data.Estimate), 64); err == nil && price > 0 {
		// HJB is a US dealer and lists every price in US dollars.
		lines = append(lines, fmt.Sprintf("Price: %.2f USD", price))
	}
	add("Quantity available", data.Quantity)
	if data.IsCurrentBBS == "1" {
		add("Sale", firstWishlistNonEmpty(data.BBSTitle, data.PointOfSale))
		add("Lot", data.LotNumber)
		add("Sale closes", data.CloseTime)
	}
	add("Description", data.Description)

	metadata := WishlistURLPageMetadata{SiteName: hjbSiteName}
	for _, lot := range data.Lot {
		for _, image := range lot.ImageUrls {
			if strings.TrimSpace(image) != "" {
				metadata.ImageURL = strings.TrimSpace(image)
				break
			}
		}
		if metadata.ImageURL != "" {
			break
		}
	}
	return title, truncateWishlistURLText(strings.Join(lines, "\n")), metadata
}
