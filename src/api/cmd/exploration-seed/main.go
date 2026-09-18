package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/testutil"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type seedResult struct {
	UserID    uint
	CoinCount int
	DB        *gorm.DB
}

func validateExplorationDatabasePath(path, allowedRoot string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(allowedRoot) == "" {
		return fmt.Errorf("database path and ephemeral root are required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absoluteRoot, err := filepath.Abs(allowedRoot)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return fmt.Errorf("database path must be beneath the supplied ephemeral root")
	}
	lower := strings.ToLower(filepath.ToSlash(absolutePath))
	if filepath.Base(lower) == "ancientcoins.db" || strings.Contains(lower, "/production/") || strings.Contains(lower, "/prod/") || strings.Contains(lower, "/beta/") {
		return fmt.Errorf("production-like database path is forbidden")
	}
	return nil
}

func seedExplorationDatabase(path, allowedRoot, username, email, password string) (*seedResult, error) {
	if err := validateExplorationDatabasePath(path, allowedRoot); err != nil {
		return nil, err
	}
	if username == "" || email == "" || len(password) < 12 {
		return nil, fmt.Errorf("dedicated test account values are invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path+"?"+models.SQLiteConcurrencyDSNParams), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open ephemeral database: %w", err)
	}
	succeeded := false
	defer func() {
		if !succeeded {
			if sqlDB, closeErr := db.DB(); closeErr == nil {
				_ = sqlDB.Close()
			}
		}
	}()
	if err := db.AutoMigrate(
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{},
		&models.CoinReference{}, &models.Tag{}, &models.CoinTag{}, &models.CoinSet{},
		&models.CoinSetMembership{},
	); err != nil {
		return nil, fmt.Errorf("migrate ephemeral database: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := models.User{
		Username: username, Email: email, PasswordHash: string(hash), Role: models.RoleAdmin,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("create dedicated test user: %w", err)
	}
	collection, err := testutil.PersistGoldenCollection(db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("persist golden collection: %w", err)
	}
	succeeded = true
	return &seedResult{UserID: user.ID, CoinCount: len(collection.Coins), DB: db}, nil
}

func main() {
	if !strings.EqualFold(os.Getenv("AI_BROWSER_EXPLORATION_ENABLED"), "true") {
		log.Fatal("exploration seed is disabled")
	}
	path := os.Getenv("DB_PATH")
	root := os.Getenv("AI_BROWSER_EPHEMERAL_ROOT")
	result, err := seedExplorationDatabase(
		path,
		root,
		os.Getenv("AI_BROWSER_TEST_USERNAME"),
		os.Getenv("AI_BROWSER_TEST_EMAIL"),
		os.Getenv("AI_BROWSER_TEST_PASSWORD"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("seeded exploration user=%d coins=%d\n", result.UserID, result.CoinCount)
}
