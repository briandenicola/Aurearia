package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func collectorProfileRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:collector_profile_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CollectorProfile{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCollectorProfileRepositoryOwnerIsolationAndReplacement(t *testing.T) {
	db := collectorProfileRepositoryTestDB(t)
	repo := NewCollectorProfileRepository(db)

	if profile, err := repo.Get(1); err != nil || profile != nil {
		t.Fatalf("absent profile = %#v, %v; want nil, nil", profile, err)
	}
	first := models.CollectorProfile{UserID: 999, Currency: stringPointer("USD"), PreferredPeriods: models.StringList{"Flavian"}}
	if err := repo.Replace(1, &first); err != nil {
		t.Fatal(err)
	}
	second := models.CollectorProfile{Currency: stringPointer("EUR"), PreferredPeriods: models.StringList{"Severan"}}
	if err := repo.Replace(2, &second); err != nil {
		t.Fatal(err)
	}

	owner, err := repo.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if owner == nil || owner.UserID != 1 || owner.Currency == nil || *owner.Currency != "USD" {
		t.Fatalf("owner profile mismatch: %#v", owner)
	}
	foreign, err := repo.Get(2)
	if err != nil {
		t.Fatal(err)
	}
	if foreign == nil || foreign.UserID != 2 || foreign.Currency == nil || *foreign.Currency != "EUR" {
		t.Fatalf("foreign profile mismatch: %#v", foreign)
	}

	replacement := models.CollectorProfile{UserID: 2, Currency: stringPointer("GBP"), PreferredPeriods: models.StringList{}}
	if err := repo.Replace(1, &replacement); err != nil {
		t.Fatal(err)
	}
	owner, _ = repo.Get(1)
	foreign, _ = repo.Get(2)
	if owner == nil || owner.UserID != 1 || owner.Currency == nil || *owner.Currency != "GBP" {
		t.Fatalf("replacement escaped owner scope: %#v", owner)
	}
	if foreign == nil || foreign.Currency == nil || *foreign.Currency != "EUR" {
		t.Fatalf("owner replacement changed foreign row: %#v", foreign)
	}
	var count int64
	if err := db.Model(&models.CollectorProfile{}).Where("user_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("same-owner replacement produced %d rows", count)
	}
}

func stringPointer(value string) *string { return &value }
