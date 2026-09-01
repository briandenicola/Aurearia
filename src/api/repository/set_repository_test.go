package repository

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestSetRepository_GetCoinsInSet_UsesManualSortOrder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeStandard}
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

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeStandard}
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

	set := models.CoinSet{UserID: 1, Name: "Emperors", SetType: models.CoinSetTypeStandard}
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

func TestSetRepository_GetAgenticSetCompletionRequiresExplicitAssignments(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "US Silver Quarters", SetType: models.CoinSetTypeAgentic, CreationMode: models.CoinSetCreationModeDynamic}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	coins := []models.Coin{
		{Name: "1940 Washington Quarter A", UserID: 1, Denomination: "Quarter", Material: "Silver"},
		{Name: "1940 Washington Quarter B", UserID: 1, Denomination: "Quarter", Material: "Silver"},
		{Name: "1941 Washington Quarter", UserID: 1, Denomination: "Quarter", Material: "Silver", IsWishlist: true},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}
	year1940 := 1940
	year1941 := 1941
	denomination := "Quarter"
	material := "Silver"
	targets := []models.CoinSetTarget{
		{SetID: set.ID, Label: "1940 US Silver Quarter", Year: &year1940, Denomination: &denomination, Material: &material, SortOrder: 1},
		{SetID: set.ID, Label: "Duplicate 1940 Slot", Year: &year1940, Denomination: &denomination, Material: &material, SortOrder: 2},
		{SetID: set.ID, Label: "1941 US Silver Quarter", Year: &year1941, Denomination: &denomination, Material: &material, SortOrder: 3},
	}
	if err := db.Create(&targets).Error; err != nil {
		t.Fatalf("create targets: %v", err)
	}

	completion, err := repo.GetSetCompletion(set.ID, 1)
	if err != nil {
		t.Fatalf("GetSetCompletion: %v", err)
	}
	if completion["completedTargets"] != 0 {
		t.Fatalf("expected no implicit auto-matching for agentic sets, got %#v", completion)
	}
	missing, ok := completion["missingTargets"].([]models.CoinSetTarget)
	if !ok || len(missing) != 3 {
		t.Fatalf("expected all unassigned targets to be missing, missing=%#v", completion["missingTargets"])
	}
	matches, ok := completion["targetMatches"].([]AgenticTargetMatch)
	if !ok || len(matches) != 3 {
		t.Fatalf("expected target matches in completion, got %#v", completion["targetMatches"])
	}
	if matches[0].Coin != nil || matches[1].Coin != nil || matches[2].Coin != nil {
		t.Fatalf("expected no auto-assigned coins in agentic completion, got %#v", matches)
	}
}

func TestSetRepository_AddCoinToSet_AssignsAgenticSlots(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Byzantine Emperors", SetType: models.CoinSetTypeAgentic, CreationMode: models.CoinSetCreationModeDynamic}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	coins := []models.Coin{
		{Name: "Anastasius Solidus", UserID: 1, Denomination: "Solidus", Material: "Gold"},
		{Name: "Justin I Solidus", UserID: 1, Denomination: "Solidus", Material: "Gold"},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}
	targets := []models.CoinSetTarget{
		{SetID: set.ID, Label: "Anastasius I", SortOrder: 1},
		{SetID: set.ID, Label: "Justin I", SortOrder: 2},
	}
	if err := db.Create(&targets).Error; err != nil {
		t.Fatalf("create targets: %v", err)
	}

	if err := repo.AddCoinToSet(coins[0].ID, set.ID, 1, "", targets[0].ID); err != nil {
		t.Fatalf("assign first slot: %v", err)
	}
	if err := repo.AddCoinToSet(coins[0].ID, set.ID, 1, "", targets[1].ID); err != nil {
		t.Fatalf("move coin to second slot: %v", err)
	}
	if err := repo.AddCoinToSet(coins[1].ID, set.ID, 1, "", targets[1].ID); err != nil {
		t.Fatalf("replace second slot coin: %v", err)
	}

	completion, err := repo.GetSetCompletion(set.ID, 1)
	if err != nil {
		t.Fatalf("GetSetCompletion: %v", err)
	}
	if got := completion["completedTargets"].(int); got != 1 {
		t.Fatalf("expected one completed target after replacement, got %d", got)
	}
	matches := completion["targetMatches"].([]AgenticTargetMatch)
	if matches[0].Coin != nil {
		t.Fatalf("first slot should be empty after coin reassignment, got %#v", matches[0].Coin)
	}
	if matches[1].Coin == nil || matches[1].Coin.ID != coins[1].ID {
		t.Fatalf("second slot should contain replacement coin, got %#v", matches[1].Coin)
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

func TestSetRepository_GetSetCompletion_GoalUsesCollectionVsWishlistMembers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Goal Set", SetType: models.CoinSetTypeGoal}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}

	coins := []models.Coin{
		{Name: "Owned 1", UserID: 1, IsWishlist: false},
		{Name: "Owned 2", UserID: 1, IsWishlist: false},
		{Name: "Wish 1", UserID: 1, IsWishlist: true},
		{Name: "Wish 2", UserID: 1, IsWishlist: true},
		{Name: "Wish 3", UserID: 1, IsWishlist: true},
		{Name: "Wish 4", UserID: 1, IsWishlist: true},
		{Name: "Wish 5", UserID: 1, IsWishlist: true},
	}
	if err := db.Create(&coins).Error; err != nil {
		t.Fatalf("create coins: %v", err)
	}
	for i := range coins {
		if err := db.Create(&models.CoinSetMembership{
			SetID:     set.ID,
			CoinID:    coins[i].ID,
			AddedAt:   time.Now(),
			SortOrder: i,
		}).Error; err != nil {
			t.Fatalf("create membership: %v", err)
		}
	}

	completion, err := repo.GetSetCompletion(set.ID, 1)
	if err != nil {
		t.Fatalf("GetSetCompletion failed: %v", err)
	}

	if got := completion["totalTargets"].(int); got != 7 {
		t.Fatalf("expected totalTargets=7, got %d", got)
	}
	if got := completion["completedTargets"].(int); got != 2 {
		t.Fatalf("expected completedTargets=2, got %d", got)
	}
	if got := completion["collectionItems"].(int); got != 2 {
		t.Fatalf("expected collectionItems=2, got %d", got)
	}
	if got := completion["wishlistItems"].(int); got != 5 {
		t.Fatalf("expected wishlistItems=5, got %d", got)
	}
	pct := completion["completionPercentage"].(float64)
	if math.Abs(pct-28.5714285714) > 0.001 {
		t.Fatalf("expected completionPercentage≈28.57, got %.6f", pct)
	}
}

