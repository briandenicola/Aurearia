package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCoinImagePathInActiveShowcaseRequiresShowcaseOwnerCoin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_repository_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.Showcase{}, &models.ShowcaseCoin{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	owner := models.User{Username: "owner", Email: "owner@example.com", PasswordHash: "x"}
	other := models.User{Username: "other", Email: "other@example.com", PasswordHash: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	ownerCoin := models.Coin{Name: "Owner Coin", Category: models.CategoryRoman, UserID: owner.ID}
	otherCoin := models.Coin{Name: "Other Coin", Category: models.CategoryGreek, UserID: other.ID}
	if err := db.Create(&ownerCoin).Error; err != nil {
		t.Fatalf("failed to create owner coin: %v", err)
	}
	if err := db.Create(&otherCoin).Error; err != nil {
		t.Fatalf("failed to create other coin: %v", err)
	}
	ownerImage := models.CoinImage{CoinID: ownerCoin.ID, FilePath: "coins/owner.webp", ImageType: models.ImageTypeObverse}
	otherImage := models.CoinImage{CoinID: otherCoin.ID, FilePath: "coins/other.webp", ImageType: models.ImageTypeObverse}
	if err := db.Create(&ownerImage).Error; err != nil {
		t.Fatalf("failed to create owner image: %v", err)
	}
	if err := db.Create(&otherImage).Error; err != nil {
		t.Fatalf("failed to create other image: %v", err)
	}

	showcase := models.Showcase{UserID: owner.ID, Slug: "featured-set", Title: "Featured Set", IsActive: true}
	if err := db.Create(&showcase).Error; err != nil {
		t.Fatalf("failed to create showcase: %v", err)
	}
	if err := db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: ownerCoin.ID}).Error; err != nil {
		t.Fatalf("failed to link owner coin: %v", err)
	}
	if err := db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: otherCoin.ID}).Error; err != nil {
		t.Fatalf("failed to link other owner coin: %v", err)
	}

	repo := NewImageRepository(db)
	allowed, err := repo.CoinImagePathInActiveShowcase("featured-set", "coins/owner.webp")
	if err != nil {
		t.Fatalf("failed checking owner image: %v", err)
	}
	if !allowed {
		t.Fatal("expected showcase owner's linked coin image to be public")
	}

	allowed, err = repo.CoinImagePathInActiveShowcase("featured-set", "coins/other.webp")
	if err != nil {
		t.Fatalf("failed checking other owner image: %v", err)
	}
	if allowed {
		t.Fatal("expected linked image from a different coin owner to stay private")
	}
}
