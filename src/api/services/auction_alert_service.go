package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrAuctionLotNotWatchable = errors.New("auction lot is not watchable")
	ErrInvalidAlertDirection  = errors.New("invalid alert direction")
	ErrInvalidTargetPrice     = errors.New("invalid target price")
	ErrInvalidReminderWindow  = errors.New("invalid reminder window")
)

type PriceAlertCreateRequest struct {
	AuctionLotID uint    `json:"auctionLotId" binding:"required"`
	TargetPrice  float64 `json:"targetPrice" binding:"required"`
	Direction    string  `json:"direction"`
}

type BidReminderCreateRequest struct {
	AuctionLotID  uint `json:"auctionLotId" binding:"required"`
	MinutesBefore int  `json:"minutesBefore"`
}

type AuctionAlertService struct {
	alertRepo    *repository.PriceAlertRepository
	reminderRepo *repository.BidReminderRepository
	auctionRepo  *repository.AuctionLotRepository
}

func NewAuctionAlertService(
	alertRepo *repository.PriceAlertRepository,
	reminderRepo *repository.BidReminderRepository,
	auctionRepo *repository.AuctionLotRepository,
) *AuctionAlertService {
	return &AuctionAlertService{alertRepo: alertRepo, reminderRepo: reminderRepo, auctionRepo: auctionRepo}
}

func (s *AuctionAlertService) ListAlerts(userID uint) ([]models.PriceAlert, error) {
	return s.alertRepo.ListByUser(userID)
}

func (s *AuctionAlertService) CreateAlert(userID uint, req PriceAlertCreateRequest) (*models.PriceAlert, error) {
	if req.TargetPrice <= 0 {
		return nil, ErrInvalidTargetPrice
	}
	direction := strings.ToLower(strings.TrimSpace(req.Direction))
	if direction == "" {
		direction = "above"
	}
	if direction != "above" && direction != "below" {
		return nil, ErrInvalidAlertDirection
	}
	if err := s.ensureWatchableLot(req.AuctionLotID, userID); err != nil {
		return nil, err
	}

	alert := &models.PriceAlert{
		AuctionLotID: req.AuctionLotID,
		UserID:       userID,
		TargetPrice:  req.TargetPrice,
		Direction:    direction,
	}
	if err := s.alertRepo.Create(alert); err != nil {
		return nil, err
	}
	return alert, nil
}

func (s *AuctionAlertService) DeleteAlert(id, userID uint) error {
	return s.alertRepo.Delete(id, userID)
}

func (s *AuctionAlertService) ListReminders(userID uint) ([]models.BidReminder, error) {
	return s.reminderRepo.ListByUser(userID)
}

func (s *AuctionAlertService) CreateReminder(userID uint, req BidReminderCreateRequest) (*models.BidReminder, error) {
	minutes := req.MinutesBefore
	if minutes == 0 {
		minutes = 30
	}
	if minutes < 1 || minutes > 10080 {
		return nil, ErrInvalidReminderWindow
	}
	if err := s.ensureWatchableLot(req.AuctionLotID, userID); err != nil {
		return nil, err
	}

	reminder := &models.BidReminder{
		AuctionLotID:  req.AuctionLotID,
		UserID:        userID,
		MinutesBefore: minutes,
	}
	if err := s.reminderRepo.Create(reminder); err != nil {
		return nil, err
	}
	return reminder, nil
}

func (s *AuctionAlertService) DeleteReminder(id, userID uint) error {
	return s.reminderRepo.Delete(id, userID)
}

func (s *AuctionAlertService) ensureWatchableLot(lotID, userID uint) error {
	lot, err := s.auctionRepo.GetByID(lotID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrAuctionLotNotWatchable
		}
		return err
	}
	if lot.Status != models.AuctionStatusWatching && lot.Status != models.AuctionStatusBidding {
		return ErrAuctionLotNotWatchable
	}
	return nil
}

type AuctionAlertEvaluationResult struct {
	LotsChecked          int
	PriceAlertsTriggered int
	BidRemindersSent     int
}

type AuctionAlertEvaluator struct {
	alertRepo       *repository.PriceAlertRepository
	reminderRepo    *repository.BidReminderRepository
	notificationSvc *NotificationService
	logger          *Logger
}

func NewAuctionAlertEvaluator(
	alertRepo *repository.PriceAlertRepository,
	reminderRepo *repository.BidReminderRepository,
	notificationSvc *NotificationService,
	logger *Logger,
) *AuctionAlertEvaluator {
	return &AuctionAlertEvaluator{
		alertRepo:       alertRepo,
		reminderRepo:    reminderRepo,
		notificationSvc: notificationSvc,
		logger:          logger,
	}
}

