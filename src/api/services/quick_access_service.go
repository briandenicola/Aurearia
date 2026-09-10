package services

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const maxQuickAccessCoinSets = 5

var (
	ErrInvalidQuickAccessTarget  = errors.New("invalid quick access target")
	ErrQuickAccessTargetNotFound = errors.New("quick access item not found")
)

type QuickAccessCoinDTO struct {
	Name            string `json:"name"`
	Classification  string `json:"classification"`
	PrimaryImageURL string `json:"primaryImageUrl"`
}

type QuickAccessCoinSetDTO struct {
	Name    string             `json:"name"`
	SetType models.CoinSetType `json:"setType"`
	Color   string             `json:"color"`
	Icon    string             `json:"icon"`
}

type QuickAccessAuctionLotDTO struct {
	Title          string                  `json:"title"`
	Status         models.AuctionLotStatus `json:"status"`
	AuctionHouse   string                  `json:"auctionHouse"`
	SaleDate       *time.Time              `json:"saleDate"`
	AuctionEndTime *time.Time              `json:"auctionEndTime"`
	ImageURL       string                  `json:"imageUrl"`
}

type QuickAccessCalendarEventDTO struct {
	Title        string     `json:"title"`
	AuctionHouse string     `json:"auctionHouse"`
	StartDate    *time.Time `json:"startDate"`
	EndDate      *time.Time `json:"endDate"`
}

type QuickAccessItemDTO struct {
	Type          models.QuickAccessTargetType `json:"type"`
	ID            uint                         `json:"id"`
	PinnedAt      time.Time                    `json:"pinnedAt"`
	Coin          *QuickAccessCoinDTO          `json:"coin,omitempty"`
	CoinSet       *QuickAccessCoinSetDTO       `json:"coinSet,omitempty"`
	AuctionLot    *QuickAccessAuctionLotDTO    `json:"auctionLot,omitempty"`
	CalendarEvent *QuickAccessCalendarEventDTO `json:"calendarEvent,omitempty"`
}

type QuickAccessListDTO struct {
	Items []QuickAccessItemDTO `json:"items"`
}

type QuickAccessService struct {
	repo   *repository.QuickAccessRepository
	logger *Logger
}

func NewQuickAccessService(repo *repository.QuickAccessRepository, logger ...*Logger) *QuickAccessService {
	var serviceLogger *Logger
	if len(logger) > 0 {
		serviceLogger = logger[0]
	}
	return &QuickAccessService{repo: repo, logger: serviceLogger}
}

func ParseQuickAccessTargetType(value string) (models.QuickAccessTargetType, error) {
	targetType := models.QuickAccessTargetType(value)
	if !targetType.Valid() {
		return "", ErrInvalidQuickAccessTarget
	}
	return targetType, nil
}

