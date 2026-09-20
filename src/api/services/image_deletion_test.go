package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func deletionFixture(t *testing.T) (*gorm.DB, *ImageService, models.CoinImage, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "images.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}, &models.ImageCleanup{}); err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: 1, Name: "Fixture"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	image := models.CoinImage{CoinID: coin.ID, FilePath: "fixture.jpg"}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"fixture.jpg", "fixture_thumb.jpg", "fixture_medium.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return db, NewImageService(repository.NewImageRepository(db), dir), image, dir
}

func TestImageDeleteMetadataFailurePreservesFiles(t *testing.T) {
	db, svc, image, dir := deletionFixture(t)
	if err := db.Callback().Delete().Before("gorm:delete").Register("reject", func(tx *gorm.DB) {
		tx.AddError(errors.New("injected metadata failure"))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); err == nil {
		t.Error("metadata failure reported success")
	}
	for _, name := range []string{"fixture.jpg", "fixture_thumb.jpg", "fixture_medium.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("file removed before commit: %s", name)
		}
	}
	var count int64
	db.Model(&models.ImageCleanup{}).Count(&count)
	if count != 0 {
		t.Fatal("cleanup intent was not rolled back")
	}
}

func TestImageDeleteCleanupFailureAndRecovery(t *testing.T) {
	for _, restart := range []bool{false, true} {
		t.Run(fmt.Sprint("restart=", restart), func(t *testing.T) {
			db, svc, image, dir := deletionFixture(t)
			blocked := filepath.Join(dir, "fixture_thumb.jpg")
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(blocked, 0700); err != nil {
				t.Fatal(err)
			}
			unrelated := filepath.Join(blocked, "keep.txt")
			if err := os.WriteFile(unrelated, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); !errors.Is(err, ErrImageCleanupPending) {
				t.Fatalf("expected observable cleanup failure: %v", err)
			}
			var count int64
			db.Model(&models.CoinImage{}).Count(&count)
			if count != 0 {
				t.Fatal("metadata still authoritative")
			}
			db.Model(&models.ImageCleanup{}).Count(&count)
			if count != 1 {
				t.Fatal("lost durable recovery intent")
			}
			if _, err := svc.DeleteImage(image.CoinID, image.ID, 2); !errors.Is(err, ErrCoinNotFound) {
				t.Fatal("different owner could retry cleanup")
			}
			if _, err := os.Stat(unrelated); err != nil {
				t.Fatal("unrelated file removed")
			}
			if err := os.Remove(unrelated); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			if restart {
				fresh := NewImageService(repository.NewImageRepository(db), dir)
				if err := fresh.RetryPendingCleanup(); err != nil {
					t.Fatal(err)
				}
			} else if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); err != nil {
				t.Fatal(err)
			}
			db.Model(&models.ImageCleanup{}).Count(&count)
			if count != 0 {
				t.Fatal("completed intent retained")
			}
			for _, name := range []string{"fixture.jpg", "fixture_thumb.jpg", "fixture_medium.jpg"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("%s not cleaned", name)
				}
			}
			if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); !errors.Is(err, ErrImageNotFound) {
				t.Fatal(err)
			}
			if err := svc.RetryPendingCleanup(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestImageCleanupAcknowledgementFailureIsRetryable(t *testing.T) {
	db, svc, image, _ := deletionFixture(t)
	if err := db.Callback().Delete().Before("gorm:delete").Register("reject_ack", func(tx *gorm.DB) {
		if tx.Statement.Table == "image_cleanups" {
			tx.AddError(errors.New("injected acknowledgement failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); !errors.Is(err, ErrImageCleanupPending) {
		t.Fatal(err)
	}
	if err := db.Callback().Delete().Remove("reject_ack"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); err != nil {
		t.Fatal(err)
	}
}

func TestImageCleanupRejectsEscapingPath(t *testing.T) {
	db, svc, image, dir := deletionFixture(t)
	if err := db.Model(&image).Update("file_path", "../outside.jpg").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteImage(image.CoinID, image.ID, 1); !errors.Is(err, ErrImageCleanupPending) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fixture.jpg")); err != nil {
		t.Fatal("unrelated original removed")
	}
}
