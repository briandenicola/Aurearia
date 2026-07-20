package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupEmperorTrackerServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.RomanImperialFigure{}, &models.RomanImperialFigureHighlight{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func emperorTrackerServiceFor(db *gorm.DB) *EmperorTrackerService {
	return NewEmperorTrackerService(
		repository.NewRomanImperialFigureRepository(db),
		repository.NewCoinRepository(db),
		repository.NewRomanImperialFigureHighlightRepository(db),
	)
}

func seedEmperorTrackerFigures(t *testing.T, db *gorm.DB) map[string]uint {
	t.Helper()
	figures := []models.RomanImperialFigure{
		{Name: "Augustus", NormalizedName: "augustus", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 1, RarityTier: models.RarityTierCommon},
		{Name: "Tiberius", NormalizedName: "tiberius", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: 14, ReignEnd: 37, SortOrder: 2, RarityTier: models.RarityTierCommon},
		{Name: "Gordian I", NormalizedName: "gordiani", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Crisis of the Third Century", ReignStart: 238, ReignEnd: 238, SortOrder: 3, RarityTier: models.RarityTierVeryRare},
		{Name: "Nerva", NormalizedName: "nerva", Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Nerva-Antonine", ReignStart: 96, ReignEnd: 98, SortOrder: 4, RarityTier: models.RarityTierScarce},
		{Name: "Livia", NormalizedName: "livia", Role: models.ImperialFigureRoleEmpress, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 5, RarityTier: models.RarityTierCommon},
		{Name: "Basiliscus", NormalizedName: "basiliscus", Role: models.ImperialFigureRoleUsurper, Region: models.ImperialFigureRegionEast, Dynasty: "Leonid", ReignStart: 475, ReignEnd: 476, SortOrder: 6, RarityTier: models.RarityTierRare},
		{Name: "Julius Caesar", NormalizedName: "juliuscaesar", Role: models.ImperialFigureRoleOther, Region: models.ImperialFigureRegionWest, Dynasty: "Late Republic (precursor)", ReignStart: -49, ReignEnd: -44, SortOrder: 7, RarityTier: models.RarityTierRare},
		{Name: "Crispus", NormalizedName: "crispus", Role: models.ImperialFigureRoleCaesar, Region: models.ImperialFigureRegionWest, Dynasty: "Constantinian", ReignStart: 317, ReignEnd: 326, SortOrder: 8, RarityTier: models.RarityTierScarce},
	}
	if err := db.Create(&figures).Error; err != nil {
		t.Fatalf("failed to seed figures: %v", err)
	}
	ids := make(map[string]uint, len(figures))
	for _, f := range figures {
		ids[f.Name] = f.ID
	}
	return ids
}

func TestEmperorTrackerService_ProgressComputesOverallAndPerDynasty(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	ids := seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	augustusID := ids["Augustus"]
	if err := db.Create(&models.Coin{Name: "My Augustus", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &augustusID}).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}

	if progress.Total != 4 {
		t.Fatalf("expected 4 total emperors, got %d", progress.Total)
	}
	if progress.Owned != 1 {
		t.Fatalf("expected 1 owned emperor, got %d", progress.Owned)
	}
	if progress.Percentage != 25.0 {
		t.Fatalf("expected 25%% completion, got %v", progress.Percentage)
	}

	var julioClaudian, crisis DynastyProgress
	for _, d := range progress.Dynasties {
		switch d.Dynasty {
		case "Julio-Claudian":
			julioClaudian = d
		case "Crisis of the Third Century":
			crisis = d
		}
	}
	if julioClaudian.Total != 2 || julioClaudian.Owned != 1 {
		t.Fatalf("expected Julio-Claudian 1/2, got %d/%d", julioClaudian.Owned, julioClaudian.Total)
	}
	if crisis.Total != 1 || crisis.Owned != 0 {
		t.Fatalf("expected Crisis 0/1, got %d/%d", crisis.Owned, crisis.Total)
	}
}

func TestEmperorTrackerService_ProgressExcludesWishlistAndSoldCoins(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	ids := seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	augustusID := ids["Augustus"]
	tiberiusID := ids["Tiberius"]
	if err := db.Create(&models.Coin{Name: "Wishlist Augustus", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &augustusID, IsWishlist: true}).Error; err != nil {
		t.Fatalf("seed wishlist coin: %v", err)
	}
	if err := db.Create(&models.Coin{Name: "Sold Tiberius", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &tiberiusID, IsSold: true}).Error; err != nil {
		t.Fatalf("seed sold coin: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}
	if progress.Owned != 0 {
		t.Fatalf("expected 0 owned (wishlist/sold coins don't count), got %d", progress.Owned)
	}
}

func TestEmperorTrackerService_ProgressExcludesNonRomanMatchedCoins(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	ids := seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	augustusID := ids["Augustus"]
	if err := db.Create(&models.Coin{Name: "Greek Augustus", Category: models.CategoryGreek, UserID: 1, RomanImperialFigureID: &augustusID}).Error; err != nil {
		t.Fatalf("seed non-Roman coin: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}
	if progress.Owned != 0 {
		t.Fatalf("expected 0 owned because non-Roman matched coins do not count, got %d", progress.Owned)
	}
}

func TestEmperorTrackerService_ProgressCombinesMultipleRoles(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleCaesar, models.ImperialFigureRoleOther)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}
	if progress.Total != 2 {
		t.Fatalf("expected 2 total (Crispus + Julius Caesar combined), got %d", progress.Total)
	}
}

func TestEmperorTrackerService_SuggestionsSortsByRarityTierThenChronology(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	suggestions, err := svc.Suggestions(1, 10)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	// All 4 emperors are missing. Common (Augustus, Tiberius) should sort
	// before scarce (Nerva) before very_rare (Gordian I); Augustus before
	// Tiberius as a chronological tie-break within the same tier.
	wantOrder := []string{"Augustus", "Tiberius", "Nerva", "Gordian I"}
	if len(suggestions) != len(wantOrder) {
		t.Fatalf("expected %d suggestions, got %d: %v", len(wantOrder), len(suggestions), suggestions)
	}
	for i, name := range wantOrder {
		if suggestions[i].Name != name {
			t.Errorf("suggestion[%d] = %s, want %s", i, suggestions[i].Name, name)
		}
	}
}

func TestEmperorTrackerService_SuggestionsRespectsLimit(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	suggestions, err := svc.Suggestions(1, 2)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
	}
}

func TestEmperorTrackerService_FullProgressOnlyIncludesEnabledCategories(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := emperorTrackerServiceFor(db)

	result, err := svc.FullProgress(1, false, false, false, 10)
	if err != nil {
		t.Fatalf("FullProgress failed: %v", err)
	}
	if result.Usurpers != nil || result.Empresses != nil || result.Other != nil {
		t.Fatalf("expected no optional categories when all disabled, got %+v", result)
	}
	if result.Emperor.Total != 4 {
		t.Fatalf("expected emperor total 4, got %d", result.Emperor.Total)
	}
	if len(result.Suggestions) != 4 {
		t.Fatalf("expected 4 suggestions, got %d", len(result.Suggestions))
	}

	full, err := svc.FullProgress(1, true, true, true, 10)
	if err != nil {
		t.Fatalf("FullProgress failed: %v", err)
	}
	if full.Usurpers == nil || full.Usurpers.Total != 1 {
		t.Fatalf("expected usurpers enabled with total 1, got %+v", full.Usurpers)
	}
	if full.Empresses == nil || full.Empresses.Total != 1 {
		t.Fatalf("expected empresses enabled with total 1, got %+v", full.Empresses)
	}
	if full.Other == nil || full.Other.Total != 2 {
		t.Fatalf("expected other enabled with total 2, got %+v", full.Other)
	}
}

func TestEmperorTrackerService_ProgressReturnsCoinDataForOwnedFigures(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	ids := seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	augustusID := ids["Augustus"]
	diameter := 18.5
	coin := models.Coin{Name: "My Augustus Denarius", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &augustusID, DiameterMm: &diameter}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "/uploads/augustus.jpg", ImageType: models.ImageTypeObverse}).Error; err != nil {
		t.Fatalf("seed coin image: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}

	var augustusSlot *ImperialFigureSlot
	for _, d := range progress.Dynasties {
		for i := range d.Figures {
			if d.Figures[i].Figure.Name == "Augustus" {
				augustusSlot = &d.Figures[i]
			}
		}
	}
	if augustusSlot == nil || augustusSlot.Coin == nil {
		t.Fatal("expected Augustus slot to have a matched coin")
	}
	if augustusSlot.Coin.Name != "My Augustus Denarius" {
		t.Errorf("coin name = %s, want 'My Augustus Denarius'", augustusSlot.Coin.Name)
	}
	if len(augustusSlot.Coin.Images) != 1 || augustusSlot.Coin.Images[0].FilePath != "/uploads/augustus.jpg" {
		t.Errorf("expected 1 preloaded image, got %+v", augustusSlot.Coin.Images)
	}
	if len(augustusSlot.Coins) != 1 || augustusSlot.HighlightedCoinID == nil || *augustusSlot.HighlightedCoinID != coin.ID {
		t.Errorf("expected one candidate with highlighted coin %d, got coins=%+v highlighted=%v", coin.ID, augustusSlot.Coins, augustusSlot.HighlightedCoinID)
	}
}

func TestEmperorTrackerService_ProgressUsesUserSelectedHighlightWithoutInflatingOwnedCount(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	ids := seedEmperorTrackerFigures(t, db)
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	augustusID := ids["Augustus"]
	first := models.Coin{Name: "First Augustus", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &augustusID}
	second := models.Coin{Name: "Preferred Augustus", Category: models.CategoryRoman, UserID: 1, RomanImperialFigureID: &augustusID}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("seed first coin: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("seed second coin: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	if err := svc.SetHighlight(1, augustusID, second.ID); err != nil {
		t.Fatalf("SetHighlight failed: %v", err)
	}
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}
	if progress.Owned != 1 {
		t.Fatalf("expected duplicate coins to count as one owned figure, got %d", progress.Owned)
	}
	var slot *ImperialFigureSlot
	for _, dynasty := range progress.Dynasties {
		for i := range dynasty.Figures {
			if dynasty.Figures[i].Figure.ID == augustusID {
				slot = &dynasty.Figures[i]
			}
		}
	}
	if slot == nil || slot.Coin == nil {
		t.Fatal("expected Augustus slot")
	}
	if slot.Coin.ID != second.ID {
		t.Fatalf("expected highlighted coin %d, got %+v", second.ID, slot.Coin)
	}
	if len(slot.Coins) != 2 {
		t.Fatalf("expected both candidate coins, got %d", len(slot.Coins))
	}
}

func TestEmperorTrackerService_ProgressHandlesZeroFiguresWithoutDivideByZero(t *testing.T) {
	db := setupEmperorTrackerServiceDB(t)
	// No figures seeded at all for this test — an empty dataset/role combo
	// must not panic or produce NaN/Inf.
	if err := db.Create(&models.User{ID: 1, Username: "u1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := emperorTrackerServiceFor(db)
	progress, err := svc.Progress(1, models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("Progress failed: %v", err)
	}
	if progress.Total != 0 || progress.Owned != 0 || progress.Percentage != 0 {
		t.Fatalf("expected all-zero progress for an empty dataset, got %+v", progress)
	}
	if len(progress.Dynasties) != 0 {
		t.Fatalf("expected no dynasties, got %+v", progress.Dynasties)
	}
}
