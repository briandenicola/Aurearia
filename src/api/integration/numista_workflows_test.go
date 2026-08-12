package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/handlers"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type integrationNumistaSettings struct {
	key    string
	config services.NumistaSettings
}

func (s *integrationNumistaSettings) GetSetting(string) string { return s.key }
func (s *integrationNumistaSettings) GetNumistaSettings() services.NumistaSettings {
	return s.config
}

type integrationNumistaProvider struct {
	mu         sync.Mutex
	candidates []models.NumistaCandidate
	details    map[int]models.NumistaCandidate
	searches   int
	detailIDs  []int
}

func (p *integrationNumistaProvider) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.searches++
	return append([]models.NumistaCandidate(nil), p.candidates...), nil
}

func (p *integrationNumistaProvider) Detail(_ context.Context, id int) (models.NumistaCandidate, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.detailIDs = append(p.detailIDs, id)
	detail, ok := p.details[id]
	if !ok {
		return models.NumistaCandidate{}, fmt.Errorf("detail unavailable")
	}
	return detail, nil
}

type numistaIntegrationHarness struct {
	db       *gorm.DB
	router   *gin.Engine
	provider *integrationNumistaProvider
	userID   uint
}

func newNumistaIntegrationHarness(t *testing.T) *numistaIntegrationHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf(
		"file:numista_integration_%d?mode=memory&cache=shared", time.Now().UnixNano(),
	)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.CatalogRegistry{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{},
		&models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{},
		&models.DraftLifecycleEvent{},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CatalogRegistry{
		Catalog: "Numista", DisplayName: "Numista", Era: models.EraModern,
	}).Error; err != nil {
		t.Fatal(err)
	}

	provider := &integrationNumistaProvider{candidates: []models.NumistaCandidate{
		{ID: 101, Title: "Trajan Sestertius"},
		{ID: 202, Title: "Trajan Denarius", Issuer: "Trajan"},
		{ID: 303, Title: "Hadrian Denarius"},
	}}
	settings := &integrationNumistaSettings{
		key: "configured",
		config: services.NumistaSettings{
			SearchTTL: time.Hour, DetailTTL: time.Hour, SearchResultLimit: 20,
			EnrichmentLimit: 5, Valid: true,
		},
	}
	lookup := services.NewNumistaLookupService(
		provider, services.NewNumistaCache(nil, 50, 50),
		services.NewNumistaV1Scorer(), services.NewNumistaTelemetry(50), settings, nil,
	)
	numistaHandler := handlers.NewNumistaHandler(lookup)

	refRepo := repository.NewCoinReferenceRepository(db)
	refSvc := services.NewCoinReferenceService(refRepo, repository.NewCatalogRegistryRepository(db))
	refHandler := handlers.NewCoinReferenceHandler(refRepo, refSvc, nil)

	quickSvc := services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()).
		WithReferenceValidation(refSvc)
	quickHandler := handlers.NewQuickCaptureHandler(quickSvc, nil)

	harness := &numistaIntegrationHarness{db: db, provider: provider, userID: 7}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", harness.userID)
		c.Set("userRole", string(models.RoleUser))
		c.Next()
	})
	router.POST("/api/numista/lookup", numistaHandler.Lookup)
	router.GET("/api/numista/search", numistaHandler.Search)
	router.POST("/api/coins/:id/references", refHandler.Create)
	router.GET("/api/coins/:id/references", refHandler.List)
	router.POST("/api/quick-capture/drafts", quickHandler.CreateDraft)
	router.GET("/api/quick-capture/drafts/:id", quickHandler.GetDraft)
	router.PUT("/api/quick-capture/drafts/:id", quickHandler.UpdateDraft)
	router.POST("/api/quick-capture/drafts/:id/promote", quickHandler.PromoteDraft)
	harness.router = router
	return harness
}

