package repository

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestSetRepository_GetCoinsInSet_UsesManualSortOrder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeOpen}
	coins := []models.Coin{
		{Name: "Trajan", UserID: 1},
		{Name: "Augustus", UserID: 1},
		{Name: "Hadrian", UserID: 1},
	}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	addedAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	memberships := []models.CoinSetMembership{
		{SetID: set.ID, CoinID: coins[0].ID, AddedAt: addedAt, SortOrder: 2},
		{SetID: set.ID, CoinID: coins[1].ID, AddedAt: addedAt, SortOrder: 0},
		{SetID: set.ID, CoinID: coins[2].ID, AddedAt: addedAt, SortOrder: 1},
	}
	if err := db.Create(&memberships).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}

	got, err := repo.GetCoinsInSet(set.ID, 1)
	if err != nil {
		t.Fatalf("GetCoinsInSet failed: %v", err)
	}
	names := coinNames(got)
	want := []string{"Augustus", "Hadrian", "Trajan"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected order %v, got %v", want, names)
	}
}

func TestSetRepository_GetCoinsInSet_DefaultSortOrderFallsBackToName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeOpen}
	coins := []models.Coin{
		{Name: "Trajan", UserID: 1},
		{Name: "Augustus", UserID: 1},
		{Name: "Hadrian", UserID: 1},
	}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	addedAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	memberships := []models.CoinSetMembership{
		{SetID: set.ID, CoinID: coins[0].ID, AddedAt: addedAt},
		{SetID: set.ID, CoinID: coins[1].ID, AddedAt: addedAt},
		{SetID: set.ID, CoinID: coins[2].ID, AddedAt: addedAt},
	}
	if err := db.Create(&memberships).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}

	got, err := repo.GetCoinsInSet(set.ID, 1)
	if err != nil {
		t.Fatalf("GetCoinsInSet failed: %v", err)
	}
	names := coinNames(got)
	want := []string{"Augustus", "Hadrian", "Trajan"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected fallback order %v, got %v", want, names)
	}
}

func TestSetRepository_ReorderCoinsInSet_RejectsInvalidMembersWithoutPartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeOpen}
	memberA := models.Coin{Name: "Augustus", UserID: 1}
	memberB := models.Coin{Name: "Trajan", UserID: 1}
	nonMember := models.Coin{Name: "Nero", UserID: 1}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	if err := db.Create(&[]*models.Coin{&memberA, &memberB, &nonMember}).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}

	addedAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	memberships := []models.CoinSetMembership{
		{SetID: set.ID, CoinID: memberA.ID, AddedAt: addedAt, SortOrder: 0},
		{SetID: set.ID, CoinID: memberB.ID, AddedAt: addedAt, SortOrder: 1},
	}
	if err := db.Create(&memberships).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}

	err := repo.ReorderCoinsInSet(set.ID, 1, []uint{memberB.ID, nonMember.ID})
	if !errors.Is(err, ErrInvalidSetOrder) {
		t.Fatalf("expected ErrInvalidSetOrder, got %v", err)
	}

	got, err := repo.GetCoinsInSet(set.ID, 1)
	if err != nil {
		t.Fatalf("GetCoinsInSet failed: %v", err)
	}
	names := coinNames(got)
	want := []string{"Augustus", "Trajan"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("order changed after rejected reorder: want %v, got %v", want, names)
	}
}

func coinNames(coins []models.Coin) []string {
	names := make([]string, 0, len(coins))
	for _, coin := range coins {
		names = append(names, coin.Name)
	}
	return names
}
