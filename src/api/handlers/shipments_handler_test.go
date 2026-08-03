package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type stubShipmentHandlerCarrierClient struct {
	carrier  models.ShipmentCarrier
	snapshot services.ShipmentTrackingSnapshot
}

func (c *stubShipmentHandlerCarrierClient) Carrier() models.ShipmentCarrier { return c.carrier }

func (c *stubShipmentHandlerCarrierClient) GetTracking(context.Context, string) (services.ShipmentTrackingSnapshot, error) {
	return c.snapshot, nil
}

func setupShipmentHandlerTestRouter(t *testing.T, clients ...services.ShipmentCarrierClient) (*gin.Engine, *gorm.DB, models.User, models.User, models.Coin) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.StorageLocation{},
		&models.MintLocation{},
		&models.Tag{},
		&models.CoinSet{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CoinReference{},
		&models.Notification{},
		&models.Shipment{},
		&models.ShipmentEvent{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := models.User{Username: "owner", Email: "owner-ship@test.com", PasswordHash: "hash"}
	other := models.User{Username: "other", Email: "other-ship@test.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}
	coin := models.Coin{Name: "Shipment Coin", UserID: owner.ID}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	coinRepo := repository.NewCoinRepository(db)
	shipmentRepo := repository.NewShipmentRepository(db)
	notifSvc := services.NewNotificationService(repository.NewNotificationRepository(db), nil, repository.NewUserRepository(db), nil, services.NewLogger(10))
	shipmentSvc := services.NewShipmentService(shipmentRepo, coinRepo, services.NewShipmentCarrierClientRegistry(clients...), notifSvc, services.NewLogger(10))
	shipmentHandler := NewShipmentHandler(shipmentSvc)

	router := gin.New()
	api := router.Group("/api")
	api.Use(func(c *gin.Context) {
		switch c.GetHeader("Authorization") {
		case "owner":
			c.Set("userId", owner.ID)
		case "other":
			c.Set("userId", other.ID)
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})
	api.GET("/coins/:id/shipment", shipmentHandler.GetForCoin)
	api.PUT("/coins/:id/shipment", shipmentHandler.UpsertForCoin)
	api.DELETE("/coins/:id/shipment", shipmentHandler.DeleteForCoin)
	api.PUT("/coins/:id/shipment/manual-override", shipmentHandler.SetManualOverride)
	api.POST("/coins/:id/shipment/sync", shipmentHandler.SyncForCoin)

	return router, db, owner, other, coin
}

func TestShipmentHandler_UpsertAndGetForCoin(t *testing.T) {
	router, _, owner, _, coin := setupShipmentHandlerTestRouter(t)

	body := map[string]any{
		"carrier":        "usps",
		"trackingNumber": "94001000",
		"notes":          "seller packed securely",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "owner")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upsert status=%d body=%s", w.Code, w.Body.String())
	}

	var upsertResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &upsertResp); err != nil {
		t.Fatalf("parse upsert response: %v", err)
	}
	if upsertResp["trackingUrl"] == "" {
		t.Fatalf("expected trackingUrl in response: %+v", upsertResp)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/coins/"+toUintParam(coin.ID)+"/shipment", nil)
	getReq.Header.Set("Authorization", "owner")
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getW.Code, getW.Body.String())
	}

	var getResp map[string]any
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("parse get response: %v", err)
	}
	shipment := getResp["shipment"].(map[string]any)
	if shipment["trackingNumber"] != "94001000" {
		t.Fatalf("trackingNumber=%v want 94001000", shipment["trackingNumber"])
	}
	if shipment["userId"].(float64) != float64(owner.ID) {
		t.Fatalf("userId=%v want %d", shipment["userId"], owner.ID)
	}
}

func TestShipmentHandler_GetForCoin_OwnerScoped(t *testing.T) {
	router, _, _, _, coin := setupShipmentHandlerTestRouter(t)

	body := `{"carrier":"usps","trackingNumber":"94001000"}`
	req := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "owner")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed upsert status=%d body=%s", w.Code, w.Body.String())
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/api/coins/"+toUintParam(coin.ID)+"/shipment", nil)
	otherReq.Header.Set("Authorization", "other")
	otherW := httptest.NewRecorder()
	router.ServeHTTP(otherW, otherReq)
	if otherW.Code != http.StatusNotFound {
		t.Fatalf("non-owner get status=%d body=%s", otherW.Code, otherW.Body.String())
	}
}

