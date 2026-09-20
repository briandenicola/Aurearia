package main

import (
	"path/filepath"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestValidateExplorationDatabasePath(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "data", "exploration.db")
	if err := validateExplorationDatabasePath(valid, root); err != nil {
		t.Fatalf("ephemeral path rejected: %v", err)
	}
	for _, unsafe := range []string{"", "ancientcoins.db", filepath.Join(root, "..", "production.db")} {
		if err := validateExplorationDatabasePath(unsafe, root); err == nil {
			t.Fatalf("unsafe path accepted: %q", unsafe)
		}
	}
}

func TestSeedCreatesUniqueAccountAndGoldenAssociations(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "exploration.db")
	result, err := seedExplorationDatabase(path, root, "explorer-one", "explorer-one@example.test", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := result.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if result.UserID == 0 || result.CoinCount == 0 {
		t.Fatalf("unexpected seed result: %+v", result)
	}
	if _, err := seedExplorationDatabase(path, root, "explorer-one", "other@example.test", "test-password-123"); err == nil {
		t.Fatal("duplicate account was accepted")
	}
	var associations int64
	if err := result.DB.Model(&models.CoinSetMembership{}).Count(&associations).Error; err != nil {
		t.Fatal(err)
	}
	if associations == 0 {
		t.Fatal("golden set associations were not persisted")
	}
}
