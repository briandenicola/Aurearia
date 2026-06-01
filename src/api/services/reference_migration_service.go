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
		ref.Certainty = "legacy-import"

		var existing models.CoinReference
		err := s.db.Where("coin_id = ? AND catalog = ? AND volume = ? AND number = ?",
			ref.CoinID, ref.Catalog, ref.Volume, ref.Number).
			First(&existing).Error
		if err == nil {
			s.journalSkip(coin.ID, coin.UserID, "Already has matching reference: "+s.formatReference(ref))
			result.Skipped++
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
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
func (s *ReferenceMigrationService) parseLegacyReference(text string, registry map[string]*models.CatalogRegistry) (*models.CoinReference, bool, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false, ""
	}

	parts := strings.SplitN(text, ";", 2)
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return nil, false, ""
	}

	tokens := strings.Fields(first)
	if len(tokens) == 0 {
		return nil, false, "unparseable legacy reference"
	}

	catalogToken := strings.ToUpper(tokens[0])
	catalogNormalized := s.normalizeCatalogAlias(catalogToken)
	if catalogNormalized == "" {
		return nil, false, "unrecognized catalog: " + catalogToken
	}

	regEntry, ok := registry[catalogNormalized]
	if !ok {
		return nil, false, "catalog not in registry: " + catalogNormalized
	}

	var volume, number string
	remaining := tokens[1:]

	if regEntry.VolumeRequired {
		if len(remaining) == 0 {
			origText := text
			if len(parts) > 1 {
				origText = first
			}
			return &models.CoinReference{
					Catalog: catalogNormalized,
					Volume:  "0",
					Number:  "",
				}, true,
				"Legacy " + catalogNormalized + " reference imported with placeholder volume 0 — manual review needed: " + origText
		}

		volCandidate := remaining[0]
		if s.isRomanNumeral(volCandidate) || s.isPlausibleVolumeToken(volCandidate) {
			volume = volCandidate
			remaining = remaining[1:]

			if len(remaining) == 0 {
				origText := text
				if len(parts) > 1 {
					origText = first
				}
				return &models.CoinReference{
						Catalog: catalogNormalized,
						Volume:  "0",
						Number:  "",
					}, true,
					"Legacy " + catalogNormalized + " reference imported with placeholder volume 0 — manual review needed: " + origText
			}
		} else {
			origText := text
			if len(parts) > 1 {
				origText = first
			}
			return &models.CoinReference{
					Catalog: catalogNormalized,
					Volume:  "0",
					Number:  "",
				}, true,
				"Legacy " + catalogNormalized + " reference imported with placeholder volume 0 — manual review needed: " + origText
		}
	} else {
		if len(remaining) == 0 {
			origText := text
			if len(parts) > 1 {
				origText = first
			}
			return nil, false, "no number found in reference: " + origText
		}
	}

	number = strings.Join(remaining, " ")

	return &models.CoinReference{
		Catalog: catalogNormalized,
		Volume:  volume,
		Number:  number,
	}, false, ""
}

// normalizeCatalogAlias maps known aliases to canonical catalog codes.
func (s *ReferenceMigrationService) normalizeCatalogAlias(token string) string {
	upper := strings.ToUpper(token)
	switch upper {
	case "RIC", "RPC", "SNG", "CRAWFORD", "CNI", "KM", "Y", "CRAIG", "REDBOOK":
		return upper
	case "SEAR", "SRCV":
		return "SEAR"
	case "SPINK":
		return "SPINK"
	case "DUPLESSY":
		return "DUPLESSY"
	default:
		return ""
	}
}

// isRomanNumeral checks if a token is a valid Roman numeral.
func (s *ReferenceMigrationService) isRomanNumeral(str string) bool {
	str = strings.ToUpper(str)
	for _, ch := range str {
		if ch != 'I' && ch != 'V' && ch != 'X' && ch != 'L' && ch != 'C' && ch != 'D' && ch != 'M' {
			return false
		}
	}
	return len(str) > 0
}

// isPlausibleVolumeToken checks if a token looks like it could be a volume (not purely numeric like a catalog number).
// Accepts Roman numerals, short numeric strings (1-3 digits), or alphabetic tokens (e.g. "Cop" for SNG Copenhagen).
func (s *ReferenceMigrationService) isPlausibleVolumeToken(str string) bool {
	if len(str) == 0 {
		return false
	}

	if s.isRomanNumeral(str) {
		return true
	}

	if len(str) <= 3 {
		allDigits := true
		for _, ch := range str {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}

	allLetters := true
	for _, ch := range str {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
			allLetters = false
			break
		}
	}
	return allLetters
}
