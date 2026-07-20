package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupEmperorTrackerHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.RomanImperialFigure{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	figures := []models.RomanImperialFigure{
		{Name: "Augustus", NormalizedName: "augustus", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 1, RarityTier: models.RarityTierCommon},
		{Name: "Tiberius", NormalizedName: "tiberius", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: 14, ReignEnd: 37, SortOrder: 2, RarityTier: models.RarityTierCommon},
		{Name: "Livia", NormalizedName: "livia", Role: models.ImperialFigureRoleEmpress, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 3, RarityTier: models.RarityTierCommon},
	}
	if err := db.Create(&figures).Error; err != nil {
		t.Fatalf("failed to seed figures: %v", err)
	}
	return db
}

func emperorTrackerHandlerFor(db *gorm.DB) *EmperorTrackerHandler {
	svc := services.NewEmperorTrackerService(repository.NewRomanImperialFigureRepository(db), repository.NewCoinRepository(db))
	return NewEmperorTrackerHandler(svc, repository.NewUserRepository(db))
}

func TestEmperorTrackerHandler_GetProgressRejectsWhenNotEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupEmperorTrackerHandlerDB(t)
	user := models.User{Username: "u1", PasswordHash: "hash", EmperorTrackerEnabled: false}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := emperorTrackerHandlerFor(db)

	req := httptest.NewRequest(http.MethodGet, "/stats/emperors", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", user.ID)

	handler.GetProgress(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body.String())
	}
}

func TestEmperorTrackerHandler_GetProgressReturnsEmperorAndSuggestionsWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupEmperorTrackerHandlerDB(t)
	user := models.User{Username: "u1", PasswordHash: "hash", EmperorTrackerEnabled: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := emperorTrackerHandlerFor(db)

	req := httptest.NewRequest(http.MethodGet, "/stats/emperors", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", user.ID)

	handler.GetProgress(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	emperor, ok := body["emperor"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected emperor object, got %v", body["emperor"])
	}
	if emperor["total"] != float64(2) {
		t.Errorf("emperor.total = %v, want 2", emperor["total"])
	}
	suggestions, ok := body["suggestions"].([]interface{})
	if !ok || len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %v", body["suggestions"])
	}
	if body["usurpers"] != nil || body["empresses"] != nil || body["other"] != nil {
		t.Errorf("expected optional categories omitted by default, got usurpers=%v empresses=%v other=%v",
			body["usurpers"], body["empresses"], body["other"])
	}
}

func TestEmperorTrackerHandler_GetProgressIncludesEnabledOptionalCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupEmperorTrackerHandlerDB(t)
	user := models.User{
		Username:                    "u1",
		PasswordHash:                "hash",
		EmperorTrackerEnabled:       true,
		EmperorTrackerShowEmpresses: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := emperorTrackerHandlerFor(db)

	req := httptest.NewRequest(http.MethodGet, "/stats/emperors", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", user.ID)

	handler.GetProgress(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	empresses, ok := body["empresses"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected empresses object since ShowEmpresses is enabled, got %v", body["empresses"])
	}
	if empresses["total"] != float64(1) {
		t.Errorf("empresses.total = %v, want 1", empresses["total"])
	}
	if body["usurpers"] != nil || body["other"] != nil {
		t.Errorf("expected usurpers/other to stay omitted, got usurpers=%v other=%v", body["usurpers"], body["other"])
	}
}

func TestEmperorTrackerHandler_GetProgressReturns404ForMissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupEmperorTrackerHandlerDB(t)
	handler := emperorTrackerHandlerFor(db)

	req := httptest.NewRequest(http.MethodGet, "/stats/emperors", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", uint(999999))

	handler.GetProgress(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
