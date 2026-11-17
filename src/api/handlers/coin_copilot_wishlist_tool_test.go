package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCoinCopilotWishlistURLProposalIsOwnerScopedAndReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CollectionUpdateProposal{}, &models.AppSetting{},
		&models.CoinCopilotThread{}, &models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{},
		&models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{}, &models.CoinCopilotRunGrade{},
	); err != nil {
		t.Fatal(err)
	}
	const listingURL = "https://www.hjbltd.com/#!/inventory/item-detail/ancient-coins/119010"
	_ = db.Create(&models.User{ID: 7, Username: "owner", Email: "owner@example.test", PasswordHash: "x"}).Error
	_ = db.Create(&models.User{ID: 8, Username: "other", Email: "other@example.test", PasswordHash: "x"}).Error
	owned := models.Coin{ID: 11, UserID: 7, Name: "Theodosius I. AE 3", IsWishlist: true, ReferenceURL: listingURL}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatal(err)
	}

	copilotRepo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_wishlist", UserID: 7, Title: "Test"}
	run := &models.CoinCopilotRun{
		ID: "ccr_wishlist", ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunRunning,
		Goal: "test", StartIdempotencyKeyHash: "key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_wishlist", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
		ExecutionStartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(),
	}
	if err := copilotRepo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	coinRepo := repository.NewCoinRepository(db)
	tokenSvc := services.NewInternalTokenService("01234567890123456789012345678901")
	settings := services.NewSettingsService(repository.NewSettingsRepository(db))
	logger := services.NewLogger(10)
	copilotSvc := services.NewCoinCopilotService(copilotRepo, settings, services.NewAgentProxy("", "", logger), tokenSvc, logger, "")
	collectionSvc := services.NewCollectionToolsService(coinRepo, repository.NewCollectionUpdateRepository(db))
	wishlistSvc := services.NewWishlistURLService(coinRepo, services.NewAgentProxy("", "", logger), settings)
	handler := NewCoinCopilotInternalToolsHandler(collectionSvc, copilotSvc, logger).WithWishlistURLService(wishlistSvc)
	router := gin.New()
	router.POST("/wishlist_url_proposal", middleware.CoinCopilotExecutionTokenRequired(tokenSvc, "wishlist_url_proposal"), handler.WishlistURLProposal)

	token, err := tokenSvc.MintForCopilotExecution(7, run.ID, run.ExecutionID, []string{"wishlist_url_proposal"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	call := func(token, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/wishlist_url_proposal", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		return recorder
	}
	decode := func(recorder *httptest.ResponseRecorder) services.WishlistURLAnalysis {
		t.Helper()
		var result services.WishlistURLAnalysis
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode %s: %v", recorder.Body.String(), err)
		}
		return result
	}

	recorder := call(token, `{"tool_call_id":"call_dup","url":"`+listingURL+`?fromBbs=237th"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("duplicate status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decode(recorder); result.Outcome != "duplicate" || result.ExistingCoinID == nil || *result.ExistingCoinID != owned.ID {
		t.Fatalf("duplicate result = %+v", result)
	}

	recorder = call(token, `{"tool_call_id":"call_private","url":"http://127.0.0.1/admin"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("invalid URL status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decode(recorder); result.Outcome != services.WishlistURLOutcomeInvalidURL || result.SourceURL != "" {
		t.Fatalf("invalid URL result = %+v", result)
	}

	for name, body := range map[string]string{
		"missing url":   `{"tool_call_id":"call_empty"}`,
		"oversized url": `{"tool_call_id":"call_long","url":"https://dealer.example/` + strings.Repeat("a", 2000) + `"}`,
		"unknown field": `{"tool_call_id":"call_extra","url":"https://dealer.example/1","coin":{"name":"x"}}`,
	} {
		if recorder := call(token, body); recorder.Code != http.StatusBadRequest {
			t.Errorf("%s status=%d body=%s", name, recorder.Code, recorder.Body.String())
		}
	}

	foreignToken, err := tokenSvc.MintForCopilotExecution(8, run.ID, run.ExecutionID, []string{"wishlist_url_proposal"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := call(foreignToken, `{"tool_call_id":"call_foreign","url":"`+listingURL+`"}`); recorder.Code != http.StatusConflict {
		t.Fatalf("foreign owner status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	otherToolToken, err := tokenSvc.MintForCopilotExecution(7, run.ID, run.ExecutionID, []string{"get_coin"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := call(otherToolToken, `{"tool_call_id":"call_scope","url":"`+listingURL+`"}`); recorder.Code == http.StatusOK {
		t.Fatalf("token without the tool was accepted: %s", recorder.Body.String())
	}

	var count int64
	db.Model(&models.Coin{}).Count(&count)
	if count != 1 {
		t.Fatalf("proposal tool wrote coins: count=%d", count)
	}
}
