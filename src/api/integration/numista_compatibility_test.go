package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNumistaCompatibilityLegacyGETAndOldDraftInputs(t *testing.T) {
	h := newNumistaIntegrationHarness(t)
	response := performRequest(
		t, h.router, http.MethodGet, "/api/numista/search?q=Trajan", "", nil,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("legacy GET status=%d body=%s", response.Code, response.Body.String())
	}
	var legacy map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &legacy); err != nil ||
		legacy["count"] == nil || legacy["types"] == nil || legacy["candidates"] != nil {
		t.Fatalf("legacy shape changed: body=%s err=%v", response.Body.String(), err)
	}

	oldForm := url.Values{
		"workingTitle": {"Legacy draft"}, "notes": {"old client payload"},
		"ngcCertNumber": {"823160-093"},
		"ngcLookupUrl":  {"https://www.ngccoin.com/certlookup/823160093/NGCAncients/"},
		"ngcGrade":      {"Ch AU"},
	}
	response = performRequest(
		t, h.router, http.MethodPost, "/api/quick-capture/drafts",
		"application/x-www-form-urlencoded", []byte(oldForm.Encode()),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("old draft input status=%d body=%s", response.Code, response.Body.String())
	}
	var draft models.QuickCaptureDraft
	if err := json.Unmarshal(response.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	if draft.SelectedNumistaReference != nil || draft.NGCCertNumber != "823160-093" ||
		draft.NGCGrade != "Ch AU" {
		t.Fatalf("old draft/NGC fields changed: %+v", draft)
	}
}

func newPhotoLookupService(t *testing.T, analysis string, calls *int) *services.CoinLookupService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if r.URL.Path != "/api/analyze" {
			t.Fatalf("unexpected photo request path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(services.AnalyzeProxyResponse{Analysis: analysis})
	}))
	t.Cleanup(server.Close)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf(
		"file:numista_photo_compat_%d?mode=memory&cache=shared", time.Now().UnixNano(),
	)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatal(err)
	}
	settings := []models.AppSetting{
		{Key: services.SettingAIProvider, Value: "anthropic"},
		{Key: services.SettingAnthropicAPIKey, Value: "test-only"},
		{Key: services.SettingAnthropicModel, Value: "test-model"},
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatal(err)
	}
	return services.NewCoinLookupService(
		services.NewAgentProxy(server.URL, "internal-test-token", services.NewLogger(10)),
		services.NewSettingsService(repository.NewSettingsRepository(db)),
		services.NewLogger(10),
	)
}

func TestNumistaCompatibilityPhotoFieldsAreAdditiveAndNGCReferencesRemain(t *testing.T) {
	t.Run("non NGC adds proposal without populating legacy candidates", func(t *testing.T) {
		calls := 0
		service := newPhotoLookupService(t,
			`{"ngcCert":null,"labelText":"IMP TRAIANO","name":"Trajan Denarius","ruler":"Trajan","mint":"Rome","denomination":"Denarius","material":"Silver"}`,
			&calls,
		)
		result, err := service.Lookup(context.Background(), 7, services.CoinLookupRequest{
			Images: []string{"data:image/png;base64,AA=="},
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || result.ProposedNumistaQuery == "" ||
			result.NumistaEvidence.Title != "Trajan Denarius" ||
			result.NumistaLookup != nil || len(result.NumistaCandidates) != 0 ||
			len(result.CandidateReferences) != 0 {
			t.Fatalf("additive non-NGC response mismatch: %+v calls=%d", result, calls)
		}
	})

	t.Run("NGC first retains legacy NGC reference", func(t *testing.T) {
		calls := 0
		service := newPhotoLookupService(t,
			`{"ngcCert":"823160-093","ngcGrade":"Ch AU","ngcDescription":"Trajan","labelText":"NGC"}`,
			&calls,
		)
		result, err := service.Lookup(context.Background(), 7, services.CoinLookupRequest{
			Images: []string{"data:image/png;base64,AA=="},
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || result.ProposedNumistaQuery != "" ||
			result.NumistaEvidence != (models.NumistaEvidence{}) ||
			len(result.CandidateReferences) != 1 ||
			result.CandidateReferences[0].Catalog != "NGC" ||
			!strings.Contains(result.CandidateReferences[0].URI, "823160093") {
			t.Fatalf("NGC compatibility mismatch: %+v calls=%d", result, calls)
		}
	})
}

func TestNumistaCompatibilityOwnershipDedupAndRollbackReadableRecords(t *testing.T) {
	h := newNumistaIntegrationHarness(t)
	coin := models.Coin{
		UserID: h.userID, Name: "Owner coin", Category: models.CategoryRoman,
		Material: models.MaterialSilver, Era: models.EraAncient,
	}
	if err := h.db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"catalog":"numista","number":"202","uri":"https://en.numista.com/catalogue/pieces202.html"}`)
	response := performRequest(
		t, h.router, http.MethodPost, fmt.Sprintf("/api/coins/%d/references", coin.ID),
		"application/json", payload,
	)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"catalog":"Numista"`) {
		t.Fatalf("normalized reference status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(
		t, h.router, http.MethodPost, fmt.Sprintf("/api/coins/%d/references", coin.ID),
		"application/json", payload,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("structured duplicate status=%d body=%s", response.Code, response.Body.String())
	}

	h.userID = 99
	response = performRequest(
		t, h.router, http.MethodGet, fmt.Sprintf("/api/coins/%d/references", coin.ID), "", nil,
	)
	if response.Code != http.StatusNotFound {
		t.Fatalf("other owner reference list status=%d body=%s", response.Code, response.Body.String())
	}
	h.userID = 7

	_ = createDraft(t, h, "")
	var rows []struct {
		ID             uint
		UserID         uint
		WorkingTitle   string
		Status         string
		PromotedCoinID *uint
	}
	if err := h.db.Table("quick_capture_drafts").Find(&rows).Error; err != nil {
		t.Fatalf("old binary draft projection cannot read additive table: %v", err)
	}
	if len(rows) != 1 || rows[0].WorkingTitle != "Trajan Denarius" {
		t.Fatalf("rollback draft projection mismatch: %+v", rows)
	}
	var oldCoin struct {
		ID     uint
		UserID uint
		Name   string
	}
	if err := h.db.Table("coins").First(&oldCoin, coin.ID).Error; err != nil ||
		oldCoin.Name != coin.Name || oldCoin.UserID != h.userID {
		t.Fatalf("rollback coin projection mismatch: %+v err=%v", oldCoin, err)
	}
	var oldReference struct {
		CoinID  uint
		Catalog string
		Number  string
		URI     string
	}
	if err := h.db.Table("coin_references").Where("coin_id = ?", coin.ID).First(&oldReference).Error; err != nil ||
		oldReference.Catalog != "Numista" || oldReference.Number != "202" {
		t.Fatalf("rollback reference projection mismatch: %+v err=%v", oldReference, err)
	}
}
