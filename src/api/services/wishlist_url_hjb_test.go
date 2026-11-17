package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const hjbRecordedListingURL = "https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010?fromBbs=237th%20Buy%20Or%20Bid%20Sale"

func TestCanonicalizeWishlistURLNormalizesHJBListing(t *testing.T) {
	want := "https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010"
	for _, rawURL := range []string{
		hjbRecordedListingURL,
		"https://hjbltd.com/#!/inventory/item-detail/ancient-coins/119010",
		"HTTPS://WWW.HJBLTD.COM/#!/inventory/item-detail/ancient-coins/119010?fromBbs=x&utm_source=y",
		"https://www.hjbltd.com#!/inventory/item-detail/ancient-coins/119010",
	} {
		got, err := CanonicalizeWishlistURL(rawURL)
		if err != nil {
			t.Fatalf("CanonicalizeWishlistURL(%q) error = %v", rawURL, err)
		}
		if got != want {
			t.Errorf("CanonicalizeWishlistURL(%q) = %q, want %q", rawURL, got, want)
		}
	}
}

func TestParseHJBListingURLRejectsNonListings(t *testing.T) {
	for _, rawURL := range []string{
		"https://www.hjbltd.com/",
		"https://www.hjbltd.com/#!/inventory/ancient-coins",
		"https://www.hjbltd.com/#!/inventory/item-detail/unknown-type/119010",
		"https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/abc",
		"https://www.hjbltd.com/other/#!/inventory/item-detail/ancient-coins/119010",
		"https://evil.example/#!/inventory/item-detail/ancient-coins/119010",
		"https://hjbltd.com.evil.example/#!/inventory/item-detail/ancient-coins/119010",
	} {
		parsed, err := ValidateOutboundURL(rawURL)
		if err != nil {
			t.Fatalf("ValidateOutboundURL(%q) error = %v", rawURL, err)
		}
		if listing, ok := parseHJBListingURL(parsed); ok {
			t.Errorf("parseHJBListingURL(%q) unexpectedly matched %+v", rawURL, listing)
		}
	}
}

func hjbJSONResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestWishlistURLAnalyzeReadsRecordedHJBListing(t *testing.T) {
	recorded, err := os.ReadFile("testdata/hjb/inventory_lot_detail_cc_119010.json")
	if err != nil {
		t.Fatal(err)
	}
	db := setupTestDB(t)
	extractor := &wishlistURLExtractorStub{resp: WishlistURLExtractionResponse{
		Hypothesis: WishlistURLHypothesis{
			Name: &WishlistURLHypothesisField{Value: "Theodosius I. AE 3", Confidence: 0.9, Evidence: []string{"Theodosius I. AE 3"}},
		},
	}}
	service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
	var requests atomic.Int32
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		if got := req.URL.String(); got != "https://www.hjbltd.com/api/cms/inventory_lot_detail/cc/119010" {
			t.Fatalf("unexpected request URL %q", got)
		}
		return hjbJSONResponse(req, http.StatusOK, string(recorded)), nil
	})}

	result, err := service.Analyze(context.Background(), 3, hjbRecordedListingURL)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected exactly one request, got %d", requests.Load())
	}
	if result.Outcome != "ready" ||
		result.SourceURL != "https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010" ||
		result.PageTitle != "Theodosius I. AE 3" ||
		result.ImageURL != "https://images.hjbltd.com/ancient-coins/cc119010a.jpg?cfcache=2" ||
		result.ReverseImageURL != "https://images.hjbltd.com/ancient-coins/cc119010b.jpg?cfcache=2" {
		t.Fatalf("unexpected analysis: %+v", result)
	}
	if extractor.req.SourceURL != result.SourceURL || extractor.req.PageTitle != "Theodosius I. AE 3" {
		t.Fatalf("unexpected extraction request: %+v", extractor.req)
	}
	for _, want := range []string{
		"Category: JOVIAN-ROMULUS AUGUSTUS (363-476 AD)",
		"Era: Ancient Roman",
		"Metal: Bronze",
		"Grade: EF",
		"Price: 200.00 USD",
		"Quantity available: 1",
		"Sale: 237th Buy or Bid Sale",
		"Lot: 405",
		"Sale closes: 2026-12-03 16:00:00",
		"Description: Theodosius I; 379-395 AD, Antioch",
		"3.35g",
		"ANTΔ in exergue",
	} {
		if !strings.Contains(extractor.req.PageText, want) {
			t.Errorf("page text missing %q:\n%s", want, extractor.req.PageText)
		}
	}
	if strings.Contains(extractor.req.PageText, "Certified") || strings.Contains(extractor.req.PageText, "vimeo") {
		t.Errorf("page text contains unexpected content:\n%s", extractor.req.PageText)
	}
}

