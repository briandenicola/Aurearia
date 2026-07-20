package services

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func newScraperClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	return &http.Client{Jar: jar}, nil
}

func newScraperRequest(method, rawURL string, body io.Reader, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper request: %w", err)
	}
	applyScraperHeaders(req, headers)
	return req, nil
}

func newScraperFormRequest(rawURL string, form url.Values, headers map[string]string) (*http.Request, error) {
	req, err := newScraperRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()), headers)
	if err != nil {
		return nil, err
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return req, nil
}

func doScraperRequest(client *http.Client, req *http.Request, operation string, okStatuses ...int) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("%s request failed: nil HTTP client", scraperOperation(operation))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", scraperOperation(operation), err)
	}
	return readScraperResponseBody(resp, operation, okStatuses...)
}

func readScraperResponseBody(resp *http.Response, operation string, okStatuses ...int) ([]byte, error) {
	op := scraperOperation(operation)
	if resp == nil {
		return nil, fmt.Errorf("%s returned no response", op)
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("%s returned no response body", op)
	}
	defer resp.Body.Close()

	if !scraperStatusOK(resp.StatusCode, okStatuses...) {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("%s returned HTTP %d", op, resp.StatusCode)
	}

	// Handle Content-Encoding explicitly. Go transparently decompresses gzip only
	// when it set Accept-Encoding itself — not when the caller set it. NumisBids
	// requests set Accept-Encoding: gzip, deflate explicitly to prevent Cloudflare
	// from sending Brotli (verified: real /watchlist response returns
	// content-encoding: br when a Chrome user-agent is used). Go has no built-in
	// Brotli support, so we must control the negotiation and decompress gzip manually
	// for requests where we own the Accept-Encoding header.
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil, fmt.Errorf("failed to create gzip reader for %s response: %w", op, err)
		}
		body, err := io.ReadAll(gr)
		_ = gr.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip %s response: %w", op, err)
		}
		return body, nil
	case "br":
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("%s returned brotli-compressed response; ensure Accept-Encoding excludes br", op)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s body: %w", op, err)
	}
	return body, nil
}

func applyScraperHeaders(req *http.Request, headers map[string]string) {
	for name, value := range headers {
		req.Header.Set(name, value)
	}
}

func scraperStatusOK(status int, okStatuses ...int) bool {
	if len(okStatuses) == 0 {
		return status == http.StatusOK
	}
	for _, okStatus := range okStatuses {
		if status == okStatus {
			return true
		}
	}
	return false
}

func scraperOperation(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return "scraper"
	}
	return operation
}
