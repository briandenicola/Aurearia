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

	err = DB.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.MintLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{}, &models.ApiKey{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.SecurityEvent{}, &models.IPRule{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.Note{}, &models.CoinIntakeDraft{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.DraftLifecycleEvent{}, &models.AgentConversation{}, &models.CollectionUpdateProposal{}, &models.Follow{}, &models.CoinComment{}, &models.CoinValueHistory{}, &models.AuctionLot{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{}, &models.Notification{}, &models.AIJob{}, &models.Tag{}, &models.CoinTag{}, &models.CoinSet{}, &models.CoinSetMembership{}, &models.CoinSetTarget{}, &models.CoinSetValuationSnapshot{}, &models.CoinSetMilestoneAlert{}, &models.SmartCriteriaTemplate{}, &models.Showcase{}, &models.ShowcaseCoin{}, &models.AuctionEvent{}, &models.PriceAlert{}, &models.BidReminder{}, &models.AuctionAlertRun{}, &models.ValuationRun{}, &models.ValuationResult{}, &models.AuctionEndingRun{}, &models.AuctionWatchBidDigestRun{}, &models.FeaturedCoin{}, &models.CoinOfDayRun{}, &models.CollectionHealthSnapshot{}, &models.CollectionHealthSnapshotRun{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	if err := DB.Migrator().AlterColumn(&models.User{}, "NumisBidsPassword"); err != nil {
		log.Fatalf("Failed to widen NumisBids password column: %v", err)
	}
	if err := DB.Migrator().AlterColumn(&models.User{}, "CNGPassword"); err != nil {
		log.Fatalf("Failed to widen CNG password column: %v", err)
	}

	// Note: CurrentValueUpdatedAt is a new nullable time.Time column.
	// SQLite AutoMigrate adds it as a plain NULL column without FK constraints — safe additive change.

	// Backfill existing api_keys with default read-only capability
	DB.Exec("UPDATE api_keys SET capabilities='read' WHERE capabilities IS NULL OR capabilities=''")
	DB.Exec("UPDATE auction_lots SET source='numisbids' WHERE source IS NULL OR source=''")
	DB.Exec("UPDATE auction_lots SET source_url=numis_bids_url WHERE (source_url IS NULL OR source_url='') AND numis_bids_url IS NOT NULL AND numis_bids_url<>''")

	if err := seedCatalogRegistry(DB); err != nil {
		log.Fatalf("Failed to seed catalog registry: %v", err)
	}
	if err := seedMintLocations(DB); err != nil {
		log.Fatalf("Failed to seed mint locations: %v", err)
	}

	log.Println("Database connected and migrated")
}

const mintLocationSeedVersionKey = "MintLocationSeedVersion"
const currentMintLocationSeedVersion = "1"

func seedMintLocations(db *gorm.DB) error {
	var existingSetting models.AppSetting
	err := db.First(&existingSetting, "key = ?", mintLocationSeedVersionKey).Error
	if err == nil && existingSetting.Value == currentMintLocationSeedVersion {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	seed := []models.MintLocation{
		{DisplayName: "Rome", Lat: 41.9028, Lng: 12.4964, Region: "Italy", Aliases: models.StringList{"Roma", "Rome mint"}},
		{DisplayName: "Athens", Lat: 37.9838, Lng: 23.7275, Region: "Greece", Aliases: models.StringList{"Athenai", "Athenae", "Athens mint"}},
		{DisplayName: "Constantinople", Lat: 41.0082, Lng: 28.9784, Region: "Thrace", Aliases: models.StringList{"Byzantium", "Istanbul", "Constantinopolis", "Constantinople mint"}},
		{DisplayName: "Alexandria", Lat: 31.2001, Lng: 29.9187, Region: "Egypt", Aliases: models.StringList{"Alexandria Egypt", "Alexandria ad Aegyptum"}},
		{DisplayName: "Antioch", Lat: 36.2021, Lng: 36.1603, Region: "Syria", Aliases: models.StringList{"Antiochia", "Antioch on the Orontes", "Antioch mint"}},
		{DisplayName: "Syracuse", Lat: 37.0755, Lng: 15.2866, Region: "Sicily", Aliases: models.StringList{"Syracusa", "Syracuse Sicily"}},
		{DisplayName: "Trier", Lat: 49.7499, Lng: 6.6371, Region: "Gaul", Aliases: models.StringList{"Treveri", "Treves", "Augusta Treverorum"}},
		{DisplayName: "Lugdunum", Lat: 45.764, Lng: 4.8357, Region: "Gaul", Aliases: models.StringList{"Lyon", "Lyons", "Lugdunum Lyon"}},
		{DisplayName: "Siscia", Lat: 45.4872, Lng: 16.376, Region: "Pannonia", Aliases: models.StringList{"Sisak", "Siscia mint"}},
		{DisplayName: "Nicomedia", Lat: 40.7654, Lng: 29.9408, Region: "Bithynia", Aliases: models.StringList{"Nikomedia", "Izmit"}},
		{DisplayName: "Cyzicus", Lat: 40.3991, Lng: 27.7936, Region: "Mysia", Aliases: models.StringList{"Kyzikos", "Cyzicus mint"}},
		{DisplayName: "Carthage", Lat: 36.8528, Lng: 10.3233, Region: "Africa", Aliases: models.StringList{"Carthago", "Qart Hadasht"}},
		{DisplayName: "Thessalonica", Lat: 40.6401, Lng: 22.9444, Region: "Macedonia", Aliases: models.StringList{"Thessalonika", "Thessaloniki"}},
		{DisplayName: "Heraclea", Lat: 41.2797, Lng: 27.9553, Region: "Thrace", Aliases: models.StringList{"Heraclea Thraciae", "Herakleia"}},
		{DisplayName: "Aquileia", Lat: 45.7686, Lng: 13.3678, Region: "Italy", Aliases: models.StringList{"Aquileia mint"}},
		{DisplayName: "Arelate", Lat: 43.6766, Lng: 4.6278, Region: "Gaul", Aliases: models.StringList{"Arles", "Arelate Arles", "Constantina"}},
		{DisplayName: "Ephesus", Lat: 37.9393, Lng: 27.3416, Region: "Ionia", Aliases: models.StringList{"Ephesos", "Ephesus mint"}},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range seed {
			entry.NormalizedName = models.NormalizeMintLocationName(entry.DisplayName)
			var existing models.MintLocation
			err := tx.Where("normalized_name = ?", entry.NormalizedName).First(&existing).Error
			if err == nil {
				updates := map[string]interface{}{
					"display_name": entry.DisplayName,
					"lat":          entry.Lat,
					"lng":          entry.Lng,
					"region":       entry.Region,
					"aliases":      entry.Aliases,
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		return tx.Save(&models.AppSetting{Key: mintLocationSeedVersionKey, Value: currentMintLocationSeedVersion}).Error
	})
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
