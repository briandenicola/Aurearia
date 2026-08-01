package services

import (
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSetServiceTest(t *testing.T) *SetService {
	t.Helper()
	service, _ := setupSetServiceTestWithDB(t)
	return service
}

func setupSetServiceTestWithDB(t *testing.T) (*SetService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Tag{},
		&models.CoinTag{},
		&models.Coin{},
		&models.CoinSet{},
		&models.CoinSetMembership{},
		&models.CoinSetTarget{},
		&models.SmartCriteriaTemplate{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return NewSetService(repository.NewSetRepository(db), repository.NewTagRepository(db)), db
}

func TestSetService_CreateSet_NormalizesLegacySetTypes(t *testing.T) {
	service := setupSetServiceTest(t)

	openSet, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Legacy Open",
		"setType": "open",
	})
	if err != nil {
		t.Fatalf("create open set failed: %v", err)
	}
	if openSet.SetType != models.CoinSetTypeStandard {
		t.Fatalf("expected open to normalize to %q, got %q", models.CoinSetTypeStandard, openSet.SetType)
	}

	definedSet, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Legacy Defined",
		"setType": "defined",
	})
	if err != nil {
		t.Fatalf("create defined set failed: %v", err)
	}
	if definedSet.SetType != models.CoinSetTypeGoal {
		t.Fatalf("expected defined to normalize to %q, got %q", models.CoinSetTypeGoal, definedSet.SetType)
	}
}

func TestSetService_CreateSet_RejectsDynamicAsSetType(t *testing.T) {
	service := setupSetServiceTest(t)
	_, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Invalid Dynamic Type",
		"setType": "dynamic",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid set type") {
		t.Fatalf("expected invalid set type error, got %v", err)
	}
}

func TestSetService_CreateSet_GeneratesDefaultPaletteColor(t *testing.T) {
	service := setupSetServiceTest(t)

	set, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Palette Default",
		"setType": "standard",
	})
	if err != nil {
		t.Fatalf("create set failed: %v", err)
	}

	expected := defaultSetColor(1, "Palette Default", 0)
	if set.Color != expected {
		t.Fatalf("expected generated color %q, got %q", expected, set.Color)
	}
	if set.Color == "#6b7280" {
		t.Fatalf("expected palette color instead of legacy gray default")
	}
}

func TestSetService_CreateSet_DisablesDirectAgenticCreation(t *testing.T) {
	service := setupSetServiceTest(t)

	_, err := service.CreateSet(1, map[string]interface{}{
		"name":         "Invalid Dynamic Goal",
		"setType":      "goal",
		"creationMode": "dynamic",
	})
	if err == nil || !strings.Contains(err.Error(), "only valid for agentic sets") {
		t.Fatalf("expected agentic mode validation error, got %v", err)
	}

	_, err = service.CreateSet(1, map[string]interface{}{
		"name":          "US Silver Quarters",
		"setType":       "agentic",
		"agenticPrompt": "All US Silver Quarters from 1940s to 1960s",
	})
	if err == nil || !strings.Contains(err.Error(), "proposal review workflow") {
		t.Fatalf("expected direct agentic creation disabled error, got %v", err)
	}

	count, countErr := service.repo.CountByUser(1)
	if countErr != nil {
		t.Fatalf("count sets: %v", countErr)
	}
	if count != 0 {
		t.Fatalf("expected no set to be created, got %d", count)
	}
}

func TestSetService_AddCoinToSet_AgenticRequiresTargetID(t *testing.T) {
	service, db := setupSetServiceTestWithDB(t)

	set := models.CoinSet{UserID: 1, Name: "Agentic Slot Set", SetType: models.CoinSetTypeAgentic, CreationMode: models.CoinSetCreationModeDynamic}
	if err := service.repo.Create(&set); err != nil {
		t.Fatalf("create set: %v", err)
	}
	coin := models.Coin{Name: "Test Coin", UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	err := service.AddCoinToSet(coin.ID, set.ID, 1, "")
	if err == nil || !strings.Contains(err.Error(), "targetId is required") {
		t.Fatalf("expected agentic targetId validation error, got %v", err)
	}
}
