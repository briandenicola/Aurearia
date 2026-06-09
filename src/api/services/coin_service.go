package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrCoinInvalidEra = errors.New("era is not supported")
)

var builtInCoinEras = map[models.Era]struct{}{
	models.EraAncient:  {},
	models.EraMedieval: {},
	models.EraModern:   {},
}

// CoinService handles coin business logic and orchestrates repository calls.
type CoinService struct {
	repo                *repository.CoinRepository
	notifSvc            *NotificationService
	refRepo             *repository.CoinReferenceRepository
	refSvc              *CoinReferenceService
	storageLocationRepo *repository.StorageLocationRepository
	catalogRegistryRepo *repository.CatalogRegistryRepository
}

// NewCoinService creates a new CoinService.
func NewCoinService(repo *repository.CoinRepository, notifSvc *NotificationService) *CoinService {
	return &CoinService{repo: repo, notifSvc: notifSvc}
}

// WithReferenceSupport enables structured reference orchestration during coin create/update workflows.
func (s *CoinService) WithReferenceSupport(
	refRepo *repository.CoinReferenceRepository,
	refSvc *CoinReferenceService,
) *CoinService {
	s.refRepo = refRepo
	s.refSvc = refSvc
	return s
}

// WithStorageLocationSupport enables storage-location ownership validation during coin create/update workflows.
func (s *CoinService) WithStorageLocationSupport(storageLocationRepo *repository.StorageLocationRepository) *CoinService {
	s.storageLocationRepo = storageLocationRepo
	return s
}

// WithCatalogRegistrySupport enables data-driven coin era validation.
func (s *CoinService) WithCatalogRegistrySupport(catalogRegistryRepo *repository.CatalogRegistryRepository) *CoinService {
	s.catalogRegistryRepo = catalogRegistryRepo
	return s
}

// CreateCoin creates a coin and records a value snapshot in a single transaction.
func (s *CoinService) CreateCoin(coin *models.Coin) error {
	if err := s.validateStorageLocation(coin.StorageLocationID, coin.UserID); err != nil {
		return err
	}
	coin.Era = models.Era(strings.TrimSpace(string(coin.Era)))
	if err := s.validateCoinEra(coin.Era); err != nil {
		return err
	}
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.Create(coin); err != nil {
			return err
		}

		if s.refRepo != nil && s.refSvc != nil && coin.References != nil {
			normalized, err := s.refSvc.NormalizeAndValidate(coin.References)
			if err != nil {
				return err
			}

			for i := range normalized {
				normalized[i].CoinID = coin.ID
			}

			txRefRepo := s.refRepo.WithTx(tx)
			if err := txRefRepo.CreateBatch(normalized); err != nil {
				return err
			}

			coin.References = normalized
		}

		return txRepo.RecordValueSnapshot(coin.UserID)
	})
	if err != nil {
		return err
	}

	// Notify followers after commit (async to avoid slowing the response)
	if s.notifSvc != nil {
		go s.notifSvc.NotifyNewCoin(coin.UserID, *coin)
	}

	return nil
}

