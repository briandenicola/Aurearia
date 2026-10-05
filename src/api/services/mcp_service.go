package services

import (
	"errors"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var ErrMCPInvalidRequest = errors.New("invalid MCP request")

type MCPAuctionLotSummary struct {
	ID             uint                    `json:"id"`
	Source         models.AuctionSource    `json:"source"`
	SourceURL      string                  `json:"sourceUrl,omitempty"`
	LotNumber      int                     `json:"lotNumber,omitempty"`
	AuctionHouse   string                  `json:"auctionHouse,omitempty"`
	SaleName       string                  `json:"saleName,omitempty"`
	SaleDate       *time.Time              `json:"saleDate,omitempty"`
	AuctionEndTime *time.Time              `json:"auctionEndTime,omitempty"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description,omitempty"`
	Category       models.Category         `json:"category,omitempty"`
	Estimate       *float64                `json:"estimate,omitempty"`
	CurrentBid     *float64                `json:"currentBid,omitempty"`
	MaxBid         *float64                `json:"maxBid,omitempty"`
	WinningBid     *float64                `json:"winningBid,omitempty"`
	Currency       string                  `json:"currency,omitempty"`
	Status         models.AuctionLotStatus `json:"status"`
	IsOutbid       bool                    `json:"isOutbid"`
	ImageURL       string                  `json:"imageUrl,omitempty"`
	CoinID         *uint                   `json:"coinId,omitempty"`
	EventID        *uint                   `json:"eventId,omitempty"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

type MCPAuctionListInput struct {
	Status string
	Source string
	Search string
	Page   int
	Limit  int
}

type MCPAuctionListResult struct {
	Lots  []MCPAuctionLotSummary `json:"lots"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
}

type MCPService struct {
	collectionSvc *CollectionToolsService
	coinRepo      *repository.CoinRepository
	auctionRepo   *repository.AuctionLotRepository
	copilotSvc    *CoinCopilotService
}

func NewMCPService(
	collectionSvc *CollectionToolsService,
	coinRepo *repository.CoinRepository,
	auctionRepo *repository.AuctionLotRepository,
	copilotSvc *CoinCopilotService,
) *MCPService {
	return &MCPService{
		collectionSvc: collectionSvc,
		coinRepo:      coinRepo,
		auctionRepo:   auctionRepo,
		copilotSvc:    copilotSvc,
	}
}

func (s *MCPService) SearchCollection(userID uint, query string, limit *int) ([]CollectionCoinSummary, error) {
	if err := validateMCPQueryLimit(query, limit, 20); err != nil {
		return nil, err
	}
	return s.collectionSvc.SearchActiveCollection(userID, query, limit)
}

func (s *MCPService) GetCoin(userID, coinID uint) (*CollectionCoinSummary, error) {
	return s.collectionSvc.GetCoin(userID, coinID)
}

func (s *MCPService) ListWishlist(userID uint, query string, limit *int) ([]CollectionCoinSummary, error) {
	if err := validateMCPQueryLimit(query, limit, 50); err != nil {
		return nil, err
	}
	return s.collectionSvc.SearchWishlist(userID, query, limit)
}

func (s *MCPService) CollectionStats(userID uint) (*repository.CollectionStats, error) {
	return s.coinRepo.GetStats(userID)
}

func (s *MCPService) TopCoinsByValue(userID uint, limit *int) ([]CollectionCoinSummary, error) {
	if limit != nil && (*limit < 1 || *limit > 10) {
		return nil, ErrMCPInvalidRequest
	}
	return s.collectionSvc.TopCoinsByValue(userID, limit)
}

func (s *MCPService) ListAuctionLots(userID uint, input MCPAuctionListInput) (*MCPAuctionListResult, error) {
	if !validAuctionStatus(input.Status) || !validAuctionSource(input.Source) {
		return nil, ErrMCPInvalidRequest
	}
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	if page < 1 || limit < 1 || limit > 50 || len(input.Search) > 200 {
		return nil, ErrMCPInvalidRequest
	}
	lots, total, err := s.auctionRepo.List(userID, repository.AuctionLotListFilters{
		Status:    input.Status,
		Source:    input.Source,
		Search:    strings.TrimSpace(input.Search),
		SortField: "updated_at",
		SortOrder: "desc",
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	result := &MCPAuctionListResult{
		Lots:  make([]MCPAuctionLotSummary, 0, len(lots)),
		Total: total,
		Page:  page,
		Limit: limit,
	}
	for _, lot := range lots {
		result.Lots = append(result.Lots, toMCPAuctionLotSummary(lot))
	}
	return result, nil
}

func (s *MCPService) GetAuctionLot(userID, lotID uint) (*MCPAuctionLotSummary, error) {
	lot, err := s.auctionRepo.GetByID(lotID, userID)
	if err != nil {
		return nil, err
	}
	summary := toMCPAuctionLotSummary(*lot)
	return &summary, nil
}

func (s *MCPService) AuctionCounts(userID uint, source string) (map[string]int64, error) {
	if !validAuctionSource(source) {
		return nil, ErrMCPInvalidRequest
	}
	if source == "" {
		return s.auctionRepo.CountByStatus(userID)
	}
	return s.auctionRepo.CountByStatusForSource(userID, models.AuctionSource(source))
}

func (s *MCPService) StartCopilot(userID uint, input CoinCopilotStartInput) (*models.CoinCopilotRun, bool, error) {
	return s.copilotSvc.Start(userID, input)
}

func (s *MCPService) GetCopilotRun(userID uint, runID string) (*models.CoinCopilotRun, error) {
	return s.copilotSvc.GetRun(userID, runID)
}

func (s *MCPService) ResumeCopilot(userID uint, runID string, input CoinCopilotResumeInput) (*models.CoinCopilotRun, bool, error) {
	return s.copilotSvc.Resume(userID, runID, input)
}

func (s *MCPService) CancelCopilot(userID uint, runID string) (*models.CoinCopilotRun, bool, error) {
	return s.copilotSvc.Cancel(userID, runID)
}

func validAuctionStatus(value string) bool {
	switch models.AuctionLotStatus(value) {
	case "", models.AuctionStatusWatching, models.AuctionStatusBidding,
		models.AuctionStatusWon, models.AuctionStatusLost, models.AuctionStatusPassed:
		return true
	default:
		return false
	}
}

func validAuctionSource(value string) bool {
	switch models.AuctionSource(value) {
	case "", models.AuctionSourceNumisBids, models.AuctionSourceCNG:
		return true
	default:
		return false
	}
}

func validateMCPQueryLimit(query string, limit *int, max int) error {
	if len(query) > 200 || (limit != nil && (*limit < 1 || *limit > max)) {
		return ErrMCPInvalidRequest
	}
	return nil
}

func toMCPAuctionLotSummary(lot models.AuctionLot) MCPAuctionLotSummary {
	return MCPAuctionLotSummary{
		ID: lot.ID, Source: lot.Source, SourceURL: boundMCPText(lot.SourceURL, 2048),
		LotNumber: lot.LotNumber, AuctionHouse: boundMCPText(lot.AuctionHouse, 300), SaleName: boundMCPText(lot.SaleName, 300),
		SaleDate: lot.SaleDate, AuctionEndTime: lot.AuctionEndTime,
		Title: boundMCPText(lot.Title, 500), Description: boundMCPText(lot.Description, 2000), Category: lot.Category,
		Estimate: lot.Estimate, CurrentBid: lot.CurrentBid, MaxBid: lot.MaxBid,
		WinningBid: lot.WinningBid, Currency: lot.Currency, Status: lot.Status,
		IsOutbid: lot.IsOutbid, ImageURL: boundMCPText(lot.ImageURL, 2048), CoinID: lot.CoinID,
		EventID: lot.EventID, UpdatedAt: lot.UpdatedAt,
	}
}

func boundMCPText(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
