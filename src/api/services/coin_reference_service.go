package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrReferenceCatalogRequired = errors.New("catalog is required")
	ErrReferenceNumberRequired  = errors.New("number is required")
	ErrReferenceVolumeRequired  = errors.New("volume is required for this catalog")
	ErrReferenceUnknownCatalog  = errors.New("catalog is not supported")
	ErrReferenceDuplicate       = errors.New("duplicate references are not allowed")
)

// CoinReferenceService validates and normalizes structured references.
type CoinReferenceService struct {
	repo         *repository.CoinReferenceRepository
	registryRepo *repository.CatalogRegistryRepository
}

// NewCoinReferenceService creates a new CoinReferenceService.
func NewCoinReferenceService(
	repo *repository.CoinReferenceRepository,
	registryRepo *repository.CatalogRegistryRepository,
) *CoinReferenceService {
	return &CoinReferenceService{
		repo:         repo,
		registryRepo: registryRepo,
	}
}

// NormalizeAndValidate normalizes a reference list and validates catalog rules.
func (s *CoinReferenceService) NormalizeAndValidate(
	refs []models.CoinReference,
) ([]models.CoinReference, error) {
	normalized := make([]models.CoinReference, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))

	for _, ref := range refs {
		n, err := s.NormalizeAndValidateOne(ref)
		if err != nil {
			return nil, err
		}

		key := dedupeKey(n)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf(
				"%w: catalog=%s volume=%s number=%s",
				ErrReferenceDuplicate, n.Catalog, n.Volume, n.Number,
			)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, n)
	}

	return normalized, nil
}

// NormalizeAndValidateOne validates a single reference against registry rules.
func (s *CoinReferenceService) NormalizeAndValidateOne(
	ref models.CoinReference,
) (models.CoinReference, error) {
	ref.Catalog = strings.ToUpper(strings.TrimSpace(ref.Catalog))
	ref.Volume = strings.TrimSpace(ref.Volume)
	ref.Number = strings.TrimSpace(ref.Number)
	ref.InvoiceNumber = strings.TrimSpace(ref.InvoiceNumber)
	ref.URI = strings.TrimSpace(ref.URI)

	if ref.Catalog == "" {
		return ref, ErrReferenceCatalogRequired
	}
	if ref.Number == "" {
		return ref, ErrReferenceNumberRequired
	}

	registry, err := s.registryRepo.FindByCatalog(ref.Catalog)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ref, fmt.Errorf("%w: %s", ErrReferenceUnknownCatalog, ref.Catalog)
		}
		return ref, err
	}
	if registry.VolumeRequired && ref.Volume == "" {
		return ref, fmt.Errorf("%w: %s", ErrReferenceVolumeRequired, ref.Catalog)
	}

	return ref, nil
}

// ReplaceForCoin validates and then replaces all references for a coin.
func (s *CoinReferenceService) ReplaceForCoin(
	coinID uint,
	userID uint,
	refs []models.CoinReference,
) error {
	normalized, err := s.NormalizeAndValidate(refs)
	if err != nil {
		return err
	}
	return s.repo.ReplaceForCoin(coinID, userID, normalized)
}

func dedupeKey(ref models.CoinReference) string {
	return strings.ToUpper(strings.TrimSpace(ref.Catalog)) + "|" +
		strings.ToUpper(strings.TrimSpace(ref.Volume)) + "|" +
		strings.ToUpper(strings.TrimSpace(ref.Number))
}