func TestShipmentHandler_SyncForCoin(t *testing.T) {
	now := time.Now().UTC()
	client := &stubShipmentHandlerCarrierClient{
		carrier: models.ShipmentCarrierUSPS,
		snapshot: services.ShipmentTrackingSnapshot{
			Carrier:             models.ShipmentCarrierUSPS,
			TrackingNumber:      "94001000",
			CurrentStatus:       models.ShipmentStatusOutForDelivery,
			CurrentStatusSource: models.ShipmentStatusSourceAPI,
			Events: []services.ShipmentTrackingEvent{
				{
					EventKey:     "evt-1",
					Status:       models.ShipmentStatusOutForDelivery,
					StatusSource: models.ShipmentStatusSourceAPI,
					OccurredAt:   now,
					Location:     "Boston",
					Description:  "Out for delivery",
				},
			},
			SyncedAt: now,
		},
	}
	router, _, _, _, coin := setupShipmentHandlerTestRouter(t, client)

	body := `{"carrier":"usps","trackingNumber":"94001000"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment", bytes.NewBufferString(body))
	putReq.Header.Set("Authorization", "owner")
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	router.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("seed upsert status=%d body=%s", putW.Code, putW.Body.String())
	}

	overrideReq := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment/manual-override", bytes.NewBufferString(`{"enabled":false,"status":"pending","note":""}`))
	overrideReq.Header.Set("Authorization", "owner")
	overrideReq.Header.Set("Content-Type", "application/json")
	overrideW := httptest.NewRecorder()
	router.ServeHTTP(overrideW, overrideReq)
	if overrideW.Code != http.StatusOK {
		t.Fatalf("disable override status=%d body=%s", overrideW.Code, overrideW.Body.String())
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/coins/"+toUintParam(coin.ID)+"/shipment/sync", nil)
	syncReq.Header.Set("Authorization", "owner")
	syncW := httptest.NewRecorder()
	router.ServeHTTP(syncW, syncReq)
	if syncW.Code != http.StatusOK {
		t.Fatalf("sync status=%d body=%s", syncW.Code, syncW.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(syncW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse sync response: %v", err)
	}
	shipment := resp["shipment"].(map[string]any)
	if shipment["currentStatus"] != string(models.ShipmentStatusOutForDelivery) {
		t.Fatalf("currentStatus=%v want %s", shipment["currentStatus"], models.ShipmentStatusOutForDelivery)
	}
}

func TestShipmentHandler_SyncForCoin_NotConfiguredCarrierReturnsBadRequest(t *testing.T) {
	router, _, _, _, coin := setupShipmentHandlerTestRouter(t)

	body := `{"carrier":"fedex","trackingNumber":"94001000"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment", bytes.NewBufferString(body))
	putReq.Header.Set("Authorization", "owner")
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	router.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("seed upsert status=%d body=%s", putW.Code, putW.Body.String())
	}

	overrideReq := httptest.NewRequest(http.MethodPut, "/api/coins/"+toUintParam(coin.ID)+"/shipment/manual-override", bytes.NewBufferString(`{"enabled":false,"status":"pending","note":""}`))
	overrideReq.Header.Set("Authorization", "owner")
	overrideReq.Header.Set("Content-Type", "application/json")
	overrideW := httptest.NewRecorder()
	router.ServeHTTP(overrideW, overrideReq)
	if overrideW.Code != http.StatusOK {
		t.Fatalf("disable override status=%d body=%s", overrideW.Code, overrideW.Body.String())
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/coins/"+toUintParam(coin.ID)+"/shipment/sync", nil)
	syncReq.Header.Set("Authorization", "owner")
	syncW := httptest.NewRecorder()
	router.ServeHTTP(syncW, syncReq)
	if syncW.Code != http.StatusBadRequest {
		t.Fatalf("sync status=%d body=%s", syncW.Code, syncW.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(syncW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse sync error response: %v", err)
	}
	if resp["error"] != "shipment carrier integration not configured: fedex" {
		t.Fatalf("error=%q want %q", resp["error"], "shipment carrier integration not configured: fedex")
	}
}

func toUintParam(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
