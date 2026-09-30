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
	scheduleLoop{
		category:     "scheduler",
		name:         "auction alerts check",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 30 * time.Second,
		next:         s.nextRun,
		run:          s.runCycle,
	}.Run()
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
		stats := s.syncSvc.SyncAllConfiguredUsers()
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
	return max(s.nextRun(now, false).Sub(now), 0)
}

func (s *AuctionAlertScheduler) nextRun(now time.Time, catchUp bool) time.Time {
	var last *time.Time
	if run := s.runRepo.GetLastScheduledRun(); run != nil {
		last = run.CompletedAt
	}
	h, m := s.getStartTime()
	schedule := dailySchedule{Hour: h, Minute: m, Interval: s.getInterval(), Location: scheduleLocation(s.settingsSvc, s.logger)}
	return schedule.next(now, last, catchUp)
}

func (s *AuctionAlertScheduler) getStartTime() (int, int) {
	return parseStartTime(s.settingsSvc.GetSetting(SettingAuctionAlertsCheckStartTime), 8, 0)
}

func (s *AuctionAlertScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingAuctionAlertsCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 60
	}
	return time.Duration(mins) * time.Minute
}
