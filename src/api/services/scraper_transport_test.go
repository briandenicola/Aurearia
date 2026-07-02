package services

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNewScraperClientCreatesCookieJarClient(t *testing.T) {
	client, err := newScraperClient()
	if err != nil {
		t.Fatalf("newScraperClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("newScraperClient returned nil client")
	}
	if client.Jar == nil {
		t.Fatal("newScraperClient returned client without cookie jar")
	}

	parsed, err := url.Parse("https://example.test/login")
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}
	client.Jar.SetCookies(parsed, []*http.Cookie{{Name: "session", Value: "abc"}})
	if got := client.Jar.Cookies(parsed); len(got) != 1 || got[0].Name != "session" || got[0].Value != "abc" {
		t.Fatalf("cookie jar did not retain cookie: %#v", got)
	}
}

func TestNewScraperRequestAppliesHeaders(t *testing.T) {
	req, err := newScraperRequest(http.MethodGet, "https://example.test/watchlist", nil, map[string]string{
		"User-Agent":       "Aurearia Test Agent",
		"Referer":          "https://example.test/",
		"X-Requested-With": "XMLHttpRequest",
	})
	if err != nil {
		t.Fatalf("newScraperRequest returned error: %v", err)
	}
	if req.Method != http.MethodGet {
		t.Fatalf("Method = %q, want GET", req.Method)
	}
	if req.Header.Get("User-Agent") != "Aurearia Test Agent" {
		t.Fatalf("User-Agent = %q", req.Header.Get("User-Agent"))
	}
	if req.Header.Get("Referer") != "https://example.test/" {
		t.Fatalf("Referer = %q", req.Header.Get("Referer"))
	}
	if req.Header.Get("X-Requested-With") != "XMLHttpRequest" {
		t.Fatalf("X-Requested-With = %q", req.Header.Get("X-Requested-With"))
	}
}

func TestNewScraperFormRequestAppliesFormContentType(t *testing.T) {
	req, err := newScraperFormRequest("https://example.test/login", url.Values{"email": {"user@example.com"}}, nil)
	if err != nil {
		t.Fatalf("newScraperFormRequest returned error: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", req.Method)
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(body) != "email=user%40example.com" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestDoScraperRequestHandlesNonOKStatus(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("temporarily unavailable")),
		}, nil
	})}
	req, err := newScraperRequest(http.MethodGet, "https://example.test/watchlist", nil, nil)
	if err != nil {
		t.Fatalf("newScraperRequest returned error: %v", err)
	}

	body, err := doScraperRequest(client, req, "watchlist")
	if err == nil {
		t.Fatal("doScraperRequest returned nil error for non-OK status")
	}
	if body != nil {
		t.Fatalf("body = %q, want nil", string(body))
	}
	if !strings.Contains(err.Error(), "watchlist returned HTTP 503") {
		t.Fatalf("error = %v, want HTTP 503 context", err)
	}
}

func TestReadScraperResponseBodyReadsAndClosesBody(t *testing.T) {
	tracker := &trackingReadCloser{Reader: strings.NewReader("watchlist html")}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       tracker,
	}

	body, err := readScraperResponseBody(resp, "watchlist")
	if err != nil {
		t.Fatalf("readScraperResponseBody returned error: %v", err)
	}
	if string(body) != "watchlist html" {
		t.Fatalf("body = %q", string(body))
	}
	if !tracker.closed {
		t.Fatal("response body was not closed")
	}
}

func TestDoScraperRequestWrapsRequestFailure(t *testing.T) {
	requestErr := errors.New("network unavailable")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, requestErr
	})}
	req, err := newScraperRequest(http.MethodGet, "https://example.test/watchlist", nil, nil)
	if err != nil {
		t.Fatalf("newScraperRequest returned error: %v", err)
	}

	_, err = doScraperRequest(client, req, "watchlist")
	if !errors.Is(err, requestErr) {
		t.Fatalf("error = %v, want wrapped request error", err)
	}
	if !strings.Contains(err.Error(), "watchlist request failed") {
		t.Fatalf("error = %v, want operation context", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type trackingReadCloser struct {
	*strings.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
