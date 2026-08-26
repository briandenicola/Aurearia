package services

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// AuctionWatchBidDigestScheduler refreshes watched auction lots and sends a current-bid digest.
type AuctionWatchBidDigestScheduler struct {
	auctionRepo *repository.AuctionLotRepository
	runRepo     *repository.AuctionWatchBidDigestRepository
	userRepo    *repository.UserRepository
	pushoverSvc *PushoverService
	syncSvc     *AuctionWatchlistSyncService
	settingsSvc *SettingsService
	logger      *Logger

	stopChan  chan struct{}
	isRunning bool
	mu        sync.RWMutex
}

func NewAuctionWatchBidDigestScheduler(
	auctionRepo *repository.AuctionLotRepository,
	runRepo *repository.AuctionWatchBidDigestRepository,
	userRepo *repository.UserRepository,
	pushoverSvc *PushoverService,
	syncSvc *AuctionWatchlistSyncService,
	settingsSvc *SettingsService,
	logger *Logger,
) *AuctionWatchBidDigestScheduler {
	return &AuctionWatchBidDigestScheduler{
		auctionRepo: auctionRepo,
		runRepo:     runRepo,
		userRepo:    userRepo,
		pushoverSvc: pushoverSvc,
		syncSvc:     syncSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

func (s *AuctionWatchBidDigestScheduler) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	s.logger.Info("scheduler", "Auction watch bid digest scheduler started")
	for {
		select {
		case <-s.stopChan:
			s.logger.Info("scheduler", "Auction watch bid digest scheduler stopped")
			return
		case <-time.After(s.timeUntilNextRun()):
			if !s.isEnabled() {
				s.logger.Debug("scheduler", "Auction watch bid digest disabled, skipping")
				continue
			}
			s.runDigest("scheduled", nil)
		}
	}
}

func (s *AuctionWatchBidDigestScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning {
		s.isRunning = false
		close(s.stopChan)
	}
}

func (s *AuctionWatchBidDigestScheduler) RunNow() error {
	if !s.isEnabled() {
		s.logger.Info("scheduler", "Manual auction watch bid digest requested while disabled; running anyway")
	}
	s.runDigest("manual", nil)
	return nil
}

func (s *AuctionWatchBidDigestScheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	running := s.isRunning
	s.mu.RUnlock()

	return SchedulerStatus{
		Name:      "auction-watch-bid-digest",
		Enabled:   s.isEnabled(),
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

func (s *AuctionWatchBidDigestScheduler) isEnabled() bool {
	return s.settingsSvc.GetSetting(SettingAuctionWatchBidDigestEnabled) == "true"
}

func (s *AuctionWatchBidDigestScheduler) getStartTime() string {
	value := s.settingsSvc.GetSetting(SettingAuctionWatchBidDigestStartTime)
	if value == "" {
		return "08:00"
	}
	return value
}

func (s *AuctionWatchBidDigestScheduler) getIntervalMinutes() int {
	value := s.settingsSvc.GetSetting(SettingAuctionWatchBidDigestInterval)
	if value == "" {
		return 1440
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes < 60 {
		return 1440
	}
	return minutes
}

func (s *AuctionWatchBidDigestScheduler) timeUntilNextRun() time.Duration {
	lastRun := s.runRepo.GetLastScheduledRun()
	interval := time.Duration(s.getIntervalMinutes()) * time.Minute
	now := time.Now()

	if lastRun != nil && lastRun.CompletedAt != nil {
		nextRun := lastRun.CompletedAt.Add(interval)
		if nextRun.After(now) {
			return nextRun.Sub(now)
		}
		return 0
	}

	startTime := s.getStartTime()
	parsed, err := time.Parse("15:04", startTime)
	if err != nil {
		parsed, _ = time.Parse("15:04", "08:00")
	}

	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	if !nextRun.After(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}
	return nextRun.Sub(now)
}

func (s *AuctionWatchBidDigestScheduler) runDigest(triggerType string, triggerUserID *uint) {
	start := time.Now()
	run := &models.AuctionWatchBidDigestRun{
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        "running",
		StartedAt:     start,
	}
	if err := s.runRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create auction watch bid digest run: %s", err)
		return
	}

	if s.syncSvc != nil {
		stats := s.syncSvc.SyncAllConfiguredUsers()
		s.logger.Info("scheduler", "Auction watchlist sync complete — %d users checked, %d lots synced, %d errors", stats.UsersChecked, stats.LotsSynced, stats.Errors)
	}

	lots, err := s.auctionRepo.GetActiveWatchBidDigestLots()
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch active auction watch lots: %s", err)
		run.Status = "error"
		run.ErrorMessage = err.Error()
	} else {
		run.LotsChecked = len(lots)
		if len(lots) > 0 {
			userLots := make(map[uint][]models.AuctionLot)
			for _, lot := range lots {
				userLots[lot.UserID] = append(userLots[lot.UserID], lot)
			}
			for userID, lots := range userLots {
				if s.notifyUser(userID, lots) {
					run.DigestsSent++
				}
			}
		}
		run.Status = "success"
	}

	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.DurationMs = completedAt.Sub(start).Milliseconds()
	if err := s.runRepo.CompleteRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to complete auction watch bid digest run: %s", err)
	}
	s.logger.Info("scheduler", "%s auction watch bid digest complete — %d lots checked, %d digests sent", triggerType, run.LotsChecked, run.DigestsSent)
}

func (s *AuctionWatchBidDigestScheduler) notifyUser(userID uint, lots []models.AuctionLot) bool {
	if s.userRepo == nil || s.pushoverSvc == nil {
		return false
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil || !user.PushoverEnabled || user.PushoverUserKey == "" {
		return false
	}

	message := buildAuctionWatchBidDigestMessage(lots)

	if err := s.pushoverSvc.SendNotification(user.PushoverUserKey, "Auction Watch Bid Digest", message, ""); err != nil {
		s.logger.Error("scheduler", "Failed to send auction watch bid digest to user %d: %s", userID, err)
		return false
	}
	return true
}

// buildAuctionWatchBidDigestMessage renders the multi-lot digest body. Each watched lot
// leads with its title (falling back to "Untitled lot" via the shared auctionLotTitle
// helper), then a separate line with auction metadata and the current bid, reusing
// auctionLotLabel/formatAuctionBid so this stays consistent with the single-lot alert
// wording. Lots are dropped (with a trailing summary line) once the message would exceed
// Pushover's message length limit, so a long watchlist can never cause an API rejection
// (specs/_backlog/F027).
func buildAuctionWatchBidDigestMessage(lots []models.AuctionLot) string {
	return buildBatchedLotMessage(
		fmt.Sprintf("%d watched auction lot(s):\n\n", len(lots)),
		lots,
		func(lot models.AuctionLot) string {
			return fmt.Sprintf("%s\n%s: %s\n\n", auctionLotTitle(lot), auctionLotLabel(lot), formatAuctionBid(lot.CurrentBid, lot.Currency))
		},
	)
}

func formatAuctionBid(bid *float64, currency string) string {
	if bid == nil {
		return "current high bid unavailable"
	}
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "USD"
	}
	return fmt.Sprintf("current high bid %.2f %s", *bid, currency)
}
