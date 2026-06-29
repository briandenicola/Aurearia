package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestValidateImageDataRejectsNonImageBytes(t *testing.T) {
	if err := ValidateImageData([]byte("not an image")); err == nil {
		t.Fatal("expected non-image bytes to be rejected")
	}

	if _, err := NormalizeImageExt(".jpg"); err != nil {
		t.Fatalf("jpg extension should be accepted: %v", err)
	}
	if _, err := NormalizeImageType("obverse"); err != nil {
		t.Fatalf("obverse image type should be accepted: %v", err)
	}
}

func TestResolveAuthorizedMediaPathAllowsOnlyDraftOwner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_service_draft_media_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.Follow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	uploadDir := t.TempDir()
	relPath := filepath.ToSlash(filepath.Join("quick-capture-draft-1", "obverse.png"))
	fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("png"), 0644); err != nil {
		t.Fatal(err)
	}
	draft := models.QuickCaptureDraft{UserID: 10, WorkingTitle: "Draft", Status: models.QuickCaptureDraftStatusActive}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	image := models.QuickCaptureDraftImage{DraftID: draft.ID, UserID: 10, FilePath: relPath, ImageType: models.ImageTypeObverse}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewImageService(repository.NewImageRepository(db), uploadDir)
	if _, err := svc.ResolveAuthorizedMediaPath(relPath, 10); err != nil {
		t.Fatalf("owner should resolve draft image: %v", err)
	}
	if _, err := svc.ResolveAuthorizedMediaPath(relPath, 11); err == nil {
		t.Fatal("non-owner should not resolve draft image")
	}
}
