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

func TestAuctionLotRepository_GetEndingToday(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	yesterday := today.Add(-24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	tests := []struct {
		name     string
		lot      *models.AuctionLot
		expected bool
	}{
		{
			name: "bidding lot ending today",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot1",
				Title:        "Lot 1",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &today,
				LotNumber:    1,
				UserID:       1,
			},
			expected: true,
		},
		{
			name: "watching lot ending today",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot2",
				Title:        "Lot 2",
				Status:       models.AuctionStatusWatching,
				SaleDate:     &today,
				LotNumber:    2,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot ending yesterday",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot3",
				Title:        "Lot 3",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &yesterday,
				LotNumber:    3,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot ending tomorrow",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot4",
				Title:        "Lot 4",
				Status:       models.AuctionStatusBidding,
				SaleDate:     &tomorrow,
				LotNumber:    4,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot with no sale date",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot5",
				Title:        "Lot 5",
				Status:       models.AuctionStatusBidding,
				SaleDate:     nil,
				LotNumber:    5,
				UserID:       1,
			},
			expected: false,
		},
		{
			name: "bidding lot with auction_end_time today (no sale_date)",
			lot: &models.AuctionLot{
				NumisBidsURL:   "https://example.com/lot5b",
				Title:          "Lot 5b - Heritage",
				Status:         models.AuctionStatusBidding,
				SaleDate:       nil,
				AuctionEndTime: &today,
				LotNumber:      99,
				UserID:         1,
			},
			expected: true,
		},
		{
			name: "won lot ending today",
			lot: &models.AuctionLot{
				NumisBidsURL: "https://example.com/lot6",
				Title:        "Lot 6",
				Status:       models.AuctionStatusWon,
				SaleDate:     &today,
				LotNumber:    6,
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
	lots, err := repo.GetEndingToday()
	if err != nil {
		t.Fatalf("GetEndingToday failed: %v", err)
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

func TestAuctionLotRepository_GetEndingToday_MultipleUsers(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())

	// Create lots for multiple users
	lot1 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user1-lot1",
		Title:        "User 1 Lot 1",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &today,
		LotNumber:    1,
		UserID:       1,
	}
	lot2 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user2-lot1",
		Title:        "User 2 Lot 1",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &today,
		LotNumber:    2,
		UserID:       2,
	}
	lot3 := &models.AuctionLot{
		NumisBidsURL: "https://example.com/user1-lot2",
		Title:        "User 1 Lot 2",
		Status:       models.AuctionStatusBidding,
		SaleDate:     &today,
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

	lots, err := repo.GetEndingToday()
	if err != nil {
		t.Fatalf("GetEndingToday failed: %v", err)
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
