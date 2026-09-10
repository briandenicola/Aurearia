package services

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var calendarServiceDBCounter uint64

func setupCalendarServiceTest(t *testing.T) (*CalendarService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:calendar_service_%d_%d?mode=memory&cache=shared",
		time.Now().UnixNano(), atomic.AddUint64(&calendarServiceDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.AuctionEvent{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return NewCalendarService(repository.NewAuctionEventRepository(db), nil), db
}

func TestCalendarServiceUpdateClassifiesOnlyMissingRowsAsNotFound(t *testing.T) {
	service, db := setupCalendarServiceTest(t)

	err := service.Update(99, 1, func(*models.AuctionEvent) {})
	if !errors.Is(err, ErrCalendarEventNotFound) {
		t.Fatalf("missing event error=%v, want ErrCalendarEventNotFound", err)
	}

	if err := db.Migrator().DropTable(&models.AuctionEvent{}); err != nil {
		t.Fatalf("drop auction_events: %v", err)
	}
	err = service.Update(99, 1, func(*models.AuctionEvent) {})
	if err == nil {
		t.Fatal("expected repository failure")
	}
	if errors.Is(err, ErrCalendarEventNotFound) {
		t.Fatalf("repository failure was misclassified as not found: %v", err)
	}
}
