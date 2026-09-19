package services

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func collectorProfileServiceTestDB(t *testing.T) (*CollectorProfileService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:collector_profile_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CollectorProfile{}, &models.AppSetting{}); err != nil {
		t.Fatal(err)
	}
	return NewCollectorProfileService(repository.NewCollectorProfileRepository(db)), db
}

func collectorFloatPointer(value float64) *float64 { return &value }

func validCollectorProfileInput() CollectorProfileInput {
	return CollectorProfileInput{
		BudgetMin:           collectorFloatPointer(100),
		BudgetMax:           collectorFloatPointer(500),
		Currency:            stringPointer("USD"),
		PreferredPeriods:    []string{"  Flavian   dynasty "},
		PreferredCategories: []string{" Roman "},
		ExcludedCategories:  []string{" Modern "},
		PreferredDealers:    []string{" Example   Dealer "},
		CollectingGoals:     []string{" Add a documented denarius "},
	}
}

func TestCollectorProfileServiceNeutralDefaultsAndNormalizedReplacement(t *testing.T) {
	service, db := collectorProfileServiceTestDB(t)
	neutral, err := service.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if !neutral.IsDefault || neutral.UpdatedAt != nil || neutral.BudgetMin != nil || neutral.Currency != nil {
		t.Fatalf("non-neutral absent profile: %#v", neutral)
	}
	for _, list := range [][]string{neutral.PreferredPeriods, neutral.PreferredCategories, neutral.ExcludedCategories, neutral.PreferredDealers, neutral.CollectingGoals} {
		if list == nil || len(list) != 0 {
			t.Fatalf("neutral list must be non-nil and empty: %#v", neutral)
		}
		var absentRows int64
		if err := db.Model(&models.CollectorProfile{}).Count(&absentRows).Error; err != nil {
			t.Fatal(err)
		}
		if absentRows != 0 {
			t.Fatalf("neutral GET created %d rows", absentRows)
		}
	}

	saved, err := service.Replace(1, validCollectorProfileInput())
	if err != nil {
		t.Fatal(err)
	}
	if saved.IsDefault || !reflect.DeepEqual(saved.PreferredPeriods, []string{"Flavian dynasty"}) ||
		!reflect.DeepEqual(saved.PreferredDealers, []string{"Example Dealer"}) {
		t.Fatalf("normalization mismatch: %#v", saved)
	}
	cleared, err := service.Replace(1, CollectorProfileInput{})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.IsDefault || cleared.Currency != nil || len(cleared.PreferredPeriods) != 0 {
		t.Fatalf("clear must persist a non-default empty row: %#v", cleared)
	}
}

func TestCollectorProfileServiceRejectsBoundsAndDuplicatesAtomically(t *testing.T) {
	service, _ := collectorProfileServiceTestDB(t)
	if _, err := service.Replace(1, validCollectorProfileInput()); err != nil {
		t.Fatal(err)
	}
	tooMany := make([]string, 21)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("Period %d", i)
	}
	tests := []struct {
		name  string
		input CollectorProfileInput
		field string
	}{
		{"negative minimum", CollectorProfileInput{BudgetMin: collectorFloatPointer(-1)}, "budgetMin"},
		{"maximum too large", CollectorProfileInput{BudgetMax: collectorFloatPointer(100000001)}, "budgetMax"},
		{"non-finite", CollectorProfileInput{BudgetMin: collectorFloatPointer(math.Inf(1))}, "budgetMin"},
		{"reversed budget", CollectorProfileInput{BudgetMin: collectorFloatPointer(2), BudgetMax: collectorFloatPointer(1)}, "budgetMin"},
		{"bad currency", CollectorProfileInput{Currency: stringPointer("usd")}, "currency"},
		{"too many values", CollectorProfileInput{PreferredPeriods: tooMany}, "preferredPeriods"},
		{"blank value", CollectorProfileInput{PreferredCategories: []string{" "}}, "preferredCategories"},
		{"long dealer", CollectorProfileInput{PreferredDealers: []string{strings.Repeat("x", 201)}}, "preferredDealers"},
		{"long goal", CollectorProfileInput{CollectingGoals: []string{strings.Repeat("x", 501)}}, "collectingGoals"},
		{"normalized duplicate", CollectorProfileInput{ExcludedCategories: []string{"Ancient Greek", " ancient   greek "}}, "excludedCategories"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Replace(1, tc.input)
			var validation *CollectorProfileValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("error = %#v; want validation for %s", err, tc.field)
			}
			current, getErr := service.Get(1)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if current.Currency == nil || *current.Currency != "USD" {
				t.Fatalf("invalid replacement changed stored profile: %#v", current)
			}
		})
	}
}

func TestCollectorProfileServiceIsOwnerScopedAndDoesNotChangeGlobalSettings(t *testing.T) {
	service, db := collectorProfileServiceTestDB(t)
	setting := models.AppSetting{Key: "site_name", Value: "Aurearia"}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	a := validCollectorProfileInput()
	b := validCollectorProfileInput()
	b.Currency = stringPointer("EUR")
	if _, err := service.Replace(1, a); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Replace(2, b); err != nil {
		t.Fatal(err)
	}
	gotA, _ := service.Get(1)
	gotB, _ := service.Get(2)
	if gotA.Currency == nil || *gotA.Currency != "USD" || gotB.Currency == nil || *gotB.Currency != "EUR" {
		t.Fatalf("owner isolation failed: A=%#v B=%#v", gotA, gotB)
	}
	context, err := service.Context(1)
	if err != nil {
		t.Fatal(err)
	}
	if context.Currency == nil || *context.Currency != "USD" || context.CapturedAt.IsZero() {
		t.Fatalf("bounded context mismatch: %#v", context)
	}
	var after models.AppSetting
	if err := db.First(&after, "key = ?", "site_name").Error; err != nil {
		t.Fatal(err)
	}
	if after.Value != setting.Value {
		t.Fatalf("collector profile changed global setting: %#v", after)
	}
}
