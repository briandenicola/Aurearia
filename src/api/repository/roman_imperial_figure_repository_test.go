package repository

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRomanImperialFigureTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.RomanImperialFigure{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func seedTestImperialFigures(t *testing.T, db *gorm.DB) {
	t.Helper()
	figures := []models.RomanImperialFigure{
		{Name: "Augustus", NormalizedName: "augustus", Aliases: models.StringList{"Octavian"}, Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 1, RarityTier: models.RarityTierCommon},
		{Name: "Livia", NormalizedName: "livia", Aliases: models.StringList{}, Role: models.ImperialFigureRoleEmpress, Region: models.ImperialFigureRegionWest, Dynasty: "Julio-Claudian", ReignStart: -27, ReignEnd: 14, SortOrder: 2, RarityTier: models.RarityTierCommon},
		{Name: "Zeno", NormalizedName: "zeno", Aliases: models.StringList{}, Role: models.ImperialFigureRoleEmperor, Region: models.ImperialFigureRegionEast, Dynasty: "Leonid", ReignStart: 474, ReignEnd: 491, SortOrder: 3, RarityTier: models.RarityTierScarce},
		{Name: "Basiliscus", NormalizedName: "basiliscus", Aliases: models.StringList{}, Role: models.ImperialFigureRoleUsurper, Region: models.ImperialFigureRegionEast, Dynasty: "Leonid", ReignStart: 475, ReignEnd: 476, SortOrder: 4, RarityTier: models.RarityTierRare},
	}
	if err := db.Create(&figures).Error; err != nil {
		t.Fatalf("failed to seed figures: %v", err)
	}
}

func TestRomanImperialFigureRepositoryList(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	figures, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(figures) != 4 {
		t.Fatalf("expected 4 figures, got %d", len(figures))
	}
	if figures[0].Name != "Augustus" {
		t.Errorf("expected first figure by sort_order to be Augustus, got %s", figures[0].Name)
	}
}

func TestRomanImperialFigureRepositoryListByRole(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	emperors, err := repo.ListByRole(models.ImperialFigureRoleEmperor)
	if err != nil {
		t.Fatalf("ListByRole failed: %v", err)
	}
	if len(emperors) != 2 {
		t.Fatalf("expected 2 emperors, got %d", len(emperors))
	}
	for _, f := range emperors {
		if f.Role != models.ImperialFigureRoleEmperor {
			t.Errorf("expected only role=emperor, got %s", f.Role)
		}
	}
}

func TestRomanImperialFigureRepositoryFindByID(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	var augustus models.RomanImperialFigure
	if err := db.Where("name = ?", "Augustus").First(&augustus).Error; err != nil {
		t.Fatalf("failed to load seeded Augustus: %v", err)
	}

	found, err := repo.FindByID(augustus.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Augustus" {
		t.Errorf("expected Augustus, got %s", found.Name)
	}

	if _, err := repo.FindByID(999999); err == nil {
		t.Error("expected error for missing ID, got nil")
	}
}

func TestRomanImperialFigureRepositorySearchByNameQuery(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	results, err := repo.Search("aug", "", 50)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Augustus" {
		t.Fatalf("expected only Augustus for query 'aug', got %v", results)
	}
}

func TestRomanImperialFigureRepositorySearchByAlias(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	results, err := repo.Search("Octavian", "", 50)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Augustus" {
		t.Fatalf("expected Augustus matched via alias 'Octavian', got %v", results)
	}
}

func TestRomanImperialFigureRepositorySearchByRoleFilter(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	results, err := repo.Search("", models.ImperialFigureRoleUsurper, 50)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Basiliscus" {
		t.Fatalf("expected only Basiliscus for role=usurper, got %v", results)
	}
}

func TestRomanImperialFigureRepositorySearchCombinesQueryAndRole(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	results, err := repo.Search("zeno", models.ImperialFigureRoleUsurper, 50)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for name=zeno + role=usurper (Zeno is role=emperor), got %v", results)
	}
}

func TestRomanImperialFigureRepositorySearchRespectsLimit(t *testing.T) {
	db := setupRomanImperialFigureTestDB(t)
	seedTestImperialFigures(t, db)
	repo := NewRomanImperialFigureRepository(db)

	results, err := repo.Search("", "", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected limit of 2 results, got %d", len(results))
	}
}
