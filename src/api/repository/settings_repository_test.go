package repository

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSettingsRepositoryExpectedMissingRowsAreNotLoggedAsWarnings(t *testing.T) {
	var output bytes.Buffer
	dbLogger := logger.New(log.New(&output, "", 0), logger.Config{LogLevel: logger.Warn})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: dbLogger})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatal(err)
	}

	repo := NewSettingsRepository(db)
	if _, err := repo.FindByKey("MissingSetting"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindByKey error = %v, want record not found", err)
	}
	if err := repo.Upsert("NewSetting", "value"); err != nil {
		t.Fatalf("Upsert new setting: %v", err)
	}
	if strings.Contains(output.String(), "record not found") {
		t.Fatalf("expected missing settings must not be logged as warnings: %s", output.String())
	}
}
