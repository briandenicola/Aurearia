package services

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// testdata/cng/*.html are trimmed copies of real CNG lot pages: the head
// metadata plus the viewVars.lot fields the CNG parser reads.

func TestCanonicalizeWishlistURLReducesCNGLotToItsID(t *testing.T) {
	want := "https://auctions.cngcoins.com/lots/view/4-6WINT"
	for _, rawURL := range []string{
		"https://auctions.cngcoins.com/lots/view/4-6WINT/lucania-thourioi-circa-443-400-bc-ar-nomos-20mm-712-g-3h",
		"https://auctions.cngcoins.com/lots/view/4-6WINT/old-slug?utm_source=mail",
		"https://AUCTIONS.cngcoins.com/lots/view/4-6WINT",
	} {
		got, err := CanonicalizeWishlistURL(rawURL)
		if err != nil || got != want {
			t.Errorf("CanonicalizeWishlistURL(%q) = %q, %v; want %q", rawURL, got, err, want)
		}
	}
	for _, rawURL := range []string{
		"https://auctions.cngcoins.com/auctions/4-NLJYNX/electronic-auction-621",
		"https://www.cngcoins.com/lots/view/4-6WINT",
		"https://auctions.cngcoins.com.evil.example/lots/view/4-6WINT",
	} {
		if got, _ := CanonicalizeWishlistURL(rawURL); got == want {
			t.Errorf("CanonicalizeWishlistURL(%q) was reduced to the CNG lot link", rawURL)
		}
	}
}

func cngFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/cng/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCNGWishlistLotLines(t *testing.T) {
	sold := "https://auctions.cngcoins.com/lots/view/4-6WINT/lucania-thourioi-circa-443-400-bc-ar-nomos-20mm-712-g-3h"
	title, lines := cngWishlistLotLines(sold, cngFixture(t, "lot_sold_lucania_nomos.html"))
	if title != "LUCANIA, Thourioi. Circa 443-400 BC. AR Nomos (20mm, 7.12 g, 3h)." {
		t.Fatalf("sold title = %q", title)
	}
	if got, want := strings.Join(lines, "|"), "Auction: CNG Keystone Auction 1: Part I of the Style Collection|Lot: 1|Estimate: 1500 USD|Lot status: sold|Sold for: 1400 USD"; got != want {
		t.Fatalf("sold lines = %q, want %q", got, want)
	}

	active := "https://auctions.cngcoins.com/lots/view/4-NLKUR9/eastern-europe"
	_, lines = cngWishlistLotLines(active, cngFixture(t, "lot_active_kreuzelreiter_tetradrachm.html"))
	if got, want := strings.Join(lines, "|"), "Auction: Electronic Auction 621|Lot: 1|Estimate: 500 USD|Current bid: 400 USD|Lot closes: 2026-11-18T15:00:00Z|Lot status: active"; got != want {
		t.Fatalf("active lines = %q, want %q", got, want)
	}

	if title, lines := cngWishlistLotLines("https://dealer.example/lots/view/4-6WINT", cngFixture(t, "lot_sold_lucania_nomos.html")); title != "" || lines != nil {
		t.Fatalf("non-CNG host produced %q %q", title, lines)
	}
	if title, lines := cngWishlistLotLines(sold, "<html><title>No lot</title></html>"); title != "" || lines != nil {
		t.Fatalf("page without viewVars produced %q %q", title, lines)
	}
}

func TestWishlistURLAnalyzeReadsCNGLotFromOneRequest(t *testing.T) {
	db := setupTestDB(t)
	extractor := &wishlistURLExtractorStub{resp: WishlistURLExtractionResponse{Hypothesis: WishlistURLHypothesis{
		Name: &WishlistURLHypothesisField{Value: "Kreuzelreiter tetradrachm", Confidence: 0.9, Evidence: []string{"AR Tetradrachm"}},
	}}}
	service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
	page := cngFixture(t, "lot_active_kreuzelreiter_tetradrachm.html")
	var requests atomic.Int32
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:       io.NopCloser(strings.NewReader(page)),
			Request:    req,
		}, nil
	})}

	result, err := service.Analyze(context.Background(), 3, "https://auctions.cngcoins.com/lots/view/4-NLKUR9/eastern-europe-imitations")
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected one request, got %d", requests.Load())
	}
	if result.SourceURL != "https://auctions.cngcoins.com/lots/view/4-NLKUR9" ||
		!strings.HasPrefix(result.PageTitle, "EASTERN EUROPE, Imitations of Philip II of Macedon.") ||
		!strings.HasPrefix(result.ImageURL, "https://images4-cdn.auctionmobility.com/") {
		t.Fatalf("unexpected analysis: %+v", result)
	}
	for _, want := range []string{"Estimate: 500 USD", "Current bid: 400 USD", "Lot status: active", "Kreuzelreiter"} {
		if !strings.Contains(extractor.req.PageText, want) {
			t.Errorf("page text missing %q:\n%s", want, extractor.req.PageText)
		}
	}
	for _, excluded := range []string{"{{", "Optimized for your Mobile Device", "redacted"} {
		if strings.Contains(extractor.req.PageText, excluded) {
			t.Errorf("page text retained %q:\n%s", excluded, extractor.req.PageText)
		}
	}
}

func TestWishlistURLAnalyzeFindsCNGDuplicateAcrossSlugs(t *testing.T) {
	db := setupTestDB(t)
	existing := models.Coin{Name: "Lucania nomos", IsWishlist: true, UserID: 3, ReferenceURL: "https://auctions.cngcoins.com/lots/view/4-6WINT"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	service := NewWishlistURLService(repository.NewCoinRepository(db), &wishlistURLExtractorStub{}, nil)
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("duplicate CNG lot should not be fetched")
		return nil, nil
	})}
	result, err := service.Analyze(context.Background(), 3, "https://auctions.cngcoins.com/lots/view/4-6WINT/a-different-slug")
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "duplicate" || result.ExistingCoinID == nil || *result.ExistingCoinID != existing.ID {
		t.Fatalf("unexpected analysis: %+v", result)
	}
}

func TestCleanWishlistListingHTMLPrefersRenderedTitle(t *testing.T) {
	title, text, _, err := cleanWishlistListingHTML(strings.NewReader(
		`<html><head><title>Real lot</title><title>{{template.title}}</title></head><body><p>{{ "Bid now" | translate }}</p><p>Coin</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if title != "Real lot" || strings.Contains(text, "{{") || !strings.Contains(text, "Coin") {
		t.Fatalf("title=%q text=%q", title, text)
	}
}
