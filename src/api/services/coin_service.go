package services

import (
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

// CoinService handles coin business logic and orchestrates repository calls.
type CoinService struct {
	repo     *repository.CoinRepository
	notifSvc *NotificationService
}

// NewCoinService creates a new CoinService.
func NewCoinService(repo *repository.CoinRepository, notifSvc *NotificationService) *CoinService {
	return &CoinService{repo: repo, notifSvc: notifSvc}
}

// CreateCoin creates a coin and records a value snapshot in a single transaction.
func (s *CoinService) CreateCoin(coin *models.Coin) error {
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.Create(coin); err != nil {
			return err
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
func (s *CoinService) UpdateCoin(existing *models.Coin, updates *models.Coin, userID uint, source string) error {
	oldValue := existing.CurrentValue

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)

		if err := txRepo.Update(existing, updates); err != nil {
			return err
		}

		if updates.CurrentValue != nil {
			newVal := *updates.CurrentValue
			oldVal := 0.0
			if oldValue != nil {
				oldVal = *oldValue
			}
			if newVal != oldVal && source != "estimate" {
				if err := txRepo.RecordValueHistory(&models.CoinValueHistory{
					CoinID:     existing.ID,
					UserID:     userID,
					Value:      newVal,
					Confidence: "manual",
					RecordedAt: time.Now(),
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
