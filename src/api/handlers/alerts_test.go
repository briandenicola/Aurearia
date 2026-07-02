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
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionAlertsHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuctionLot{}, &models.PriceAlert{}, &models.BidReminder{}, &models.AuctionAlertRun{}, &models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestAlertHandlerCreateRejectsUnownedLot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuctionAlertsHandlerDB(t)
	owner := models.User{Username: "owner", Email: "owner@example.com", PasswordHash: "hash"}
	other := models.User{Username: "other", Email: "other@example.com", PasswordHash: "hash"}
	db.Create(&owner)
	db.Create(&other)
	lot := models.AuctionLot{
		UserID:       owner.ID,
		NumisBidsURL: "https://example.com/lot",
		SourceURL:    "https://example.com/lot",
		Title:        "Watched lot",
		Status:       models.AuctionStatusWatching,
	}
	db.Create(&lot)

	service := services.NewAuctionAlertService(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		repository.NewAuctionLotRepository(db),
	)
	handler := NewAlertHandler(service)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", other.ID)
		c.Next()
	})
	router.POST("/api/alerts", handler.CreateAlert)

	body := strings.NewReader(`{"auctionLotId":` + strconvAuctionAlertUint(lot.ID) + `,"targetPrice":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuctionAlertAdminHandlerRunNowAndListRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuctionAlertsHandlerDB(t)
	admin := models.User{Username: "admin", Email: "admin@example.com", PasswordHash: "hash", Role: models.RoleAdmin}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	logger := services.NewLogger(100)
	evaluator := services.NewAuctionAlertEvaluator(
		repository.NewPriceAlertRepository(db),
		repository.NewBidReminderRepository(db),
		repository.NewUserRepository(db),
		services.NewPushoverService(settingsSvc, logger),
		logger,
	)
	runRepo := repository.NewAuctionAlertRunRepository(db)
	scheduler := services.NewAuctionAlertScheduler(evaluator, runRepo, nil, settingsSvc, logger)
	handler := NewAuctionAlertAdminHandler(scheduler, runRepo)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", admin.ID)
		c.Next()
	})
	router.POST("/api/admin/auction-alerts/run", handler.RunNow)
	router.GET("/api/admin/auction-alert-runs", handler.ListRuns)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-alerts/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var runResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("failed to parse run response: %v", err)
	}
	if runResp["runId"] == nil || runResp["priceAlertsTriggered"] == nil || runResp["bidRemindersSent"] == nil {
		t.Fatalf("run response missing stable fields: %#v", runResp)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/auction-alert-runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d: %s", w.Code, w.Body.String())
	}
	var listResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if listResp["total"].(float64) != 1 {
		t.Fatalf("total = %v, want 1", listResp["total"])
	}
}

func strconvAuctionAlertUint(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
