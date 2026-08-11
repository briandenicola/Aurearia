package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const numistaStatusJWTSecret = "numista-status-test-secret"

func numistaStatusToken(t *testing.T, role models.UserRole) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(42),
		"role":   string(role),
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
	})
	signed, err := token.SignedString([]byte(numistaStatusJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func newAuthenticatedNumistaStatusRouter(client services.NumistaClient, key string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	lookup := services.NewNumistaLookupService(
		client,
		services.NewNumistaCache(nil, 20, 20),
		services.NewNumistaV1Scorer(),
		services.NewNumistaTelemetry(20),
		handlerNumistaSettings{key: key},
		nil,
	)
	handler := NewNumistaHandler(lookup)
	router := gin.New()
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired(numistaStatusJWTSecret, nil))
	protected.POST("/numista/lookup", handler.Lookup)
	protected.GET("/numista/search", handler.Search)
	return router
}

func performNumistaStatusLookup(
	t *testing.T,
	router http.Handler,
	role models.UserRole,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/numista/lookup", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+numistaStatusToken(t, role))
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestNumistaStatusHandlerReturnsExpectedDomainOutcomesWithRoleSafeGuidance(t *testing.T) {
	retryAfter := 60
	tests := []struct {
		name         string
		key          string
		client       services.NumistaClient
		role         models.UserRole
		wantStatus   models.NumistaLookupStatus
		wantGuidance string
		wantRetry    *int
	}{
		{
			name: "success", key: "configured", role: models.RoleUser,
			client:       &handlerNumistaClient{candidates: []models.NumistaCandidate{{ID: 1, Title: "Coin"}}},
			wantStatus:   models.NumistaStatusSuccess,
			wantGuidance: "",
		},
		{
			name: "empty", key: "configured", role: models.RoleUser,
			client: &handlerNumistaClient{}, wantStatus: models.NumistaStatusEmpty,
			wantGuidance: "revise_numista_query",
		},
		{
			name: "unconfigured admin", role: models.RoleAdmin,
			client: &handlerNumistaClient{}, wantStatus: models.NumistaStatusUnconfigured,
			wantGuidance: "numista_configuration_required",
		},
		{
			name: "unconfigured user", role: models.RoleUser,
			client: &handlerNumistaClient{}, wantStatus: models.NumistaStatusUnconfigured,
			wantGuidance: "numista_contact_administrator",
		},
		{
			name: "quota", key: "configured", role: models.RoleUser,
			client: &handlerNumistaClient{err: &services.NumistaError{
				Kind: services.NumistaErrorQuotaLimited, RetryAfterSeconds: &retryAfter,
			}},
			wantStatus: models.NumistaStatusQuotaLimited, wantGuidance: "numista_quota_limited",
			wantRetry: &retryAfter,
		},
		{
			name: "timeout", key: "configured", role: models.RoleUser,
			client:     &handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorTimeout}},
			wantStatus: models.NumistaStatusTimeout, wantGuidance: "retry_numista_lookup",
		},
		{
			name: "provider unauthorized admin", key: "configured", role: models.RoleAdmin,
			client:     &handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorUnauthorized}},
			wantStatus: models.NumistaStatusUnconfigured, wantGuidance: "numista_configuration_required",
		},
		{
			name: "provider unauthorized user", key: "configured", role: models.RoleUser,
			client:     &handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorUnauthorized}},
			wantStatus: models.NumistaStatusUnconfigured, wantGuidance: "numista_contact_administrator",
		},
		{
			name: "provider invalid request", key: "configured", role: models.RoleUser,
			client:     &handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorInvalidRequest}},
			wantStatus: models.NumistaStatusEmpty, wantGuidance: "revise_numista_query",
		},
		{
			name: "unavailable", key: "configured", role: models.RoleUser,
			client:     &handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorUnavailable}},
			wantStatus: models.NumistaStatusUnavailable, wantGuidance: "retry_numista_lookup",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := newAuthenticatedNumistaStatusRouter(test.client, test.key)
			recorder := performNumistaStatusLookup(
				t, router, test.role,
				`{"query":"coin","path":"direct","evidence":{}}`,
			)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var outcome models.NumistaLookupOutcome
			if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil {
				t.Fatal(err)
			}
			if outcome.Status != test.wantStatus || outcome.GuidanceCode != test.wantGuidance {
				t.Fatalf("outcome=%+v, want status %q guidance %q", outcome, test.wantStatus, test.wantGuidance)
			}
			if test.wantRetry != nil &&
				(outcome.RetryAfterSeconds == nil || *outcome.RetryAfterSeconds != *test.wantRetry) {
				t.Fatalf("retryAfterSeconds=%v, want %d", outcome.RetryAfterSeconds, *test.wantRetry)
			}
			if test.role != models.RoleAdmin {
				body := strings.ToLower(recorder.Body.String())
				for _, forbidden := range []string{"settings", "api key", "credential", "connections"} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("ordinary user response leaked privileged guidance %q: %s", forbidden, recorder.Body.String())
					}
				}
			}
		})
	}
}