func (e *AuctionAlertEvaluator) Evaluate(now time.Time) (AuctionAlertEvaluationResult, error) {
	result := AuctionAlertEvaluationResult{}
	seenLots := make(map[uint]struct{})
	var notificationFailures []string

	alerts, err := e.alertRepo.ListPendingWithLots()
	if err != nil {
		return result, err
	}
	for _, alert := range alerts {
		seenLots[alert.AuctionLotID] = struct{}{}
		if alert.AuctionLot.CurrentBid == nil || !priceAlertCrossed(alert) {
			continue
		}
		sent, err := e.notifyPriceAlert(alert, now)
		if err != nil {
			notificationFailures = append(notificationFailures, err.Error())
			continue
		}
		if sent {
			result.PriceAlertsTriggered++
		}
	}

	reminders, err := e.reminderRepo.ListPendingWithLots()
	if err != nil {
		return result, err
	}
	for _, reminder := range reminders {
		seenLots[reminder.AuctionLotID] = struct{}{}
		if reminder.AuctionLot.AuctionEndTime == nil || !bidReminderDue(reminder, now) {
			continue
		}
		sent, err := e.notifyBidReminder(reminder, now)
		if err != nil {
			notificationFailures = append(notificationFailures, err.Error())
			continue
		}
		if sent {
			result.BidRemindersSent++
		}
	}

	result.LotsChecked = len(seenLots)
	if len(notificationFailures) > 0 {
		return result, fmt.Errorf("%d auction alert notification(s) failed: %s", len(notificationFailures), strings.Join(notificationFailures, "; "))
	}
	return result, nil
}

func priceAlertCrossed(alert models.PriceAlert) bool {
	bid := alert.AuctionLot.CurrentBid
	if bid == nil {
		return false
	}
	switch alert.Direction {
	case "below":
		return *bid <= alert.TargetPrice
	default:
		return *bid >= alert.TargetPrice
	}
}

func bidReminderDue(reminder models.BidReminder, now time.Time) bool {
	if reminder.AuctionLot.AuctionEndTime == nil {
		return false
	}
	start := reminder.AuctionLot.AuctionEndTime.Add(-time.Duration(reminder.MinutesBefore) * time.Minute)
	return (now.Equal(start) || now.After(start)) && now.Before(*reminder.AuctionLot.AuctionEndTime)
}

// notifyPriceAlert claims the alert (so it fires once) and notifies the user. The in-app
// notification (via notificationSvc) always fires once claimed; Pushover delivery inside it
// is best-effort and its failure does not un-claim the alert — unlike the old Pushover-only
// design, a delivery hiccup on one channel should not suppress the other (specs/_backlog/F027).
func (e *AuctionAlertEvaluator) notifyPriceAlert(alert models.PriceAlert, now time.Time) (bool, error) {
	claimed, err := e.alertRepo.MarkTriggeredIfPending(alert.ID, now)
	if err != nil {
		e.logger.Error("scheduler", "Failed to mark price alert %d triggered: %s", alert.ID, err)
		return false, fmt.Errorf("price alert %d: %w", alert.ID, err)
	}
	if !claimed {
		return false, nil
	}
	if e.notificationSvc != nil {
		e.notificationSvc.NotifyAuctionPriceAlert(alert.UserID, alert.AuctionLot, alert.TargetPrice)
	}
	return true, nil
}

// notifyBidReminder claims the reminder and notifies the user. See notifyPriceAlert for why
// Pushover delivery failure no longer un-claims the reminder.
func (e *AuctionAlertEvaluator) notifyBidReminder(reminder models.BidReminder, now time.Time) (bool, error) {
	claimed, err := e.reminderRepo.MarkNotifiedIfPending(reminder.ID, now)
	if err != nil {
		e.logger.Error("scheduler", "Failed to mark bid reminder %d notified: %s", reminder.ID, err)
		return false, fmt.Errorf("bid reminder %d: %w", reminder.ID, err)
	}
	if !claimed {
		return false, nil
	}
	if e.notificationSvc != nil {
		e.notificationSvc.NotifyAuctionBidReminder(reminder.UserID, reminder.AuctionLot, reminder.MinutesBefore)
	}
	return true, nil
}

func auctionLotLabel(lot models.AuctionLot) string {
	house := strings.TrimSpace(lot.AuctionHouse)
	if house == "" {
		house = "Auction"
	}
	sale := strings.TrimSpace(lot.SaleName)
	if sale == "" {
		sale = "Sale"
	}
	if lot.LotNumber > 0 {
		return fmt.Sprintf("%s - %s (Lot %d)", house, sale, lot.LotNumber)
	}
	title := strings.TrimSpace(lot.Title)
	if title == "" {
		title = "auction lot"
	}
	return fmt.Sprintf("%s - %s", house, title)
}

func auctionLotURL(lot models.AuctionLot) string {
	if strings.TrimSpace(lot.SourceURL) != "" {
		return strings.TrimSpace(lot.SourceURL)
	}
	return strings.TrimSpace(lot.NumisBidsURL)
}

func auctionCurrency(currency string) string {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return "USD"
	}
	return currency
}
