package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func feature358StorageRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	db := setupCoinHandlerTestDB(t)
	repo := repository.NewStorageLocationRepository(db)
	handler := NewStorageLocationHandler(services.NewStorageLocationService(repo))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)); c.Next() })
	router.GET("/storage-locations", handler.List)
	router.POST("/storage-locations", handler.Create)
	router.PUT("/storage-locations/:id", handler.Update)
	router.DELETE("/storage-locations/:id", handler.Delete)
	router.GET("/storage-locations/:id/occupancy", handler.Occupancy)
	router.GET("/storage-trays", handler.ListTrays)
	return router, db
}

func TestStorageLocationHandlerFeature358Contracts(t *testing.T) {
	router, db := feature358StorageRouter(t)
	create := httptest.NewRequest(http.MethodPost, "/storage-locations", strings.NewReader(`{"name":"Cabinet A","type":"tray","rows":3,"columns":3}`))
	create.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, create)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var tray models.StorageLocation
	if err := json.Unmarshal(response.Body.Bytes(), &tray); err != nil {
		t.Fatal(err)
	}
	slot := 6
	coin := models.Coin{UserID: 1, Name: "Denarius", StorageLocationID: &tray.ID, StorageSlot: &slot}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	occupancy := httptest.NewRecorder()
	router.ServeHTTP(occupancy, httptest.NewRequest(http.MethodGet, "/storage-locations/"+itoa(tray.ID)+"/occupancy?coinId="+itoa(coin.ID), nil))
	if occupancy.Code != http.StatusOK || !strings.Contains(occupancy.Body.String(), `"occupiedSlots":[6]`) || !strings.Contains(occupancy.Body.String(), `"currentCoinSlot":6`) {
		t.Fatalf("occupancy status=%d body=%s", occupancy.Code, occupancy.Body.String())
	}
	aggregate := httptest.NewRecorder()
	router.ServeHTTP(aggregate, httptest.NewRequest(http.MethodGet, "/storage-trays", nil))
	if aggregate.Code != http.StatusOK || !strings.Contains(aggregate.Body.String(), `"storageSlot":6`) || strings.Contains(aggregate.Body.String(), `"purchasePrice"`) {
		t.Fatalf("aggregate status=%d body=%s", aggregate.Code, aggregate.Body.String())
	}
	resize := httptest.NewRequest(http.MethodPut, "/storage-locations/"+itoa(tray.ID), strings.NewReader(`{"rows":2,"columns":2}`))
	resize.Header.Set("Content-Type", "application/json")
	conflict := httptest.NewRecorder()
	router.ServeHTTP(conflict, resize)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"tray_occupied"`) {
		t.Fatalf("resize status=%d body=%s", conflict.Code, conflict.Body.String())
	}
}

func TestStorageLocationHandlerFeature358OmissionAndPrivacy(t *testing.T) {
	router, db := feature358StorageRouter(t)
	empty := httptest.NewRecorder()
	router.ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/storage-trays", nil))
	if empty.Code != http.StatusOK || strings.TrimSpace(empty.Body.String()) != `{"trays":[]}` {
		t.Fatalf("empty aggregate status=%d body=%s", empty.Code, empty.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/storage-locations", strings.NewReader(`{"name":"Safe"}`))
	create.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, create)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"type":"standard"`) || !strings.Contains(response.Body.String(), `"rows":null`) {
		t.Fatalf("compatibility status=%d body=%s", response.Code, response.Body.String())
	}
	var standard models.StorageLocation
	if err := json.Unmarshal(response.Body.Bytes(), &standard); err != nil {
		t.Fatal(err)
	}
	referenced := models.Coin{UserID: 1, Name: "Stored", StorageLocationID: &standard.ID}
	if err := db.Create(&referenced).Error; err != nil {
		t.Fatal(err)
	}
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/storage-locations/"+itoa(standard.ID), nil))
	if deleteResponse.Code != http.StatusConflict || !strings.Contains(deleteResponse.Body.String(), `"code":"location_referenced"`) {
		t.Fatalf("delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	invalid := httptest.NewRequest(http.MethodPost, "/storage-locations", strings.NewReader(`{"name":"Too Large","type":"tray","rows":21,"columns":20}`))
	invalid.Header.Set("Content-Type", "application/json")
	invalidResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || !strings.Contains(invalidResponse.Body.String(), `"field":"rows"`) {
		t.Fatalf("invalid status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
	rows, columns := 1, 1
	other := models.StorageLocation{UserID: 2, Name: "Private", Type: "tray", Rows: &rows, Columns: &columns}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	hidden := httptest.NewRecorder()
	router.ServeHTTP(hidden, httptest.NewRequest(http.MethodGet, "/storage-locations/"+itoa(other.ID)+"/occupancy", nil))
	if hidden.Code != http.StatusNotFound {
		t.Fatalf("cross-owner status=%d body=%s", hidden.Code, hidden.Body.String())
	}
	ownerAggregate := httptest.NewRecorder()
	router.ServeHTTP(ownerAggregate, httptest.NewRequest(http.MethodGet, "/storage-trays", nil))
	if ownerAggregate.Code != http.StatusOK || strings.Contains(ownerAggregate.Body.String(), "Private") {
		t.Fatalf("owner aggregate status=%d body=%s", ownerAggregate.Code, ownerAggregate.Body.String())
	}
}

func TestStorageLocationHandlerDuplicateNameHasStableConflictCode(t *testing.T) {
	router, _ := feature358StorageRouter(t)
	for index, name := range []string{"Cabinet A", "cabinet a"} {
		request := httptest.NewRequest(http.MethodPost, "/storage-locations", strings.NewReader(`{"name":"`+name+`"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if index == 0 {
			if response.Code != http.StatusCreated {
				t.Fatalf("first create status=%d body=%s", response.Code, response.Body.String())
			}
			continue
		}
		if response.Code != http.StatusConflict ||
			!strings.Contains(response.Body.String(), `"code":"duplicate_location"`) ||
			!strings.Contains(response.Body.String(), `"message":"Choose a different storage location name."`) {
			t.Fatalf("duplicate status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func itoa(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
