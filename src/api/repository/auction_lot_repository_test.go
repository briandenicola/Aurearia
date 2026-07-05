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
	err = db.AutoMigrate(&models.User{}, &models.AuctionEvent{}, &models.AuctionLot{})
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

func TestAuctionLotRepository_GetActiveWatchBidDigestLots(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	now := time.Now()
	in12Hours := now.Add(12 * time.Hour)
	in48Hours := now.Add(48 * time.Hour)
	ended := now.Add(-1 * time.Hour)
	bid := 300.0

	lots := []models.AuctionLot{
		{
			NumisBidsURL: "https://example.com/watching-future",
			Title:        "Watching Future",
			Status:       models.AuctionStatusWatching,
			SaleDate:     &in12Hours,
			CurrentBid:   &bid,
			LotNumber:    1,
			UserID:       1,
		},
		{
			NumisBidsURL:   "https://example.com/bidding-future",
			Title:          "Bidding Future",
			Status:         models.AuctionStatusBidding,
			AuctionEndTime: &in48Hours,
			CurrentBid:     &bid,
			LotNumber:      2,
			UserID:         1,
		},
		{
			NumisBidsURL: "https://example.com/ended",
			Title:        "Ended",
			Status:       models.AuctionStatusWatching,
			SaleDate:     &ended,
			CurrentBid:   &bid,
			LotNumber:    3,
			UserID:       1,
		},
		{
			NumisBidsURL: "https://example.com/passed",
			Title:        "Passed",
			Status:       models.AuctionStatusPassed,
			SaleDate:     &in12Hours,
			CurrentBid:   &bid,
			LotNumber:    4,
			UserID:       1,
		},
		{
			NumisBidsURL: "https://example.com/no-date",
			Title:        "No Date",
			Status:       models.AuctionStatusWatching,
			CurrentBid:   &bid,
			LotNumber:    5,
			UserID:       1,
		},
	}
	for i := range lots {
		if err := repo.Create(&lots[i]); err != nil {
			t.Fatalf("create lot %d: %v", i, err)
		}
	}

	got, err := repo.GetActiveWatchBidDigestLots()
	if err != nil {
		t.Fatalf("GetActiveWatchBidDigestLots: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d lots, want 2: %#v", len(got), got)
	}
	found := map[string]bool{}
	for _, lot := range got {
		found[lot.NumisBidsURL] = true
	}
	for _, url := range []string{"https://example.com/watching-future", "https://example.com/bidding-future"} {
		if !found[url] {
			t.Fatalf("expected active digest lot %s", url)
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

	if _, err := repo.Upsert(numisLot); err != nil {
		t.Fatalf("upsert numis lot: %v", err)
	}
	if _, err := repo.Upsert(cngLot); err != nil {
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
	if _, err := repo.Upsert(cngLot); err != nil {
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

func TestAuctionLotRepository_UpsertRefreshesCurrentBid(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)

	originalBid := 125.0
	lot := &models.AuctionLot{
		NumisBidsURL: "https://example.com/bid-refresh",
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    "https://example.com/bid-refresh",
		Title:        "Tracked Lot",
		Status:       models.AuctionStatusWatching,
		CurrentBid:   &originalBid,
		LotNumber:    30,
		UserID:       1,
	}
	if _, err := repo.Upsert(lot); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	refreshedBid := 300.0
	lot.CurrentBid = &refreshedBid
	if _, err := repo.Upsert(lot); err != nil {
		t.Fatalf("refresh upsert: %v", err)
	}

	found, err := repo.GetBySourceURL(models.AuctionSourceNumisBids, lot.SourceURL, 1)
	if err != nil {
		t.Fatalf("GetBySourceURL: %v", err)
	}
	if found.CurrentBid == nil || *found.CurrentBid != refreshedBid {
		t.Fatalf("CurrentBid = %v, want %v", found.CurrentBid, refreshedBid)
	}
}

func TestAuctionLotRepository_UpsertWithCalendarEventCreatesOnlyForNewWatchableLots(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)
	endTime := time.Date(2026, 7, 2, 15, 30, 0, 0, time.UTC)
	estimate := 250.0
	lot := &models.AuctionLot{
		NumisBidsURL:   "https://www.numisbids.com/n.php?p=lot&sid=1&lot=100",
		Source:         models.AuctionSourceNumisBids,
		SourceURL:      "https://www.numisbids.com/n.php?p=lot&sid=1&lot=100",
		SourceSaleID:   "sale-1",
		Title:          "Aurelian Antoninianus",
		AuctionHouse:   "Numis House",
		SaleName:       "Summer Sale",
		LotNumber:      100,
		AuctionEndTime: &endTime,
		Estimate:       &estimate,
		Status:         models.AuctionStatusWatching,
		UserID:         7,
	}

	result, err := repo.UpsertWithCalendarEvent(lot)
	if err != nil {
		t.Fatalf("upsert with event: %v", err)
	}
	if !result.Created || !result.EventCreated || result.EventID == nil {
		t.Fatalf("result = %#v, want created lot and event", result)
	}

	upserted, err := repo.GetBySourceURL(models.AuctionSourceNumisBids, lot.SourceURL, 7)
	if err != nil {
		t.Fatalf("reload upserted lot: %v", err)
	}
	if upserted.EventID == nil || *upserted.EventID != *result.EventID {
		t.Fatalf("lot event id = %v, want %d", upserted.EventID, *result.EventID)
	}

	var event models.AuctionEvent
	if err := db.First(&event, *result.EventID).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if event.UserID != 7 || event.Title != "Summer Sale" || event.AuctionHouse != "Numis House" {
		t.Fatalf("unexpected event fields: %#v", event)
	}
	if event.StartDate == nil || !event.StartDate.Equal(endTime) || event.EndDate == nil || !event.EndDate.Equal(endTime) {
		t.Fatalf("event dates = %v/%v, want %v", event.StartDate, event.EndDate, endTime)
	}
	if event.URL != lot.SourceURL {
		t.Fatalf("event url = %q, want %q", event.URL, lot.SourceURL)
	}

	lot.Title = "Updated title"
	second, err := repo.UpsertWithCalendarEvent(lot)
	if err != nil {
		t.Fatalf("second upsert with event: %v", err)
	}
	if second.Created || second.EventCreated || second.EventID != nil {
		t.Fatalf("second result = %#v, want update without duplicate event", second)
	}

	var eventCount int64
	if err := db.Model(&models.AuctionEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("event count = %d, want 1", eventCount)
	}
}

func TestAuctionLotRepository_UpsertWithCalendarEventSkipsPassedAndExistingLots(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)
	closedLots := []*models.AuctionLot{
		{
			NumisBidsURL: "https://www.numisbids.com/passed",
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    "https://www.numisbids.com/passed",
			Title:        "Passed lot",
			Status:       models.AuctionStatusPassed,
			UserID:       8,
		},
		{
			NumisBidsURL: "https://www.numisbids.com/won",
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    "https://www.numisbids.com/won",
			Title:        "Won lot",
			Status:       models.AuctionStatusWon,
			UserID:       8,
		},
		{
			NumisBidsURL: "https://www.numisbids.com/lost",
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    "https://www.numisbids.com/lost",
			Title:        "Lost lot",
			Status:       models.AuctionStatusLost,
			UserID:       8,
		},
	}
	for _, lot := range closedLots {
		if result, err := repo.UpsertWithCalendarEvent(lot); err != nil {
			t.Fatalf("upsert %s lot: %v", lot.Status, err)
		} else if !result.Created || result.EventCreated {
			t.Fatalf("%s result = %#v, want created lot without event", lot.Status, result)
		}
	}

	existing := &models.AuctionLot{
		NumisBidsURL: "https://www.numisbids.com/existing",
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    "https://www.numisbids.com/existing",
		Title:        "Already tracked",
		Status:       models.AuctionStatusWatching,
		UserID:       8,
	}
	if err := repo.Create(existing); err != nil {
		t.Fatalf("create existing lot: %v", err)
	}
	existing.Title = "Already tracked updated"
	if result, err := repo.UpsertWithCalendarEvent(existing); err != nil {
		t.Fatalf("upsert existing lot: %v", err)
	} else if result.Created || result.EventCreated {
		t.Fatalf("existing result = %#v, want update without event", result)
	}

	linkedEvent := models.AuctionEvent{
		UserID: 8,
		Title:  "Manually linked event",
	}
	if err := db.Create(&linkedEvent).Error; err != nil {
		t.Fatalf("create linked event: %v", err)
	}
	existingWithEvent := &models.AuctionLot{
		NumisBidsURL: "https://www.numisbids.com/existing-linked",
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    "https://www.numisbids.com/existing-linked",
		Title:        "Already tracked and linked",
		Status:       models.AuctionStatusWatching,
		EventID:      &linkedEvent.ID,
		UserID:       8,
	}
	if err := repo.Create(existingWithEvent); err != nil {
		t.Fatalf("create existing linked lot: %v", err)
	}
	existingWithEvent.Title = "Already tracked and linked updated"
	if result, err := repo.UpsertWithCalendarEvent(existingWithEvent); err != nil {
		t.Fatalf("upsert existing linked lot: %v", err)
	} else if result.Created || result.EventCreated {
		t.Fatalf("existing linked result = %#v, want update without event", result)
	}
	reloadedLinked, err := repo.GetBySourceURL(models.AuctionSourceNumisBids, existingWithEvent.SourceURL, 8)
	if err != nil {
		t.Fatalf("reload existing linked lot: %v", err)
	}
	if reloadedLinked.EventID == nil || *reloadedLinked.EventID != linkedEvent.ID {
		t.Fatalf("existing linked event id = %v, want %d", reloadedLinked.EventID, linkedEvent.ID)
	}

	var eventCount int64
	if err := db.Model(&models.AuctionEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("event count = %d, want 1 existing manual event only", eventCount)
	}
}

func TestAuctionLotRepository_UpsertWithCalendarEventReusesSaleNameEvent(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)
	firstLot := &models.AuctionLot{
		NumisBidsURL: "https://www.numisbids.com/n.php?p=lot&sid=11&lot=1",
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    "https://www.numisbids.com/n.php?p=lot&sid=11&lot=1",
		SourceSaleID: "sale-11",
		Title:        "First lot title",
		SaleName:     "Numis Sale 11",
		Status:       models.AuctionStatusWatching,
		UserID:       10,
	}
	firstResult, err := repo.UpsertWithCalendarEvent(firstLot)
	if err != nil {
		t.Fatalf("upsert first lot: %v", err)
	}
	if !firstResult.Created || !firstResult.EventCreated || firstResult.EventID == nil {
		t.Fatalf("first result = %#v, want created lot and event", firstResult)
	}

	secondLot := &models.AuctionLot{
		NumisBidsURL: "https://www.numisbids.com/n.php?p=lot&sid=11&lot=2",
		Source:       models.AuctionSourceNumisBids,
		SourceURL:    "https://www.numisbids.com/n.php?p=lot&sid=11&lot=2",
		SourceSaleID: "sale-11",
		Title:        "Second lot title",
		SaleName:     "Numis Sale 11",
		Status:       models.AuctionStatusBidding,
		UserID:       10,
	}
	secondResult, err := repo.UpsertWithCalendarEvent(secondLot)
	if err != nil {
		t.Fatalf("upsert second lot: %v", err)
	}
	if !secondResult.Created || secondResult.EventCreated || secondResult.EventID == nil {
		t.Fatalf("second result = %#v, want created lot linked to existing event", secondResult)
	}
	if *secondResult.EventID != *firstResult.EventID {
		t.Fatalf("second event id = %d, want %d", *secondResult.EventID, *firstResult.EventID)
	}

	reloadedSecond, err := repo.GetBySourceURL(models.AuctionSourceNumisBids, secondLot.SourceURL, 10)
	if err != nil {
		t.Fatalf("reload second lot: %v", err)
	}
	if reloadedSecond.EventID == nil || *reloadedSecond.EventID != *firstResult.EventID {
		t.Fatalf("second lot event id = %v, want %d", reloadedSecond.EventID, *firstResult.EventID)
	}

	var eventCount int64
	if err := db.Model(&models.AuctionEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("event count = %d, want 1 shared sale event", eventCount)
	}
}

func TestAuctionLotRepository_UpsertWithCalendarEventIsSourceAwareForCNG(t *testing.T) {
	db := setupAuctionTestDB(t)
	repo := NewAuctionLotRepository(db)
	sharedURL := "https://example.com/shared-lot"
	cngEndTime := time.Date(2026, 7, 3, 18, 0, 0, 0, time.UTC)
	lots := []*models.AuctionLot{
		{
			NumisBidsURL: sharedURL,
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    sharedURL,
			Title:        "NumisBids Shared Lot",
			Status:       models.AuctionStatusWatching,
			UserID:       9,
		},
		{
			NumisBidsURL:   sharedURL,
			Source:         models.AuctionSourceCNG,
			SourceURL:      sharedURL,
			SourceLotID:    "4-CNGLOT",
			Title:          "CNG Shared Lot",
			Status:         models.AuctionStatusBidding,
			AuctionEndTime: &cngEndTime,
			UserID:         9,
		},
	}

	for _, lot := range lots {
		result, err := repo.UpsertWithCalendarEvent(lot)
		if err != nil {
			t.Fatalf("upsert %s lot: %v", lot.Source, err)
		}
		if !result.Created || !result.EventCreated {
			t.Fatalf("%s result = %#v, want provider-specific event", lot.Source, result)
		}
	}

	var lotCount int64
	if err := db.Model(&models.AuctionLot{}).Count(&lotCount).Error; err != nil {
		t.Fatalf("count lots: %v", err)
	}
	if lotCount != 2 {
		t.Fatalf("lot count = %d, want 2 provider-specific rows", lotCount)
	}
	var eventCount int64
	if err := db.Model(&models.AuctionEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 2 {
		t.Fatalf("event count = %d, want 2 source-aware events", eventCount)
	}

	cngLot, err := repo.GetBySourceURL(models.AuctionSourceCNG, sharedURL, 9)
	if err != nil {
		t.Fatalf("reload CNG lot: %v", err)
	}
	if cngLot.EventID == nil {
		t.Fatalf("CNG lot EventID is nil, want linked event")
	}
	var cngEvent models.AuctionEvent
	if err := db.First(&cngEvent, *cngLot.EventID).Error; err != nil {
		t.Fatalf("reload CNG event: %v", err)
	}
	if cngEvent.StartDate == nil || !cngEvent.StartDate.Equal(cngEndTime) {
		t.Fatalf("CNG event start date = %v, want %v", cngEvent.StartDate, cngEndTime)
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
