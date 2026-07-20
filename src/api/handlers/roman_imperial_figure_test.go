package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRomanImperialFigureHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.RomanImperialFigure{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	figures := []models.RomanImperialFigure{
		{Name: "Augustus", NormalizedName: "augustus", Aliases: models.StringList{"Octavian"}, Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 1, RarityTier: models.RarityTierCommon},
		{Name: "Livia", NormalizedName: "livia", Aliases: models.StringList{}, Role: models.ImperialFigureRoleEmpress, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 2, RarityTier: models.RarityTierCommon},
	}
	if err := db.Create(&figures).Error; err != nil {
		t.Fatalf("failed to seed figures: %v", err)
	}
	return db
}

func romanImperialFigureRequest(t *testing.T, handler *RomanImperialFigureHandler, query string) (int, map[string]interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/roman-imperial-figures"+query, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Search(c)

	var body map[string]interface{}
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
	}
	return w.Code, body
}

func TestRomanImperialFigureSearchReturnsAllByDefault(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	code, body := romanImperialFigureRequest(t, handler, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	figures, ok := body["figures"].([]interface{})
	if !ok || len(figures) != 2 {
		t.Fatalf("expected 2 figures, got %v", body["figures"])
	}
}

func TestRomanImperialFigureSearchFiltersByRole(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	code, body := romanImperialFigureRequest(t, handler, "?role=empress")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	figures := body["figures"].([]interface{})
	if len(figures) != 1 {
		t.Fatalf("expected 1 figure, got %d", len(figures))
	}
	figure := figures[0].(map[string]interface{})
	if figure["name"] != "Livia" {
		t.Errorf("name = %v, want Livia", figure["name"])
	}
}

func TestRomanImperialFigureSearchRejectsInvalidRole(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	code, _ := romanImperialFigureRequest(t, handler, "?role=notarole")
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestRomanImperialFigureSearchRejectsInvalidLimit(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	code, _ := romanImperialFigureRequest(t, handler, "?limit=notanumber")
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestRomanImperialFigureSearchByQuery(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	code, body := romanImperialFigureRequest(t, handler, "?q=oct")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	figures := body["figures"].([]interface{})
	if len(figures) != 1 {
		t.Fatalf("expected 1 figure matched via alias, got %d", len(figures))
	}
}

func TestRomanImperialFigureGetReturnsFigure(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	var augustus models.RomanImperialFigure
	if err := db.Where("name = ?", "Augustus").First(&augustus).Error; err != nil {
		t.Fatalf("failed to load seeded Augustus: %v", err)
	}

	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/roman-imperial-figures/1", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(augustus.ID), 10)}}

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var got models.RomanImperialFigure
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if got.Name != "Augustus" {
		t.Errorf("name = %v, want Augustus", got.Name)
	}
}

func TestRomanImperialFigureGetReturns404ForMissingID(t *testing.T) {
	db := setupRomanImperialFigureHandlerDB(t)
	svc := services.NewRomanImperialFigureService(repository.NewRomanImperialFigureRepository(db))
	handler := NewRomanImperialFigureHandler(svc)

	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/roman-imperial-figures/999999", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "999999"}}

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
