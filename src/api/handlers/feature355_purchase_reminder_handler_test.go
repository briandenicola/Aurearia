package handlers

// Independent QA handler-level contract tests for Feature 355.
// Owned by Brutus (Tester/QA).
//
// Build tag: feature355
// References PurchaseReminderHandler, PurchaseReminderService -- types that
// do not exist until Cassius lands Phase 4 (handler). Remove the build tag
// once implementation is complete and verify all tests pass.
//
// Frozen contract under test (spec.md API Contract):
//   - POST /coins/:id/reminder: 201 new, 200 update, 400 past/invalid, 404 missing, 409 non-wishlist.
//   - GET  /coins/:id/reminder: 200 active, 404 when none.
//   - DELETE /coins/:id/reminder: 204 success, 404 when none.
//   - GET  /purchase-reminders: 200 list of active (pending+notified) for current user.
//   - Cross-user isolation: another user's coin returns 404 on all operations.
//   - Upsert: second POST on same coin returns 200 with same reminder ID (not 201).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupReminderHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupCoinHandlerTestDB(t)
	db.AutoMigrate(&models.PurchaseReminder{}, &models.Notification{})
	return db
}

func setupReminderHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := setupReminderHandlerDB(t)
	reminderRepo := repository.NewPurchaseReminderRepository(db)
	coinRepo := repository.NewCoinRepository(db)
	logger := services.NewLogger(100)
	svc := services.NewPurchaseReminderService(reminderRepo, coinRepo, logger)
	handler := NewPurchaseReminderHandler(svc, logger)

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.POST("/coins/:id/reminder", handler.CreateOrUpdate)
	protected.GET("/coins/:id/reminder", handler.Get)
	protected.DELETE("/coins/:id/reminder", handler.Cancel)
	protected.GET("/purchase-reminders", handler.List)
	return r, db
}

func reminderPayload(remindDate, timezone string) *bytes.Reader {
	body, _ := json.Marshal(map[string]any{
		"remindDate": remindDate,
		"timezone":   timezone,
	})
	return bytes.NewReader(body)
}

func sendReminderRequest(router *gin.Engine, method, path string, userID uint, body *bytes.Reader) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Authorization", authHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func seedWishlistCoinForUser(t *testing.T, db *gorm.DB, userID uint) models.Coin {
	t.Helper()
	coin := models.Coin{
		UserID:     userID,
		Name:       "Trajan Denarius",
		Category:   models.CategoryRoman,
		Material:   models.MaterialSilver,
		Era:        models.EraAncient,
		IsWishlist: true,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed wishlist coin: %v", err)
	}
	return coin
}

func seedNonWishlistCoinForUser(t *testing.T, db *gorm.DB, userID uint) models.Coin {
	t.Helper()
	coin := models.Coin{
		UserID:     userID,
		Name:       "Augustus Aureus",
		Category:   models.CategoryRoman,
		Material:   models.MaterialGold,
		Era:        models.EraAncient,
		IsWishlist: false,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed non-wishlist coin: %v", err)
	}
	return coin
}

func futureDateUTC() string {
	return time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
}

// TestFeature355_ReminderHandler_CreateReturns201ForNewReminder
// FR-001/US1 AC1: first POST creates reminder, returns 201 with PurchaseReminder JSON.
func TestFeature355_ReminderHandler_CreateReturns201ForNewReminder(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))

	if w.Code != http.StatusCreated {
		t.Fatalf("POST create expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Error("response must include id field")
	}
	if body["status"] != "pending" {
		t.Errorf("new reminder status=%v, want pending", body["status"])
	}
}

