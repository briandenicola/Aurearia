package handlers

import (
	"bytes"
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

func TestAuctionLotHandlerUpdateStatusPersistsChangedMaxBidWithoutStatusChange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuctionLot{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	initialMaxBid := 100.0
	lot := models.AuctionLot{
		Title:        "CNG test lot",
		NumisBidsURL: "https://auctions.cngcoins.com/lots/view/4-LOT/test",
		Source:       models.AuctionSourceCNG,
		SourceURL:    "https://auctions.cngcoins.com/lots/view/4-LOT/test",
		Status:       models.AuctionStatusBidding,
		MaxBid:       &initialMaxBid,
		UserID:       42,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("create lot: %v", err)
	}

	auctionRepo := repository.NewAuctionLotRepository(db)
	handler := NewAuctionLotHandler(auctionRepo, services.NewAuctionLotService(auctionRepo, nil), nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/auctions/1/status", bytes.NewBufferString(`{"status":"bidding","maxBid":150}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("userId", uint(42))

	handler.UpdateStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateStatus status = %d body=%s", w.Code, w.Body.String())
	}

	var updated models.AuctionLot
	if err := db.First(&updated, lot.ID).Error; err != nil {
		t.Fatalf("reload lot: %v", err)
	}
	if updated.Status != models.AuctionStatusBidding {
		t.Fatalf("status = %q, want bidding", updated.Status)
	}
	if updated.MaxBid == nil || *updated.MaxBid != 150 {
		t.Fatalf("max bid = %v, want 150", updated.MaxBid)
	}
}

func TestAuctionLotHandlerUpdateStatusPersistsWinningBidWhenWon(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuctionLot{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	lot := models.AuctionLot{
		Title:        "CNG test lot",
		NumisBidsURL: "https://auctions.cngcoins.com/lots/view/4-LOT/test",
		Source:       models.AuctionSourceCNG,
		SourceURL:    "https://auctions.cngcoins.com/lots/view/4-LOT/test",
		Status:       models.AuctionStatusBidding,
		UserID:       42,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("create lot: %v", err)
	}

	auctionRepo := repository.NewAuctionLotRepository(db)
	handler := NewAuctionLotHandler(auctionRepo, services.NewAuctionLotService(auctionRepo, nil), nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/auctions/1/status", bytes.NewBufferString(`{"status":"won","winningBid":275.5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("userId", uint(42))

	handler.UpdateStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateStatus status = %d body=%s", w.Code, w.Body.String())
	}

	var updated models.AuctionLot
	if err := db.First(&updated, lot.ID).Error; err != nil {
		t.Fatalf("reload lot: %v", err)
	}
	if updated.Status != models.AuctionStatusWon {
		t.Fatalf("status = %q, want won", updated.Status)
	}
	if updated.WinningBid == nil || *updated.WinningBid != 275.5 {
		t.Fatalf("winning bid = %v, want 275.5", updated.WinningBid)
	}
}

func TestAuctionLotHandlerGetBidRecommendationReturnsInsufficientDataWithoutHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuctionLot{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	estimate := 500.0
	lot := models.AuctionLot{
		Title:        "CNG target lot",
		NumisBidsURL: "https://auctions.cngcoins.com/lots/view/4-TARGET/test",
		Source:       models.AuctionSourceCNG,
		SourceURL:    "https://auctions.cngcoins.com/lots/view/4-TARGET/test",
		Category:     models.CategoryRoman,
		Estimate:     &estimate,
		Status:       models.AuctionStatusWatching,
		UserID:       42,
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("create lot: %v", err)
	}

	auctionRepo := repository.NewAuctionLotRepository(db)
	handler := NewAuctionLotHandler(auctionRepo, services.NewAuctionLotService(auctionRepo, nil), nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/auctions/1/bid-recommendation", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("userId", uint(42))

	handler.GetBidRecommendation(c)

	if w.Code != http.StatusOK {
		t.Fatalf("GetBidRecommendation status = %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"insufficient_data"`)) {
		t.Fatalf("expected insufficient_data confidence, got body=%s", w.Body.String())
	}
}