func (s *QuickAccessService) List(userID uint) (QuickAccessListDTO, error) {
	pins, err := s.repo.List(userID)
	if err != nil {
		return QuickAccessListDTO{}, err
	}

	idsByType := map[models.QuickAccessTargetType][]uint{}
	for _, pin := range pins {
		if pin.TargetType.Valid() {
			idsByType[pin.TargetType] = append(idsByType[pin.TargetType], pin.TargetID)
		}
	}

	coins, err := s.repo.FindCoins(userID, idsByType[models.QuickAccessTargetCoin])
	if err != nil {
		return QuickAccessListDTO{}, err
	}
	sets, err := s.repo.FindCoinSets(userID, idsByType[models.QuickAccessTargetCoinSet])
	if err != nil {
		return QuickAccessListDTO{}, err
	}
	lots, err := s.repo.FindAuctionLots(userID, idsByType[models.QuickAccessTargetAuctionLot])
	if err != nil {
		return QuickAccessListDTO{}, err
	}
	events, err := s.repo.FindCalendarEvents(userID, idsByType[models.QuickAccessTargetCalendarEvent])
	if err != nil {
		return QuickAccessListDTO{}, err
	}

	coinByID := make(map[uint]repository.QuickAccessCoinRow, len(coins))
	for _, row := range coins {
		coinByID[row.ID] = row
	}
	setByID := make(map[uint]repository.QuickAccessSetRow, len(sets))
	for _, row := range sets {
		setByID[row.ID] = row
	}
	lotByID := make(map[uint]repository.QuickAccessAuctionLotRow, len(lots))
	for _, row := range lots {
		lotByID[row.ID] = row
	}
	eventByID := make(map[uint]repository.QuickAccessCalendarEventRow, len(events))
	for _, row := range events {
		eventByID[row.ID] = row
	}

	items := make([]QuickAccessItemDTO, 0, len(pins))
	for _, pin := range pins {
		var item QuickAccessItemDTO
		var ok bool
		switch pin.TargetType {
		case models.QuickAccessTargetCoin:
			row, found := coinByID[pin.TargetID]
			if found && !row.IsSold {
				item = coinItem(pin, row)
				ok = true
			}
		case models.QuickAccessTargetCoinSet:
			row, found := setByID[pin.TargetID]
			if found {
				item = coinSetItem(pin, row)
				ok = true
			}
		case models.QuickAccessTargetAuctionLot:
			row, found := lotByID[pin.TargetID]
			if found && auctionLotIsQuickAccessEligible(row.Status) {
				item = auctionLotItem(pin, row)
				ok = true
			}
		case models.QuickAccessTargetCalendarEvent:
			row, found := eventByID[pin.TargetID]
			if found && row.Origin == models.AuctionEventOriginManual {
				item = calendarEventItem(pin, row)
				ok = true
			}
		}
		if ok {
			items = append(items, item)
		} else {
			s.logStalePin(pin)
		}
	}
	return QuickAccessListDTO{Items: items}, nil
}

func (s *QuickAccessService) Pin(userID uint, targetType models.QuickAccessTargetType, targetID uint) (QuickAccessItemDTO, bool, error) {
	if userID == 0 || targetID == 0 || !targetType.Valid() {
		return QuickAccessItemDTO{}, false, ErrInvalidQuickAccessTarget
	}

	var pin models.QuickAccessPin
	var created bool
	err := s.repo.RunInTransaction(func(tx *repository.Transaction) error {
		txRepo := s.repo.WithTransaction(tx)
		if err := validateQuickAccessTarget(txRepo, userID, targetType, targetID); err != nil {
			return err
		}
		if targetType == models.QuickAccessTargetCoinSet {
			var err error
			pin, created, err = s.setCoinSetPinned(txRepo, userID, targetID, true)
			return err
		}
		pin = models.QuickAccessPin{
			UserID:     userID,
			TargetType: targetType,
			TargetID:   targetID,
			PinnedAt:   time.Now().UTC(),
		}
		var err error
		created, err = txRepo.CreateOrGet(&pin)
		return err
	})
	if err != nil {
		return QuickAccessItemDTO{}, false, normalizeQuickAccessTargetError(err)
	}

	item, err := s.hydratePin(userID, pin)
	if err != nil {
		return QuickAccessItemDTO{}, false, err
	}
	return item, created, nil
}

func (s *QuickAccessService) Unpin(userID uint, targetType models.QuickAccessTargetType, targetID uint) error {
	if userID == 0 || targetID == 0 || !targetType.Valid() {
		return ErrInvalidQuickAccessTarget
	}
	return s.repo.RunInTransaction(func(tx *repository.Transaction) error {
		txRepo := s.repo.WithTransaction(tx)
		if err := txRepo.Delete(userID, targetType, targetID); err != nil {
			return err
		}
		if targetType == models.QuickAccessTargetCoinSet {
			err := txRepo.SetCoinSetPinnedAt(targetID, userID, nil)
			if repository.IsRecordNotFound(err) {
				return nil
			}
			return err
		}
		return nil
	})
}

