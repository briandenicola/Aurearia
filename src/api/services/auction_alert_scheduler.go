package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// AuctionAlertScheduler refreshes watched lots and evaluates price alerts and bid reminders.
type AuctionAlertScheduler struct {
	evaluator   *AuctionAlertEvaluator
	runRepo     *repository.AuctionAlertRunRepository
	syncSvc     *AuctionWatchlistSyncService
	settingsSvc *SettingsService
	logger      *Logger

	stopCh    chan struct{}
	once      sync.Once
	statusMu  sync.RWMutex
	isRunning bool
}

func NewAuctionAlertScheduler(
	evaluator *AuctionAlertEvaluator,
	runRepo *repository.AuctionAlertRunRepository,
	syncSvc *AuctionWatchlistSyncService,
	settingsSvc *SettingsService,
	logger *Logger,
) *AuctionAlertScheduler {
	return &AuctionAlertScheduler{
		evaluator:   evaluator,
		runRepo:     runRepo,
		syncSvc:     syncSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

func (s *AuctionAlertScheduler) Start() {
	s.logger.Info("scheduler", "Auction alerts scheduler started")

	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next auction alerts check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Auction alerts scheduler stopped")
			return
		}

		s.runCycle()
	}
}

func (s *AuctionAlertScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

func (s *AuctionAlertScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	return SchedulerStatus{
		Name:      "auction-alerts",
		Enabled:   s.isEnabled(),
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

func (s *AuctionAlertScheduler) RunNow() error {
	_, err := s.RunNowWithTrigger(nil)
	return err
}

func (s *AuctionAlertScheduler) RunNowWithTrigger(triggerUserID *uint) (*models.AuctionAlertRun, error) {
	return s.runCycleWithTrigger("manual", triggerUserID)
}

func (s *AuctionAlertScheduler) runCycle() {
	if !s.isEnabled() {
		s.logger.Debug("scheduler", "Auction alerts check disabled, skipping cycle")
		return
	}
	s.runCycleWithTrigger("scheduled", nil)
}

func (s *AuctionAlertScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) (*models.AuctionAlertRun, error) {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	startedAt := time.Now()
	run := &models.AuctionAlertRun{
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        "running",
		StartedAt:     startedAt,
	}
	if err := s.runRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create auction alerts run: %s", err)
		return nil, err
	}

	if s.syncSvc != nil {
		stats := s.syncSvc.SyncDigestEligibleUsers()
		s.logger.Info("scheduler", "Auction watchlist refresh before alerts complete — %d users checked, %d lots synced, %d errors", stats.UsersChecked, stats.LotsSynced, stats.Errors)
	}

	result, err := s.evaluator.Evaluate(time.Now())
	run.LotsChecked = result.LotsChecked
	run.PriceAlertsTriggered = result.PriceAlertsTriggered
	run.BidRemindersSent = result.BidRemindersSent
	if err != nil {
		run.Status = "error"
		run.ErrorMessage = fmt.Sprintf("Failed to evaluate auction alerts: %v", err)
	} else {
		run.Status = "success"
	}

	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.DurationMs = completedAt.Sub(startedAt).Milliseconds()
	if completeErr := s.runRepo.CompleteRun(run); completeErr != nil {
		s.logger.Error("scheduler", "Failed to complete auction alerts run: %s", completeErr)
	}
	s.logger.Info("scheduler", "%s auction alerts check complete — %d lots checked, %d price alerts, %d bid reminders", triggerType, run.LotsChecked, run.PriceAlertsTriggered, run.BidRemindersSent)

	return run, err
}

func (s *AuctionAlertScheduler) isEnabled() bool {
	return s.settingsSvc.GetSetting(SettingAuctionAlertsCheckEnabled) == "true"
}

func (s *AuctionAlertScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	interval := s.getInterval()

	lastRun := s.runRepo.GetLastScheduledRun()
	if lastRun != nil && lastRun.CompletedAt != nil {
		nextRun := lastRun.CompletedAt.Add(interval)
		if nextRun.After(now) {
			return nextRun.Sub(now)
		}
		return 0
	}

	startHour, startMin := s.getStartTime()
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())
	if anchor.After(now) {
		return anchor.Sub(now)
	}
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

func (s *AuctionAlertScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingAuctionAlertsCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 8, 0
	}
	return h, m
}

func (s *AuctionAlertScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingAuctionAlertsCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 60
	}
	return time.Duration(mins) * time.Minute
}
