package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestQuickCaptureCreateDraftMultipartAndDoesNotCreateCoin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(7))
		c.Next()
	})
	router.POST("/api/quick-capture/drafts", handler.CreateDraft)

	form := url.Values{}
	form.Set("notes", "Needs attribution later")
	req := httptest.NewRequest(http.MethodPost, "/api/quick-capture/drafts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var coinCount int64
	if err := db.Model(&models.Coin{}).Count(&coinCount).Error; err != nil {
		t.Fatalf("count coins: %v", err)
	}
	if coinCount != 0 {
		t.Fatalf("draft creation should not create coins, got %d", coinCount)
	}
}

func TestQuickCaptureCreateDraftValidationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_validation_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(7))
		c.Next()
	})
	router.POST("/api/quick-capture/drafts", handler.CreateDraft)

	req := httptest.NewRequest(http.MethodPost, "/api/quick-capture/drafts", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestQuickCaptureCreateDraftRejectsOversizedUploadBeforeContentValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_oversized_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint(7))
		c.Next()
	})
	router.POST("/api/quick-capture/drafts", handler.CreateDraft)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("notes", "oversized image should fail by size"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	part, err := writer.CreateFormFile("obverseImage", "oversized.jpg")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("x"), services.MaxImageUploadBytes+1)); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/quick-capture/drafts", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "20MB") {
		t.Fatalf("expected size validation error, got %s", rec.Body.String())
	}
	var draftCount int64
	if err := db.Model(&models.QuickCaptureDraft{}).Count(&draftCount).Error; err != nil {
		t.Fatalf("count drafts: %v", err)
	}
	if draftCount != 0 {
		t.Fatalf("oversized upload should not create a draft, got %d", draftCount)
	}
}

