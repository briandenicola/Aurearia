package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var quickAccessHandlerDBCounter uint64

func setupQuickAccessHandler(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:quick_access_handler_%d_%d?mode=memory&cache=shared",
		time.Now().UnixNano(), atomic.AddUint64(&quickAccessHandlerDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}, &models.CoinSet{}, &models.AuctionLot{}, &models.AuctionEvent{}, &models.QuickAccessPin{}); err != nil {
		t.Fatal(err)
	}
	service := services.NewQuickAccessService(repository.NewQuickAccessRepository(db))
	handler := NewQuickAccessHandler(service, nil)
	router := gin.New()
	api := router.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	api.GET("/quick-access", handler.List)
	api.PUT("/quick-access/:type/:id", handler.Pin)
	api.DELETE("/quick-access/:type/:id", handler.Unpin)
	return router, db
}

func TestQuickAccessHandlerContract(t *testing.T) {
	router, db := setupQuickAccessHandler(t)
	coin := models.Coin{UserID: 1, Name: "Denarius"}
	foreign := models.Coin{UserID: 2, Name: "Foreign"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}

	request := func(method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}
	firstPut := request(http.MethodPut, fmt.Sprintf("/api/quick-access/coin/%d", coin.ID))
	if firstPut.Code != http.StatusCreated {
		t.Fatalf("first PUT=%d body=%s", firstPut.Code, firstPut.Body.String())
	}
	secondPut := request(http.MethodPut, fmt.Sprintf("/api/quick-access/coin/%d", coin.ID))
	if secondPut.Code != http.StatusOK {
		t.Fatalf("second PUT=%d body=%s", secondPut.Code, secondPut.Body.String())
	}
	var firstItem, secondItem services.QuickAccessItemDTO
	if err := json.Unmarshal(firstPut.Body.Bytes(), &firstItem); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(secondPut.Body.Bytes(), &secondItem); err != nil {
		t.Fatal(err)
	}
	if !secondItem.PinnedAt.Equal(firstItem.PinnedAt) {
		t.Fatalf("repeated PUT changed pinnedAt: %s != %s", secondItem.PinnedAt, firstItem.PinnedAt)
	}
	w := request(http.MethodGet, "/api/quick-access")
	if w.Code != http.StatusOK {
		t.Fatalf("GET=%d body=%s", w.Code, w.Body.String())
	}
	var body services.QuickAccessListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Coin == nil || body.Items[0].CoinSet != nil {
		t.Fatalf("unexpected response: %#v", body)
	}
	if w := request(http.MethodPut, fmt.Sprintf("/api/quick-access/coin/%d", foreign.ID)); w.Code != http.StatusNotFound || w.Body.String() != "{\"error\":\"Quick access item not found\"}" {
		t.Fatalf("foreign PUT=%d body=%s", w.Code, w.Body.String())
	}
	sold := models.Coin{UserID: 1, Name: "Sold", IsSold: true}
	if err := db.Create(&sold).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/api/quick-access/coin/999999",
		fmt.Sprintf("/api/quick-access/coin/%d", sold.ID),
	} {
		if w := request(http.MethodPut, path); w.Code != http.StatusNotFound || w.Body.String() != "{\"error\":\"Quick access item not found\"}" {
			t.Fatalf("generic 404 %s=%d body=%s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/quick-access/nope/1", "/api/quick-access/coin/0", "/api/quick-access/coin/not-a-number"} {
		if w := request(http.MethodPut, path); w.Code != http.StatusBadRequest {
			t.Fatalf("invalid PUT %s=%d", path, w.Code)
		}
	}
	if w := request(http.MethodDelete, fmt.Sprintf("/api/quick-access/coin/%d", coin.ID)); w.Code != http.StatusNoContent {
		t.Fatalf("DELETE=%d body=%s", w.Code, w.Body.String())
	}
	if w := request(http.MethodDelete, fmt.Sprintf("/api/quick-access/coin/%d", coin.ID)); w.Code != http.StatusNoContent {
		t.Fatalf("second DELETE=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuickAccessHandlerListUsesExactDiscriminatedPayloads(t *testing.T) {
	router, db := setupQuickAccessHandler(t)
	coin := models.Coin{UserID: 1, Name: "Denarius", IsWishlist: true}
	set := models.CoinSet{UserID: 1, Name: "Roman Set"}
	lot := models.AuctionLot{
		UserID: 1, Title: "Auction Lot", NumisBidsURL: "https://example.test/lot",
		Status: models.AuctionStatusWatching,
	}
	event := models.AuctionEvent{UserID: 1, Title: "Coin Show", Origin: models.AuctionEventOriginManual}
	for _, target := range []interface{}{&coin, &set, &lot, &event} {
		if err := db.Create(target).Error; err != nil {
			t.Fatal(err)
		}
	}

	targets := []struct {
		targetType models.QuickAccessTargetType
		targetID   uint
		payloadKey string
	}{
		{models.QuickAccessTargetCoin, coin.ID, "coin"},
		{models.QuickAccessTargetCoinSet, set.ID, "coinSet"},
		{models.QuickAccessTargetAuctionLot, lot.ID, "auctionLot"},
		{models.QuickAccessTargetCalendarEvent, event.ID, "calendarEvent"},
	}
	for _, target := range targets {
		w := httptest.NewRecorder()
		path := fmt.Sprintf("/api/quick-access/%s/%d", target.targetType, target.targetID)
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, path, nil))
		if w.Code != http.StatusCreated {
			t.Fatalf("PUT %s=%d body=%s", path, w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/quick-access", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != len(targets) {
		t.Fatalf("items=%d want %d", len(response.Items), len(targets))
	}

	expectedPayload := make(map[string]string, len(targets))
	for _, target := range targets {
		expectedPayload[string(target.targetType)] = target.payloadKey
	}
	payloadKeys := []string{"coin", "coinSet", "auctionLot", "calendarEvent"}
	for _, item := range response.Items {
		var discriminator string
		if err := json.Unmarshal(item["type"], &discriminator); err != nil {
			t.Fatalf("invalid discriminator: %v", err)
		}
		wantPayload := expectedPayload[discriminator]
		if wantPayload == "" {
			t.Fatalf("unexpected discriminator %q", discriminator)
		}
		for _, required := range []string{"type", "id", "pinnedAt", wantPayload} {
			if _, ok := item[required]; !ok {
				t.Fatalf("%s item missing %q: %v", discriminator, required, item)
			}
		}
		if len(item) != 4 {
			t.Fatalf("%s item has unexpected fields: %v", discriminator, item)
		}
		for _, payloadKey := range payloadKeys {
			_, present := item[payloadKey]
			if payloadKey == wantPayload && !present {
				t.Fatalf("%s item missing matching payload %q", discriminator, payloadKey)
			}
			if payloadKey != wantPayload && present {
				t.Fatalf("%s item contains mismatched payload %q", discriminator, payloadKey)
			}
		}
	}
}

func TestQuickAccessHandlerCapAndGenericFailure(t *testing.T) {
	router, db := setupQuickAccessHandler(t)
	for i := 0; i < 6; i++ {
		set := models.CoinSet{UserID: 1, Name: fmt.Sprintf("Set %d", i)}
		if err := db.Create(&set).Error; err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/quick-access/coin_set/%d", set.ID), nil))
		if i < 5 && w.Code != http.StatusCreated {
			t.Fatalf("set %d status=%d body=%s", i, w.Code, w.Body.String())
		}
		if i == 5 && (w.Code != http.StatusBadRequest || w.Body.String() != "{\"error\":\"you can pin up to 5 sets\"}") {
			t.Fatalf("sixth set status=%d body=%s", w.Code, w.Body.String())
		}
	}
	if err := db.Migrator().DropTable(&models.QuickAccessPin{}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/quick-access", nil))
	if w.Code != http.StatusInternalServerError || w.Body.String() != "{\"error\":\"Failed to load quick access\"}" {
		t.Fatalf("generic failure status=%d body=%s", w.Code, w.Body.String())
	}
}
