package handlers

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEventLinkDoesNotReportFailedWriteAsSuccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "links.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.AuctionLot{}, &models.AuctionEvent{}); err != nil {
		t.Fatal(err)
	}
	lot := models.AuctionLot{UserID: 1, Title: "Fixture"}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("reject", func(tx *gorm.DB) {
		tx.AddError(errors.New("injected write failure"))
	}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAuctionLotRepository(db)
	h := NewAuctionLotHandler(repo, services.NewAuctionLotService(repo, nil), nil, nil, nil, services.NewLogger(10))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)) })
	router.PUT("/lots/:id/event", h.LinkEvent)
	router.PUT("/bulk", h.BulkLinkEvent)
	for _, tc := range []struct {
		path, body string
		status     int
		fragment   string
	}{
		{"/lots/1/event", `{"eventId":null}`, 500, "Failed"},
		{"/bulk", `{"lotIds":[1],"eventId":null}`, 200, `"updated":0`},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.fragment) {
			t.Errorf("%s: status %d body %s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestEventLinkPartialSuccessOwnershipAndReloadRollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "links.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.AuctionLot{}, &models.AuctionEvent{}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []interface{}{
		&models.AuctionLot{ID: 1, UserID: 1}, &models.AuctionLot{ID: 2, UserID: 1},
		&models.AuctionLot{ID: 3, UserID: 2}, &models.AuctionEvent{ID: 1, UserID: 1},
		&models.AuctionEvent{ID: 2, UserID: 2},
	} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := repository.NewAuctionLotRepository(db)
	svc := services.NewAuctionLotService(repo, nil)
	h := NewAuctionLotHandler(repo, svc, nil, nil, nil, services.NewLogger(10))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)) })
	router.PUT("/lots/:id/event", h.LinkEvent)
	router.PUT("/bulk", h.BulkLinkEvent)
	request := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		return rec
	}
	if err := db.Callback().Update().Before("gorm:update").Register("reject_second", func(tx *gorm.DB) {
		if lot, ok := tx.Statement.Model.(*models.AuctionLot); ok && lot.ID == 2 {
			tx.AddError(errors.New("write rejected"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	rec := request("/bulk", `{"lotIds":[1,1,2,3,999],"eventId":1}`)
	var result services.BulkEventLinkResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || result.Updated != 1 || len(result.Failures) != 3 {
		t.Fatalf("partial results: %d %s", rec.Code, rec.Body.String())
	}
	if result.Failures[0].Code != "storage_error" || result.Failures[1].Code != "not_found" {
		t.Fatal(result)
	}
	for _, body := range []string{`{"eventId":2}`, `{"eventId":999}`} {
		if rec := request("/lots/1/event", body); rec.Code != 404 {
			t.Fatalf("ownership: %s", rec.Body.String())
		}
	}
	queries := 0
	if err := db.Callback().Query().Before("gorm:query").Register("reject_reload", func(tx *gorm.DB) {
		if tx.Statement.Table == "auction_lots" {
			queries++
			if queries == 2 {
				tx.AddError(errors.New("reload rejected"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if rec := request("/lots/1/event", `{"eventId":null}`); rec.Code != 500 {
		t.Fatalf("reload: %s", rec.Body.String())
	}
	lot, err := repo.GetByID(1, 1)
	if err != nil || lot.EventID == nil || *lot.EventID != 1 {
		t.Fatal("reload failure did not roll back link")
	}
	if rec := request("/lots/1/event", `{"eventId":null}`); rec.Code != 200 {
		t.Fatalf("unlink: %s", rec.Body.String())
	}
}
