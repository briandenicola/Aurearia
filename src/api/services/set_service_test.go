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
	return NewSetService(repository.NewSetRepository(db), repository.NewTagRepository(db))
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

func TestSetService_CreateSet_ValidatesTrackerDynamicMode(t *testing.T) {
	service := setupSetServiceTest(t)

	_, err := service.CreateSet(1, map[string]interface{}{
		"name":         "Invalid Dynamic Goal",
		"setType":      "goal",
		"creationMode": "dynamic",
	})
	if err == nil || !strings.Contains(err.Error(), "only valid for tracker sets") {
		t.Fatalf("expected tracker mode validation error, got %v", err)
	}

	tracker, err := service.CreateSet(1, map[string]interface{}{
		"name":         "Dynamic Tracker",
		"setType":      "tracker",
		"creationMode": "dynamic",
	})
	if err != nil {
		t.Fatalf("create tracker failed: %v", err)
	}
	if tracker.SetType != models.CoinSetTypeTracker {
		t.Fatalf("expected set type %q, got %q", models.CoinSetTypeTracker, tracker.SetType)
	}
	if tracker.CreationMode != models.CoinSetCreationModeDynamic {
		t.Fatalf("expected creation mode %q, got %q", models.CoinSetCreationModeDynamic, tracker.CreationMode)
	}
}
