package repository

import (
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupHealthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CollectionHealthSnapshot{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedHealthTestUsers(t *testing.T, db *gorm.DB) (uint, uint) {
	t.Helper()
	user1 := models.User{Username: "collector1", Email: "c1@test.com"}
	user2 := models.User{Username: "collector2", Email: "c2@test.com"}
	db.Create(&user1)
	db.Create(&user2)
	return user1.ID, user2.ID
}

func seedHealthTestCoins(t *testing.T, db *gorm.DB, userID uint, count int, wishlist, sold bool) []models.Coin {
	t.Helper()
	coins := make([]models.Coin, count)
	for i := 0; i < count; i++ {
		coins[i] = models.Coin{
			Name:       "Test Coin",
			Category:   models.CategoryRoman,
			Material:   models.MaterialSilver,
			UserID:     userID,
			IsWishlist: wishlist,
			IsSold:     sold,
		}
		if err := db.Create(&coins[i]).Error; err != nil {
			t.Fatalf("failed to create coin %d: %v", i, err)
		}
	}
	return coins
}

// --- T009: Repository tests for snapshot upsert and 30-day baseline lookup ---

func TestHealthRepository_UpsertCollectionHealthSnapshot_Insert(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	now := time.Now().Truncate(24 * time.Hour)
	snapshot := &models.CollectionHealthSnapshot{
		UserID:            userID,
		SnapshotDate:      now,
		Score:             75,
		GradeC:            3,
		EligibleCoinCount: 3,
	}

	err := repo.UpsertCollectionHealthSnapshot(snapshot)
	if err != nil {
		t.Fatalf("UpsertCollectionHealthSnapshot failed: %v", err)
	}
	if snapshot.ID == 0 {
		t.Fatal("expected snapshot ID to be set after insert")
	}

	var found models.CollectionHealthSnapshot
	db.First(&found, snapshot.ID)
	if found.Score != 75 {
		t.Errorf("expected score 75, got %d", found.Score)
	}
	if found.GradeC != 3 {
		t.Errorf("expected GradeC=3, got %d", found.GradeC)
	}
}

func TestHealthRepository_UpsertCollectionHealthSnapshot_Update(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	now := time.Now().Truncate(24 * time.Hour)
	snapshot := &models.CollectionHealthSnapshot{
		UserID:            userID,
		SnapshotDate:      now,
		Score:             60,
		GradeD:            2,
		EligibleCoinCount: 2,
	}
	repo.UpsertCollectionHealthSnapshot(snapshot)

	// Upsert same date with new values
	// Note: GORM FirstOrCreate with Assign should update on existing record
	updated := &models.CollectionHealthSnapshot{
		UserID:            userID,
		SnapshotDate:      now,
		Score:             80,
		GradeB:            2,
		GradeD:            0,
		EligibleCoinCount: 2,
	}
	err := repo.UpsertCollectionHealthSnapshot(updated)
	if err != nil {
		t.Fatalf("UpsertCollectionHealthSnapshot update failed: %v", err)
	}

	var found []models.CollectionHealthSnapshot
	db.Where("user_id = ? AND DATE(snapshot_date) = DATE(?)", userID, now).Find(&found)
	if len(found) != 1 {
		t.Fatalf("expected 1 snapshot (upsert should not create duplicate), got %d", len(found))
	}
	
	// Verify the record was updated (not created new)
	if found[0].ID != snapshot.ID {
		t.Errorf("expected same ID %d after upsert, got %d", snapshot.ID, found[0].ID)
	}
	if found[0].Score != 80 {
		t.Errorf("expected score 80, got %d", found[0].Score)
	}
	if found[0].GradeB != 2 {
		t.Errorf("expected GradeB=2, got %d", found[0].GradeB)
	}
	if found[0].GradeD != 0 {
		t.Errorf("expected GradeD=0 after update, got %d", found[0].GradeD)
	}
}

func TestHealthRepository_GetSnapshotBaseline_Found(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	date30DaysAgo := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	snapshot := &models.CollectionHealthSnapshot{
		UserID:       userID,
		SnapshotDate: date30DaysAgo,
		Score:        65,
	}
	repo.UpsertCollectionHealthSnapshot(snapshot)

	baselineDate := time.Now().AddDate(0, 0, -28)
	row, err := repo.GetSnapshotBaseline(userID, baselineDate)
	if err != nil {
		t.Fatalf("GetSnapshotBaseline failed: %v", err)
	}
	if row == nil {
		t.Fatal("expected baseline row, got nil")
	}
	if row.Score != 65 {
		t.Errorf("expected score 65, got %d", row.Score)
	}
}

func TestHealthRepository_GetSnapshotBaseline_NotFound(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	baselineDate := time.Now().AddDate(0, 0, -30)
	row, err := repo.GetSnapshotBaseline(userID, baselineDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if row != nil {
		t.Fatalf("expected nil when no snapshot exists, got %v", row)
	}
}

func TestHealthRepository_GetSnapshotBaseline_PicksClosestBefore(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	date40 := time.Now().AddDate(0, 0, -40).Truncate(24 * time.Hour)
	date30 := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	date20 := time.Now().AddDate(0, 0, -20).Truncate(24 * time.Hour)

	repo.UpsertCollectionHealthSnapshot(&models.CollectionHealthSnapshot{
		UserID: userID, SnapshotDate: date40, Score: 50,
	})
	repo.UpsertCollectionHealthSnapshot(&models.CollectionHealthSnapshot{
		UserID: userID, SnapshotDate: date30, Score: 70,
	})
	repo.UpsertCollectionHealthSnapshot(&models.CollectionHealthSnapshot{
		UserID: userID, SnapshotDate: date20, Score: 85,
	})

	baselineDate := time.Now().AddDate(0, 0, -28)
	row, err := repo.GetSnapshotBaseline(userID, baselineDate)
	if err != nil {
		t.Fatalf("GetSnapshotBaseline failed: %v", err)
	}
	if row == nil {
		t.Fatal("expected baseline row, got nil")
	}
	if row.Score != 70 {
		t.Errorf("expected score 70 (closest before -28 days), got %d", row.Score)
	}
}

// --- ListEligibleCoins tests ---

func TestHealthRepository_ListEligibleCoins_ActiveOnly(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 2, false, false) // active
	seedHealthTestCoins(t, db, userID, 1, true, false)  // wishlist
	seedHealthTestCoins(t, db, userID, 1, false, true)  // sold

	rows, err := repo.ListEligibleCoins(userID)
	if err != nil {
		t.Fatalf("ListEligibleCoins failed: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 active coins, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoins_EmptyForNoActive(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 2, true, false) // all wishlist

	rows, err := repo.ListEligibleCoins(userID)
	if err != nil {
		t.Fatalf("ListEligibleCoins failed: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 coins, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoins_UserScoped(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	user1, user2 := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, user1, 3, false, false)
	seedHealthTestCoins(t, db, user2, 2, false, false)

	rows, err := repo.ListEligibleCoins(user1)
	if err != nil {
		t.Fatalf("ListEligibleCoins failed: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 coins for user1, got %d", len(rows))
	}

	rows, err = repo.ListEligibleCoins(user2)
	if err != nil {
		t.Fatalf("ListEligibleCoins failed: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 coins for user2, got %d", len(rows))
	}
}

// --- T027: Pagination and scope filtering tests ---

func TestHealthRepository_ListEligibleCoinsPaged_DefaultPagination(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 30, false, false)

	rows, total, err := repo.ListEligibleCoinsPaged(userID, 1, 25, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if total != 30 {
		t.Errorf("expected total=30, got %d", total)
	}
	if len(rows) != 25 {
		t.Errorf("expected 25 rows on page 1, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoinsPaged_SecondPage(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 30, false, false)

	rows, total, err := repo.ListEligibleCoinsPaged(userID, 2, 25, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if total != 30 {
		t.Errorf("expected total=30, got %d", total)
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows on page 2, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoinsPaged_InvalidPageDefaultsToOne(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 10, false, false)

	rows, _, err := repo.ListEligibleCoinsPaged(userID, 0, 10, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if len(rows) != 10 {
		t.Errorf("expected 10 rows when page=0 defaults to 1, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoinsPaged_LimitBounds(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, userID, 50, false, false)

	// limit < 1 defaults to 25
	rows, _, err := repo.ListEligibleCoinsPaged(userID, 1, 0, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if len(rows) != 25 {
		t.Errorf("expected limit=0 to default to 25, got %d", len(rows))
	}

	// limit > 100 defaults to 25
	rows, _, err = repo.ListEligibleCoinsPaged(userID, 1, 150, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if len(rows) != 25 {
		t.Errorf("expected limit=150 to default to 25, got %d", len(rows))
	}
}

func TestHealthRepository_ListEligibleCoinsPaged_EmptyResult(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	userID, _ := seedHealthTestUsers(t, db)

	rows, total, err := repo.ListEligibleCoinsPaged(userID, 1, 25, "all")
	if err != nil {
		t.Fatalf("ListEligibleCoinsPaged failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

// --- ListUsersWithEligibleCoins tests ---

func TestHealthRepository_ListUsersWithEligibleCoins_MultipleUsers(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	user1, user2 := seedHealthTestUsers(t, db)
	user3 := models.User{Username: "user3", Email: "u3@test.com"}
	db.Create(&user3)

	seedHealthTestCoins(t, db, user1, 2, false, false)
	seedHealthTestCoins(t, db, user2, 1, false, false)
	seedHealthTestCoins(t, db, user3.ID, 0, false, false)

	userIDs, err := repo.ListUsersWithEligibleCoins()
	if err != nil {
		t.Fatalf("ListUsersWithEligibleCoins failed: %v", err)
	}
	if len(userIDs) != 2 {
		t.Errorf("expected 2 users with coins, got %d", len(userIDs))
	}

	found1, found2 := false, false
	for _, id := range userIDs {
		if id == user1 {
			found1 = true
		}
		if id == user2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected user1 and user2 in result")
	}
}

func TestHealthRepository_ListUsersWithEligibleCoins_ExcludesWishlistAndSold(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)
	user1, user2 := seedHealthTestUsers(t, db)

	seedHealthTestCoins(t, db, user1, 1, false, false) // active
	seedHealthTestCoins(t, db, user2, 1, true, false)  // wishlist only

	userIDs, err := repo.ListUsersWithEligibleCoins()
	if err != nil {
		t.Fatalf("ListUsersWithEligibleCoins failed: %v", err)
	}
	if len(userIDs) != 1 {
		t.Errorf("expected 1 user, got %d", len(userIDs))
	}
	if userIDs[0] != user1 {
		t.Errorf("expected user1, got %d", userIDs[0])
	}
}

func TestHealthRepository_ListUsersWithEligibleCoins_EmptyIfNoActive(t *testing.T) {
	db := setupHealthTestDB(t)
	repo := NewHealthRepository(db)

	userIDs, err := repo.ListUsersWithEligibleCoins()
	if err != nil {
		t.Fatalf("ListUsersWithEligibleCoins failed: %v", err)
	}
	if len(userIDs) != 0 {
		t.Errorf("expected 0 users, got %d", len(userIDs))
	}
}
