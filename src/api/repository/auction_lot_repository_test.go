package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuctionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.User{}, &models.AuctionLot{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestAuctionLotRepository_GetEndingSoon(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	now := time.Now()
	// Test cases across the 24-hour rolling window
	in23Hours := now.Add(23 * time.Hour)
	in12Hours := now.Add(12 * time.Hour)
	in2Hours := now.Add(2 * time.Hour)
	in25Hours := now.Add(25 * time.Hour)
	justEnded := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		lot      *models.AuctionLot
		expected bool
	}{
		{
			name: "bidding lot ending in 23 hours (near upper bound)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot1",
				Title:        "Lot 1",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &in23Hours,
				LotNumber:    1,
				UserID:       1,
			},
			expected: true,
		},
		{
			name: "bidding lot ending in 12 hours (Brian's exact case)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot2",
				Title:        "Lot 8325 - Heritage",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &in12Hours,
				LotNumber:    8325,
				UserID:       1,
			},
			expected: true,
		},
		{
			name: "bidding lot ending in 2 hours",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot3",
				Title:        "Lot 3",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &in2Hours,
				LotNumber:    3,
				UserID:       1,
			},
			expected: true,
		},
		{
			name: "bidding lot ending in 25 hours (beyond 24h window)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot4",
				Title:        "Lot 4",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &in25Hours,
				LotNumber:    4,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot that just ended (1 hour ago)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot5",
				Title:        "Lot 5",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &justEnded,
				LotNumber:    5,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "watching lot ending in 2 hours (wrong status)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot6",
				Title:        "Lot 6",
				Status:       models.AuctionStatusWatching,
				SaleDate:     &in2Hours,
				LotNumber:    6,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot with no dates",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot7",
				Title:        "Lot 7",
				Status:       models.AuctionStatusBidding,
				SaleDate:     nil,
				LotNumber:    7,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot with auction_end_time in 2 hours (no sale_date)",
			lot: &models.AuctionLot{
				NumisBidsURL:   "https://example.com/lot8",
				Title:          "Lot 8 - Heritage",
				Status:         models.AuctionStatusBidding,
				SaleDate:       nil,
				AuctionEndTime: &in2Hours,
				LotNumber:      8,
				UserID:         1,
			},
			expected: true,
		},
		{
			name: "bidding lot with UPPERCASE status (case-insensitive test)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot9",
				Title:        "Lot 9",
				Status:       "BIDDING", // Uppercase to test case-insensitive query
				SaleDate:     &in12Hours,
				LotNumber:    9,
				UserID:       1,
			},
			expected: true,
		},
		{
			name: "won lot ending in 2 hours (wrong status)",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot10",
				Title:        "Lot 10",
				Status:       models.AuctionStatusWon,
				SaleDate:     &in2Hours,
				LotNumber:    10,
				UserID:       1,
			},
			expected: false,
		},
	}

	// Create all test lots
	for _, tt := range tests {
		if err := repo.Create(tt.lot); err != nil {
			t.Fatalf("failed to create test lot %q: %v", tt.name, err)
		}
	}

	// Run the query
	lots, err := repo.GetEndingSoon()
	if err != nil {
		t.Fatalf("GetEndingSoon failed: %v", err)
	}

	// Verify only the expected lots are returned
	expectedCount := 0
	for _, tt := range tests {
		if tt.expected {
			expectedCount++
		}
	}

	if len(lots) != expectedCount {
		t.Errorf("expected %d lots, got %d", expectedCount, len(lots))
	}

	// Verify the returned lots match expectations
	foundLots := make(map[string]bool)
	for _, lot := range lots {
		foundLots[lot.NumisBidsURL] = true
	}

	for _, tt := range tests {
		found := foundLots[tt.lot.NumisBidsURL]
		if found != tt.expected {
			if tt.expected {
				t.Errorf("expected lot %q to be returned, but it wasn't", tt.name)
			} else {
				t.Errorf("lot %q should not be returned, but it was", tt.name)
			}
		}
	}
}

