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

// validAuctionStatuses is the set of recognized lot statuses.
var validAuctionStatuses = map[models.AuctionLotStatus]bool{
	models.AuctionStatusWatching: true,
	models.AuctionStatusBidding:  true,
	models.AuctionStatusWon:      true,
	models.AuctionStatusLost:     true,
	models.AuctionStatusPassed:   true,
}

// UpdateStatus applies a manual status override. Any known status may move to any other
// known status: this is an explicit user override rather than a workflow the app enforces,
// since only the user (or a synced provider signal, applied separately by the watchlist
// sync path) actually knows a lot's real-world outcome.
func (s *AuctionLotService) UpdateStatus(id, userID uint, newStatus models.AuctionLotStatus) error {
	lot, err := s.repo.GetByID(id, userID)
	if err != nil {
		return ErrAuctionLotNotFound
	}

	if !validAuctionStatuses[newStatus] {
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
		ReferenceURL: firstNonEmptyAuctionURL(lot.SourceURL, lot.NumisBidsURL),
		ReferenceText: func() string {
			if lot.AuctionHouse != "" && lot.SaleName != "" {
				return lot.AuctionHouse + " — " + lot.SaleName
			}
			return lot.AuctionHouse
		}(),
		PurchasePrice: firstNonNilFloat(lot.WinningBid, lot.CurrentBid),
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

func firstNonEmptyAuctionURL(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNilFloat(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
