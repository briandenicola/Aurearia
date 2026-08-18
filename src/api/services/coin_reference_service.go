package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
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
		// Strip any client-supplied primary key so CreateBatch always generates a fresh row.
		n.ID = 0
		n.CoinID = 0
		normalized = append(normalized, n)
	}

	return normalized, nil
}

// NormalizeAndValidateOne validates a single reference against registry rules.
func (s *CoinReferenceService) NormalizeAndValidateOne(
	ref models.CoinReference,
) (models.CoinReference, error) {
	ref.Catalog = strings.TrimSpace(ref.Catalog)
	ref.Volume = strings.TrimSpace(ref.Volume)
	ref.Number = strings.TrimSpace(ref.Number)
	ref.URI = strings.TrimSpace(ref.URI)

	if ref.Catalog == "" {
		return ref, ErrReferenceCatalogRequired
	}
	if ref.Number == "" {
		return ref, ErrReferenceNumberRequired
	}

	registry, err := s.registryRepo.FindByCatalog(ref.Catalog)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ref, fmt.Errorf("%w: %s", ErrReferenceUnknownCatalog, ref.Catalog)
		}
		return ref, err
	}
	if registry.VolumeRequired && ref.Volume == "" {
		return ref, fmt.Errorf("%w: %s", ErrReferenceVolumeRequired, ref.Catalog)
	}
	ref.Catalog = registry.Catalog

	return ref, nil
}

// ReplaceForCoin validates and then replaces all references for a coin.
//
// This is owner-editor replacement semantics: it deletes every existing
// reference for the coin before inserting the given set. It is intended for
// the manual "these are my references" editing path only. Agent/enrichment
// paths that discover additional references must use AppendForCoin, which is
// additive and never deletes existing references.
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

// AppendForCoin validates and inserts additional references for a coin
// without deleting any existing ones. Proposed references that duplicate an
// existing reference (or another proposed reference earlier in the list),
// compared case-insensitively on (Catalog, Volume, Number) via dedupeKey, are
// silently skipped rather than rejected. It returns only the newly inserted
// rows (with generated IDs), not the merged set. Empty input, or input that
// is entirely duplicates, is a successful no-op that returns an empty slice.
func (s *CoinReferenceService) AppendForCoin(
	coinID uint,
	userID uint,
	refs []models.CoinReference,
) ([]models.CoinReference, error) {
	if len(refs) == 0 {
		return []models.CoinReference{}, nil
	}

	existing, err := s.repo.ListByCoin(coinID, userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(existing)+len(refs))
	for _, ref := range existing {
		seen[dedupeKey(ref)] = struct{}{}
	}

	survivors := make([]models.CoinReference, 0, len(refs))
	for _, ref := range refs {
		n, err := s.NormalizeAndValidateOne(ref)
		if err != nil {
			return nil, err
		}

		key := dedupeKey(n)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		// Strip any client-supplied primary key/scope so CreateBatch always
		// generates a fresh row scoped to this coin.
		n.ID = 0
		n.CoinID = coinID
		survivors = append(survivors, n)
	}

	if len(survivors) == 0 {
		return []models.CoinReference{}, nil
	}

	if err := s.repo.CreateBatch(survivors); err != nil {
		return nil, err
	}

	return survivors, nil
}

func dedupeKey(ref models.CoinReference) string {
	return strings.ToUpper(strings.TrimSpace(ref.Catalog)) + "|" +
		strings.ToUpper(strings.TrimSpace(ref.Volume)) + "|" +
		strings.ToUpper(strings.TrimSpace(ref.Number))
}