func TestAuctionLotRepository_GetEndingSoon_MultipleUsers(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	now := time.Now()
	inNext12Hours := now.Add(12 * time.Hour)

	// Create lots for multiple users
	lot1 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user1-lot1",
		Title:        "User 1 Lot 1",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &inNext12Hours,
		LotNumber:    1,
		UserID:       1,
	}
	lot2 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user2-lot1",
		Title:        "User 2 Lot 1",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &inNext12Hours,
		LotNumber:    2,
		UserID:       2,
	}
	lot3 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user1-lot2",
		Title:        "User 1 Lot 2",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &inNext12Hours,
		LotNumber:    3,
		UserID:       1,
	}

	if err := repo.Create(lot1); err != nil {
		t.Fatalf("failed to create lot1: %v", err)
	}
	if err := repo.Create(lot2); err != nil {
		t.Fatalf("failed to create lot2: %v", err)
	}
	if err := repo.Create(lot3); err != nil {
		t.Fatalf("failed to create lot3: %v", err)
	}

	lots, err := repo.GetEndingSoon()
	if err != nil {
		t.Fatalf("GetEndingSoon failed: %v", err)
	}

	if len(lots) != 3 {
		t.Errorf("expected 3 lots, got %d", len(lots))
	}

	// Verify lots are ordered by user_id then sale_date
	if len(lots) >= 2 {
		if lots[0].UserID > lots[1].UserID {
			t.Error("expected lots to be ordered by user_id")
		}
	}
}

func TestAuctionLotRepository_CountAllAndCountAllByStatus(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	lots := []models.AuctionLot{
		{
			NumisBidsURL: "https://example.com/admin-bidding-1",
			Title:        "Admin Bidding 1",
			Status:       models.AuctionStatusBidding,
			LotNumber:    1,
			UserID:       1,
		},
		{
			NumisBidsURL: "https://example.com/user-bidding-1",
			Title:        "User Bidding 1",
			Status:       models.AuctionStatusBidding,
			LotNumber:    2,
			UserID:       2,
		},
		{
			NumisBidsURL: "https://example.com/user-watching-1",
			Title:        "User Watching 1",
			Status:       models.AuctionStatusWatching,
			LotNumber:    3,
			UserID:       2,
		},
	}
	for i := range lots {
		if err := repo.Create(&lots[i]); err != nil {
			t.Fatalf("failed to create lot %d: %v", i, err)
		}
	}

	total, err := repo.CountAll()
	if err != nil {
		t.Fatalf("CountAll failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected CountAll=3, got %d", total)
	}

	counts, err := repo.CountAllByStatus()
	if err != nil {
		t.Fatalf("CountAllByStatus failed: %v", err)
	}
	expectedCounts := map[string]int64{
		string(models.AuctionStatusBidding):  2,
		string(models.AuctionStatusWatching): 1,
	}
	for status, expected := range expectedCounts {
		if counts[status] != expected {
			t.Errorf("expected CountAllByStatus[%q]=%d, got %d", status, expected, counts[status])
		}
	}
}

func TestAuctionLotRepository_UpsertUsesSourceURL(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	sharedURL := "https://example.com/shared-lot"
	numisLot := &models.AuctionLot{
		NumisBidsURL: sharedURL,
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    sharedURL,
		Title:        "NumisBids Lot",
		Status:       models.AuctionStatusWatching,
		LotNumber:    1,
		UserID:       1,
	}
	cngLot := &models.AuctionLot{
		NumisBidsURL: sharedURL,
		Source:       models.AuctionSourceCNG,
		SourceURL:    sharedURL,
		SourceLotID:  "4-CNGLOT",
		Title:        "CNG Lot",
		Status:       models.AuctionStatusWatching,
		LotNumber:    1,
		UserID:       1,
	}

	if err := repo.Upsert(numisLot); err != nil {
		t.Fatalf("upsert numis lot: %v", err)
	}
	if err := repo.Upsert(cngLot); err != nil {
		t.Fatalf("upsert cng lot: %v", err)
	}

	var count int64
	if err := db.Model(&models.AuctionLot{}).Count(&count).Error; err != nil {
		t.Fatalf("count lots: %v", err)
	}
	if count != 2 {
		t.Fatalf("lot count = %d, want 2 provider-specific rows", count)
	}

	cngLot.Title = "CNG Lot Updated"
	if err := repo.Upsert(cngLot); err != nil {
		t.Fatalf("upsert cng update: %v", err)
	}
	if err := db.Model(&models.AuctionLot{}).Count(&count).Error; err != nil {
		t.Fatalf("count lots after update: %v", err)
	}
	if count != 2 {
		t.Fatalf("lot count after update = %d, want 2", count)
	}

	found, err := repo.GetBySourceURL(models.AuctionSourceCNG, sharedURL, 1)
	if err != nil {
		t.Fatalf("GetBySourceURL CNG: %v", err)
	}
	if found.Title != "CNG Lot Updated" {
		t.Fatalf("CNG title = %q, want updated title", found.Title)
	}
}