func TestNumistaStatusHandlerValidationAuthenticationAndSafeInternalFailure(t *testing.T) {
	router := newAuthenticatedNumistaStatusRouter(&handlerNumistaClient{}, "configured")

	unauthorized := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/numista/lookup",
		bytes.NewBufferString(`{"query":"coin","path":"direct","evidence":{}}`),
	)
	router.ServeHTTP(unauthorized, request)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	invalid := performNumistaStatusLookup(
		t, router, models.RoleUser,
		`{"query":"","path":"direct","evidence":{}}`,
	)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}

	const privateError = "provider-secret database DSN"
	internalRouter := newAuthenticatedNumistaStatusRouter(
		&handlerNumistaClient{err: errors.New(privateError)},
		"configured",
	)
	internal := performNumistaStatusLookup(
		t, internalRouter, models.RoleUser,
		`{"query":"coin","path":"direct","evidence":{}}`,
	)
	if internal.Code != http.StatusInternalServerError ||
		strings.Contains(internal.Body.String(), privateError) ||
		internal.Body.String() != `{"error":"Numista lookup failed"}` {
		t.Fatalf("unsafe internal response status=%d body=%s", internal.Code, internal.Body.String())
	}
}

func TestNumistaStatusHandlerPreservesLegacyGETFailureSemantics(t *testing.T) {
	failures := []error{
		&services.NumistaError{Kind: services.NumistaErrorUnconfigured},
		&services.NumistaError{Kind: services.NumistaErrorUnauthorized},
		&services.NumistaError{Kind: services.NumistaErrorQuotaLimited},
		&services.NumistaError{Kind: services.NumistaErrorTimeout},
		&services.NumistaError{Kind: services.NumistaErrorUnavailable},
		errors.New("private internal failure"),
	}
	for _, failure := range failures {
		router := newAuthenticatedNumistaStatusRouter(&handlerNumistaClient{err: failure}, "configured")
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/numista/search?q=coin", nil)
		request.Header.Set("Authorization", "Bearer "+numistaStatusToken(t, models.RoleUser))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable ||
			recorder.Body.String() != `{"error":"Numista lookup is unavailable"}` {
			t.Fatalf("legacy failure=%v status=%d body=%s", failure, recorder.Code, recorder.Body.String())
		}
	}
}

func TestNumistaStatusHandlerLegacyGETMapsProviderInvalidRequestToEmptyResult(t *testing.T) {
	router := newAuthenticatedNumistaStatusRouter(
		&handlerNumistaClient{err: &services.NumistaError{Kind: services.NumistaErrorInvalidRequest}},
		"configured",
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/numista/search?q=coin", nil)
	request.Header.Set("Authorization", "Bearer "+numistaStatusToken(t, models.RoleUser))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"count":0,"types":[]}` {
		t.Fatalf("legacy invalid request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNumistaStatusHandlerStopsSilentlyForCancelledCaller(t *testing.T) {
	router := newAuthenticatedNumistaStatusRouter(&handlerNumistaClient{}, "configured")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/numista/lookup",
		bytes.NewBufferString(`{"query":"coin","path":"direct","evidence":{}}`),
	).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+numistaStatusToken(t, models.RoleUser))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Body.Len() != 0 {
		t.Fatalf("cancelled request wrote a response: %s", recorder.Body.String())
	}
}
