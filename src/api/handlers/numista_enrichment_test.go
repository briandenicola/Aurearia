package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type numistaEnrichmentHTTPHandler interface {
	Enrich(*gin.Context)
}

type handlerEnrichmentClient struct {
	mu       sync.Mutex
	details  map[int]models.NumistaCandidate
	failures map[int]error
	calls    []int
}

func (c *handlerEnrichmentClient) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	return nil, errors.New("broad search must not run through enrichment")
}

func (c *handlerEnrichmentClient) Detail(_ context.Context, id int) (models.NumistaCandidate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, id)
	if err := c.failures[id]; err != nil {
		return models.NumistaCandidate{}, err
	}
	return c.details[id], nil
}

func (c *handlerEnrichmentClient) calledIDs() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.calls...)
}

type handlerEnrichmentSettings struct {
	limit int
}

func (handlerEnrichmentSettings) GetSetting(string) string { return "configured" }
func (s handlerEnrichmentSettings) GetNumistaSettings() services.NumistaSettings {
	return services.NumistaSettings{
		DetailTTL: 7 * 24 * time.Hour, EnrichmentLimit: s.limit, Valid: true,
	}
}

func handlerEnrichmentCandidate(id int, title string, position int) models.NumistaCandidate {
	canonical, _ := models.CanonicalNumistaURL(id)
	return models.NumistaCandidate{
		ID: id, CanonicalURL: canonical, Title: title, ProviderPosition: position,
		EnrichmentState: models.NumistaEnrichmentNotRequested,
		Assessment: models.NumistaRelevanceAssessment{
			ScoringVersion: models.NumistaScoringVersion,
			Score:          50,
			Band:           "weak",
			Reasons:        []models.NumistaRelevanceReason{},
		},
	}
}

func handlerEnrichmentRequest(candidates []models.NumistaCandidate) models.NumistaEnrichmentRequest {
	return models.NumistaEnrichmentRequest{
		NumistaLookupRequest: models.NumistaLookupRequest{
			Query: "Trajan denarius Rome silver",
			Path:  models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{
				Title: "Trajan denarius", Mint: "Rome", Material: "Silver",
			},
		},
		Candidates: candidates,
	}
}

func newAuthenticatedNumistaEnrichmentRouter(
	t *testing.T,
	client services.NumistaClient,
	limit int,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service := services.NewNumistaLookupService(
		client,
		services.NewNumistaCache(nil, 20, 50),
		services.NewNumistaV1Scorer(),
		services.NewNumistaTelemetry(50),
		handlerEnrichmentSettings{limit: limit},
		nil,
	)
	handler := NewNumistaHandler(service)
	enrichmentHandler, ok := any(handler).(numistaEnrichmentHTTPHandler)
	if !ok {
		t.Fatal("Phase 7 missing: NumistaHandler.Enrich is not implemented")
	}
	router := gin.New()
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired(numistaStatusJWTSecret, nil))
	protected.POST("/numista/enrich", enrichmentHandler.Enrich)
	return router
}

func performEnrichmentRequest(
	t *testing.T,
	router http.Handler,
	body any,
	authenticated bool,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/numista/enrich", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	if authenticated {
		request.Header.Set("Authorization", "Bearer "+numistaStatusToken(t, models.RoleUser))
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestNumistaEnrichmentHandlerRequiresAuthentication(t *testing.T) {
	router := newAuthenticatedNumistaEnrichmentRouter(t, &handlerEnrichmentClient{}, 5)
	recorder := performEnrichmentRequest(
		t,
		router,
		handlerEnrichmentRequest([]models.NumistaCandidate{handlerEnrichmentCandidate(1, "Coin", 0)}),
		false,
	)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", recorder.Code, recorder.Body.String())
	}
}