func (s *QuickAccessService) SetCoinSetPinnedInTx(tx *repository.Transaction, setID, userID uint, pinned bool) error {
	txRepo := s.repo.WithTransaction(tx)
	if _, _, err := s.setCoinSetPinned(txRepo, userID, setID, pinned); err != nil {
		return normalizeQuickAccessTargetError(err)
	}
	return nil
}

func (s *QuickAccessService) RemoveTargetInTx(tx *repository.Transaction, userID uint, targetType models.QuickAccessTargetType, targetID uint) error {
	return s.repo.WithTransaction(tx).Delete(userID, targetType, targetID)
}

func (s *QuickAccessService) RemoveTargetsInTx(tx *repository.Transaction, userID uint, targetType models.QuickAccessTargetType, targetIDs []uint) error {
	return s.repo.WithTransaction(tx).DeleteTargets(userID, targetType, targetIDs)
}

func (s *QuickAccessService) setCoinSetPinned(repo *repository.QuickAccessRepository, userID, setID uint, pinned bool) (models.QuickAccessPin, bool, error) {
	if !pinned {
		if err := repo.Delete(userID, models.QuickAccessTargetCoinSet, setID); err != nil {
			return models.QuickAccessPin{}, false, err
		}
		if err := repo.SetCoinSetPinnedAt(setID, userID, nil); err != nil {
			return models.QuickAccessPin{}, false, err
		}
		return models.QuickAccessPin{}, false, nil
	}

	existing, err := repo.Get(userID, models.QuickAccessTargetCoinSet, setID)
	if err == nil {
		if err := repo.SetCoinSetPinnedAt(setID, userID, &existing.PinnedAt); err != nil {
			return models.QuickAccessPin{}, false, err
		}
		return *existing, false, nil
	}
	if !repository.IsRecordNotFound(err) {
		return models.QuickAccessPin{}, false, err
	}
	count, err := repo.CountByType(userID, models.QuickAccessTargetCoinSet)
	if err != nil {
		return models.QuickAccessPin{}, false, err
	}
	if count >= maxQuickAccessCoinSets {
		return models.QuickAccessPin{}, false, ErrPinLimitReached
	}
	pin := models.QuickAccessPin{
		UserID:     userID,
		TargetType: models.QuickAccessTargetCoinSet,
		TargetID:   setID,
		PinnedAt:   time.Now().UTC(),
	}
	created, err := repo.CreateOrGet(&pin)
	if err != nil {
		return models.QuickAccessPin{}, false, err
	}
	if err := repo.SetCoinSetPinnedAt(setID, userID, &pin.PinnedAt); err != nil {
		return models.QuickAccessPin{}, false, err
	}
	return pin, created, nil
}

func validateQuickAccessTarget(repo *repository.QuickAccessRepository, userID uint, targetType models.QuickAccessTargetType, targetID uint) error {
	switch targetType {
	case models.QuickAccessTargetCoin:
		row, err := repo.FindCoin(userID, targetID)
		if err != nil {
			return err
		}
		if row.IsSold {
			return ErrQuickAccessTargetNotFound
		}
	case models.QuickAccessTargetCoinSet:
		if _, err := repo.FindCoinSet(userID, targetID); err != nil {
			return err
		}
	case models.QuickAccessTargetAuctionLot:
		row, err := repo.FindAuctionLot(userID, targetID)
		if err != nil {
			return err
		}
		if !auctionLotIsQuickAccessEligible(row.Status) {
			return ErrQuickAccessTargetNotFound
		}
	case models.QuickAccessTargetCalendarEvent:
		row, err := repo.FindCalendarEvent(userID, targetID)
		if err != nil {
			return err
		}
		if row.Origin != models.AuctionEventOriginManual {
			return ErrQuickAccessTargetNotFound
		}
	default:
		return ErrInvalidQuickAccessTarget
	}
	return nil
}

func normalizeQuickAccessTargetError(err error) error {
	if repository.IsRecordNotFound(err) {
		return ErrQuickAccessTargetNotFound
	}
	return err
}

