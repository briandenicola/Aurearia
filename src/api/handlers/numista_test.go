package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type handlerNumistaClient struct {
	candidates []models.NumistaCandidate
	err        error
}

func (c *handlerNumistaClient) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	return c.candidates, c.err
}
func (c *handlerNumistaClient) Detail(context.Context, int) (models.NumistaCandidate, error) {
	return models.NumistaCandidate{}, c.err
}

type handlerNumistaSettings struct {
	key string
}

func (s handlerNumistaSettings) GetSetting(string) string { return s.key }
func (s handlerNumistaSettings) GetNumistaSettings() services.NumistaSettings {
	return services.NumistaSettings{SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true}
}

func newNumistaTestRouter(client services.NumistaClient, key string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	lookup := services.NewNumistaLookupService(
		client, services.NewNumistaCache(nil, 20, 20), services.NewNumistaV1Scorer(),
		services.NewNumistaTelemetry(20), handlerNumistaSettings{key: key}, nil,
	)
	handler := NewNumistaHandler(lookup)
	router := gin.New()
	router.POST("/numista/query-proposal", handler.QueryProposal)
	router.POST("/numista/lookup", handler.Lookup)
	router.GET("/numista/search", handler.Search)
	return router
}

func TestNumistaHandlerQueryProposalIsTypedBoundedAndLocal(t *testing.T) {
	client := &handlerNumistaClient{}
	router := newNumistaTestRouter(client, "configured")
	body := bytes.NewBufferString(`{
		"path":"direct",
		"evidence":{
			"title":"Honorius AE3 RIC IX 46",
			"issuer":"Honorius",
			"mint":"SMNT",
			"reverseInscription":"GLORIA ROMANORVM",
			"material":"Bronze",
			"visibleText":"NGC slab text"
		}
	}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/numista/query-proposal", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("proposal status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var proposal models.NumistaQueryProposal
	if err := json.Unmarshal(recorder.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.Query != "Honorius GLORIA ROMANORVM Nicomedia" ||
		proposal.QuerySource != models.NumistaQuerySourceGenerated ||
		proposal.GenerationVersion != models.NumistaQueryGenerationVersion {
		t.Fatalf("proposal=%#v", proposal)
	}

	for _, invalid := range []string{
		`{"path":"direct"}`,
		`{"path":"other","evidence":{}}`,
		`{"path":"direct","evidence":{},"unknown":true}`,
		`{"path":"direct","evidence":{"reverseType":"` + strings.Repeat("x", 501) + `"}}`,
	} {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(
			http.MethodPost, "/numista/query-proposal", strings.NewReader(invalid),
		))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid proposal status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestNumistaHandlerLookupAndLegacyShape(t *testing.T) {
	router := newNumistaTestRouter(&handlerNumistaClient{candidates: []models.NumistaCandidate{
		{ID: 123, Title: "Trajan Denarius", Issuer: "Roman Empire"},
	}}, "configured")
	body := bytes.NewBufferString(`{"query":"  Trajan   denarius  ","path":"direct","evidence":{"issuer":"Trajan"},"querySource":"manual"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/numista/lookup", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("lookup status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var outcome models.NumistaLookupOutcome
	if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil ||
		outcome.Status != models.NumistaStatusSuccess || outcome.EffectiveQuery != "  Trajan   denarius  " {
		t.Fatalf("unexpected lookup body: %s err=%v", recorder.Body.String(), err)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/numista/search?q=Trajan", nil))
	var legacy map[string]json.RawMessage
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &legacy) != nil ||
		legacy["count"] == nil || legacy["types"] == nil {
		t.Fatalf("legacy contract changed: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNumistaHandlerRejectsInvalidAndReturnsSafeLegacyFailure(t *testing.T) {
	router := newNumistaTestRouter(&handlerNumistaClient{}, "")
	for _, body := range []string{
		`{"query":"","path":"direct","evidence":{},"querySource":"manual"}`,
		`{"query":"coin","path":"direct","evidence":{}}`,
		`{"query":"coin","path":"direct","querySource":"manual"}`,
		`{"query":"coin","path":"direct","evidence":{},"querySource":"manual","unknown":true}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/numista/lookup", bytes.NewBufferString(body))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid body status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/numista/search?q=coin", nil))
	if recorder.Code != http.StatusServiceUnavailable || bytes.Contains(recorder.Body.Bytes(), []byte("key")) {
		t.Fatalf("unsafe legacy failure: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNumistaHandlerLookupRequestBoundaries(t *testing.T) {
	router := newNumistaTestRouter(&handlerNumistaClient{candidates: []models.NumistaCandidate{
		{ID: 123, Title: "Trajan Denarius"},
	}}, "configured")
	perform := func(body []byte) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/numista/lookup", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		return recorder
	}
	marshalRequest := func(query string, evidence models.NumistaEvidence) []byte {
		t.Helper()
		source := models.NumistaQuerySourceManual
		body, err := json.Marshal(numistaLookupWireRequest{
			Query: query, Path: models.NumistaLookupPathDirect, Evidence: &evidence,
			QuerySource: &source,
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	for _, test := range []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"query at 500 runes", strings.Repeat("é", models.NumistaMaxQueryLength), http.StatusOK},
		{"query at 501 runes", strings.Repeat("é", models.NumistaMaxQueryLength+1), http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := perform(marshalRequest(test.query, models.NumistaEvidence{}))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", recorder.Code, recorder.Body.String(), test.wantStatus)
			}
		})
	}

	evidenceFields := []struct {
		name string
		max  int
		set  func(*models.NumistaEvidence, string)
	}{
		{"title", 200, func(e *models.NumistaEvidence, value string) { e.Title = value }},
		{"issuer", 200, func(e *models.NumistaEvidence, value string) { e.Issuer = value }},
		{"denomination", 100, func(e *models.NumistaEvidence, value string) { e.Denomination = value }},
		{"mint", 200, func(e *models.NumistaEvidence, value string) { e.Mint = value }},
		{"dateText", 100, func(e *models.NumistaEvidence, value string) { e.DateText = value }},
		{"material", 100, func(e *models.NumistaEvidence, value string) { e.Material = value }},
		{"obverseInscription", 500, func(e *models.NumistaEvidence, value string) { e.ObverseInscription = value }},
		{"reverseInscription", 500, func(e *models.NumistaEvidence, value string) { e.ReverseInscription = value }},
		{"visibleText", 500, func(e *models.NumistaEvidence, value string) { e.VisibleText = value }},
	}
	for _, field := range evidenceFields {
		t.Run(field.name, func(t *testing.T) {
			var evidence models.NumistaEvidence
			field.set(&evidence, strings.Repeat("é", field.max))
			if recorder := perform(marshalRequest("coin", evidence)); recorder.Code != http.StatusOK {
				t.Fatalf("exact maximum status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			field.set(&evidence, strings.Repeat("é", field.max+1))
			if recorder := perform(marshalRequest("coin", evidence)); recorder.Code != http.StatusBadRequest {
				t.Fatalf("over maximum status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	base := marshalRequest("coin", models.NumistaEvidence{})
	exactLimit := append(base, bytes.Repeat([]byte(" "), numistaLookupBodyLimit-len(base))...)
	if recorder := perform(exactLimit); recorder.Code != http.StatusOK {
		t.Fatalf("body at %d bytes status=%d body=%s", numistaLookupBodyLimit, recorder.Code, recorder.Body.String())
	}
	overLimit := append(append([]byte(nil), exactLimit...), ' ')
	if recorder := perform(overLimit); recorder.Code != http.StatusBadRequest {
		t.Fatalf("body at %d bytes status=%d body=%s", numistaLookupBodyLimit+1, recorder.Code, recorder.Body.String())
	}
}

func TestNumistaLegacySearchQueryBoundaries(t *testing.T) {
	router := newNumistaTestRouter(&handlerNumistaClient{candidates: []models.NumistaCandidate{
		{ID: 123, Title: "Trajan Denarius"},
	}}, "configured")
	for _, test := range []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"query at 500 runes", strings.Repeat("é", models.NumistaMaxQueryLength), http.StatusOK},
		{"query at 501 runes", strings.Repeat("é", models.NumistaMaxQueryLength+1), http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/numista/search?q="+url.QueryEscape(test.query),
				nil,
			)
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", recorder.Code, recorder.Body.String(), test.wantStatus)
			}
		})
	}
}

func TestNumistaRoutesRequireAuthenticationAndApplyRateLimit(t *testing.T) {
	const jwtSecret = "numista-handler-test-secret"
	lookup := services.NewNumistaLookupService(
		&handlerNumistaClient{candidates: []models.NumistaCandidate{{ID: 123, Title: "Trajan Denarius"}}},
		services.NewNumistaCache(nil, 20, 20), services.NewNumistaV1Scorer(),
		services.NewNumistaTelemetry(20), handlerNumistaSettings{key: "configured"}, nil,
	)
	numistaHandler := NewNumistaHandler(lookup)
	router := gin.New()
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired(jwtSecret, nil))
	protected.Use(middleware.AuthenticatedRateLimit(2, time.Minute))
	protected.POST("/numista/query-proposal", numistaHandler.QueryProposal)
	protected.POST("/numista/lookup", numistaHandler.Lookup)
	protected.GET("/numista/search", numistaHandler.Search)

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/numista/query-proposal", `{"path":"direct","evidence":{}}`},
		{http.MethodPost, "/api/numista/lookup", `{"query":"Trajan","path":"direct","evidence":{},"querySource":"manual"}`},
		{http.MethodGet, "/api/numista/search?q=Trajan", ""},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth status=%d, want 401", route.method, route.path, recorder.Code)
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(42),
		"role":   "user",
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
	})
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/numista/lookup",
			bytes.NewBufferString(`{"query":"Trajan","path":"direct","evidence":{},"querySource":"manual"}`),
		)
		request.Header.Set("Authorization", "Bearer "+signed)
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		want := http.StatusOK
		if attempt == 3 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Fatalf("authenticated lookup attempt %d status=%d body=%s, want %d", attempt, recorder.Code, recorder.Body.String(), want)
		}
	}
}
