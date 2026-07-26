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

func TestSetService_CreateSet_ValidatesAgenticDynamicMode(t *testing.T) {
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
		"name":    "Missing Prompt",
		"setType": "agentic",
	})
	if err == nil || !strings.Contains(err.Error(), "agentic prompt is required") {
		t.Fatalf("expected agentic prompt validation error, got %v", err)
	}

	agentic, err := service.CreateSet(1, map[string]interface{}{
		"name":          "US Silver Quarters",
		"setType":       "agentic",
		"agenticPrompt": "All US Silver Quarters from 1940s to 1960s",
	})
	if err != nil {
		t.Fatalf("create agentic failed: %v", err)
	}
	if agentic.SetType != models.CoinSetTypeAgentic {
		t.Fatalf("expected set type %q, got %q", models.CoinSetTypeAgentic, agentic.SetType)
	}
	if agentic.CreationMode != models.CoinSetCreationModeDynamic {
		t.Fatalf("expected creation mode %q, got %q", models.CoinSetCreationModeDynamic, agentic.CreationMode)
	}
}

func TestBuildAgenticTargets_GeneratesBoundedSilverQuarterRoster(t *testing.T) {
	targets := buildAgenticTargets("All US Silver Quarters from 1940s to 1960s", 7)
	if len(targets) != 25 {
		t.Fatalf("expected 25 targets for 1940-1964 silver quarters, got %d", len(targets))
	}
	if targets[0].Label != "1940 US Silver Quarter" {
		t.Fatalf("unexpected first target label %q", targets[0].Label)
	}
	last := targets[len(targets)-1]
	if last.Label != "1964 US Silver Quarter" {
		t.Fatalf("unexpected last target label %q", last.Label)
	}
	if targets[0].Material == nil || *targets[0].Material != "Silver" {
		t.Fatalf("expected silver material on first target")
	}
}
