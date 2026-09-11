package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

func TestBulkHandlerStorageAssignmentsAreAtomicAndRejectTrays(t *testing.T) {
	db := setupCoinHandlerTestDB(t)
	coinRepo := repository.NewCoinRepository(db)
	locationRepo := repository.NewStorageLocationRepository(db)
	coinService := services.NewCoinService(coinRepo, nil).WithStorageLocationSupport(locationRepo)
	handler := NewBulkHandler(coinRepo, nil, locationRepo, nil).WithCoinService(coinService)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)); c.Next() })
	router.POST("/coins/bulk", handler.BulkAction)

	standard := models.StorageLocation{UserID: 1, Name: "Safe", Type: "standard"}
	rows, columns := 1, 1
	tray := models.StorageLocation{UserID: 1, Name: "Tray", Type: "tray", Rows: &rows, Columns: &columns}
	if err := db.Create(&standard).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&tray).Error; err != nil {
		t.Fatal(err)
	}
	slot := 1
	coin := models.Coin{UserID: 1, Name: "Coin", StorageLocationID: &tray.ID, StorageSlot: &slot}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/coins/bulk", strings.NewReader(`{"coinIds":[1],"action":"assign-location","storageLocationId":`+itoa(standard.ID)+`}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("standard status=%d body=%s", response.Code, response.Body.String())
	}
	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.StorageLocationID == nil || *updated.StorageLocationID != standard.ID || updated.StorageSlot != nil {
		t.Fatalf("assignment not updated atomically: %#v", updated)
	}

	request = httptest.NewRequest(http.MethodPost, "/coins/bulk", strings.NewReader(`{"coinIds":[1],"action":"assign-location","storageLocationId":`+itoa(tray.ID)+`}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Tray assignments require choosing a slot on each coin.") {
		t.Fatalf("tray status=%d body=%s", response.Code, response.Body.String())
	}
}
