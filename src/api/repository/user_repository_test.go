package repository

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestUserRepository_ListUsersWithAuctionCredentials_IncludesUsersWithoutPushover guards
// against the regression this query used to have (F026): background watchlist sync was
// gated on Pushover being configured, so a user without it set up never got their
// CurrentBid/status refreshed automatically. This must include any user with NumisBids or
// CNG credentials, regardless of Pushover configuration.
func TestUserRepository_ListUsersWithAuctionCredentials_IncludesUsersWithoutPushover(t *testing.T) {
	db := setupUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	withCredsNoPushover := models.User{
		Username: "no-pushover", Email: "no-pushover@example.com",
		NumisBidsUsername: "user@example.com", NumisBidsPassword: "secret",
		PushoverEnabled: false, PushoverUserKey: "",
	}
	withCredsAndPushover := models.User{
		Username: "with-pushover", Email: "with-pushover@example.com",
		CNGUsername: "user@example.com", CNGPassword: "secret",
		PushoverEnabled: true, PushoverUserKey: "pushover-key",
	}
	noCredsAtAll := models.User{
		Username: "no-creds", Email: "no-creds@example.com",
		PushoverEnabled: true, PushoverUserKey: "pushover-key",
	}
	for _, u := range []models.User{withCredsNoPushover, withCredsAndPushover, noCredsAtAll} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("failed to create user %q: %v", u.Username, err)
		}
	}

	users, err := repo.ListUsersWithAuctionCredentials()
	if err != nil {
		t.Fatalf("ListUsersWithAuctionCredentials returned error: %v", err)
	}

	usernames := make(map[string]bool, len(users))
	for _, u := range users {
		usernames[u.Username] = true
	}
	if !usernames["no-pushover"] {
		t.Fatal("expected user without Pushover configured to be included")
	}
	if !usernames["with-pushover"] {
		t.Fatal("expected user with Pushover configured to still be included")
	}
	if usernames["no-creds"] {
		t.Fatal("did not expect a user with no auction credentials at all to be included")
	}
}
