package database

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var feature357DBCounter uint64

type legacyFeature357Set struct {
	ID       uint `gorm:"primaryKey"`
	UserID   uint
	Name     string
	PinnedAt *time.Time
}

func (legacyFeature357Set) TableName() string { return "coin_sets" }

type legacyFeature357Event struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint
	Title  string
}

func (legacyFeature357Event) TableName() string { return "auction_events" }

type legacyFeature357Lot struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint
	Title   string
	EventID *uint
}

func (legacyFeature357Lot) TableName() string { return "auction_lots" }

func TestFeature357MigrationPreservesAndReconcilesLegacyPins(t *testing.T) {
	dsn := fmt.Sprintf("file:feature357_%d_%d?mode=memory&cache=shared",
		time.Now().UnixNano(), atomic.AddUint64(&feature357DBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&legacyFeature357Set{}, &legacyFeature357Event{}, &legacyFeature357Lot{}); err != nil {
		t.Fatal(err)
	}

	legacyTimes := make(map[uint]time.Time)
	for i := 0; i < 6; i++ {
		pinnedAt := time.Date(2026, 9, 1+i, 12, 0, 0, 123000000, time.UTC)
		set := legacyFeature357Set{UserID: 1, Name: fmt.Sprintf("Legacy %d", i), PinnedAt: &pinnedAt}
		if err := db.Create(&set).Error; err != nil {
			t.Fatal(err)
		}
		legacyTimes[set.ID] = pinnedAt
	}
	unpinned := legacyFeature357Set{UserID: 1, Name: "Unpinned"}
	if err := db.Create(&unpinned).Error; err != nil {
		t.Fatal(err)
	}
	unlinked := legacyFeature357Event{UserID: 1, Title: "Manual"}
	linked := legacyFeature357Event{UserID: 1, Title: "Generated"}
	if err := db.Create(&unlinked).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&linked).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&legacyFeature357Lot{UserID: 1, Title: "Lot", EventID: &linked.ID}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&models.CoinSet{}, &models.AuctionEvent{}, &models.QuickAccessPin{}); err != nil {
		t.Fatal(err)
	}
	authoritativeAt := time.Date(2026, 8, 31, 8, 0, 0, 456000000, time.UTC)
	if err := db.Create(&models.QuickAccessPin{
		UserID: 1, TargetType: models.QuickAccessTargetCoinSet, TargetID: 1, PinnedAt: authoritativeAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	for pass := 0; pass < 2; pass++ {
		if err := migrateQuickAccessPins(db); err != nil {
			t.Fatalf("migration pass %d: %v", pass+1, err)
		}
	}

	var pins []models.QuickAccessPin
	if err := db.Where("target_type = ?", models.QuickAccessTargetCoinSet).Order("target_id").Find(&pins).Error; err != nil {
		t.Fatal(err)
	}
	if len(pins) != 6 {
		t.Fatalf("pins=%d want 6", len(pins))
	}
	var setCount, eventCount int64
	if err := db.Model(&models.CoinSet{}).Count(&setCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.AuctionEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if setCount != 7 || eventCount != 2 {
		t.Fatalf("migration lost rows: sets=%d events=%d", setCount, eventCount)
	}
	var unpinnedCount int64
	if err := db.Model(&models.QuickAccessPin{}).
		Where("target_type = ? AND target_id = ?", models.QuickAccessTargetCoinSet, unpinned.ID).
		Count(&unpinnedCount).Error; err != nil {
		t.Fatal(err)
	}
	if unpinnedCount != 0 {
		t.Fatal("migration created a pin for an unpinned set")
	}
	for _, pin := range pins {
		want := legacyTimes[pin.TargetID]
		if pin.TargetID == 1 {
			want = authoritativeAt
		}
		if !pin.PinnedAt.Equal(want) {
			t.Fatalf("set %d timestamp=%s want %s", pin.TargetID, pin.PinnedAt, want)
		}
		var set models.CoinSet
		if err := db.First(&set, pin.TargetID).Error; err != nil {
			t.Fatal(err)
		}
		if set.PinnedAt == nil || !set.PinnedAt.Equal(want) {
			t.Fatalf("set %d mirror=%v want %s", set.ID, set.PinnedAt, want)
		}
	}

	var migratedLinked, migratedUnlinked models.AuctionEvent
	if err := db.First(&migratedLinked, linked.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&migratedUnlinked, unlinked.ID).Error; err != nil {
		t.Fatal(err)
	}
	if migratedLinked.Origin != models.AuctionEventOriginAuction || migratedUnlinked.Origin != models.AuctionEventOriginManual {
		t.Fatalf("origins linked=%q unlinked=%q", migratedLinked.Origin, migratedUnlinked.Origin)
	}
	for _, index := range []string{"idx_quick_access_owner_target", "idx_quick_access_owner_time", "idx_quick_access_target"} {
		if !db.Migrator().HasIndex(&models.QuickAccessPin{}, index) {
			t.Fatalf("missing index %s", index)
		}
	}
}
