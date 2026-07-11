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

func TestSetRepository_CriteriaTemplate_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	criteria := models.JSONObject{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Silver"},
		},
	}

	tmpl := &models.SmartCriteriaTemplate{
		UserID:      1,
		Name:        "My Silver Rules",
		Description: "Silver coins filter",
		Criteria:    criteria,
	}

	// Create
	if err := repo.CreateCriteriaTemplate(tmpl); err != nil {
		t.Fatalf("CreateCriteriaTemplate failed: %v", err)
	}
	if tmpl.ID == 0 {
		t.Fatal("expected template ID to be set")
	}

	// List
	list, err := repo.ListCriteriaTemplates(1)
	if err != nil {
		t.Fatalf("ListCriteriaTemplates failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 template, got %d", len(list))
	}
	if list[0].Name != "My Silver Rules" {
		t.Fatalf("expected name %q, got %q", "My Silver Rules", list[0].Name)
	}

	// GetCriteriaTemplate
	got, err := repo.GetCriteriaTemplate(tmpl.ID, 1)
	if err != nil {
		t.Fatalf("GetCriteriaTemplate failed: %v", err)
	}
	if got.Name != tmpl.Name {
		t.Fatalf("expected name %q, got %q", tmpl.Name, got.Name)
	}

	// Delete
	if err := repo.DeleteCriteriaTemplate(tmpl.ID, 1); err != nil {
		t.Fatalf("DeleteCriteriaTemplate failed: %v", err)
	}

	// Confirm gone
	list, err = repo.ListCriteriaTemplates(1)
	if err != nil {
		t.Fatalf("ListCriteriaTemplates after delete failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 templates after delete, got %d", len(list))
	}
}

func TestSetRepository_CriteriaTemplate_UserScoped(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	criteria := models.JSONObject{"operator": "and", "rules": []interface{}{
		map[string]interface{}{"field": "material", "op": "eq", "value": "Gold"},
	}}

	user1Tmpl := &models.SmartCriteriaTemplate{UserID: 1, Name: "User1 Template", Criteria: criteria}
	user2Tmpl := &models.SmartCriteriaTemplate{UserID: 2, Name: "User2 Template", Criteria: criteria}

	if err := repo.CreateCriteriaTemplate(user1Tmpl); err != nil {
		t.Fatalf("create user1 template: %v", err)
	}
	if err := repo.CreateCriteriaTemplate(user2Tmpl); err != nil {
		t.Fatalf("create user2 template: %v", err)
	}

	// User 1 should only see their own template
	list, err := repo.ListCriteriaTemplates(1)
	if err != nil {
		t.Fatalf("list templates for user1: %v", err)
	}
	if len(list) != 1 || list[0].UserID != 1 {
		t.Fatalf("user1 should see exactly 1 template, got %d", len(list))
	}

	// User 2 cannot delete user 1's template
	if err := repo.DeleteCriteriaTemplate(user1Tmpl.ID, 2); err == nil {
		t.Fatal("expected error when deleting another user's template, got nil")
	}
}
