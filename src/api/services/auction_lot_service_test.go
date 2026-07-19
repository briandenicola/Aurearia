package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionLotServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuctionEvent{}, &models.AuctionLot{}, &models.Coin{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func float64Ptr(v float64) *float64 { return &v }

func TestAuctionLotService_RecommendReturnsInsufficientDataWithoutHistory(t *testing.T) {
	db := setupAuctionLotServiceDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	svc := NewAuctionLotService(auctionRepo, repository.NewCoinRepository(db))

	lot := &models.AuctionLot{
		Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-TARGET/test",
		Title: "Target lot", Category: models.CategoryRoman, Estimate: float64Ptr(500), Status: models.AuctionStatusWatching, UserID: 1,
	}
	if err := auctionRepo.Create(lot); err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	rec, err := svc.Recommend(lot.ID, 1)
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if rec.Confidence != ConfidenceInsufficientData {
		t.Fatalf("Confidence = %q, want insufficient_data", rec.Confidence)
	}
	if rec.SuggestedMaxBid != nil {
		t.Fatalf("SuggestedMaxBid = %v, want nil when there's no history", rec.SuggestedMaxBid)
	}
}

func TestAuctionLotService_RecommendReturnsInsufficientDataWithoutEstimate(t *testing.T) {
	db := setupAuctionLotServiceDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	svc := NewAuctionLotService(auctionRepo, repository.NewCoinRepository(db))

	lot := &models.AuctionLot{
		Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-NOEST/test",
		Title: "No estimate lot", Category: models.CategoryRoman, Status: models.AuctionStatusWatching, UserID: 1,
	}
	if err := auctionRepo.Create(lot); err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	rec, err := svc.Recommend(lot.ID, 1)
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if rec.Confidence != ConfidenceInsufficientData {
		t.Fatalf("Confidence = %q, want insufficient_data", rec.Confidence)
	}
}

func TestAuctionLotService_RecommendUsesOwnWonAndLostHistory(t *testing.T) {
	db := setupAuctionLotServiceDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	svc := NewAuctionLotService(auctionRepo, repository.NewCoinRepository(db))

	// Two won lots at 150% of estimate, one lost lot where the winning bid (captured as
	// CurrentBid at close) was 200% of estimate — average ratio should land at 166.67%.
	won1 := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-WON1/test", Title: "Won 1", Category: models.CategoryRoman, Estimate: float64Ptr(100), WinningBid: float64Ptr(150), Status: models.AuctionStatusWon, UserID: 1}
	won2 := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-WON2/test", Title: "Won 2", Category: models.CategoryRoman, Estimate: float64Ptr(200), WinningBid: float64Ptr(300), Status: models.AuctionStatusWon, UserID: 1}
	lost1 := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-LOST1/test", Title: "Lost 1", Category: models.CategoryRoman, Estimate: float64Ptr(100), CurrentBid: float64Ptr(200), Status: models.AuctionStatusLost, UserID: 1}
	// Different category — must not be included in the average.
	otherCategory := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-GREEK/test", Title: "Greek won", Category: models.CategoryGreek, Estimate: float64Ptr(100), WinningBid: float64Ptr(1000), Status: models.AuctionStatusWon, UserID: 1}
	target := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-TARGET/test", Title: "Target lot", Category: models.CategoryRoman, Estimate: float64Ptr(500), Status: models.AuctionStatusWatching, UserID: 1}

	for _, lot := range []*models.AuctionLot{won1, won2, lost1, otherCategory, target} {
		if err := auctionRepo.Create(lot); err != nil {
			t.Fatalf("failed to create lot %s: %v", lot.Title, err)
		}
	}

	rec, err := svc.Recommend(target.ID, 1)
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if rec.Confidence != ConfidenceLow {
		t.Fatalf("Confidence = %q, want low (3 comparable lots)", rec.Confidence)
	}
	if rec.SampleSize != 3 {
		t.Fatalf("SampleSize = %d, want 3 (excludes the different-category lot)", rec.SampleSize)
	}
	if rec.SuggestedMaxBid == nil {
		t.Fatal("SuggestedMaxBid was nil, want a value")
	}
	// avg ratio = (1.5 + 1.5 + 2.0) / 3 = 1.6667; applied to estimate 500 = 833.33
	want := 833.3333333333334
	if diff := *rec.SuggestedMaxBid - want; diff > 0.01 || diff < -0.01 {
		t.Fatalf("SuggestedMaxBid = %v, want ~%v", *rec.SuggestedMaxBid, want)
	}
}

func TestAuctionLotService_RecommendExcludesTargetLotFromItsOwnHistory(t *testing.T) {
	db := setupAuctionLotServiceDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	svc := NewAuctionLotService(auctionRepo, repository.NewCoinRepository(db))

	// A lot that is itself already "lost" and being asked about again should not count
	// itself as a comparable — otherwise a single resolved lot could recommend against itself.
	target := &models.AuctionLot{Source: models.AuctionSourceCNG, SourceURL: "https://auctions.cngcoins.com/lots/view/4-SELF/test", Title: "Self", Category: models.CategoryRoman, Estimate: float64Ptr(100), CurrentBid: float64Ptr(200), Status: models.AuctionStatusLost, UserID: 1}
	if err := auctionRepo.Create(target); err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	rec, err := svc.Recommend(target.ID, 1)
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if rec.Confidence != ConfidenceInsufficientData {
		t.Fatalf("Confidence = %q, want insufficient_data (the lot must not count itself)", rec.Confidence)
	}
}

func TestAuctionLotService_UpdateStatusTagsManualOverrideSource(t *testing.T) {
	db := setupAuctionLotServiceDB(t)
	auctionRepo := repository.NewAuctionLotRepository(db)
	svc := NewAuctionLotService(auctionRepo, repository.NewCoinRepository(db))

	lot := &models.AuctionLot{
		Source: models.AuctionSourceNumisBids, SourceURL: "https://www.numisbids.com/sale/1/lot/1",
		Title: "Manual override target", Status: models.AuctionStatusWatching,
		StatusSource: models.AuctionLotStatusSourceSync, UserID: 1,
	}
	if err := auctionRepo.Create(lot); err != nil {
		t.Fatalf("failed to create lot: %v", err)
	}

	if err := svc.UpdateStatus(lot.ID, 1, models.AuctionStatusWon); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	found, err := auctionRepo.GetByID(lot.ID, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.Status != models.AuctionStatusWon {
		t.Fatalf("Status = %q, want won", found.Status)
	}
	if found.StatusSource != models.AuctionLotStatusSourceManual {
		t.Fatalf("StatusSource = %q, want manual (an explicit override must not read as sync-detected)", found.StatusSource)
	}
}
