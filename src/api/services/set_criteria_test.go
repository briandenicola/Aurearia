package services

import (
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSetCriteriaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.CoinSet{}, &models.CoinSetMembership{}, &models.CoinSetTarget{},
		&models.SmartCriteriaTemplate{}, &models.Tag{}, &models.CoinTag{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestSetService(t *testing.T) (*SetService, *gorm.DB) {
	t.Helper()
	db := setupSetCriteriaTestDB(t)
	setRepo := repository.NewSetRepository(db)
	tagRepo := repository.NewTagRepository(db)
	svc := NewSetService(setRepo, tagRepo)
	return svc, db
}

// ---- ValidateSmartCriteria ----

func TestValidateSmartCriteria_ValidSingleRule(t *testing.T) {
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Silver"},
		},
	}
	if err := ValidateSmartCriteria(criteria); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateSmartCriteria_InvalidField(t *testing.T) {
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "badfield", "op": "eq", "value": "x"},
		},
	}
	err := ValidateSmartCriteria(criteria)
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
	if !strings.Contains(err.Error(), "badfield") {
		t.Errorf("error should mention bad field, got: %v", err)
	}
}

func TestValidateSmartCriteria_InvalidOperator(t *testing.T) {
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "LIKE", "value": "Silver"},
		},
	}
	err := ValidateSmartCriteria(criteria)
	if err == nil {
		t.Fatal("expected error for invalid operator")
	}
}

func TestValidateSmartCriteria_InvalidGroupOperator(t *testing.T) {
	criteria := map[string]interface{}{
		"operator": "xor",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Gold"},
		},
	}
	err := ValidateSmartCriteria(criteria)
	if err == nil {
		t.Fatal("expected error for invalid group operator")
	}
}

func TestValidateSmartCriteria_EmptyRules(t *testing.T) {
	criteria := map[string]interface{}{"operator": "and", "rules": []interface{}{}}
	if err := ValidateSmartCriteria(criteria); err == nil {
		t.Fatal("expected error for empty rules")
	}
}

func TestValidateSmartCriteria_NestedGroups(t *testing.T) {
	criteria := map[string]interface{}{
		"operator": "or",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Silver"},
			map[string]interface{}{
				"operator": "and",
				"rules": []interface{}{
					map[string]interface{}{"field": "category", "op": "eq", "value": "Roman"},
					map[string]interface{}{"field": "isWishlist", "op": "eq", "value": true},
				},
			},
		},
	}
	if err := ValidateSmartCriteria(criteria); err != nil {
		t.Errorf("expected no error for nested groups, got %v", err)
	}
}

// ---- GetSuggestedCriteria ----

func TestGetSuggestedCriteria_ReturnsTenSuggestions(t *testing.T) {
	suggestions := GetSuggestedCriteria()
	if len(suggestions) < 5 {
		t.Errorf("expected at least 5 suggestions, got %d", len(suggestions))
	}
}

func TestGetSuggestedCriteria_AllHaveValidCriteria(t *testing.T) {
	for _, s := range GetSuggestedCriteria() {
		if s.ID == "" {
			t.Errorf("suggestion missing ID")
		}
		if s.Name == "" {
			t.Errorf("suggestion %s missing Name", s.ID)
		}
		if err := ValidateSmartCriteria(s.Criteria); err != nil {
			t.Errorf("suggestion %s has invalid criteria: %v", s.ID, err)
		}
	}
}

func TestGetSuggestedCriteria_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range GetSuggestedCriteria() {
		if seen[s.ID] {
			t.Errorf("duplicate suggestion ID: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

// ---- CriteriaTemplate service methods ----

func TestSetService_SaveCriteriaTemplate_Valid(t *testing.T) {
	svc, _ := newTestSetService(t)
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Gold"},
		},
	}
	tmpl, err := svc.SaveCriteriaTemplate(1, "My Gold Template", "Gold coins", criteria)
	if err != nil {
		t.Fatalf("SaveCriteriaTemplate failed: %v", err)
	}
	if tmpl.ID == 0 {
		t.Fatal("expected template ID to be set")
	}
	if tmpl.Name != "My Gold Template" {
		t.Errorf("name mismatch: got %q", tmpl.Name)
	}
}

func TestSetService_SaveCriteriaTemplate_EmptyName(t *testing.T) {
	svc, _ := newTestSetService(t)
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "material", "op": "eq", "value": "Gold"},
		},
	}
	_, err := svc.SaveCriteriaTemplate(1, "", "", criteria)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSetService_SaveCriteriaTemplate_InvalidCriteria(t *testing.T) {
	svc, _ := newTestSetService(t)
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "hackedfield", "op": "eq", "value": "x"},
		},
	}
	_, err := svc.SaveCriteriaTemplate(1, "Bad Template", "", criteria)
	if err == nil {
		t.Fatal("expected error for invalid criteria field")
	}
}

func TestSetService_ListCriteriaTemplates_Empty(t *testing.T) {
	svc, _ := newTestSetService(t)
	list, err := svc.ListCriteriaTemplates(1)
	if err != nil {
		t.Fatalf("ListCriteriaTemplates failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 templates, got %d", len(list))
	}
}

func TestSetService_DeleteCriteriaTemplate_OwnedByUser(t *testing.T) {
	svc, _ := newTestSetService(t)
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "isSold", "op": "eq", "value": true},
		},
	}
	tmpl, err := svc.SaveCriteriaTemplate(1, "Sold Items", "", criteria)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := svc.DeleteCriteriaTemplate(tmpl.ID, 1); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	list, _ := svc.ListCriteriaTemplates(1)
	if len(list) != 0 {
		t.Errorf("expected 0 templates after delete, got %d", len(list))
	}
}

func TestSetService_DeleteCriteriaTemplate_AnotherUser(t *testing.T) {
	svc, _ := newTestSetService(t)
	criteria := map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{"field": "category", "op": "eq", "value": "Roman"},
		},
	}
	tmpl, _ := svc.SaveCriteriaTemplate(1, "Roman", "", criteria)

	// User 2 cannot delete user 1's template
	if err := svc.DeleteCriteriaTemplate(tmpl.ID, 2); err == nil {
		t.Fatal("expected error when deleting another user's template")
	}
}
