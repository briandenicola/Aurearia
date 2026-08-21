package services

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var valuationTestCounter uint64

func setupValuationSourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := fmt.Sprintf("file:val_source_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&valuationTestCounter, 1))
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.CatalogRegistry{}, &models.AppSetting{}, &models.MintLocation{},
		&models.StorageLocation{}, &models.ValueSnapshot{}, &models.CoinJournal{},
		&models.CoinValueHistory{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestValuationService_UpdateCoinValuation_WritesAIScheduledSource verifies D1+D2:
// the scheduled valuation writer records Source="ai_scheduled" in value history
// and writes NO journal entry.
func TestValuationService_UpdateCoinValuation_WritesAIScheduledSource(t *testing.T) {
	db := setupValuationSourceTestDB(t)
	coinRepo := repository.NewCoinRepository(db)
	svc := &ValuationService{coinRepo: coinRepo, logger: NewLogger(100)}

	coin := &models.Coin{
		Name:     "Caracalla denarius",
		Ruler:    "Caracalla",
		Category: models.CategoryRoman,
		UserID:   1,
	}
	if err := db.Create(coin).Error; err != nil {
		t.Fatalf("failed to create coin: %v", err)
	}

	estimate := &ValueEstimate{EstimatedValue: 350.0, Confidence: "high"}
	svc.updateCoinValuation(coin, 1, estimate)

	var history []models.CoinValueHistory
	if err := db.Where("coin_id = ?", coin.ID).Find(&history).Error; err != nil {
		t.Fatalf("failed to query value history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 value history row, got %d", len(history))
	}
	if history[0].Source != models.ValueHistorySourceAIScheduled {
		t.Errorf("expected Source %q, got %q", models.ValueHistorySourceAIScheduled, history[0].Source)
	}
	if history[0].Confidence != "high" {
		t.Errorf("expected Confidence 'high', got %q", history[0].Confidence)
	}
	if history[0].Value != 350.0 {
		t.Errorf("expected Value 350.0, got %f", history[0].Value)
	}

	var journalCount int64
	if err := db.Model(&models.CoinJournal{}).Where("coin_id = ?", coin.ID).Count(&journalCount).Error; err != nil {
		t.Fatalf("failed to count journal entries: %v", err)
	}
	if journalCount != 0 {
		t.Errorf("expected 0 journal entries for scheduled valuation (D1), got %d", journalCount)
	}
}
