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

	message, reported := buildAuctionWatchBidDigestMessage(lots)

	if err := s.pushoverSvc.SendNotification(user.PushoverUserKey, "Auction Watch Bid Digest", message, ""); err != nil {
		s.logger.Error("scheduler", "Failed to send auction watch bid digest to user %d: %s", userID, err)
		return false
	}
	s.recordDigestedBids(lots[:reported])
	return true
}

// recordDigestedBids stores the bids this digest just reported as the baseline the next one
// compares against. Only lots the delivered message actually named are recorded, and only
// after a successful send, so a failed push or a lot trimmed for length keeps its previous
// baseline and its change surfaces in the next digest instead of being lost.
func (s *AuctionWatchBidDigestScheduler) recordDigestedBids(lots []models.AuctionLot) {
	if s.auctionRepo == nil || len(lots) == 0 {
		return
	}
	if err := s.auctionRepo.SaveWatchBidDigestBids(lots); err != nil {
		s.logger.Error("scheduler", "Failed to record digested auction bids: %s", err)
	}
}

// buildAuctionWatchBidDigestMessage renders the multi-lot digest body and reports how many
// lots it named. Each watched lot gets three lines: a shortened title with its lot number,
// the sale it belongs to, and its current high bid compared against the bid the previous
// digest reported ("up from 75.00", "down from 95.00", "no change") so the digest answers
// "did anything move?" without the user remembering yesterday's numbers (specs/_backlog/F032).
// Lots are dropped (with a trailing summary line) once the message would exceed Pushover's
// message length limit, so a long watchlist can never cause an API rejection
// (specs/_backlog/F027); the returned count is what the caller snapshots.
func buildAuctionWatchBidDigestMessage(lots []models.AuctionLot) (string, int) {
	return buildBatchedLotMessageWithIncluded(
		fmt.Sprintf("%d watched auction lot(s):\n\n", len(lots)),
		lots,
		func(lot models.AuctionLot) string {
			return fmt.Sprintf(
				"%s\n%s\n- Current high bid: %s\n\n",
				auctionWatchBidDigestHeadline(lot),
				auctionLotSaleLabel(lot),
				formatAuctionDigestBid(lot),
			)
		},
	)
}

// auctionWatchBidDigestHeadline is the lot's leading line: a shortened title plus its lot
// number. The lot number moves up here because the sale line below it no longer carries one.
func auctionWatchBidDigestHeadline(lot models.AuctionLot) string {
	title := auctionLotShortTitle(lot)
	if lot.LotNumber > 0 {
		return fmt.Sprintf("%s (Lot %d)", title, lot.LotNumber)
	}
	return title
}

// auctionLotShortTitle trims a scraped catalog description down to its leading clause so a
// digest of a dozen lots stays scannable. Provider titles read
// "PAMPHYLIA, Aspendos. Circa 380/75-330/25 BC. AR Stater (20mm, 10.85 g, 2h). VF." — the
// first sentence is the identifying half and the rest is detail the user can look up in the
// app. Titles with no sentence break are capped instead, so one runaway title cannot crowd
// every other lot out of the length-limited body.
func auctionLotShortTitle(lot models.AuctionLot) string {
	title := auctionLotTitle(lot)
	if index := strings.Index(title, ". "); index > 0 {
		title = strings.TrimSpace(title[:index])
	}
	title = strings.TrimRight(title, ".")
	if title == "" {
		return auctionLotTitle(lot)
	}
	return truncateRunes(title, auctionLotShortTitleLimit)
}

// auctionLotShortTitleLimit caps a digest title line. Chosen so a lot's three lines stay well
// under a phone notification's readable width while leaving room for several lots inside
// pushoverMessageLimit.
const auctionLotShortTitleLimit = 60

// formatAuctionDigestBid renders the bid line's value: the current high bid, followed by how
// it compares with the bid the last digest reported for this lot. A lot that has never been
// reported before has nothing to compare against, so it shows the bid alone.
func formatAuctionDigestBid(lot models.AuctionLot) string {
	if lot.CurrentBid == nil {
		return "unavailable"
	}
	amount := formatAuctionBidAmount(*lot.CurrentBid, lot.Currency)
	if lot.LastDigestBid == nil {
		return amount
	}
	switch previous := *lot.LastDigestBid; {
	case *lot.CurrentBid > previous:
		return fmt.Sprintf("%s (up from %.2f)", amount, previous)
	case *lot.CurrentBid < previous:
		return fmt.Sprintf("%s (down from %.2f)", amount, previous)
	default:
		return fmt.Sprintf("%s (no change)", amount)
	}
}

func formatAuctionBid(bid *float64, currency string) string {
	if bid == nil {
		return "current high bid unavailable"
	}
	return fmt.Sprintf("current high bid %s", formatAuctionBidAmount(*bid, currency))
}

// formatAuctionBidAmount renders a bid as "80.00 USD", defaulting a blank currency to USD.
func formatAuctionBidAmount(bid float64, currency string) string {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "USD"
	}
	return fmt.Sprintf("%.2f %s", bid, currency)
}