func TestNumistaEnrichmentHandlerRejectsInvalidAndDuplicateCandidatesWithoutProviderCalls(t *testing.T) {
	client := &handlerEnrichmentClient{}
	router := newAuthenticatedNumistaEnrichmentRouter(t, client, 5)

	fiftyOne := make([]models.NumistaCandidate, 51)
	for index := range fiftyOne {
		fiftyOne[index] = handlerEnrichmentCandidate(index+1, "Coin", index)
	}
	tests := []struct {
		name string
		body any
	}{
		{name: "empty candidates", body: handlerEnrichmentRequest([]models.NumistaCandidate{})},
		{name: "more than fifty", body: handlerEnrichmentRequest(fiftyOne)},
		{name: "duplicate IDs", body: handlerEnrichmentRequest([]models.NumistaCandidate{
			handlerEnrichmentCandidate(1, "First", 0),
			handlerEnrichmentCandidate(1, "Duplicate", 1),
		})},
		{name: "forged canonical URL", body: func() models.NumistaEnrichmentRequest {
			candidate := handlerEnrichmentCandidate(1, "Coin", 0)
			candidate.CanonicalURL = "https://attacker.example/pieces1.html"
			return handlerEnrichmentRequest([]models.NumistaCandidate{candidate})
		}()},
		{name: "missing evidence object", body: map[string]any{
			"query": "coin", "path": "direct",
			"candidates": []models.NumistaCandidate{handlerEnrichmentCandidate(1, "Coin", 0)},
		}},
		{name: "unknown field", body: map[string]any{
			"query": "coin", "path": "direct", "evidence": map[string]any{},
			"candidates":      []models.NumistaCandidate{handlerEnrichmentCandidate(1, "Coin", 0)},
			"privateOverride": true,
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := performEnrichmentRequest(t, router, test.body, true)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want 400", recorder.Code, recorder.Body.String())
			}
		})
	}
	if calls := client.calledIDs(); len(calls) != 0 {
		t.Fatalf("invalid requests reached provider details: %v", calls)
	}
}

