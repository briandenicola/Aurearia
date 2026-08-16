package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestCoinLookupFastPath_UnaffectedByDeepIdentification is the Feature 344
// fast-path regression guard (T050, FR-001/SC-008): the quick Identify Coin
// endpoint (`POST /api/coins/lookup`) must keep its exact pre-existing
// response shape and must never create a DeepIdentificationJob row, since
// Deep Analysis is an entirely separate, additive, opt-in flow.
func TestCoinLookupFastPath_UnaffectedByDeepIdentification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	analysis := `{"ngcCert":null,"labelText":"IMP TRAIANO","name":"Trajan Denarius","ruler":"Trajan","mint":"Rome","denomination":"Denarius","material":"Silver","obverseInscription":"IMP TRAIANO","reverseInscription":"PAX"}`
	agentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request services.AnalyzeProxyRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode agent request: %v", err)
		}
		if request.Coin.Notes != "Weight 3.2 g" {
			t.Fatalf("collector notes = %q, want %q", request.Coin.Notes, "Weight 3.2 g")
		}
		if !strings.Contains(request.Prompt, "image 1 is obverse") {
			t.Fatalf("image role was not included in prompt: %q", request.Prompt)
		}
		_ = json.NewEncoder(w).Encode(services.AnalyzeProxyResponse{Analysis: analysis})
	}))
	t.Cleanup(agentServer.Close)

	dsn := fmt.Sprintf("file:coin_lookup_fastpath_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}, &models.DeepIdentificationJob{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, setting := range []models.AppSetting{
		{Key: services.SettingAIProvider, Value: "anthropic"},
		{Key: services.SettingAnthropicAPIKey, Value: "test-key"},
		{Key: services.SettingAnthropicModel, Value: "test-model"},
	} {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("seed setting: %v", err)
		}
	}

	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	lookupSvc := services.NewCoinLookupService(
		services.NewAgentProxy(agentServer.URL, "internal-token", services.NewLogger(10)),
		settingsSvc,
		services.NewLogger(10),
	)
	handler := NewCoinLookupHandler(lookupSvc, services.NewLogger(10))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	router.POST("/api/coins/lookup", handler.Lookup)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("images", "coin.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(deepTestPNGBytes(t)); err != nil {
		t.Fatalf("write image bytes: %v", err)
	}
	if err := writer.WriteField("notes", "  Weight 3.2 g  "); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	if err := writer.WriteField("imageRoles", "obverse"); err != nil {
		t.Fatalf("write image role: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/coins/lookup", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// Response shape unchanged: the four core CoinLookupResponse keys are
	// still present, and no deep-identification-specific keys leaked in.
	for _, key := range []string{"extractedData", "numistaCandidates", "numistaEvidence", "numistaLookup"} {
		if _, ok := resp[key]; !ok {
			t.Fatalf("expected response to retain key %q, got %v", key, resp)
		}
	}
	for _, key := range []string{"job", "reused", "report", "proposal", "deepJobId"} {
		if _, ok := resp[key]; ok {
			t.Fatalf("fast-path lookup response unexpectedly contains deep-identification key %q", key)
		}
	}

	var jobCount int64
	if err := db.Model(&models.DeepIdentificationJob{}).Count(&jobCount).Error; err != nil {
		t.Fatalf("count deep identification jobs: %v", err)
	}
	if jobCount != 0 {
		t.Fatalf("fast-path lookup must never create a DeepIdentificationJob row, found %d", jobCount)
	}
}

func TestCoinLookupRejectsNotesOverCharacterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewCoinLookupHandler(nil, services.NewLogger(10))
	router := gin.New()
	router.POST("/api/coins/lookup", handler.Lookup)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("images", "coin.png")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write(deepTestPNGBytes(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("notes", strings.Repeat("a", 2001)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/coins/lookup", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2000 characters or fewer") {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
}

func TestCoinLookupRejectsInvalidImageRoles(t *testing.T) {
	for _, test := range []struct {
		name      string
		roles     []string
		wantError string
	}{
		{name: "unknown role", roles: []string{"unknown"}, wantError: "obverse, reverse, or notes"},
		{name: "role count mismatch", roles: []string{"obverse", "reverse"}, wantError: "one image role"},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			handler := NewCoinLookupHandler(nil, services.NewLogger(10))
			router := gin.New()
			router.POST("/api/coins/lookup", handler.Lookup)

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("images", "coin.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(deepTestPNGBytes(t)); err != nil {
				t.Fatal(err)
			}
			for _, role := range test.roles {
				if err := writer.WriteField("imageRoles", role); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/coins/lookup", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), test.wantError) {
				t.Fatalf("unexpected response: %s", rec.Body.String())
			}
		})
	}
}
