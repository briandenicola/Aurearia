package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func signedNumistaToken(t *testing.T, secret, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(7), "role": role, "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestNumistaSecurityProviderKeyUsesHeaderOnlyAndCannotLeak(t *testing.T) {
	const secret = "numista-provider-secret-value"
	var requestSnapshot string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Numista-API-Key") != secret {
			t.Fatalf("provider key header=%q", request.Header.Get("Numista-API-Key"))
		}
		if request.Header.Get("Authorization") != "" ||
			strings.Contains(request.URL.String(), secret) {
			t.Fatalf("provider key leaked outside approved header: %s headers=%v", request.URL, request.Header)
		}
		requestSnapshot = request.Method + " " + request.URL.String()
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"` + secret + ` raw provider evidence"}`)),
		}, nil
	})
	client, err := services.NewHTTPNumistaClient(services.NumistaClientConfig{
		BaseURL:      "https://numista.test",
		HTTPClient:   &http.Client{Transport: transport},
		APIKey:       func() string { return secret },
		RetrySleeper: func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), "private IMP TRAIANO", 20)
	if err == nil || strings.Contains(err.Error(), secret) ||
		strings.Contains(err.Error(), "raw provider evidence") ||
		strings.Contains(requestSnapshot, secret) {
		t.Fatalf("unsafe provider failure: request=%q err=%v", requestSnapshot, err)
	}
}

func TestNumistaSecurityCacheTelemetryScoringAndCanonicalLinksAreRedacted(t *testing.T) {
	const (
		query       = "IMP TRAIANO private inscription"
		visibleText = "dealer cabinet label 42"
		secret      = "server-only-key"
		rawError    = "provider raw body"
	)
	searchKey := services.NumistaSearchCacheKey(query, 20)
	detailKey := services.NumistaDetailCacheKey(123)
	for _, forbidden := range []string{query, "TRAIANO", visibleText, secret, rawError} {
		if strings.Contains(searchKey, forbidden) || strings.Contains(detailKey, forbidden) {
			t.Fatalf("cache key leaked %q: search=%q detail=%q", forbidden, searchKey, detailKey)
		}
	}
	if len(searchKey) != 64 || len(detailKey) != 64 {
		t.Fatalf("cache identities are not SHA-256 digests: %q %q", searchKey, detailKey)
	}

	request := models.NumistaLookupRequest{
		Query: query, Path: models.NumistaLookupPathPhoto,
		Evidence: models.NumistaEvidence{
			ObverseInscription: query, VisibleText: visibleText, Material: "Silver",
		},
	}
	assessment := services.NewNumistaV1Scorer().Score(request, models.NumistaCandidate{
		ID: 123, Title: "Trajan Denarius", Material: "Bronze",
	})
	encodedAssessment, _ := json.Marshal(assessment)
	for _, forbidden := range []string{"IMP TRAIANO", "private inscription", visibleText} {
		if strings.Contains(string(encodedAssessment), forbidden) {
			t.Fatalf("scorer explanation leaked %q: %s", forbidden, encodedAssessment)
		}
	}

	telemetry := services.NewNumistaTelemetry(10)
	telemetry.Record(services.NumistaTelemetryEvent{
		Path: models.NumistaLookupPathPhoto, Operation: "broad",
		Status: models.NumistaStatusUnavailable, CacheOutcome: services.NumistaCacheOutcomeLoader,
		CorrelationDigest: strings.Join([]string{query, visibleText, secret, rawError}, "|"),
	})
	encodedHealth, _ := json.Marshal(telemetry.Health(true, true))
	for _, forbidden := range []string{query, visibleText, secret, rawError, "correlationDigest"} {
		if strings.Contains(string(encodedHealth), forbidden) {
			t.Fatalf("telemetry response leaked %q: %s", forbidden, encodedHealth)
		}
	}

	canonical, err := models.CanonicalNumistaURL(123)
	if err != nil || canonical != "https://en.numista.com/catalogue/pieces123.html" {
		t.Fatalf("canonical URL=%q err=%v", canonical, err)
	}
	if _, err := models.ParseSelectedNumistaReference(
		"123", "https://evil.example/catalogue/pieces123.html",
	); err == nil {
		t.Fatal("arbitrary selected link was accepted")
	}
}

func TestNumistaSecurityAuthAdminAndOversizedBodies(t *testing.T) {
	const jwtSecret = "numista-integration-jwt-secret"
	provider := &integrationNumistaProvider{candidates: []models.NumistaCandidate{
		{ID: 123, Title: "Trajan Denarius"},
	}}
	settings := &integrationNumistaSettings{
		key: "configured-provider-key",
		config: services.NumistaSettings{
			SearchTTL: time.Hour, DetailTTL: time.Hour, SearchResultLimit: 20, Valid: true,
		},
	}
	telemetry := services.NewNumistaTelemetry(20)
	lookup := services.NewNumistaLookupService(
		provider, services.NewNumistaCache(nil, 20, 20), services.NewNumistaV1Scorer(),
		telemetry, settings, nil,
	)
	numistaHandler := handlers.NewNumistaHandler(lookup)
	adminHandler := handlers.NewAdminNumistaHandler(telemetry, settings)
	router := gin.New()
	api := router.Group("/api")
	api.Use(middleware.AuthRequired(jwtSecret, nil))
	api.POST("/numista/lookup", numistaHandler.Lookup)
	admin := api.Group("/admin")
	admin.Use(handlers.AdminRequired())
	admin.GET("/numista/health", adminHandler.Health)

	validBody := []byte(`{"query":"Trajan","path":"direct","evidence":{},"querySource":"manual"}`)
	if response := performRequest(t, router, http.MethodPost, "/api/numista/lookup", "application/json", validBody); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated lookup status=%d body=%s", response.Code, response.Body.String())
	}

	userToken := signedNumistaToken(t, jwtSecret, string(models.RoleUser))
	request := httptest.NewRequest(http.MethodGet, "/api/admin/numista/health", nil)
	request.Header.Set("Authorization", "Bearer "+userToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin health status=%d body=%s", response.Code, response.Body.String())
	}
	adminToken := signedNumistaToken(t, jwtSecret, string(models.RoleAdmin))
	request = httptest.NewRequest(http.MethodGet, "/api/admin/numista/health", nil)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "configured-provider-key") {
		t.Fatalf("admin health status=%d body=%s", response.Code, response.Body.String())
	}

	oversizedQuery, _ := json.Marshal(models.NumistaLookupRequest{
		Query: strings.Repeat("x", models.NumistaMaxQueryLength+1),
		Path:  models.NumistaLookupPathDirect, Evidence: models.NumistaEvidence{},
	})
	request = httptest.NewRequest(http.MethodPost, "/api/numista/lookup", bytes.NewReader(oversizedQuery))
	request.Header.Set("Authorization", "Bearer "+userToken)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || provider.searches != 0 {
		t.Fatalf("oversized input status=%d searches=%d body=%s", response.Code, provider.searches, response.Body.String())
	}

	limited := gin.New()
	limited.Use(middleware.RequestBodyLimit(1024))
	limited.POST("/api/numista/lookup", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"unexpected": true})
	})
	response = performRequest(
		t, limited, http.MethodPost, "/api/numista/lookup", "application/json",
		bytes.Repeat([]byte("x"), 1025),
	)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNumistaSecurityCollectorResponsesNeverExposeProviderSecretsOrRawEvidence(t *testing.T) {
	const secret = "provider-secret-341"
	var logs bytes.Buffer
	previousLogWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousLogWriter) })
	provider := &failingIntegrationProvider{err: errors.New(secret + " raw evidence")}
	settings := &integrationNumistaSettings{
		key:    secret,
		config: services.NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true},
	}
	lookup := services.NewNumistaLookupService(
		provider, services.NewNumistaCache(nil, 10, 10), services.NewNumistaV1Scorer(),
		services.NewNumistaTelemetry(10), settings, nil,
	)
	router := gin.New()
	router.POST("/lookup", handlers.NewNumistaHandler(lookup).Lookup)
	response := performRequest(
		t, router, http.MethodPost, "/lookup", "application/json",
		[]byte(`{"query":"private inscription","path":"direct","evidence":{"visibleText":"dealer label"},"querySource":"manual"}`),
	)
	if response.Code != http.StatusInternalServerError ||
		responseContainsAny(response.Body.String(), secret, "raw evidence", "private inscription", "dealer label") ||
		responseContainsAny(logs.String(), secret, "raw evidence", "private inscription", "dealer label") {
		t.Fatalf(
			"collector surface leaked sensitive data: status=%d body=%s logs=%s",
			response.Code, response.Body.String(), logs.String(),
		)
	}
}

type failingIntegrationProvider struct{ err error }

func (p *failingIntegrationProvider) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	return nil, p.err
}
func (p *failingIntegrationProvider) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, p.err
}