func TestSetRepository_MigrateTagsToSets_SyncsNewTaggedCoinMemberships(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	tag := models.Tag{UserID: 1, Name: "Roman", Color: "#c9a84c"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	initialCoin := models.Coin{Name: "Initial", UserID: 1}
	if err := db.Create(&initialCoin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	if err := db.Create(&models.CoinTag{CoinID: initialCoin.ID, TagID: tag.ID}).Error; err != nil {
		t.Fatalf("create initial coin tag: %v", err)
	}

	if err := repo.MigrateTagsToSets(1); err != nil {
		t.Fatalf("first MigrateTagsToSets failed: %v", err)
	}

	var set models.CoinSet
	if err := db.Where("user_id = ? AND name = ?", 1, tag.Name).First(&set).Error; err != nil {
		t.Fatalf("load migrated set: %v", err)
	}

	newTaggedWishlist := models.Coin{Name: "Wishlist Add", UserID: 1, IsWishlist: true}
	if err := db.Create(&newTaggedWishlist).Error; err != nil {
		t.Fatalf("create new tagged coin: %v", err)
	}
	if err := db.Create(&models.CoinTag{CoinID: newTaggedWishlist.ID, TagID: tag.ID}).Error; err != nil {
		t.Fatalf("create new coin tag: %v", err)
	}

	if err := repo.MigrateTagsToSets(1); err != nil {
		t.Fatalf("second MigrateTagsToSets failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.CoinSetMembership{}).Where("set_id = ? AND coin_id = ?", set.ID, newTaggedWishlist.ID).Count(&count).Error; err != nil {
		t.Fatalf("count membership: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected newly tagged coin to be added to set membership, got %d", count)
	}
}

func TestSetRepository_CountPinned_ScopesToUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	now := time.Now().UTC()
	userOneSets := []models.CoinSet{
		{UserID: 1, Name: "Pinned One", SetType: models.CoinSetTypeStandard, PinnedAt: &now},
		{UserID: 1, Name: "Pinned Two", SetType: models.CoinSetTypeStandard, PinnedAt: &now},
		{UserID: 1, Name: "Unpinned", SetType: models.CoinSetTypeStandard},
	}
	if err := db.Create(&userOneSets).Error; err != nil {
		t.Fatalf("create user 1 sets: %v", err)
	}
	userTwoSets := []models.CoinSet{
		{UserID: 2, Name: "Other User Pinned", SetType: models.CoinSetTypeStandard, PinnedAt: &now},
	}
	if err := db.Create(&userTwoSets).Error; err != nil {
		t.Fatalf("create user 2 sets: %v", err)
	}

	count, err := repo.CountPinned(1)
	if err != nil {
		t.Fatalf("CountPinned(1) failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 pinned sets for user 1, got %d", count)
	}

	count, err = repo.CountPinned(2)
	if err != nil {
		t.Fatalf("CountPinned(2) failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 pinned set for user 2, got %d", count)
	}
}

func TestSetRepository_Update_PinnedAtRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSetRepository(db)

	set := models.CoinSet{UserID: 1, Name: "Round Trip Set", SetType: models.CoinSetTypeStandard}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}

	now := time.Now().UTC()
	if err := repo.Update(&set, map[string]interface{}{"pinned_at": &now}); err != nil {
		t.Fatalf("update pinned_at: %v", err)
	}

	fetched, err := repo.GetByID(set.ID, 1)
	if err != nil {
		t.Fatalf("get by id after pin: %v", err)
	}
	if fetched.PinnedAt == nil {
		t.Fatalf("expected pinned_at to persist through a map update")
	}
	if !fetched.PinnedAt.Equal(now) {
		t.Fatalf("expected pinned_at %v, got %v", now, *fetched.PinnedAt)
	}

	if err := repo.Update(fetched, map[string]interface{}{"pinned_at": nil}); err != nil {
		t.Fatalf("update pinned_at to nil: %v", err)
	}
	unpinned, err := repo.GetByID(set.ID, 1)
	if err != nil {
		t.Fatalf("get by id after unpin: %v", err)
	}
	if unpinned.PinnedAt != nil {
		t.Fatalf("expected pinned_at to be nulled out, got %v", unpinned.PinnedAt)
	}
}
