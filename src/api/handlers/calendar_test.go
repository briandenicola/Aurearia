package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCalendarHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuctionEvent{}, &models.AuctionLot{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func calendarRequest(t *testing.T, handler *CalendarHandler, query string) (int, map[string]interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/calendar"+query, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userId", uint(1))

	handler.GetCalendar(c)

	var body map[string]interface{}
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
	}
	return w.Code, body
}

func TestCalendarDefaultDateRangeSpansStartOfMonthToThreeMonthsOut(t *testing.T) {
	db := setupCalendarHandlerDB(t)
	handler := NewCalendarHandler(repository.NewAuctionEventRepository(db), repository.NewAuctionLotRepository(db))

	code, body := calendarRequest(t, handler, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	rng, ok := body["range"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a range object, got %v", body["range"])
	}

	now := time.Now()
	wantStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	wantEnd := wantStart.AddDate(0, 3, 0)

	if rng["start"] != wantStart.Format("2006-01-02") {
		t.Errorf("range.start = %v, want %v", rng["start"], wantStart.Format("2006-01-02"))
	}
	if rng["end"] != wantEnd.Format("2006-01-02") {
		t.Errorf("range.end = %v, want %v", rng["end"], wantEnd.Format("2006-01-02"))
	}
}

func TestCalendarCustomDateRangeIsEchoedBackAndFiltersResults(t *testing.T) {
	db := setupCalendarHandlerDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)

	inRange := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	outOfRange := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	if err := auctionRepo.Create(&models.AuctionLot{
		Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-IN/test",
		Title: "In range lot", Status: models.AuctionStatusWatching, SaleDate: &inRange, UserID: 1,
	}); err != nil {
		t.Fatalf("failed to create in-range lot: %v", err)
	}
	if err := auctionRepo.Create(&models.AuctionLot{
		Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-OUT/test",
		Title: "Out of range lot", Status: models.AuctionStatusWatching, SaleDate: &outOfRange, UserID: 1,
	}); err != nil {
		t.Fatalf("failed to create out-of-range lot: %v", err)
	}

	handler := NewCalendarHandler(repository.NewAuctionEventRepository(db), auctionRepo)
	code, body := calendarRequest(t, handler, "?start=2026-08-01&end=2026-08-31")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	rng := body["range"].(map[string]interface{})
	if rng["start"] != "2026-08-01" || rng["end"] != "2026-08-31" {
		t.Errorf("range = %v, want start=2026-08-01 end=2026-08-31", rng)
	}

	lots := body["lots"].([]interface{})
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot in range, got %d: %v", len(lots), lots)
	}
	lot := lots[0].(map[string]interface{})
	if lot["title"] != "In range lot" {
		t.Errorf("title = %v, want 'In range lot'", lot["title"])
	}
}

func TestCalendarRejectsInvalidStartDate(t *testing.T) {
	db := setupCalendarHandlerDB(t)
	handler := NewCalendarHandler(repository.NewAuctionEventRepository(db), repository.NewAuctionLotRepository(db))

	code, _ := calendarRequest(t, handler, "?start=not-a-date")
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestCalendarRejectsInvalidEndDate(t *testing.T) {
	db := setupCalendarHandlerDB(t)
	handler := NewCalendarHandler(repository.NewAuctionEventRepository(db), repository.NewAuctionLotRepository(db))

	code, _ := calendarRequest(t, handler, "?start=2026-08-01&end=not-a-date")
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestCalendarReturnsMixedLotAndEventShape(t *testing.T) {
	db := setupCalendarHandlerDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	eventRepo := repository.NewAuctionEventRepository(db)

	saleDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if err := auctionRepo.Create(&models.AuctionLot{
		Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-MIX/test",
		Title: "Mixed shape lot", AuctionHouse: "CNG", Status: models.AuctionStatusWatching, SaleDate: &saleDate, UserID: 1,
	}); err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	startDate := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	if err := eventRepo.Create(&models.AuctionEvent{
		UserID: 1, Title: "Mixed shape event", AuctionHouse: "CNG", StartDate: &startDate,
	}); err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	handler := NewCalendarHandler(eventRepo, auctionRepo)
	code, body := calendarRequest(t, handler, "?start=2026-08-01&end=2026-08-31")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	lots := body["lots"].([]interface{})
	events := body["events"].([]interface{})
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	lot := lots[0].(map[string]interface{})
	if lot["type"] != "lot" {
		t.Errorf("lot type = %v, want 'lot'", lot["type"])
	}
	if lot["title"] != "Mixed shape lot" {
		t.Errorf("lot title = %v, want 'Mixed shape lot'", lot["title"])
	}

	event := events[0].(map[string]interface{})
	if event["type"] != "event" {
		t.Errorf("event type = %v, want 'event'", event["type"])
	}
	if event["title"] != "Mixed shape event" {
		t.Errorf("event title = %v, want 'Mixed shape event'", event["title"])
	}
}
