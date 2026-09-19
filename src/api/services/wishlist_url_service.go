package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"golang.org/x/net/html"
)

const (
	wishlistURLMaxResponseBytes = 1 << 20
	wishlistURLMaxCleanedChars  = 50_000
	wishlistURLUserAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)

var (
	ErrWishlistURLInvalid     = errors.New("invalid wishlist URL")
	ErrWishlistURLFetchFailed = errors.New("wishlist URL fetch failed")
	wishlistURLTrackingKeys   = []string{"fbclid", "gclid", "dclid", "msclkid", "mc_cid", "mc_eid", "_ga"}
	wishlistURLSkipTags       = map[string]bool{
		"script": true, "style": true, "noscript": true, "svg": true, "nav": true,
		"footer": true, "header": true, "aside": true, "form": true, "button": true,
	}
	wishlistURLNoisePattern = regexp.MustCompile(`(?i)(cookie|consent|shipping|delivery|related|recommend|similar|carousel|newsletter|countdown|bid-controls?|error-message)`)
)

type WishlistURLService struct {
	repo      *repository.CoinRepository
	extractor WishlistURLExtractor
	settings  *SettingsService
	client    *http.Client
	slots     chan struct{}
}

type WishlistURLPageMetadata struct {
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
}

type WishlistURLAnalysis struct {
	SchemaVersion  int                    `json:"schemaVersion"`
	Outcome        string                 `json:"outcome"`
	SourceURL      string                 `json:"sourceUrl"`
	ExistingCoinID *uint                  `json:"existingCoinId,omitempty"`
	PageTitle      string                 `json:"pageTitle,omitempty"`
	ImageURL       string                 `json:"imageUrl,omitempty"`
	Hypothesis     *WishlistURLHypothesis `json:"hypothesis,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
}

func NewWishlistURLService(repo *repository.CoinRepository, extractor WishlistURLExtractor, settings *SettingsService) *WishlistURLService {
	return &WishlistURLService{
		repo:      repo,
		extractor: extractor,
		settings:  settings,
		client:    NewRestrictedHTTPClient(nil, 12*time.Second, 5),
		slots:     make(chan struct{}, 4),
	}
}

func CanonicalizeWishlistURL(rawURL string) (string, error) {
	parsed, err := ValidateOutboundURL(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrWishlistURLInvalid, err)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port != "" && !((parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443")) {
		host += ":" + port
	}
	parsed.Host = host
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || slices.Contains(wishlistURLTrackingKeys, lower) {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (s *WishlistURLService) Analyze(ctx context.Context, userID uint, rawURL string) (*WishlistURLAnalysis, error) {
	canonicalURL, err := CanonicalizeWishlistURL(rawURL)
	if err != nil {
		return nil, err
	}
	if duplicate, err := s.findDuplicate(userID, canonicalURL); err != nil {
		return nil, err
	} else if duplicate != nil {
		return duplicate, nil
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	title, text, metadata, finalURL, err := s.fetchAndClean(ctx, canonicalURL)
	if err != nil {
		return nil, err
	}
	if finalURL != canonicalURL {
		if duplicate, err := s.findDuplicate(userID, finalURL); err != nil {
			return nil, err
		} else if duplicate != nil {
			return duplicate, nil
		}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal listing metadata: %w", err)
	}
	var llm LLMConfig
	if s.settings != nil {
		llm, err = s.settings.ResolveLLMConfig()
		if err != nil {
			return nil, fmt.Errorf("resolve AI provider: %w", err)
		}
	}
	extracted, err := s.extractor.ExtractWishlistURL(ctx, WishlistURLExtractionRequest{
		LLM:          llm,
		SourceURL:    finalURL,
		PageTitle:    title,
		PageText:     text,
		PageMetadata: metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("extract wishlist listing: %w", err)
	}

	outcome := "needs_review"
	if extracted.Hypothesis.Name != nil && strings.TrimSpace(extracted.Hypothesis.Name.Value) != "" {
		outcome = "ready"
	}
	return &WishlistURLAnalysis{
		SchemaVersion: WishlistURLProposalSchemaVersion,
		Outcome:       outcome,
		SourceURL:     finalURL,
		PageTitle:     title,
		ImageURL:      metadata.ImageURL,
		Hypothesis:    &extracted.Hypothesis,
		Warnings:      extracted.Warnings,
	}, nil
}

func (s *WishlistURLService) findDuplicate(userID uint, referenceURL string) (*WishlistURLAnalysis, error) {
	coin, err := s.repo.FindWishlistByReferenceURL(userID, referenceURL)
	if err == nil {
		return duplicateWishlistURLAnalysis(referenceURL, coin), nil
	}
	if repository.IsRecordNotFound(err) {
		return nil, nil
	}
	return nil, fmt.Errorf("check wishlist URL duplicate: %w", err)
}

func duplicateWishlistURLAnalysis(sourceURL string, coin *models.Coin) *WishlistURLAnalysis {
	return &WishlistURLAnalysis{
		SchemaVersion:  WishlistURLProposalSchemaVersion,
		Outcome:        "duplicate",
		SourceURL:      sourceURL,
		ExistingCoinID: &coin.ID,
	}
}

func (s *WishlistURLService) fetchAndClean(ctx context.Context, sourceURL string) (string, string, WishlistURLPageMetadata, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: %v", ErrWishlistURLInvalid, err)
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
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
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: unsupported content type", ErrWishlistURLFetchFailed)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, wishlistURLMaxResponseBytes+1))
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: read response: %v", ErrWishlistURLFetchFailed, err)
	}
	if len(body) > wishlistURLMaxResponseBytes {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: response exceeds size limit", ErrWishlistURLFetchFailed)
	}

	title, text, metadata, err := cleanWishlistListingHTML(strings.NewReader(string(body)))
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", fmt.Errorf("%w: parse HTML: %v", ErrWishlistURLFetchFailed, err)
	}
	finalURL, err := CanonicalizeWishlistURL(response.Request.URL.String())
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, "", err
	}
	metadata.ImageURL = resolvePublicWishlistImage(finalURL, metadata.ImageURL)
	return title, text, metadata, finalURL, nil
}

func cleanWishlistListingHTML(reader io.Reader) (string, string, WishlistURLPageMetadata, error) {
	document, err := html.Parse(reader)
	if err != nil {
		return "", "", WishlistURLPageMetadata{}, err
	}
	var title string
	var metadata WishlistURLPageMetadata
	var lines []string
	seen := make(map[string]bool)

	var walk func(*html.Node, bool)
	walk = func(node *html.Node, skip bool) {
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			if tag == "script" {
				if structured := wishlistURLStructuredData(node); structured != "" && !seen[structured] {
					seen[structured] = true
					lines = append(lines, structured)
				}
			}
			skip = skip || wishlistURLSkipTags[tag] || wishlistURLNoiseNode(node)
			if tag == "meta" {
				readWishlistURLMeta(node, &metadata)
			}
			if tag == "title" && node.FirstChild != nil {
				title = strings.TrimSpace(node.FirstChild.Data)
			}
		}
		if !skip && node.Type == html.TextNode {
			line := strings.Join(strings.Fields(node.Data), " ")
			if line != "" && !seen[line] {
				seen[line] = true
				lines = append(lines, line)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, skip)
		}
	}
	walk(document, false)

	text := strings.Join(lines, "\n")
	text = truncateWishlistURLText(text)
	if title == "" {
		title = firstWishlistNonEmpty(metadata.SiteName, metadata.Description)
	}
	return title, text, metadata, nil
}

func wishlistURLStructuredData(node *html.Node) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, "type") && strings.EqualFold(strings.TrimSpace(attr.Val), "application/ld+json") {
			if node.FirstChild == nil {
				return ""
			}
			raw := strings.TrimSpace(node.FirstChild.Data)
			if raw == "" || !json.Valid([]byte(raw)) {
				return ""
			}
			return truncateWishlistURLText("Structured listing data: " + raw)
		}
	}
	return ""
}

func truncateWishlistURLText(value string) string {
	if len(value) > wishlistURLMaxCleanedChars {
		return value[:wishlistURLMaxCleanedChars]
	}
	return value
}

func wishlistURLNoiseNode(node *html.Node) bool {
	for _, attr := range node.Attr {
		if (attr.Key == "class" || attr.Key == "id" || attr.Key == "role" || attr.Key == "aria-label") &&
			wishlistURLNoisePattern.MatchString(attr.Val) {
			return true
		}
	}
	return false
}

func readWishlistURLMeta(node *html.Node, metadata *WishlistURLPageMetadata) {
	attrs := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		attrs[strings.ToLower(attr.Key)] = strings.TrimSpace(attr.Val)
	}
	key := strings.ToLower(firstWishlistNonEmpty(attrs["property"], attrs["name"]))
	value := attrs["content"]
	switch key {
	case "description", "og:description", "twitter:description":
		if metadata.Description == "" {
			metadata.Description = value
		}
	case "og:image", "twitter:image":
		if metadata.ImageURL == "" {
			metadata.ImageURL = value
		}
	case "og:site_name":
		metadata.SiteName = value
	}
}

func resolvePublicWishlistImage(pageURL, imageURL string) string {
	if strings.TrimSpace(imageURL) == "" {
		return ""
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	image, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(image)
	if _, err := ValidateOutboundURL(resolved.String()); err != nil {
		return ""
	}
	return resolved.String()
}

func firstWishlistNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
