package models

import (
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCollectorProfilePersistsJSONListsAndEnforcesOneRowPerOwner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &CollectorProfile{}); err != nil {
		t.Fatal(err)
	}
	user := User{Username: "collector", Email: "collector@example.test", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	profile := CollectorProfile{
		UserID:              user.ID,
		PreferredPeriods:    StringList{"Flavian", "Severan"},
		PreferredCategories: StringList{"Roman"},
		ExcludedCategories:  StringList{},
		PreferredDealers:    StringList{"Example Dealer"},
		CollectingGoals:     StringList{"Add a documented denarius"},
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	var got CollectorProfile
	if err := db.First(&got, "user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.PreferredPeriods, profile.PreferredPeriods) ||
		!reflect.DeepEqual(got.ExcludedCategories, StringList{}) {
		t.Fatalf("JSON list round trip mismatch: %#v", got)
	}
	if err := db.Create(&CollectorProfile{UserID: user.ID}).Error; err == nil {
		t.Fatal("expected unique owner constraint")
	}
}
