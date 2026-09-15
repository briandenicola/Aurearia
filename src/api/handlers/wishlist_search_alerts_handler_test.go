package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupWishlistSearchAlertHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := setupCoinHandlerTestDB(t)
	db.AutoMigrate(&models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{})
	handler := NewWishlistSearchAlertHandler(services.NewWishlistSearchAlertService(repository.NewWishlistSearchAlertRepository(db)))
	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.GET("/wishlist/search-alerts", handler.List)
	protected.POST("/wishlist/search-alerts", handler.Create)
	protected.GET("/wishlist/search-alerts/:alertId", handler.Get)
	protected.PUT("/wishlist/search-alerts/:alertId", handler.Update)
	protected.DELETE("/wishlist/search-alerts/:alertId", handler.Delete)
	protected.GET("/admin/wishlist-search-alert-runs", handler.ListAdminRuns)
	return r, db
}

func alertPayload(name string) *bytes.Reader {
	body, _ := json.Marshal(map[string]any{
		"name":     name,
		"cadence":  "manual",
		"isActive": true,
		"criteria": map[string]any{
			"rulerOrIssuer": "Domitian",
			"coinType":      "Denarius",
			"priceMax":      300,
			"currency":      "USD",
			"sourceFilters": []string{"vcoins.com"},
		},
	})
	return bytes.NewReader(body)
}

func sendAlertRequest(router *gin.Engine, method, path string, userID uint, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", authHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestWishlistSearchAlertHandler_CRUDAndOwnerScoping(t *testing.T) {
	router, db := setupWishlistSearchAlertHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "other")
	w := sendAlertRequest(router, http.MethodPost, "/api/wishlist/search-alerts", 1, alertPayload("Domitian discovery"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.WishlistSearchAlert
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	w = sendAlertRequest(router, http.MethodGet, "/api/wishlist/search-alerts", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d", w.Code)
	}
	w = sendAlertRequest(router, http.MethodGet, "/api/wishlist/search-alerts/"+uintString(created.ID), 2, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("non-owner read expected 404, got %d: %s", w.Code, w.Body.String())
	}
	w = sendAlertRequest(router, http.MethodPut, "/api/wishlist/search-alerts/"+uintString(created.ID), 1, alertPayload("Updated discovery"))
	if w.Code != http.StatusOK {
		t.Fatalf("owner update expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = sendAlertRequest(router, http.MethodDelete, "/api/wishlist/search-alerts/"+uintString(created.ID), 1, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWishlistSearchAlertHandler_Validation(t *testing.T) {
	router, db := setupWishlistSearchAlertHandlerRouter(t)
	createTestUser(t, db, 1, "validator")
	body, _ := json.Marshal(map[string]any{"name": "Invalid", "cadence": "hourly", "criteria": map[string]any{"rulerOrIssuer": "Domitian"}})
	w := sendAlertRequest(router, http.MethodPost, "/api/wishlist/search-alerts", 1, bytes.NewReader(body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid cadence expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWishlistSearchAlertHandler_ListAdminRuns(t *testing.T) {
	router, db := setupWishlistSearchAlertHandlerRouter(t)
	createTestUser(t, db, 1, "collector")
	alert := models.WishlistSearchAlert{
		UserID: 1, Name: "Greek bronze", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true,
	}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if err := db.Create(&models.AlertRun{
		AlertID: alert.ID, UserID: 1, TriggerType: models.AlertRunTriggerScheduled,
		Status: models.AlertRunStatusCompleted, StartedAt: alert.CreatedAt, CriteriaSnapshot: "{}",
		ResultCount: 4, NewCount: 2,
	}).Error; err != nil {
		t.Fatalf("create run: %v", err)
	}

	w := sendAlertRequest(router, http.MethodGet, "/api/admin/wishlist-search-alert-runs?page=1&limit=5", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list admin runs expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var response AdminWishlistSearchAlertRunListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Total != 1 || len(response.Runs) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Runs[0].AlertName != "Greek bronze" || response.Runs[0].UserName != "collector" || response.Runs[0].NewCount != 2 {
		t.Fatalf("missing admin run details: %+v", response.Runs[0])
	}
	if response.Runs[0].PartialWarnings == nil {
		t.Fatal("partialWarnings must serialize as an empty array")
	}
}
