package database

import (
	"log"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dbPath string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Enable WAL mode for better concurrent performance
	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA foreign_keys=ON")

	err = DB.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{}, &models.ApiKey{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.AgentConversation{}, &models.Follow{}, &models.CoinComment{}, &models.CoinValueHistory{}, &models.AuctionLot{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.Notification{}, &models.Tag{}, &models.CoinTag{}, &models.Showcase{}, &models.ShowcaseCoin{}, &models.AuctionEvent{}, &models.PriceAlert{}, &models.BidReminder{}, &models.ValuationRun{}, &models.ValuationResult{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database connected and migrated")
}