func auctionLotIsQuickAccessEligible(status models.AuctionLotStatus) bool {
	return status == models.AuctionStatusWatching || status == models.AuctionStatusBidding
}

func (s *QuickAccessService) hydratePin(userID uint, pin models.QuickAccessPin) (QuickAccessItemDTO, error) {
	switch pin.TargetType {
	case models.QuickAccessTargetCoin:
		row, err := s.repo.FindCoin(userID, pin.TargetID)
		if err != nil {
			return QuickAccessItemDTO{}, normalizeQuickAccessTargetError(err)
		}
		return coinItem(pin, *row), nil
	case models.QuickAccessTargetCoinSet:
		row, err := s.repo.FindCoinSet(userID, pin.TargetID)
		if err != nil {
			return QuickAccessItemDTO{}, normalizeQuickAccessTargetError(err)
		}
		return coinSetItem(pin, *row), nil
	case models.QuickAccessTargetAuctionLot:
		row, err := s.repo.FindAuctionLot(userID, pin.TargetID)
		if err != nil {
			return QuickAccessItemDTO{}, normalizeQuickAccessTargetError(err)
		}
		return auctionLotItem(pin, *row), nil
	case models.QuickAccessTargetCalendarEvent:
		row, err := s.repo.FindCalendarEvent(userID, pin.TargetID)
		if err != nil {
			return QuickAccessItemDTO{}, normalizeQuickAccessTargetError(err)
		}
		return calendarEventItem(pin, *row), nil
	default:
		return QuickAccessItemDTO{}, ErrInvalidQuickAccessTarget
	}
}

func coinItem(pin models.QuickAccessPin, row repository.QuickAccessCoinRow) QuickAccessItemDTO {
	classification := "owned"
	if row.IsWishlist {
		classification = "wishlist"
	}
	return QuickAccessItemDTO{
		Type:     pin.TargetType,
		ID:       pin.TargetID,
		PinnedAt: pin.PinnedAt,
		Coin: &QuickAccessCoinDTO{
			Name:            row.Name,
			Classification:  classification,
			PrimaryImageURL: row.PrimaryImageURL,
		},
	}
}

func coinSetItem(pin models.QuickAccessPin, row repository.QuickAccessSetRow) QuickAccessItemDTO {
	return QuickAccessItemDTO{
		Type:     pin.TargetType,
		ID:       pin.TargetID,
		PinnedAt: pin.PinnedAt,
		CoinSet: &QuickAccessCoinSetDTO{
			Name: row.Name, SetType: row.SetType, Color: row.Color, Icon: row.Icon,
		},
	}
}

func auctionLotItem(pin models.QuickAccessPin, row repository.QuickAccessAuctionLotRow) QuickAccessItemDTO {
	return QuickAccessItemDTO{
		Type:     pin.TargetType,
		ID:       pin.TargetID,
		PinnedAt: pin.PinnedAt,
		AuctionLot: &QuickAccessAuctionLotDTO{
			Title: row.Title, Status: row.Status, AuctionHouse: row.AuctionHouse,
			SaleDate: row.SaleDate, AuctionEndTime: row.AuctionEndTime, ImageURL: row.ImageURL,
		},
	}
}

func calendarEventItem(pin models.QuickAccessPin, row repository.QuickAccessCalendarEventRow) QuickAccessItemDTO {
	return QuickAccessItemDTO{
		Type:     pin.TargetType,
		ID:       pin.TargetID,
		PinnedAt: pin.PinnedAt,
		CalendarEvent: &QuickAccessCalendarEventDTO{
			Title: row.Title, AuctionHouse: row.AuctionHouse, StartDate: row.StartDate, EndDate: row.EndDate,
		},
	}
}

func (s *QuickAccessService) logStalePin(pin models.QuickAccessPin) {
	if s.logger != nil {
		s.logger.Warn("quick-access", "Omitting stale or ineligible pin id=%d type=%s target_id=%d", pin.ID, pin.TargetType, pin.TargetID)
	}
}
