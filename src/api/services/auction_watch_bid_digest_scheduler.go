package services

import (
	"fmt"
	"html"
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
	message.UserKey = user.PushoverUserKey

	if err := s.pushoverSvc.SendMessage(message); err != nil {
		s.logger.Error("scheduler", "Failed to send auction watch bid digest to user %d: %s", userID, err)
		return false
	}
	s.recordDigestedBids(reported)
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

// buildAuctionWatchBidDigestMessage renders the multi-lot digest as Pushover HTML and returns
// the lots it actually named, in the order it named them. Lots are grouped under the sale
// they belong to, and each one gets two lines: a shortened title with its lot number linked
// to the lot on the auction site, then its current high bid compared against the bid the
// previous digest reported ("up from 75.00", "down from 95.00", "no change") so the digest
// answers "did anything move?" without the user remembering yesterday's numbers
// (specs/_backlog/F032). Every interpolated value is HTML-escaped — lot titles and sale names
// are scraped from third-party auction sites, so they can carry markup — matching the
// newly-tracked and now-bidding pushes (F031).
//
// Lots are dropped (with a trailing summary line) once the message would exceed Pushover's
// message length limit, so a long watchlist can never cause an API rejection
// (specs/_backlog/F027). The limit counts markup, so naming each sale once instead of once
// per lot is what pays for the per-lot links.
func buildAuctionWatchBidDigestMessage(lots []models.AuctionLot) (PushoverMessage, []models.AuctionLot) {
	grouped := groupAuctionLotsBySale(lots)

	renderedSale := ""
	message, included := buildBatchedLotMessageWithIncluded(
		fmt.Sprintf("%d watched auction lot(s):\n\n", len(grouped)),
		grouped,
		func(lot models.AuctionLot) string {
			entry := ""
			// The heading belongs to the same entry as the first lot under it, so the
			// length trim can never leave a sale heading with no lots beneath it.
			if sale := auctionLotSaleLabel(lot); sale != renderedSale {
				renderedSale = sale
				entry = fmt.Sprintf("<i>%s</i>\n", html.EscapeString(sale))
			}
			return entry + fmt.Sprintf(
				"%s\n- Current high bid: %s\n\n",
				auctionWatchBidDigestHeadline(lot),
				html.EscapeString(formatAuctionDigestBid(lot)),
			)
		},
	)

	return PushoverMessage{
		Title:   "Auction Watch Bid Digest",
		Message: message,
		HTML:    true,
	}, grouped[:included]
}

// groupAuctionLotsBySale reorders lots so every lot in one sale is contiguous, keeping sales
// in the order they first appear and lots in their original order within a sale. The digest
// names a sale once and lists its lots under it, which only works if they are adjacent, and
// the repository's end-time ordering cannot guarantee that: lots in one sale close at
// staggered times (CNG electronic auctions do), so two sales closing the same evening would
// otherwise interleave.
func groupAuctionLotsBySale(lots []models.AuctionLot) []models.AuctionLot {
	saleOrder := make([]string, 0, len(lots))
	bySale := make(map[string][]models.AuctionLot, len(lots))
	for _, lot := range lots {
		sale := auctionLotSaleLabel(lot)
		if _, seen := bySale[sale]; !seen {
			saleOrder = append(saleOrder, sale)
		}
		bySale[sale] = append(bySale[sale], lot)
	}

	grouped := make([]models.AuctionLot, 0, len(lots))
	for _, sale := range saleOrder {
		grouped = append(grouped, bySale[sale]...)
	}
	return grouped
}

// auctionWatchBidDigestHeadline is the lot's leading line: a shortened title in bold plus its
// lot number, linked to that lot on the auction site so the digest is one tap from the page
// that can be bid on. The lot number moves up here because the sale is now a heading above
// the lot, and it is left unlinked when the lot has no usable provider URL.
func auctionWatchBidDigestHeadline(lot models.AuctionLot) string {
	headline := fmt.Sprintf("<b>%s</b>", html.EscapeString(auctionLotShortTitle(lot)))
	if lot.LotNumber <= 0 {
		return headline
	}

	lotNumber := fmt.Sprintf("Lot %d", lot.LotNumber)
	if lotURL := auctionLotProviderURL(lot); lotURL != "" {
		lotNumber = fmt.Sprintf("<a href=\"%s\">%s</a>", html.EscapeString(lotURL), lotNumber)
	}
	return fmt.Sprintf("%s (%s)", headline, lotNumber)
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