func TestWishlistURLAnalyzeFindsHJBDuplicateAcrossReferrers(t *testing.T) {
	db := setupTestDB(t)
	existing := models.Coin{
		Name:         "Theodosius I. AE 3",
		IsWishlist:   true,
		ReferenceURL: "https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010",
		UserID:       3,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	extractor := &wishlistURLExtractorStub{}
	service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("duplicate HJB listing should not be fetched")
		return nil, nil
	})}

	result, err := service.Analyze(context.Background(), 3, hjbRecordedListingURL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "duplicate" || result.ExistingCoinID == nil || *result.ExistingCoinID != existing.ID {
		t.Fatalf("unexpected analysis: %+v", result)
	}
}

func TestWishlistURLAnalyzeRejectsMissingOrMalformedHJBListing(t *testing.T) {
	for name, respond := range map[string]func(*http.Request) *http.Response{
		"not found": func(req *http.Request) *http.Response {
			return hjbJSONResponse(req, http.StatusOK, `{"status":"error","message":"No data found"}`)
		},
		"empty title": func(req *http.Request) *http.Response {
			return hjbJSONResponse(req, http.StatusOK, `{"status":"ok","data":{"WebTitle":"  "}}`)
		},
		"html shell": func(req *http.Request) *http.Response {
			resp := hjbJSONResponse(req, http.StatusOK, `<html><title>{{ngMeta.title}}</title></html>`)
			resp.Header.Set("Content-Type", "text/html")
			return resp
		},
		"server error": func(req *http.Request) *http.Response {
			return hjbJSONResponse(req, http.StatusBadGateway, `{}`)
		},
		"oversized": func(req *http.Request) *http.Response {
			resp := hjbJSONResponse(req, http.StatusOK, `{}`)
			resp.ContentLength = wishlistURLMaxResponseBytes + 1
			return resp
		},
	} {
		t.Run(name, func(t *testing.T) {
			db := setupTestDB(t)
			extractor := &wishlistURLExtractorStub{}
			service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
			service.client = &http.Client{Transport: wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
				return respond(req), nil
			})}
			_, err := service.Analyze(context.Background(), 3, hjbRecordedListingURL)
			if !errors.Is(err, ErrWishlistURLFetchFailed) {
				t.Fatalf("Analyze() error = %v, want ErrWishlistURLFetchFailed", err)
			}
			if extractor.calls != 0 {
				t.Fatalf("extractor should not run, got %d calls", extractor.calls)
			}
		})
	}
}

func TestHJBListingContentMarksPriceOnRequest(t *testing.T) {
	_, text, _ := hjbListingContent(&hjbDetailFields{
		WebTitle:       "Augustus Aureus",
		Estimate:       "0.0000",
		PriceOnRequest: "1",
		IsCertified:    "1",
		BBSTitle:       "237th Buy or Bid Sale",
	})
	if !strings.Contains(text, "Price: on request") || !strings.Contains(text, "Certified: yes") {
		t.Fatalf("unexpected text:\n%s", text)
	}
	if strings.Contains(text, "USD") || strings.Contains(text, "Sale:") {
		t.Fatalf("text should not invent a price or a current sale:\n%s", text)
	}
}
