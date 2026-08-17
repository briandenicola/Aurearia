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
	// Root cause of "database is locked (SQLITE_BUSY)" on job claim: every
	// GORM db.Transaction() begins a *deferred* SQLite transaction, which
	// takes no lock until its first statement, and only a *read* lock on
	// the initial SELECT. When the later UPDATE in the same transaction
	// tries to upgrade that read lock to a write lock, and a concurrent
	// writer has taken the write lock (or already committed a change to
	// the WAL) in the meantime, SQLite has no choice but to fail the
	// upgrade immediately (SQLITE_BUSY / SQLITE_BUSY_SNAPSHOT) - a
	// deferred transaction cannot "wait" its way out of a lock upgrade
	// conflict, because waiting could deadlock two transactions that each
	// hold the other's needed read lock. This is exactly the
	// SELECT-then-UPDATE shape of ClaimNextQueuedJob, and is why it
	// reproduces reliably under concurrent claims (verified with a
	// standalone probe: 30 goroutines racing 300 claims against a real
	// on-disk WAL db hit ~29 SQLITE_BUSY/SQLITE_BUSY_SNAPSHOT errors).
	//
	// busy_timeout alone (a PRAGMA that only makes an *outright lock
	// acquisition* wait, not a lock-upgrade conflict) measurably did NOT
	// fix this in the same probe (still ~29 errors) - it is necessary but
	// not sufficient here, confirming the deferred-transaction upgrade
	// race, not a slow writer, is the true cause.
	//
	// The fix is `_txlock=immediate`: every transaction acquires SQLite's
	// write lock (RESERVED) at BEGIN time instead of deferring and later
	// trying to upgrade, so competing writers simply queue for the lock
	// instead of racing to upgrade an existing read lock. Combined with
	// busy_timeout so that queuing waits (default 0ms) rather than
	// failing immediately, this eliminated all errors in the same probe
	// (0/300 failures, all claims eventually succeeded).
	//
	// Both settings are per-connection SQLite driver options, not
	// something a one-off PRAGMA exec after Open can guarantee across a
	// pooled *sql.DB (each pooled connection needs it applied as it is
	// opened), so they are encoded in the DSN rather than exec'd once.
	//
	// This is a global change: every scheduler in this process (deep
	// identification, valuation, health, auction, coin-of-day, shipments,
	// wishlist, ...) shares this one *gorm.DB/*sql.DB. Making every write
	// transaction acquire its lock immediately plus wait up to 5s under
	// contention is strictly safer for all of them than the previous
	// fail-fast default - none of those schedulers currently retry a bare
	// write failure themselves, and none holds a transaction open long
	// enough for a 5s busy_timeout to introduce a meaningful stall.
	dsn := dbPath + "?_txlock=immediate&_pragma=busy_timeout(5000)"
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
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

	// Note: MintLocation.NomismaURI/-Label/-LinkedAt are new nullable/
	// optional columns (343-nomisma-mint-authority-linking). SQLite
	// AutoMigrate adds them additively with no backfill and no destructive
	// migration - every existing row simply starts unlinked.
	err = DB.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.MintLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.CatalogRegistry{}, &models.AppSetting{}, &models.ApiKey{}, &models.RefreshToken{}, &models.WebAuthnCredential{}, &models.SecurityEvent{}, &models.IPRule{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.Note{}, &models.CoinIntakeDraft{}, &models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{}, &models.AgentConversation{}, &models.CollectionUpdateProposal{}, &models.SetBuilderRun{}, &models.SetProposal{}, &models.ProposalSlot{}, &models.Follow{}, &models.CoinComment{}, &models.CoinValueHistory{}, &models.Shipment{}, &models.ShipmentEvent{}, &models.AuctionLot{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{}, &models.Notification{}, &models.AIJob{}, &models.Tag{}, &models.CoinTag{}, &models.CoinSet{}, &models.CoinSetMembership{}, &models.CoinSetTarget{}, &models.CoinSetValuationSnapshot{}, &models.CoinSetMilestoneAlert{}, &models.SmartCriteriaTemplate{}, &models.CoinRecommendation{}, &models.RecommendationFeedback{}, &models.Showcase{}, &models.ShowcaseCoin{}, &models.AuctionEvent{}, &models.PriceAlert{}, &models.BidReminder{}, &models.AuctionAlertRun{}, &models.ValuationRun{}, &models.ValuationResult{}, &models.AuctionEndingRun{}, &models.AuctionWatchBidDigestRun{}, &models.FeaturedCoin{}, &models.CoinOfDayRun{}, &models.CollectionHealthSnapshot{}, &models.CollectionHealthSnapshotRun{}, &models.RomanImperialFigure{}, &models.RomanImperialFigureHighlight{}, &models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{}, &models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	// 344-deep-agentic-coin-identification: partial unique index enforcing
	// at most one obverse and one reverse artifact per job. Hint artifacts
	// (role='hint') are explicitly excluded - up to 3 are allowed per job,
	// enforced in service-layer validation instead (data-model.md §5).
	if err := DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uix_deep_artifact_job_role ON deep_identification_artifacts(job_id, role) WHERE role <> 'hint'`).Error; err != nil {
		log.Fatalf("Failed to create deep identification artifact role index: %v", err)
	}
	if err := DB.Migrator().AlterColumn(&models.User{}, "NumisBidsPassword"); err != nil {
		log.Fatalf("Failed to widen NumisBids password column: %v", err)
	}
	if err := DB.Migrator().AlterColumn(&models.User{}, "CNGPassword"); err != nil {
		log.Fatalf("Failed to widen CNG password column: %v", err)
	}
	if err := migrateCoinSetTypes(DB); err != nil {
		log.Fatalf("Failed to migrate coin set types: %v", err)
	}
	if err := backfillVendorInvoiceFromCoinReferences(DB); err != nil {
		log.Fatalf("Failed to backfill vendor invoice values: %v", err)
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
	if err := backfillCoinMintLocations(DB); err != nil {
		log.Fatalf("Failed to backfill coin mint locations: %v", err)
	}
	if err := seedRomanImperialFigures(DB); err != nil {
		log.Fatalf("Failed to seed Roman imperial figures: %v", err)
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

const coinMintLocationBackfillVersionKey = "CoinMintLocationBackfillVersion"
const currentCoinMintLocationBackfillVersion = "1"

// backfillCoinMintLocations links existing coins' free-text Mint values to
// a MintLocation (global or the coin owner's own private one) whenever the
// normalized text exactly matches a display name or alias. Coins with no
// match are left untouched (never auto-creates a mint location) - the user
// gets a chance to link or create one via the unlinked-mint nudge instead.
// Idempotent and versioned, run once per version bump like seedMintLocations.
func backfillCoinMintLocations(db *gorm.DB) error {
	var existingSetting models.AppSetting
	err := db.First(&existingSetting, "key = ?", coinMintLocationBackfillVersionKey).Error
	if err == nil && existingSetting.Value == currentCoinMintLocationBackfillVersion {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	var locations []models.MintLocation
	if err := db.Find(&locations).Error; err != nil {
		return err
	}

	type lookupKey struct {
		ownerUserID uint // 0 means global (visible to everyone)
		normalized  string
	}
	lookup := make(map[lookupKey]uint)
	for _, loc := range locations {
		var owner uint
		if loc.UserID != nil {
			owner = *loc.UserID
		}
		for key := range mintLocationBackfillKeys(loc) {
			lookup[lookupKey{ownerUserID: owner, normalized: key}] = loc.ID
		}
	}

	var coins []models.Coin
	if err := db.Where("mint_location_id IS NULL AND mint <> ''").Find(&coins).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, coin := range coins {
			normalized := models.NormalizeMintLocationName(coin.Mint)
			if normalized == "" {
				continue
			}
			matchID, ok := lookup[lookupKey{ownerUserID: 0, normalized: normalized}]
			if !ok {
				matchID, ok = lookup[lookupKey{ownerUserID: coin.UserID, normalized: normalized}]
			}
			if !ok {
				continue
			}
			if err := tx.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("mint_location_id", matchID).Error; err != nil {
				return err
			}
		}
		return tx.Save(&models.AppSetting{Key: coinMintLocationBackfillVersionKey, Value: currentCoinMintLocationBackfillVersion}).Error
	})
}

func mintLocationBackfillKeys(loc models.MintLocation) map[string]bool {
	keys := make(map[string]bool, len(loc.Aliases)+1)
	name := loc.NormalizedName
	if name == "" {
		name = models.NormalizeMintLocationName(loc.DisplayName)
	}
	if name != "" {
		keys[name] = true
	}
	for _, alias := range loc.Aliases {
		if n := models.NormalizeMintLocationName(alias); n != "" {
			keys[n] = true
		}
	}
	return keys
}

func migrateCoinSetTypes(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE coin_sets SET set_type='goal' WHERE LOWER(set_type)='defined'").Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE coin_sets SET set_type='standard' WHERE LOWER(set_type)='open'").Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE coin_sets SET set_type='agentic', creation_mode='dynamic' WHERE LOWER(set_type) IN ('dynamic', 'tracker')").Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE coin_sets SET creation_mode='manual' WHERE creation_mode IS NULL OR creation_mode=''").Error; err != nil {
			return err
		}
		return nil
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

const vendorInvoiceBackfillVersionKey = "VendorInvoiceBackfillVersion"
const currentVendorInvoiceBackfillVersion = "1"

// backfillVendorInvoiceFromCoinReferences copies existing invoice_number values
// from structured coin references into the new top-level coins.vendor_invoice
// field when vendor_invoice is blank. It is idempotent and versioned.
func backfillVendorInvoiceFromCoinReferences(db *gorm.DB) error {
	var existingSetting models.AppSetting
	err := db.First(&existingSetting, "key = ?", vendorInvoiceBackfillVersionKey).Error
	if err == nil && existingSetting.Value == currentVendorInvoiceBackfillVersion {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// Fresh installs created from the new model no longer include invoice_number
	// on coin_references, so there is nothing to backfill.
	var columns []struct {
		Name string
	}
	if err := db.Raw("PRAGMA table_info(coin_references)").Scan(&columns).Error; err != nil {
		return err
	}
	hasInvoiceNumber := false
	for _, col := range columns {
		if col.Name == "invoice_number" {
			hasInvoiceNumber = true
			break
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if hasInvoiceNumber {
			if err := tx.Exec(`
				UPDATE coins
				SET vendor_invoice = (
					SELECT cr.invoice_number
					FROM coin_references cr
					WHERE cr.coin_id = coins.id
						AND cr.invoice_number IS NOT NULL
						AND TRIM(cr.invoice_number) <> ''
					ORDER BY cr.updated_at DESC, cr.id DESC
					LIMIT 1
				)
				WHERE (vendor_invoice IS NULL OR TRIM(vendor_invoice) = '')
					AND EXISTS (
						SELECT 1
						FROM coin_references cr
						WHERE cr.coin_id = coins.id
							AND cr.invoice_number IS NOT NULL
							AND TRIM(cr.invoice_number) <> ''
					)
			`).Error; err != nil {
				return err
			}
		}
		return tx.Save(&models.AppSetting{
			Key:   vendorInvoiceBackfillVersionKey,
			Value: currentVendorInvoiceBackfillVersion,
		}).Error
	})
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
