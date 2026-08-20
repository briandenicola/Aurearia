package services

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// Sentinel errors for PurchaseReminderService.
var (
	ErrCoinNotWishlist  = errors.New("reminders are only available for wishlist coins")
	ErrRemindDatePast   = errors.New("remind date must be today or in the future")
	ErrInvalidTimezone  = errors.New("invalid timezone")
	ErrReminderNotFound = errors.New("no active reminder found")
)

// PurchaseReminderService handles business logic for purchase reminders.
type PurchaseReminderService struct {
	repo     *repository.PurchaseReminderRepository
	coinRepo *repository.CoinRepository
	logger   *Logger
}

// NewPurchaseReminderService constructs the service with its dependencies.
func NewPurchaseReminderService(
	repo *repository.PurchaseReminderRepository,
	coinRepo *repository.CoinRepository,
	logger *Logger,
) *PurchaseReminderService {
	return &PurchaseReminderService{
		repo:     repo,
		coinRepo: coinRepo,
		logger:   logger,
	}
}

// CreateOrUpdate upserts a purchase reminder for a wishlist coin owned by userID.
// Returns (reminder, isNew, error). If an active reminder already exists for
// this coin, it is updated in-place (same ID, status reset to pending).
// Validation: timezone must be a valid IANA string; remindDate must be today-or-future
// in that timezone; coin must exist, be owned by the user, and have IsWishlist=true.
func (s *PurchaseReminderService) CreateOrUpdate(
	userID, coinID uint,
	remindDate, timezone string,
) (*models.PurchaseReminder, bool, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, false, ErrInvalidTimezone
	}

	// Validate date format and that it is today-or-future in the given timezone.
	parsed, err := time.ParseInLocation("2006-01-02", remindDate, loc)
	if err != nil {
		return nil, false, ErrRemindDatePast // malformed date treated as past
	}
	today := time.Now().In(loc)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	if parsed.Before(todayDate) {
		return nil, false, ErrRemindDatePast
	}

	coin, err := s.coinRepo.FindByID(coinID, userID)
	if err != nil || coin == nil {
		return nil, false, ErrReminderNotFound // treat as 404 — coin not found or not owned
	}
	if !coin.IsWishlist {
		return nil, false, ErrCoinNotWishlist
	}

	// Upsert: check for an existing active reminder on this coin.
	existing, err := s.repo.FindActiveByCoinAndUser(coinID, userID)
	if err != nil {
		return nil, false, err
	}

	if existing != nil {
		// Update in-place: reset status to pending, clear notifiedAt.
		existing.RemindDate = remindDate
		existing.Timezone = timezone
		existing.Status = "pending"
		existing.NotifiedAt = nil
		if err := s.repo.UpdateReminder(existing); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}

	// Create new reminder.
	reminder := &models.PurchaseReminder{
		CoinID:     coinID,
		UserID:     userID,
		RemindDate: remindDate,
		Timezone:   timezone,
		Status:     "pending",
	}
	if err := s.repo.CreateReminder(reminder); err != nil {
		return nil, false, err
	}
	return reminder, true, nil
}

// Cancel cancels the active reminder for a coin owned by userID.
// Returns ErrReminderNotFound if no active reminder exists.
func (s *PurchaseReminderService) Cancel(userID, coinID uint) error {
	existing, err := s.repo.FindActiveByCoinAndUser(coinID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrReminderNotFound
	}
	now := time.Now()
	existing.Status = "cancelled"
	existing.CancelledAt = &now
	return s.repo.UpdateReminder(existing)
}

// GetForCoin returns the active reminder for a coin owned by userID, or nil if none.
// Returns ErrReminderNotFound if the coin is not found (404 semantics).
func (s *PurchaseReminderService) GetForCoin(userID, coinID uint) (*models.PurchaseReminder, error) {
	// Verify coin ownership (returns 404 rather than leaking existence).
	coin, err := s.coinRepo.FindByID(coinID, userID)
	if err != nil || coin == nil {
		return nil, ErrReminderNotFound
	}
	return s.repo.FindActiveByCoinAndUser(coinID, userID)
}

// ListForUser returns all active (pending + notified) reminders for userID.
func (s *PurchaseReminderService) ListForUser(userID uint) ([]models.PurchaseReminder, error) {
	reminders, err := s.repo.ListActiveByUser(userID)
	if err != nil {
		return nil, err
	}
	// Populate CoinName from preloaded Coin for API response convenience.
	for i := range reminders {
		reminders[i].CoinName = reminders[i].Coin.Name
	}
	return reminders, nil
}
