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

	err = DB.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{}, &models.ApiKey{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.AgentConversation{}, &models.CollectionUpdateProposal{}, &models.Follow{}, &models.CoinComment{}, &models.CoinValueHistory{}, &models.AuctionLot{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.Notification{}, &models.Tag{}, &models.CoinTag{}, &models.Showcase{}, &models.ShowcaseCoin{}, &models.AuctionEvent{}, &models.PriceAlert{}, &models.BidReminder{}, &models.ValuationRun{}, &models.ValuationResult{}, &models.AuctionEndingRun{}, &models.FeaturedCoin{}, &models.CollectionHealthSnapshot{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := seedCatalogRegistry(DB); err != nil {
		log.Fatalf("Failed to seed catalog registry: %v", err)
	}

	log.Println("Database connected and migrated")
}

func seedCatalogRegistry(db *gorm.DB) error {
	seed := []models.CatalogRegistry{
		{Catalog: "RIC", DisplayName: "Roman Imperial Coinage", Era: models.EraAncient, VolumeRequired: true},
		{Catalog: "RPC", DisplayName: "Roman Provincial Coinage", Era: models.EraAncient, VolumeRequired: true},
		{Catalog: "SEAR", DisplayName: "Sear", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "CRAWFORD", DisplayName: "Crawford", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "SNG", DisplayName: "Sylloge Nummorum Graecorum", Era: models.EraAncient, VolumeRequired: true},
		{Catalog: "SPINK", DisplayName: "Spink", Era: models.EraMedieval, VolumeRequired: false},
		{Catalog: "DUPLESSY", DisplayName: "Duplessy", Era: models.EraMedieval, VolumeRequired: false},
		{Catalog: "CNI", DisplayName: "Corpus Nummorum Italicorum", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "KM", DisplayName: "Krause-Mishler", Era: models.EraModern, VolumeRequired: false},
		{Catalog: "Y", DisplayName: "Y Number", Era: models.EraModern, VolumeRequired: false},
		{Catalog: "CRAIG", DisplayName: "Craig", Era: models.EraMedieval, VolumeRequired: false},
		{Catalog: "REDBOOK", DisplayName: "Red Book", Era: models.EraModern, VolumeRequired: false},
	}

	for _, entry := range seed {
		var existing models.CatalogRegistry
		err := db.Where("catalog = ?", entry.Catalog).First(&existing).Error
		if err == nil {
			existing.DisplayName = entry.DisplayName
			existing.Era = entry.Era
			existing.VolumeRequired = entry.VolumeRequired
			if err := db.Save(&existing).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(&entry).Error; err != nil {
			return err
		}
	}

	return nil
}
