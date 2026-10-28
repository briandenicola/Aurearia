package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeImageExtCanonicalAllowlist(t *testing.T) {
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		for _, input := range []string{ext, ext[1:], " " + strings.ToUpper(ext) + " "} {
			t.Run(input, func(t *testing.T) {
				got, err := NormalizeImageExt(input)
				if err != nil || got != ext {
					t.Fatalf("NormalizeImageExt(%q) = %q, %v; want %q, nil", input, got, err, ext)
				}
			})
		}
	}
	for _, input := range []string{
		"", ".", ".svg", ".jpg/../../escape", "../escape.jpg",
		`..\escape.jpg`, `C:\escape.jpg`, "/tmp/escape.png",
		".png\x00.jpg", "%2e%2e%2fescape.jpg",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := NormalizeImageExt(input)
			if got != "" || !errors.Is(err, ErrInvalidFileExt) {
				t.Fatalf("NormalizeImageExt(%q) = %q, %v; want empty extension and ErrInvalidFileExt", input, got, err)
			}
		})
	}
}

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

func TestLoadOwnedCoinImagesEnforcesPerCoinAndByteBudget(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_service_inline_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	uploadDir := t.TempDir()
	first := models.Coin{UserID: 10, Name: "First"}
	second := models.Coin{UserID: 10, Name: "Second"}
	if err := db.Create(&[]*models.Coin{&first, &second}).Error; err != nil {
		t.Fatal(err)
	}
	for _, coin := range []models.Coin{first, second} {
		for _, name := range []string{"a", "b"} {
			relPath := fmt.Sprintf("coin-%d/%s.jpg", coin.ID, name)
			fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(strings.TrimSuffix(fullPath, ".jpg")+"_thumb.jpg", []byte("0123456789"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fullPath, []byte("orig"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: models.ImageTypeOther}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	svc := NewImageService(repository.NewImageRepository(db), uploadDir)

	images, truncated, err := svc.LoadOwnedCoinImages(10, []uint{first.ID, second.ID},
		InlineCoinImageLimits{Size: VariantSizeThumb, PerCoin: 1, MaxTotalBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 || images[0].CoinID != first.ID || images[1].CoinID != second.ID || truncated {
		t.Fatalf("images=%+v truncated=%v, want one per coin within budget", images, truncated)
	}

	images, truncated, err = svc.LoadOwnedCoinImages(10, []uint{first.ID, second.ID},
		InlineCoinImageLimits{Size: VariantSizeThumb, PerCoin: 2, MaxTotalBytes: 25})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 || !truncated {
		t.Fatalf("images=%d truncated=%v, want budget to stop after two 10-byte thumbs", len(images), truncated)
	}

	if images, _, err := svc.LoadOwnedCoinImages(11, []uint{first.ID}, InlineCoinImageLimits{Size: VariantSizeThumb, PerCoin: 1, MaxTotalBytes: 100}); err != nil || len(images) != 0 {
		t.Fatalf("non-owner images=%+v err=%v, want none", images, err)
	}
}
