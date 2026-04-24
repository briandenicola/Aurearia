package services

import (
	"errors"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrAuctionLotNotFound = errors.New("auction lot not found")
	ErrInvalidStatus      = errors.New("invalid auction lot status transition")
)

// AuctionLotService handles auction lot business logic.
type AuctionLotService struct {
	repo     *repository.AuctionLotRepository
	coinRepo *repository.CoinRepository
}

// NewAuctionLotService creates a new AuctionLotService.
func NewAuctionLotService(repo *repository.AuctionLotRepository, coinRepo *repository.CoinRepository) *AuctionLotService {
	return &AuctionLotService{repo: repo, coinRepo: coinRepo}
}

// validTransitions defines which status transitions are allowed.
var validTransitions = map[models.AuctionLotStatus][]models.AuctionLotStatus{
	models.AuctionStatusWatching: {models.AuctionStatusBidding, models.AuctionStatusPassed},
	models.AuctionStatusBidding:  {models.AuctionStatusWon, models.AuctionStatusLost, models.AuctionStatusWatching},
	models.AuctionStatusWon:      {},
	models.AuctionStatusLost:     {models.AuctionStatusWatching},
	models.AuctionStatusPassed:   {models.AuctionStatusWatching},
}

// UpdateStatus transitions an auction lot to a new status.
func (s *AuctionLotService) UpdateStatus(id, userID uint, newStatus models.AuctionLotStatus) error {
	lot, err := s.repo.GetByID(id, userID)
	if err != nil {
		return ErrAuctionLotNotFound
	}

	allowed := validTransitions[lot.Status]
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return ErrInvalidStatus
	}

	return s.repo.UpdateFields(lot, map[string]interface{}{"status": newStatus})
}

// ConvertToCoin creates an owned Coin from a won auction lot.
func (s *AuctionLotService) ConvertToCoin(lotID, userID uint) (*models.Coin, error) {
	lot, err := s.repo.GetByID(lotID, userID)
	if err != nil {
		return nil, ErrAuctionLotNotFound
	}

	if lot.Status != models.AuctionStatusWon {
		return nil, ErrInvalidStatus
	}

	if lot.CoinID != nil {
		// Already converted
		coin, err := s.coinRepo.FindByID(*lot.CoinID, userID)
		if err != nil {
			return nil, err
		}
		return coin, nil
	}

	coin := &models.Coin{
		Name:         lot.Title,
		Notes:        lot.Description,
		Category:     lot.Category,
		ReferenceURL: lot.NumisBidsURL,
		ReferenceText: func() string {
			if lot.AuctionHouse != "" && lot.SaleName != "" {
				return lot.AuctionHouse + " — " + lot.SaleName
			}
			return lot.AuctionHouse
		}(),
		PurchasePrice: lot.CurrentBid,
		PurchaseDate:  lot.SaleDate,
		UserID:        userID,
	}

	err = s.repo.Transaction(func(tx *gorm.DB) error {
		txCoinRepo := s.coinRepo.WithTx(tx)
		txLotRepo := s.repo.WithTx(tx)

		if err := txCoinRepo.Create(coin); err != nil {
			return err
		}
		return txLotRepo.UpdateFields(lot, map[string]interface{}{"coin_id": coin.ID})
	})
	if err != nil {
		return nil, err
	}

	return coin, nil
}
