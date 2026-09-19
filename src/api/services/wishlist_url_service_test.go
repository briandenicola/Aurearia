package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

type wishlistURLExtractorStub struct {
	calls int
	req   WishlistURLExtractionRequest
	resp  WishlistURLExtractionResponse
}

func (s *wishlistURLExtractorStub) ExtractWishlistURL(_ context.Context, req WishlistURLExtractionRequest) (*WishlistURLExtractionResponse, error) {
	s.calls++
	s.req = req
	return &s.resp, nil
}

type wishlistURLRoundTripper func(*http.Request) (*http.Response, error)

func (f wishlistURLRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCanonicalizeWishlistURL(t *testing.T) {
	got, err := CanonicalizeWishlistURL("HTTPS://Example.COM:443/lot/42?utm_source=x&id=7&fbclid=y#details")
	if err != nil {
		t.Fatalf("CanonicalizeWishlistURL() error = %v", err)
	}
	if want := "https://example.com/lot/42?id=7#details"; got != want {
		t.Fatalf("CanonicalizeWishlistURL() = %q, want %q", got, want)
	}

	for _, rawURL := range []string{
		"http://localhost/lot",
		"http://127.0.0.1/lot",
		"http://[::1]/lot",
		"ftp://example.com/lot",
		"https://user:pass@example.com/lot",
	} {
		if _, err := CanonicalizeWishlistURL(rawURL); err == nil {
			t.Errorf("CanonicalizeWishlistURL(%q) unexpectedly succeeded", rawURL)
		}
	}
}

func TestCleanWishlistListingHTMLRemovesChromeAndRepeatedText(t *testing.T) {
	title, text, metadata, err := cleanWishlistListingHTML(strings.NewReader(`
		<html><head><title>Roman Denarius</title>
		<meta property="og:image" content="/coin.jpg">
		<meta name="description" content="Dealer description"></head>
		<body><nav>Browse auctions</nav><main>
		<h1>Hadrian Denarius</h1><p>HADRIANVS AVG COS III P P</p>
		<p>HADRIANVS AVG COS III P P</p><form><button>Bid now</button></form>
		<section class="related-listings">Other coin</section></main>
		<footer>Shipping policy</footer></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if title != "Roman Denarius" || metadata.Description != "Dealer description" {
		t.Fatalf("unexpected metadata: title=%q metadata=%+v", title, metadata)
	}
	if strings.Count(text, "HADRIANVS AVG COS III P P") != 1 {
		t.Fatalf("expected deduplicated legend, got %q", text)
	}
	for _, excluded := range []string{"Browse auctions", "Bid now", "Other coin", "Shipping policy"} {
		if strings.Contains(text, excluded) {
			t.Fatalf("cleaned text retained %q: %q", excluded, text)
		}
	}
}

func TestWishlistURLAnalyzeReturnsDuplicateBeforeRetrieval(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewCoinRepository(db)
	coin := models.Coin{UserID: 9, Name: "Existing", IsWishlist: true, ReferenceURL: "https://dealer.example/lot/42?id=7"}
	if err := repo.Create(&coin); err != nil {
		t.Fatal(err)
	}
	extractor := &wishlistURLExtractorStub{}
	service := NewWishlistURLService(repo, extractor, nil)
	var requests atomic.Int32
	service.client.Transport = wishlistURLRoundTripper(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, nil
	})

	result, err := service.Analyze(context.Background(), 9, "https://DEALER.example/lot/42?utm_source=x&id=7")
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "duplicate" || result.ExistingCoinID == nil || *result.ExistingCoinID != coin.ID {
		t.Fatalf("unexpected duplicate result: %+v", result)
	}
	if requests.Load() != 0 || extractor.calls != 0 {
		t.Fatalf("duplicate path made network/extractor calls: requests=%d extractor=%d", requests.Load(), extractor.calls)
	}
}

func TestWishlistURLAnalyzeFetchesOneBoundedPage(t *testing.T) {
	db := setupTestDB(t)
	extractor := &wishlistURLExtractorStub{resp: WishlistURLExtractionResponse{
		Hypothesis: WishlistURLHypothesis{
			Name: &WishlistURLHypothesisField{Value: "Hadrian Denarius", Confidence: 0.9, Evidence: []string{"page title"}},
		},
	}}
	service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
	var requests atomic.Int32
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:       io.NopCloser(strings.NewReader(`<html><head><title>Hadrian Denarius</title><meta property="og:image" content="/coin.jpg"></head><body><main>Silver denarius <a href="https://other.example/next">next</a></main></body></html>`)),
			Request:    req,
		}, nil
	})}

	result, err := service.Analyze(context.Background(), 3, "https://dealer.example/lot/42")
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected exactly one request, got %d", requests.Load())
	}
	if result.Outcome != "ready" || result.ImageURL != "https://dealer.example/coin.jpg" {
		t.Fatalf("unexpected analysis: %+v", result)
	}
	if extractor.calls != 1 || strings.Contains(extractor.req.PageText, "https://other.example/next") {
		t.Fatalf("unexpected extraction request: %+v", extractor.req)
	}
}

func TestWishlistURLAnalyzeRejectsOversizedAndNonHTMLResponses(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "non HTML", contentType: "application/json", body: `{}`},
		{name: "oversized", contentType: "text/html", body: strings.Repeat("x", wishlistURLMaxResponseBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewWishlistURLService(repository.NewCoinRepository(setupTestDB(t)), &wishlistURLExtractorStub{}, nil)
			service.client = &http.Client{Transport: wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{test.contentType}},
					Body:       io.NopCloser(strings.NewReader(test.body)),
					Request:    req,
				}, nil
			})}
			if _, err := service.Analyze(context.Background(), 1, "https://dealer.example/lot/42"); !errors.Is(err, ErrWishlistURLFetchFailed) {
				t.Fatalf("Analyze() error = %v, want ErrWishlistURLFetchFailed", err)
			}
		})
	}
}
