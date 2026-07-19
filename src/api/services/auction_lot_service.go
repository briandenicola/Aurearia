package services

import (
	"errors"
	"fmt"

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

	return s.repo.UpdateFields(lot, map[string]interface{}{
		"status":        newStatus,
		"status_source": string(models.AuctionLotStatusSourceManual),
	})
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

// BidRecommendationConfidence describes how much the user's own history backs a suggestion.
type BidRecommendationConfidence string

const (
	ConfidenceInsufficientData BidRecommendationConfidence = "insufficient_data"
	ConfidenceLow              BidRecommendationConfidence = "low"
	ConfidenceMedium           BidRecommendationConfidence = "medium"
	ConfidenceHigh             BidRecommendationConfidence = "high"
)

// BidRecommendation is a suggested maximum bid for a tracked lot, grounded in the user's own
// resolved (won/lost) auction history rather than a generic model. It is intentionally an
// aid, not an autofilled value: the user still places bids on the provider's own site.
type BidRecommendation struct {
	SuggestedMaxBid *float64                    `json:"suggestedMaxBid"`
	Confidence      BidRecommendationConfidence `json:"confidence"`
	SampleSize      int                         `json:"sampleSize"`
	Rationale       string                      `json:"rationale"`
}

// minComparableLotsForRecommendation is the smallest sample the recommendation will act on.
// Below this, presenting a number would look more confident than the data supports.
const minComparableLotsForRecommendation = 2

// Recommend suggests a maximum bid for the given lot based on the user's own won/lost lots
// in the same category. Won lots contribute winningBid/estimate; lost lots contribute
// currentBid/estimate — currentBid on an already-closed, lost lot reflects the final bid
// that beat the user as of the last sync (verified true for CNG post-F022-rebuild; NumisBids
// accuracy here depends on F021/F022, since its CurrentBid refresh is not yet re-verified).
func (s *AuctionLotService) Recommend(lotID, userID uint) (BidRecommendation, error) {
	lot, err := s.repo.GetByID(lotID, userID)
	if err != nil {
		return BidRecommendation{}, ErrAuctionLotNotFound
	}

	if lot.Estimate == nil || *lot.Estimate <= 0 {
		return BidRecommendation{
			Confidence: ConfidenceInsufficientData,
			Rationale:  "This lot has no estimate to compare against, so a suggested bid would just be a guess.",
		}, nil
	}

	history, err := s.repo.ListResolvedByUserAndCategory(userID, lot.Category)
	if err != nil {
		return BidRecommendation{}, err
	}

	var ratios []float64
	wonCount, lostCount := 0, 0
	for _, h := range history {
		if h.ID == lot.ID || h.Estimate == nil || *h.Estimate <= 0 {
			continue
		}
		switch h.Status {
		case models.AuctionStatusWon:
			if h.WinningBid != nil {
				ratios = append(ratios, *h.WinningBid / *h.Estimate)
				wonCount++
			}
		case models.AuctionStatusLost:
			if h.CurrentBid != nil {
				ratios = append(ratios, *h.CurrentBid / *h.Estimate)
				lostCount++
			}
		}
	}

	if len(ratios) < minComparableLotsForRecommendation {
		return BidRecommendation{
			Confidence: ConfidenceInsufficientData,
			SampleSize: len(ratios),
			Rationale: fmt.Sprintf(
				"Only %d resolved %s lot(s) in your history with an estimate to compare against — "+
					"not enough yet to base a suggestion on. This will fill in as you win or lose more lots in this category.",
				len(ratios), lot.Category,
			),
		}, nil
	}

	avgRatio := averageFloat(ratios)
	suggested := avgRatio * *lot.Estimate

	confidence := ConfidenceLow
	switch {
	case len(ratios) >= 10:
		confidence = ConfidenceHigh
	case len(ratios) >= 5:
		confidence = ConfidenceMedium
	}

	rationale := fmt.Sprintf(
		"Based on %d of your own resolved %s lot(s) (%d won, %d lost), you've historically bid to about "+
			"%.0f%% of a lot's estimate. Applied to this lot's estimate, that suggests a maximum bid around "+
			"the amount shown. This is drawn only from your own history — it does not search the wider market.",
		len(ratios), lot.Category, wonCount, lostCount, avgRatio*100,
	)

	return BidRecommendation{
		SuggestedMaxBid: &suggested,
		Confidence:      confidence,
		SampleSize:      len(ratios),
		Rationale:       rationale,
	}, nil
}

func averageFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
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
