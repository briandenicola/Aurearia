package services

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// CNG lot pages (auctions.cngcoins.com/lots/view/{lotId}/{slug}) render the
// lot description and photo in page metadata, which the generic cleaner
// already reads. The estimate, bid, sale and status live only in the page's
// viewVars script, so this adapter reads them with the parser the CNG auction
// import uses and adds them as labeled lines. It makes no extra request.

const cngWishlistHost = "auctions.cngcoins.com"

var cngWishlistStatusPattern = regexp.MustCompile(`^[a-z_]{1,30}$`)

// canonicalCNGWishlistURL reduces a CNG lot link to its lot id; CNG redirects
// the bare link to the current slug, so the same lot is one wishlist entry.
func canonicalCNGWishlistURL(parsed *url.URL) (string, bool) {
	if !strings.EqualFold(parsed.Hostname(), cngWishlistHost) {
		return "", false
	}
	if !cngLotSafePathRe.MatchString(parsed.EscapedPath()) {
		return "", false
	}
	match := cngLotPathRe.FindStringSubmatch(parsed.Path)
	if match == nil {
		return "", false
	}
	return "https://" + cngWishlistHost + "/lots/view/" + match[1], true
}

// cngWishlistLotLines extracts the auction facts for a CNG lot page body. It
// returns nil when the page is not a CNG lot or carries no lot data, so the
// generic text is used unchanged.
func cngWishlistLotLines(pageURL, body string) (string, []string) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "", nil
	}
	if _, ok := canonicalCNGWishlistURL(parsed); !ok {
		return "", nil
	}
	var root cngViewVars
	if err := parseCNGViewVars(body, &root); err != nil || root.Lot == nil {
		return "", nil
	}
	lot := *root.Lot
	title := strings.Join(strings.Fields(lot.Title), " ")
	currency := strings.ToUpper(firstNonEmpty(lot.CurrencyCode, lot.Auction.CurrencyCode))
	if !wishlistURLCurrencyPattern.MatchString(currency) {
		currency = ""
	}
	money := func(raw string) string {
		value, _ := parseCNGDecimal(raw)
		if value == nil || *value <= 0 {
			return ""
		}
		return strings.TrimSpace(strconv.FormatFloat(*value, 'f', -1, 64) + " " + currency)
	}

	var lines []string
	add := func(label, value string) {
		value = strings.Join(strings.Fields(value), " ")
		if value != "" {
			lines = append(lines, label+": "+value)
		}
	}
	add("Auction", lot.Auction.Title)
	if lot.LotNumber > 0 {
		add("Lot", fmt.Sprintf("%d%s", lot.LotNumber, strings.TrimSpace(lot.LotNumberExtension)))
	}
	low, high := money(lot.EstimateLow), money(lot.EstimateHigh)
	switch {
	case low != "" && high != "" && low != high:
		add("Estimate", strings.TrimSuffix(low, " "+currency)+" to "+high)
	case low != "":
		add("Estimate", low)
	case high != "":
		add("Estimate", high)
	}
	status := strings.ToLower(strings.TrimSpace(lot.Status))
	if status == "sold" {
		lines = append(lines, "Lot status: sold")
		add("Sold for", money(lot.SoldPrice))
	} else {
		if lot.TimedAuctionBid != nil {
			add("Current bid", money(lot.TimedAuctionBid.Amount))
		}
		add("Lot closes", firstNonEmpty(lot.ExtendedEndTime, lot.Auction.EffectiveEndTime))
		if cngWishlistStatusPattern.MatchString(status) {
			add("Lot status", status)
		}
	}
	return title, lines
}
