package services

import (
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

// ReferenceMigrationService handles legacy rarity_rating → structured CoinReference migration.
type ReferenceMigrationService struct {
	db           *gorm.DB
	coinRefRepo  *repository.CoinReferenceRepository
	registryRepo *repository.CatalogRegistryRepository
	journalRepo  *repository.JournalRepository
}

// NewReferenceMigrationService creates a new ReferenceMigrationService.
func NewReferenceMigrationService(
	db *gorm.DB,
	coinRefRepo *repository.CoinReferenceRepository,
	registryRepo *repository.CatalogRegistryRepository,
	journalRepo *repository.JournalRepository,
) *ReferenceMigrationService {
	return &ReferenceMigrationService{
		db:           db,
		coinRefRepo:  coinRefRepo,
		registryRepo: registryRepo,
		journalRepo:  journalRepo,
	}
}

// MigrationResult tracks counts for a migration run.
type MigrationResult struct {
	Succeeded int `json:"succeeded"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

// MigrateLegacyReferences processes legacy rarity_rating fields for a single user's coins.
func (s *ReferenceMigrationService) MigrateLegacyReferences(userID uint) (*MigrationResult, error) {
	result := &MigrationResult{}

	var coins []struct {
		ID           uint
		UserID       uint
		RarityRating string
	}
	if err := s.db.Model(&models.Coin{}).
		Select("id, user_id, rarity_rating").
		Where("user_id = ?", userID).
		Where("TRIM(rarity_rating) <> ''").
		Find(&coins).Error; err != nil {
		return nil, err
	}

	registry := make(map[string]*models.CatalogRegistry)
	var catalogList []models.CatalogRegistry
	if err := s.db.Find(&catalogList).Error; err != nil {
		return nil, err
	}
	for i := range catalogList {
		registry[catalogList[i].Catalog] = &catalogList[i]
	}

	for _, coin := range coins {
		ref, needsJournal, logMsg := s.parseLegacyReference(coin.RarityRating, registry)
		if ref == nil {
			if logMsg == "" {
				s.journalSkip(coin.ID, coin.UserID, "No parseable reference in rarity_rating field")
			} else if strings.Contains(logMsg, "unrecognized catalog") || strings.Contains(logMsg, "not in registry") {
				s.journalSkip(coin.ID, coin.UserID, "Skipped legacy reference migration: "+logMsg)
			} else {
				s.journalFail(coin.ID, coin.UserID, "Failed to parse legacy reference: "+logMsg)
				result.Failed++
				continue
			}
			result.Skipped++
			continue
		}

		ref.CoinID = coin.ID

		var existing models.CoinReference
		err := s.db.Where("coin_id = ? AND catalog = ? AND volume = ? AND number = ?",
			ref.CoinID, ref.Catalog, ref.Volume, ref.Number).
			First(&existing).Error
		if err == nil {
			s.journalSkip(coin.ID, coin.UserID, "Already has matching reference: "+s.formatReference(ref))
			result.Skipped++
			continue
		}
		if err != nil && !repository.IsRecordNotFound(err) {
			s.journalFail(coin.ID, coin.UserID, "Database error checking existing reference")
			result.Failed++
			continue
		}

		if err := s.db.Create(ref).Error; err != nil {
			s.journalFail(coin.ID, coin.UserID, "Failed to create reference: "+err.Error())
			result.Failed++
			continue
		}

		successMsg := "Legacy reference migrated: " + coin.RarityRating + " → " + s.formatReference(ref)
		s.journalSuccess(coin.ID, coin.UserID, successMsg)
		result.Succeeded++

		if needsJournal {
			s.journalManualReview(coin.ID, coin.UserID, logMsg)
		}
	}

	return result, nil
}

func (s *ReferenceMigrationService) journalSuccess(coinID, userID uint, msg string) {
	entry := &models.CoinJournal{
		CoinID: coinID,
		UserID: userID,
		Entry:  msg,
	}
	s.journalRepo.CreateEntry(entry)
}

func (s *ReferenceMigrationService) journalSkip(coinID, userID uint, msg string) {
	entry := &models.CoinJournal{
		CoinID: coinID,
		UserID: userID,
		Entry:  msg,
	}
	s.journalRepo.CreateEntry(entry)
}

func (s *ReferenceMigrationService) journalFail(coinID, userID uint, msg string) {
	entry := &models.CoinJournal{
		CoinID: coinID,
		UserID: userID,
		Entry:  msg,
	}
	s.journalRepo.CreateEntry(entry)
}

func (s *ReferenceMigrationService) journalManualReview(coinID, userID uint, msg string) {
	entry := &models.CoinJournal{
		CoinID: coinID,
		UserID: userID,
		Entry:  msg,
	}
	s.journalRepo.CreateEntry(entry)
}

func (s *ReferenceMigrationService) formatReference(ref *models.CoinReference) string {
	parts := []string{"catalog " + ref.Catalog}
	if ref.Volume != "" && ref.Volume != "0" {
		parts = append(parts, "vol "+ref.Volume)
	}
	if ref.Number != "" {
		parts = append(parts, "no. "+ref.Number)
	}
	return strings.Join(parts, ", ")
}

// parseLegacyReference parses the first catalog reference from a legacy rarity_rating string.
// Returns (ref, needsJournal, logMsg) where:
// - ref is nil if parsing failed or catalog not recognized
// - needsJournal is true if a volume=0 sentinel was used
// - logMsg describes what happened (for logging or journal)
//
// The token/volume/number parsing itself is delegated to the shared
// ParseCatalogReferenceText helper (Feature 352 Phase 1,
// catalog_reference_parser.go). The Volume:"0" sentinel and the
// "manual review needed" / unrecognized-catalog / no-number journal
// messages below are migration-specific policy and intentionally stay
// here rather than in the shared helper (FR-016, FR-019).
func (s *ReferenceMigrationService) parseLegacyReference(text string, registry map[string]*models.CatalogRegistry) (*models.CoinReference, bool, string) {
	parsed, found := ParseCatalogReferenceText(text, registry)

	if !found {
		switch parsed.Reason {
		case CatalogParseEmpty:
			// Empty/whitespace-only input, or (practically unreachable, since
			// a non-empty trimmed string always yields at least one token)
			// no parseable tokens at all. Matches the original silent-skip.
			return nil, false, ""
		case CatalogParseUnrecognizedCatalog:
			// Catalog is empty for this reason (see ParsedCatalogReference.Catalog
			// doc); the offending raw token is the first field of RawText.
			catalogToken := strings.ToUpper(strings.Fields(parsed.RawText)[0])
			return nil, false, "unrecognized catalog: " + catalogToken
		case CatalogParseNotInRegistry:
			return nil, false, "catalog not in registry: " + parsed.Catalog
		case CatalogParseNoNumber:
			return nil, false, "no number found in reference: " + parsed.RawText
		default:
			// Unreachable: ParseCatalogReferenceText only returns found=false
			// with one of the reasons handled above.
			return nil, false, ""
		}
	}

	if parsed.NeedsVolume {
		return &models.CoinReference{
				Catalog: parsed.Catalog,
				Volume:  "0",
				Number:  "",
			}, true,
			"Legacy " + parsed.Catalog + " reference imported with placeholder volume 0 — manual review needed: " + parsed.RawText
	}

	return &models.CoinReference{
		Catalog: parsed.Catalog,
		Volume:  parsed.Volume,
		Number:  parsed.Number,
	}, false, ""
}