// TestFeature355_ReminderHandler_UpdateReturns200WithSameID
// FR-002/US1 AC2: second POST on same coin updates in place, returns 200 with same ID.
func TestFeature355_ReminderHandler_UpdateReturns200WithSameID(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w1 := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))
	if w1.Code != http.StatusCreated {
		t.Fatalf("first POST expected 201, got %d: %s", w1.Code, w1.Body.String())
	}
	var first map[string]any
	json.Unmarshal(w1.Body.Bytes(), &first)

	newDate := time.Now().UTC().AddDate(0, 0, 14).Format("2006-01-02")
	w2 := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(newDate, "UTC"))
	if w2.Code != http.StatusOK {
		t.Fatalf("second POST expected 200 (upsert), got %d: %s", w2.Code, w2.Body.String())
	}
	var second map[string]any
	json.Unmarshal(w2.Body.Bytes(), &second)

	if first["id"] != second["id"] {
		t.Errorf("FR-002 violation: upsert must return same ID; first=%v second=%v", first["id"], second["id"])
	}
}

// TestFeature355_ReminderHandler_PastDateReturns400
// FR-005/US1 AC4: POST with a past remindDate returns 400.
func TestFeature355_ReminderHandler_PastDateReturns400(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	pastDate := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	w := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(pastDate, "UTC"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("past date expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] == "" {
		t.Error("400 response must include error field")
	}
}

// TestFeature355_ReminderHandler_InvalidTimezoneReturns400
// FR-004/US1 AC5: POST with an invalid IANA timezone returns 400.
func TestFeature355_ReminderHandler_InvalidTimezoneReturns400(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "Not/A/Zone"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid timezone expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestFeature355_ReminderHandler_NonWishlistCoinReturns409
// FR-001/US1 AC3: POST on a non-wishlist coin returns 409 Conflict.
func TestFeature355_ReminderHandler_NonWishlistCoinReturns409(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedNonWishlistCoinForUser(t, db, 1)

	w := sendReminderRequest(router, http.MethodPost,
		fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))

	if w.Code != http.StatusConflict {
		t.Fatalf("non-wishlist coin expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestFeature355_ReminderHandler_MissingCoinReturns404
// Spec: coin not found or not owned by user returns 404.
func TestFeature355_ReminderHandler_MissingCoinReturns404(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")

	w := sendReminderRequest(router, http.MethodPost, "/api/coins/99999/reminder", 1,
		reminderPayload(futureDateUTC(), "UTC"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("missing coin expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestFeature355_ReminderHandler_CrossUserOwnershipIsolation
// Edge case: another user's coin must return 404 on POST, GET, and DELETE.
func TestFeature355_ReminderHandler_CrossUserOwnershipIsolation(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "intruder")
	coin := seedWishlistCoinForUser(t, db, 1) // owned by user 1

	path := fmt.Sprintf("/api/coins/%d/reminder", coin.ID)

	// POST by user 2 must be 404 (not 409/403)
	w := sendReminderRequest(router, http.MethodPost, path, 2, reminderPayload(futureDateUTC(), "UTC"))
	if w.Code != http.StatusNotFound {
		t.Errorf("cross-user POST expected 404, got %d", w.Code)
	}

	// GET by user 2 must be 404
	w = sendReminderRequest(router, http.MethodGet, path, 2, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("cross-user GET expected 404, got %d", w.Code)
	}

	// DELETE by user 2 must be 404
	w = sendReminderRequest(router, http.MethodDelete, path, 2, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("cross-user DELETE expected 404, got %d", w.Code)
	}
}

// TestFeature355_ReminderHandler_GetActiveReminder
// US5 AC2: GET /coins/:id/reminder returns 200 with active reminder JSON.
func TestFeature355_ReminderHandler_GetActiveReminder(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	// Create reminder first
	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))

	w := sendReminderRequest(router, http.MethodGet, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET active reminder expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "pending" {
		t.Errorf("GET reminder status=%v, want pending", body["status"])
	}
}

// TestFeature355_ReminderHandler_GetNoReminderReturns404
// US5 AC2 (absent case): GET when no active reminder returns 404.
func TestFeature355_ReminderHandler_GetNoReminderReturns404(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w := sendReminderRequest(router, http.MethodGet, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET with no reminder expected 404, got %d", w.Code)
	}
}

// TestFeature355_ReminderHandler_CancelReturns204
// US3 AC1: DELETE cancels the active reminder, returns 204.
func TestFeature355_ReminderHandler_CancelReturns204(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))

	w := sendReminderRequest(router, http.MethodDelete, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Subsequent GET must be 404
	w = sendReminderRequest(router, http.MethodGet, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET after cancel expected 404, got %d", w.Code)
	}
}

// TestFeature355_ReminderHandler_CancelWhenNoneReturns404
// US3 AC2: DELETE with no active reminder returns 404.
func TestFeature355_ReminderHandler_CancelWhenNoneReturns404(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w := sendReminderRequest(router, http.MethodDelete, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("DELETE with no reminder expected 404, got %d", w.Code)
	}
}

// TestFeature355_ReminderHandler_ListActiveReminders
// US5 AC3 / FR-013: GET /reminders returns pending+notified for the user.
func TestFeature355_ReminderHandler_ListActiveReminders(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin1 := seedWishlistCoinForUser(t, db, 1)
	coin2 := seedWishlistCoinForUser(t, db, 1)

	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin1.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))
	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin2.ID), 1,
		reminderPayload(futureDateUTC(), "America/Chicago"))

	w := sendReminderRequest(router, http.MethodGet, "/api/purchase-reminders", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /reminders expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	reminders, ok := body["reminders"].([]any)
	if !ok || len(reminders) != 2 {
		t.Fatalf("FR-013 violation: expected 2 reminders in list, got: %v", body)
	}
}

// TestFeature355_ReminderHandler_ListExcludesCancelledReminders
// FR-010/FR-013: cancelled reminders must not appear in GET /reminders.
func TestFeature355_ReminderHandler_ListExcludesCancelledReminders(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))
	sendReminderRequest(router, http.MethodDelete, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)

	w := sendReminderRequest(router, http.MethodGet, "/api/purchase-reminders", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	reminders, _ := body["reminders"].([]any)
	if len(reminders) != 0 {
		t.Fatalf("FR-010 violation: cancelled reminder appeared in /reminders list")
	}
}

// TestFeature355_ReminderHandler_RecreateAfterManualCancel
// US3 AC3: after manual cancel, creating a new reminder returns 201 (fresh row).
func TestFeature355_ReminderHandler_RecreateAfterManualCancel(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	coin := seedWishlistCoinForUser(t, db, 1)

	w1 := sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))
	var first map[string]any
	json.Unmarshal(w1.Body.Bytes(), &first)

	sendReminderRequest(router, http.MethodDelete, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1, nil)

	w3 := sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))
	if w3.Code != http.StatusCreated {
		t.Fatalf("POST after cancel expected 201, got %d: %s", w3.Code, w3.Body.String())
	}
	var second map[string]any
	json.Unmarshal(w3.Body.Bytes(), &second)
	// New row must be created (new ID), not reuse of the cancelled row.
	if first["id"] == second["id"] {
		t.Errorf("US3 AC3 violation: expected a new reminder ID after cancel+recreate, got same ID %v", first["id"])
	}
}

// TestFeature355_ReminderHandler_ListIsolatedByUser verifies ownership:
// user 1's reminders do not appear in user 2's list.
func TestFeature355_ReminderHandler_ListIsolatedByUser(t *testing.T) {
	router, db := setupReminderHandlerRouter(t)
	createTestUser(t, db, 1, "user1")
	createTestUser(t, db, 2, "user2")
	coin := seedWishlistCoinForUser(t, db, 1)

	// User 1 creates a reminder
	sendReminderRequest(router, http.MethodPost, fmt.Sprintf("/api/coins/%d/reminder", coin.ID), 1,
		reminderPayload(futureDateUTC(), "UTC"))

	// User 2's list must be empty
	w := sendReminderRequest(router, http.MethodGet, "/api/purchase-reminders", 2, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	reminders, _ := body["reminders"].([]any)
	if len(reminders) != 0 {
		t.Fatalf("cross-user isolation violation: user 2 can see user 1's reminders")
	}
}
