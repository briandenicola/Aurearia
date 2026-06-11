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

	// Migrate certainty → invoice_number column in coin_references (idempotent)
	if err := migrateCoinReferenceCertaintyColumn(DB); err != nil {
		log.Fatalf("Failed to migrate coin_references column: %v", err)
	}

	err = DB.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{}, &models.ApiKey{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.Note{}, &models.CoinIntakeDraft{}, &models.AgentConversation{}, &models.CollectionUpdateProposal{}, &models.Follow{}, &models.CoinComment{}, &models.CoinValueHistory{}, &models.AuctionLot{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.Notification{}, &models.Tag{}, &models.CoinTag{}, &models.CoinSet{}, &models.CoinSetMembership{}, &models.CoinSetTarget{}, &models.CoinSetValuationSnapshot{}, &models.CoinSetMilestoneAlert{}, &models.Showcase{}, &models.ShowcaseCoin{}, &models.AuctionEvent{}, &models.PriceAlert{}, &models.BidReminder{}, &models.ValuationRun{}, &models.ValuationResult{}, &models.AuctionEndingRun{}, &models.FeaturedCoin{}, &models.CollectionHealthSnapshot{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Note: CurrentValueUpdatedAt is a new nullable time.Time column.
	// SQLite AutoMigrate adds it as a plain NULL column without FK constraints — safe additive change.

	// Backfill existing api_keys with default read-only capability
	DB.Exec("UPDATE api_keys SET capabilities='read' WHERE capabilities IS NULL OR capabilities=''")

	if err := seedCatalogRegistry(DB); err != nil {
		log.Fatalf("Failed to seed catalog registry: %v", err)
	}

	log.Println("Database connected and migrated")
}

// migrateCoinReferenceCertaintyColumn renames certainty → invoice_number if needed (idempotent).
func migrateCoinReferenceCertaintyColumn(db *gorm.DB) error {
	var columns []struct {
		Name string
	}
	if err := db.Raw("PRAGMA table_info(coin_references)").Scan(&columns).Error; err != nil {
		// Table doesn't exist yet — fresh install, nothing to migrate
		return nil
	}

	hasCertainty := false
	hasInvoiceNumber := false
	for _, col := range columns {
		if col.Name == "certainty" {
			hasCertainty = true
		}
		if col.Name == "invoice_number" {
			hasInvoiceNumber = true
		}
	}

	// Rename only if old column exists and new one doesn't
	if hasCertainty && !hasInvoiceNumber {
		if err := db.Exec("ALTER TABLE coin_references RENAME COLUMN certainty TO invoice_number").Error; err != nil {
			return err
		}
		log.Println("Migrated coin_references.certainty → invoice_number")
	}

	return nil
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
		{Catalog: "PRICE", DisplayName: "Price (Coinage of Alexander the Great)", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "BM", DisplayName: "British Museum Catalogue", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "VENÈRA", DisplayName: "La Venèra Hoard", Era: models.EraAncient, VolumeRequired: false},
		{Catalog: "NGC", DisplayName: "NGC Certification", Era: models.EraModern, VolumeRequired: false},
		{Catalog: "Numista", DisplayName: "Numista", Era: models.EraModern, VolumeRequired: false},
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