// UpdateCoin applies updates to an existing coin. If the current value changed
// and the source is not "estimate", it records a value history entry and a
// journal entry. A value snapshot is always recorded afterward.
func (s *CoinService) UpdateCoin(existing *models.Coin, updates *models.Coin, userID uint, source string, storageLocationProvided ...bool) error {
	oldValue := existing.CurrentValue
	updateStorageLocation := len(storageLocationProvided) > 0 && storageLocationProvided[0]
	if updateStorageLocation {
		if err := s.validateStorageLocation(updates.StorageLocationID, userID); err != nil {
			return err
		}
	}
	updates.Era = models.Era(strings.TrimSpace(string(updates.Era)))
	if err := s.validateCoinEra(updates.Era); err != nil {
		return err
	}

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)

		if err := txRepo.Update(existing, updates); err != nil {
			return err
		}
		if updateStorageLocation {
			if err := txRepo.UpdateStorageLocationID(existing, updates.StorageLocationID); err != nil {
				return err
			}
		}

		if s.refRepo != nil && s.refSvc != nil && updates.References != nil {
			normalized, err := s.refSvc.NormalizeAndValidate(updates.References)
			if err != nil {
				return err
			}

			txRefRepo := s.refRepo.WithTx(tx)
			if err := txRefRepo.ReplaceForCoin(existing.ID, userID, normalized); err != nil {
				return err
			}

			existing.References = normalized
		}

		if updates.CurrentValue != nil {
			newVal := *updates.CurrentValue
			oldVal := 0.0
			if oldValue != nil {
				oldVal = *oldValue
			}
			if newVal != oldVal && source != "estimate" {
				// Update CurrentValueUpdatedAt whenever CurrentValue changes manually
				now := time.Now()
				existing.CurrentValueUpdatedAt = &now
				if err := txRepo.UpdateField(existing, "current_value_updated_at", now); err != nil {
					return err
				}

				if err := txRepo.RecordValueHistory(&models.CoinValueHistory{
					CoinID:     existing.ID,
					UserID:     userID,
					Value:      newVal,
					Confidence: "manual",
					RecordedAt: now,
				}); err != nil {
					return err
				}
				if err := txRepo.CreateJournalEntry(&models.CoinJournal{
					CoinID: existing.ID,
					UserID: userID,
					Entry:  fmt.Sprintf("Current value updated manually: $%.2f", newVal),
				}); err != nil {
					return err
				}
			}
		}

		return txRepo.RecordValueSnapshot(userID)
	})
}

// DeleteCoin deletes a coin and records a value snapshot if rows were affected.
// Returns the number of rows affected.
func (s *CoinService) DeleteCoin(id, userID uint) (int64, error) {
	var rows int64
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		var err error
		rows, err = txRepo.Delete(id, userID)
		if err != nil {
			return err
		}
		if rows > 0 {
			return txRepo.RecordValueSnapshot(userID)
		}
		return nil
	})
	return rows, err
}

// PurchaseCoin marks a wishlist coin as purchased and records a value snapshot.
// The coin's purchase fields (price, date, location) should be set on the model
// before calling this method.
func (s *CoinService) PurchaseCoin(coin *models.Coin, userID uint) error {
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		updates := map[string]interface{}{
			"is_wishlist":       false,
			"purchase_price":    coin.PurchasePrice,
			"purchase_date":     coin.PurchaseDate,
			"purchase_location": coin.PurchaseLocation,
		}
		if err := txRepo.UpdateFields(coin, updates); err != nil {
			return err
		}
		return txRepo.RecordValueSnapshot(userID)
	})
}

// SellCoin applies sale updates to a coin and records a value snapshot.
func (s *CoinService) SellCoin(coin *models.Coin, updates map[string]interface{}, userID uint) error {
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.UpdateFields(coin, updates); err != nil {
			return err
		}
		return txRepo.RecordValueSnapshot(userID)
	})
}

func (s *CoinService) validateStorageLocation(storageLocationID *uint, userID uint) error {
	if storageLocationID == nil || s.storageLocationRepo == nil {
		return nil
	}
	if *storageLocationID == 0 {
		return ErrStorageLocationNotFound
	}
	exists, err := s.storageLocationRepo.ExistsByID(*storageLocationID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrStorageLocationNotFound
	}
	return nil
}

func (s *CoinService) validateCoinEra(era models.Era) error {
	trimmed := strings.TrimSpace(string(era))
	if trimmed == "" {
		return nil
	}
	normalized := models.Era(trimmed)
	if _, ok := builtInCoinEras[normalized]; ok {
		return nil
	}
	if len(trimmed) > 64 {
		return ErrCoinInvalidEra
	}
	if s.catalogRegistryRepo == nil {
		return ErrCoinInvalidEra
	}
	exists, err := s.catalogRegistryRepo.EraExists(normalized)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCoinInvalidEra
	}
	return nil
}