func TestNumistaEnrichmentHandlerIgnoresClientReorderingAndReturnsFullBroadSet(t *testing.T) {
	client := &handlerEnrichmentClient{details: map[int]models.NumistaCandidate{
		2: {ID: 2, Title: "Trajan denarius", Mint: "Rome", Material: "Silver"},
	}}
	router := newAuthenticatedNumistaEnrichmentRouter(t, client, 1)
	candidates := []models.NumistaCandidate{
		handlerEnrichmentCandidate(1, "Unrelated bronze", 0),
		handlerEnrichmentCandidate(3, "Other", 2),
		handlerEnrichmentCandidate(2, "Trajan denarius", 1),
	}
	candidates[0].Assessment.Score = 100
	candidates[0].Assessment.Band = "strong"
	candidates[2].Assessment.Score = 0
	candidates[2].Assessment.Band = "weak"

	recorder := performEnrichmentRequest(t, router, handlerEnrichmentRequest(candidates), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var outcome models.NumistaLookupOutcome
	if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if calls := client.calledIDs(); !reflect.DeepEqual(calls, []int{2}) {
		t.Fatalf("detail calls=%v, want server-ranked candidate 2", calls)
	}
	if outcome.Status != models.NumistaStatusSuccess || outcome.Stage != "enriched" ||
		len(outcome.Candidates) != len(candidates) || outcome.Candidates[0].ID != 2 {
		t.Fatalf("unexpected full enrichment contract: %+v", outcome)
	}
}

func TestNumistaEnrichmentHandlerCanonicalizesQueryAndRejectsWhitespaceOnly(t *testing.T) {
	client := &handlerEnrichmentClient{details: map[int]models.NumistaCandidate{
		1: {ID: 1, Title: "Trajan denarius"},
	}}
	router := newAuthenticatedNumistaEnrichmentRouter(t, client, 1)
	request := handlerEnrichmentRequest([]models.NumistaCandidate{
		handlerEnrichmentCandidate(1, "Trajan denarius", 0),
	})
	request.Query = " \tTrajan   denarius\n"

	recorder := performEnrichmentRequest(t, router, request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var outcome models.NumistaLookupOutcome
	if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.EffectiveQuery != "Trajan   denarius" {
		t.Fatalf("effectiveQuery=%q, want canonical trimmed query", outcome.EffectiveQuery)
	}

	request.Query = " \t\r\n "
	recorder = performEnrichmentRequest(t, router, request, true)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("whitespace-only status=%d body=%s, want 400", recorder.Code, recorder.Body.String())
	}
	if calls := client.calledIDs(); !reflect.DeepEqual(calls, []int{1}) {
		t.Fatalf("whitespace-only request reached provider: %v", calls)
	}
}

func TestNumistaEnrichmentHandlerReturnsSafePartialAndAllFailureOutcomes(t *testing.T) {
	const privateFailure = "provider key abc123 and private upstream body"
	for _, failures := range []map[int]error{
		{2: errors.New(privateFailure)},
		{
			1: &services.NumistaError{Kind: services.NumistaErrorTimeout},
			2: errors.New(privateFailure),
			3: &services.NumistaError{Kind: services.NumistaErrorQuotaLimited},
		},
	} {
		client := &handlerEnrichmentClient{
			details: map[int]models.NumistaCandidate{
				1: {ID: 1, Title: "Trajan denarius"},
				2: {ID: 2, Title: "Trajan denarius"},
				3: {ID: 3, Title: "Trajan denarius"},
			},
			failures: failures,
		}
		router := newAuthenticatedNumistaEnrichmentRouter(t, client, 3)
		candidates := []models.NumistaCandidate{
			handlerEnrichmentCandidate(1, "Trajan denarius", 0),
			handlerEnrichmentCandidate(2, "Trajan denarius", 1),
			handlerEnrichmentCandidate(3, "Trajan denarius", 2),
		}
		recorder := performEnrichmentRequest(t, router, handlerEnrichmentRequest(candidates), true)
		if recorder.Code != http.StatusOK ||
			strings.Contains(recorder.Body.String(), privateFailure) ||
			strings.Contains(strings.ToLower(recorder.Body.String()), "provider key") {
			t.Fatalf("unsafe failure response status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var outcome models.NumistaLookupOutcome
		if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil {
			t.Fatal(err)
		}
		if outcome.Status != models.NumistaStatusSuccess || len(outcome.Candidates) != len(candidates) {
			t.Fatalf("failed details erased broad results: %+v", outcome)
		}
		for _, candidate := range outcome.Candidates {
			if failures[candidate.ID] != nil && candidate.EnrichmentState != models.NumistaEnrichmentFailed {
				t.Fatalf("candidate %d state=%q, want failed", candidate.ID, candidate.EnrichmentState)
			}
		}
	}
}

func TestNumistaEnrichmentRouteAndOpenAPIDocumented(t *testing.T) {
	// The protected route table moved out of main.go into routes_protected.go
	// when the composition root was split up; the assertion is on the route
	// still being registered under the authenticated group, not on which file
	// happens to hold it.
	routeSource, err := os.ReadFile("../routes_protected.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(routeSource, []byte(`protected.POST("/numista/enrich", numistaHandler.Enrich)`)) {
		t.Fatal("authenticated POST /api/numista/enrich is not registered in routes_protected.go")
	}
	handlerSource, err := os.ReadFile("numista.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(handlerSource, []byte("@Router")) ||
		!bytes.Contains(handlerSource, []byte("/numista/enrich [post]")) ||
		!bytes.Contains(handlerSource, []byte("@Security")) {
		t.Fatal("enrichment handler lacks Swagger route/security annotations")
	}
	openAPI, err := os.ReadFile("../docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(openAPI, []byte(`"/numista/enrich"`)) {
		t.Fatal("generated OpenAPI does not document POST /numista/enrich")
	}
}