func TestQuickCaptureDraftResumeDiscardPromoteContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_resume_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	viewerID := uint(7)
	router.Use(func(c *gin.Context) {
		c.Set("userId", viewerID)
		c.Next()
	})
	router.GET("/api/quick-capture/drafts", handler.ListDrafts)
	router.GET("/api/quick-capture/drafts/:id", handler.GetDraft)
	router.PUT("/api/quick-capture/drafts/:id", handler.UpdateDraft)
	router.POST("/api/quick-capture/drafts/:id/discard", handler.DiscardDraft)
	router.POST("/api/quick-capture/drafts/:id/promote", handler.PromoteDraft)

	ownerDraft := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Owner draft", Status: models.QuickCaptureDraftStatusActive}
	otherDraft := models.QuickCaptureDraft{UserID: 99, WorkingTitle: "Other draft", Status: models.QuickCaptureDraftStatusActive}
	discardDraft := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Discard draft", Status: models.QuickCaptureDraftStatusActive}
	needsName := models.QuickCaptureDraft{UserID: viewerID, Notes: "No title", Status: models.QuickCaptureDraftStatusActive}
	savedPrice := 100.0
	explicitEmptyDraft := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Saved title", Era: string(models.EraAncient), AcquisitionSource: "Saved source", Notes: "Saved notes", Status: models.QuickCaptureDraftStatusActive}
	explicitNullPriceDraft := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Saved priced title", Era: string(models.EraAncient), PurchasePrice: &savedPrice, Status: models.QuickCaptureDraftStatusActive}
	for _, draft := range []*models.QuickCaptureDraft{&ownerDraft, &otherDraft, &discardDraft, &needsName, &explicitEmptyDraft, &explicitNullPriceDraft} {
		if err := db.Create(draft).Error; err != nil {
			t.Fatalf("create draft: %v", err)
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/quick-capture/drafts", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Owner draft") || strings.Contains(rec.Body.String(), "Other draft") {
		t.Fatalf("expected owner-scoped active draft list, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/quick-capture/drafts/%d", otherDraft.ID), nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owned get to return 404, got %d: %s", rec.Code, rec.Body.String())
	}
	body, _ := json.Marshal(map[string]interface{}{"confirm": true})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", otherDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owned promote to return 404, got %d: %s", rec.Code, rec.Body.String())
	}

	updateForm := url.Values{"workingTitle": {"Updated owner draft"}, "notes": {"saved"}}
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/quick-capture/drafts/%d", ownerDraft.ID), strings.NewReader(updateForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Updated owner draft") {
		t.Fatalf("expected update success, got %d: %s", rec.Code, rec.Body.String())
	}

	for i := 0; i < 2; i++ {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/discard", discardDraft.ID), nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), string(models.QuickCaptureDraftStatusDiscarded)) {
			t.Fatalf("expected idempotent discard success, got %d: %s", rec.Code, rec.Body.String())
		}
	}
	promotedForDiscard := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Already promoted", Status: models.QuickCaptureDraftStatusPromoted}
	if err := db.Create(&promotedForDiscard).Error; err != nil {
		t.Fatalf("create promoted draft: %v", err)
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/discard", promotedForDiscard.ID), nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected promoted discard conflict, got %d: %s", rec.Code, rec.Body.String())
	}
	promotingDraft := models.QuickCaptureDraft{UserID: viewerID, WorkingTitle: "Promoting", Status: models.QuickCaptureDraftStatusPromoting}
	if err := db.Create(&promotingDraft).Error; err != nil {
		t.Fatalf("create promoting draft: %v", err)
	}
	body, _ = json.Marshal(map[string]interface{}{"confirm": true})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", promotingDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected promoting draft promotion conflict, got %d: %s", rec.Code, rec.Body.String())
	}

	body, _ = json.Marshal(map[string]interface{}{"confirm": true})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", needsName.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "fields") || !strings.Contains(rec.Body.String(), "name") {
		t.Fatalf("expected field-level promotion validation, got %d: %s", rec.Code, rec.Body.String())
	}

	body, _ = json.Marshal(map[string]interface{}{
		"confirm": true,
		"overrides": map[string]interface{}{
			"name":             "",
			"purchaseLocation": "",
			"notes":            "",
		},
	})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", explicitEmptyDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "fields") || !strings.Contains(rec.Body.String(), "name") {
		t.Fatalf("expected explicit empty JSON name override to fail validation, got %d: %s", rec.Code, rec.Body.String())
	}

	body, _ = json.Marshal(map[string]interface{}{
		"confirm": true,
		"overrides": map[string]interface{}{
			"name":             "HTTP current title",
			"purchaseLocation": "",
			"notes":            "",
		},
	})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", explicitEmptyDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected explicit optional clears to promote successfully, got %d: %s", rec.Code, rec.Body.String())
	}
	var clearedCoin models.Coin
	if err := db.Where("name = ?", "HTTP current title").First(&clearedCoin).Error; err != nil {
		t.Fatalf("load explicitly cleared promoted coin: %v", err)
	}
	if clearedCoin.PurchaseLocation != "" || clearedCoin.Notes != "" {
		t.Fatalf("expected explicit empty optional JSON overrides to clear values, got source=%q notes=%q", clearedCoin.PurchaseLocation, clearedCoin.Notes)
	}

	body, _ = json.Marshal(map[string]interface{}{
		"confirm": true,
		"overrides": map[string]interface{}{
			"name":          "HTTP current priced title",
			"purchasePrice": nil,
		},
	})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", explicitNullPriceDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected explicit null price override to promote successfully, got %d: %s", rec.Code, rec.Body.String())
	}
	var nullPriceCoin models.Coin
	if err := db.Where("name = ?", "HTTP current priced title").First(&nullPriceCoin).Error; err != nil {
		t.Fatalf("load explicitly null price promoted coin: %v", err)
	}
	if nullPriceCoin.PurchasePrice != nil || nullPriceCoin.CurrentValue != nil {
		t.Fatalf("expected explicit null price override to clear saved price, got price=%v value=%v", nullPriceCoin.PurchasePrice, nullPriceCoin.CurrentValue)
	}

	body, _ = json.Marshal(map[string]interface{}{"confirm": true, "overrides": map[string]interface{}{"era": string(models.EraAncient)}})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", ownerDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"alreadyPromoted":false`) {
		t.Fatalf("expected first promotion success, got %d: %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", ownerDraft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"alreadyPromoted":true`) {
		t.Fatalf("expected idempotent promotion success, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestQuickCapturePromotionIncrementsActiveCountExactlyOnceAndPreservesWishlistSold(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_counts_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	viewerID := uint(7)
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", viewerID)
		c.Next()
	})
	router.POST("/api/quick-capture/drafts/:id/promote", handler.PromoteDraft)

	wishlistPrice := 50.0
	soldPrice := 75.0
	existing := []models.Coin{
		{UserID: viewerID, Name: "Existing active", Category: models.CategoryRoman, Material: models.MaterialSilver, Era: models.EraAncient},
		{UserID: viewerID, Name: "Wishlist", Category: models.CategoryGreek, Material: models.MaterialGold, Era: models.EraAncient, IsWishlist: true, PurchasePrice: &wishlistPrice, CurrentValue: &wishlistPrice},
		{UserID: viewerID, Name: "Sold", Category: models.CategoryRoman, Material: models.MaterialBronze, Era: models.EraAncient, IsSold: true, PurchasePrice: &soldPrice, CurrentValue: &soldPrice},
	}
	for i := range existing {
		if err := db.Create(&existing[i]).Error; err != nil {
			t.Fatalf("seed coin: %v", err)
		}
	}
	draftPrice := 25.0
	draft := models.QuickCaptureDraft{
		UserID:            viewerID,
		WorkingTitle:      "Promoted active coin",
		Era:               string(models.EraAncient),
		AcquisitionSource: "Quick Capture tray",
		PurchasePrice:     &draftPrice,
		Status:            models.QuickCaptureDraftStatusActive,
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	activeBefore, wishlistBefore, soldBefore := quickCaptureCoinContractCounts(t, db, viewerID)
	body, _ := json.Marshal(map[string]interface{}{"confirm": true})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", draft.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("promotion attempt %d expected 200, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	activeAfter, wishlistAfter, soldAfter := quickCaptureCoinContractCounts(t, db, viewerID)
	if activeAfter != activeBefore+1 {
		t.Fatalf("active collection count should increment exactly once: before=%d after=%d", activeBefore, activeAfter)
	}
	if wishlistAfter != wishlistBefore {
		t.Fatalf("wishlist total changed after Quick Capture v1 promotion: before=%d after=%d", wishlistBefore, wishlistAfter)
	}
	if soldAfter != soldBefore {
		t.Fatalf("sold total changed after Quick Capture v1 promotion: before=%d after=%d", soldBefore, soldAfter)
	}
	var promotedRows int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND name = ?", viewerID, "Promoted active coin").Count(&promotedRows).Error; err != nil {
		t.Fatalf("count promoted rows: %v", err)
	}
	if promotedRows != 1 {
		t.Fatalf("expected one promoted coin row after repeated promotion, got %d", promotedRows)
	}
}

func TestQuickCapturePromotionCanTargetWishlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quick_capture_handler_wishlist_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.ValueSnapshot{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	viewerID := uint(7)
	handler := NewQuickCaptureHandler(services.NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()), nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", viewerID)
		c.Next()
	})
	router.POST("/api/quick-capture/drafts/:id/promote", handler.PromoteDraft)

	draft := models.QuickCaptureDraft{
		UserID:       viewerID,
		WorkingTitle: "Wishlist target coin",
		Era:          string(models.EraAncient),
		Status:       models.QuickCaptureDraftStatusActive,
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	activeBefore, wishlistBefore, soldBefore := quickCaptureCoinContractCounts(t, db, viewerID)
	body, _ := json.Marshal(map[string]interface{}{"confirm": true, "target": "wishlist"})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/quick-capture/drafts/%d/promote", draft.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"target":"wishlist"`) {
		t.Fatalf("expected wishlist promotion success, got %d: %s", rec.Code, rec.Body.String())
	}

	activeAfter, wishlistAfter, soldAfter := quickCaptureCoinContractCounts(t, db, viewerID)
	if activeAfter != activeBefore {
		t.Fatalf("wishlist promotion should not increment active collection count: before=%d after=%d", activeBefore, activeAfter)
	}
	if wishlistAfter != wishlistBefore+1 {
		t.Fatalf("wishlist promotion should increment wishlist count once: before=%d after=%d", wishlistBefore, wishlistAfter)
	}
	if soldAfter != soldBefore {
		t.Fatalf("wishlist promotion should not change sold count: before=%d after=%d", soldBefore, soldAfter)
	}

	var coin models.Coin
	if err := db.Where("user_id = ? AND name = ?", viewerID, "Wishlist target coin").First(&coin).Error; err != nil {
		t.Fatalf("load promoted wishlist coin: %v", err)
	}
	if !coin.IsWishlist || coin.IsSold {
		t.Fatalf("expected wishlist promoted coin to be wishlist and unsold, wishlist=%v sold=%v", coin.IsWishlist, coin.IsSold)
	}
}

func quickCaptureCoinContractCounts(t *testing.T, db *gorm.DB, userID uint) (active, wishlist, sold int64) {
	t.Helper()
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ? AND is_sold = ?", userID, false, false).Count(&active).Error; err != nil {
		t.Fatalf("count active coins: %v", err)
	}
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ?", userID, true).Count(&wishlist).Error; err != nil {
		t.Fatalf("count wishlist coins: %v", err)
	}
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_sold = ?", userID, true).Count(&sold).Error; err != nil {
		t.Fatalf("count sold coins: %v", err)
	}
	return active, wishlist, sold
}