func TestAuctionLotRepository_ListFiltersBySource(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	lots := []*models.AuctionLot{
		{NumisBidsURL: "https://example.com/numis", Source: models.AuctionSourceNumisBids, SourceURL: "https://example.com/numis", Title: "Numis", Status: models.AuctionStatusWatching, UserID: 1},
		{NumisBidsURL: "https://example.com/cng", Source: models.AuctionSourceCNG, SourceURL: "https://example.com/cng", Title: "CNG", Status: models.AuctionStatusWatching, UserID: 1},
		{NumisBidsURL: "https://example.com/cng-user2", Source: models.AuctionSourceCNG, SourceURL: "https://example.com/cng-user2", Title: "CNG User 2", Status: models.AuctionStatusWatching, UserID: 2},
	}
	for _, lot := range lots {
		if err := repo.Create(lot); err != nil {
			t.Fatalf("create lot %q: %v", lot.Title, err)
		}
	}

	found, total, err := repo.List(1, AuctionLotListFilters{Source: string(models.AuctionSourceCNG), Limit: 50})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 || len(found) != 1 {
		t.Fatalf("List returned total=%d len=%d, want 1/1", total, len(found))
	}
	if found[0].Source != models.AuctionSourceCNG || found[0].Title != "CNG" {
		t.Fatalf("unexpected CNG lot: %#v", found[0])
	}
}

func TestAuctionLotRepository_CountByStatusForSource(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	lots := []*models.AuctionLot{
		{NumisBidsURL: "https://example.com/numis-watching", Source: models.AuctionSourceNumisBids, SourceURL: "https://example.com/numis-watching", Title: "Numis Watching", Status: models.AuctionStatusWatching, UserID: 1},
		{NumisBidsURL: "https://example.com/cng-watching", Source: models.AuctionSourceCNG, SourceURL: "https://example.com/cng-watching", Title: "CNG Watching", Status: models.AuctionStatusWatching, UserID: 1},
		{NumisBidsURL: "https://example.com/cng-bidding", Source: models.AuctionSourceCNG, SourceURL: "https://example.com/cng-bidding", Title: "CNG Bidding", Status: models.AuctionStatusBidding, UserID: 1},
		{NumisBidsURL: "https://example.com/cng-user2", Source: models.AuctionSourceCNG, SourceURL: "https://example.com/cng-user2", Title: "CNG User 2", Status: models.AuctionStatusWatching, UserID: 2},
	}
	for _, lot := range lots {
		if err := repo.Create(lot); err != nil {
			t.Fatalf("create lot %q: %v", lot.Title, err)
		}
	}

	counts, err := repo.CountByStatusForSource(1, models.AuctionSourceCNG)
	if err != nil {
		t.Fatalf("CountByStatusForSource failed: %v", err)
	}
	if counts[string(models.AuctionStatusWatching)] != 1 {
		t.Fatalf("CNG watching count = %d, want 1", counts[string(models.AuctionStatusWatching)])
	}
	if counts[string(models.AuctionStatusBidding)] != 1 {
		t.Fatalf("CNG bidding count = %d, want 1", counts[string(models.AuctionStatusBidding)])
	}
}