func performRequest(t *testing.T, router http.Handler, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func createDraft(t *testing.T, h *numistaIntegrationHarness, selectedID string) models.QuickCaptureDraft {
	t.Helper()
	form := url.Values{
		"workingTitle": {"Trajan Denarius"},
		"era":          {"ancient"},
	}
	if selectedID != "" {
		form.Set("selectedNumistaId", selectedID)
		form.Set("selectedNumistaUrl", "https://en.numista.com/catalogue/pieces"+selectedID+".html")
	}
	response := performRequest(
		t, h.router, http.MethodPost, "/api/quick-capture/drafts",
		"application/x-www-form-urlencoded", []byte(form.Encode()),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("create draft status=%d body=%s", response.Code, response.Body.String())
	}
	var draft models.QuickCaptureDraft
	if err := json.Unmarshal(response.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	return draft
}

func promoteDraft(t *testing.T, h *numistaIntegrationHarness, draftID uint, target string, overrides bool) map[string]any {
	t.Helper()
	body := map[string]any{"confirm": true, "target": target}
	if overrides {
		body["overrides"] = map[string]any{
			"category": "Roman", "material": "Silver", "era": "ancient",
		}
	}
	encoded, _ := json.Marshal(body)
	response := performRequest(
		t, h.router, http.MethodPost,
		fmt.Sprintf("/api/quick-capture/drafts/%d/promote", draftID),
		"application/json", encoded,
	)
	var result map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	result["_status"] = float64(response.Code)
	return result
}

func TestNumistaWorkflowDirectPersistsOnlyExplicitSelection(t *testing.T) {
	h := newNumistaIntegrationHarness(t)
	coin := models.Coin{
		UserID: h.userID, Name: "Trajan Denarius", Category: models.CategoryRoman,
		Material: models.MaterialSilver, Era: models.EraAncient,
	}
	if err := h.db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	lookupBody := []byte(`{"query":"Trajan denarius","path":"direct","evidence":{"title":"Trajan Denarius","issuer":"Trajan"},"querySource":"manual"}`)
	response := performRequest(t, h.router, http.MethodPost, "/api/numista/lookup", "application/json", lookupBody)
	if response.Code != http.StatusOK {
		t.Fatalf("lookup status=%d body=%s", response.Code, response.Body.String())
	}
	var outcome models.NumistaLookupOutcome
	if err := json.Unmarshal(response.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if len(outcome.Candidates) != 3 {
		t.Fatalf("broad candidates=%d, want 3", len(outcome.Candidates))
	}

	selected := outcome.Candidates[0]
	payload, _ := json.Marshal(models.CoinReference{
		Catalog: "Numista", Number: strconv.Itoa(selected.ID), URI: selected.CanonicalURL,
	})
	response = performRequest(
		t, h.router, http.MethodPost, fmt.Sprintf("/api/coins/%d/references", coin.ID),
		"application/json", payload,
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("persist selection status=%d body=%s", response.Code, response.Body.String())
	}
	var refs []models.CoinReference
	if err := h.db.Where("coin_id = ?", coin.ID).Find(&refs).Error; err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Number != strconv.Itoa(selected.ID) {
		t.Fatalf("persisted references=%+v", refs)
	}
	for _, candidate := range outcome.Candidates[1:] {
		if refs[0].Number == strconv.Itoa(candidate.ID) {
			t.Fatalf("unselected candidate %d was persisted", candidate.ID)
		}
	}

	response = performRequest(
		t, h.router, http.MethodPost, fmt.Sprintf("/api/coins/%d/references", coin.ID),
		"application/json", payload,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("duplicate selected reference status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNumistaWorkflowPhotoDraftPromotionCollectionWishlistNoSelectionAndRetry(t *testing.T) {
	for _, target := range []string{"collection", "wishlist"} {
		t.Run(target, func(t *testing.T) {
			h := newNumistaIntegrationHarness(t)
			draft := createDraft(t, h, "202")

			failed := promoteDraft(t, h, draft.ID, "invalid-target", false)
			if int(failed["_status"].(float64)) != http.StatusBadRequest {
				t.Fatalf("intended validation failure=%v", failed)
			}
			var retained models.QuickCaptureDraftReference
			if err := h.db.Where("draft_id = ?", draft.ID).First(&retained).Error; err != nil ||
				retained.Number != "202" {
				t.Fatalf("failed promotion lost selection: ref=%+v err=%v", retained, err)
			}

			promoted := promoteDraft(t, h, draft.ID, target, true)
			if int(promoted["_status"].(float64)) != http.StatusOK || promoted["alreadyPromoted"] != false {
				t.Fatalf("promotion failed: %v", promoted)
			}
			coinID := uint(promoted["coinId"].(float64))
			var coin models.Coin
			if err := h.db.First(&coin, coinID).Error; err != nil {
				t.Fatal(err)
			}
			if coin.IsWishlist != (target == "wishlist") {
				t.Fatalf("target mismatch: wishlist=%v target=%s", coin.IsWishlist, target)
			}
			var refs []models.CoinReference
			if err := h.db.Where("coin_id = ?", coinID).Find(&refs).Error; err != nil {
				t.Fatal(err)
			}
			if len(refs) != 1 || refs[0].Catalog != "Numista" || refs[0].Number != "202" {
				t.Fatalf("promotion reference mismatch: %+v", refs)
			}

			repeated := promoteDraft(t, h, draft.ID, "collection", true)
			if int(repeated["_status"].(float64)) != http.StatusOK ||
				repeated["alreadyPromoted"] != true ||
				uint(repeated["coinId"].(float64)) != coinID ||
				repeated["target"] != target {
				t.Fatalf("repeated promotion was not idempotent: %v", repeated)
			}
			var count int64
			h.db.Model(&models.CoinReference{}).Where("coin_id = ?", coinID).Count(&count)
			if count != 1 {
				t.Fatalf("repeated promotion created %d references", count)
			}
		})
	}

	t.Run("no selection creates no reference", func(t *testing.T) {
		h := newNumistaIntegrationHarness(t)
		draft := createDraft(t, h, "")
		result := promoteDraft(t, h, draft.ID, "collection", true)
		if int(result["_status"].(float64)) != http.StatusOK {
			t.Fatalf("promotion failed: %v", result)
		}
		var count int64
		h.db.Model(&models.CoinReference{}).
			Where("coin_id = ?", uint(result["coinId"].(float64))).Count(&count)
		if count != 0 {
			t.Fatalf("no-selection promotion created %d references", count)
		}
	})
}

func responseContainsAny(body string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(body, value) {
			return true
		}
	}
	return false
}
