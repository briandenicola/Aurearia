package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// CoinOfDayScheduler runs a daily job that selects one coin per opted-in user
// and sends them a "Coin of the Day" notification + Pushover alert.
type CoinOfDayScheduler struct {
	featuredRepo *repository.FeaturedCoinRepository
	userRepo     *repository.UserRepository
	coinRepo     *repository.CoinRepository
	notifSvc     *NotificationService
	settingsSvc  *SettingsService
	logger       *Logger
	stopCh       chan struct{}
	once         sync.Once
	lastPicked   map[uint]string // userID -> "YYYY-MM-DD" idempotency cache
	mu           sync.RWMutex
}

// NewCoinOfDayScheduler creates a new scheduler.
func NewCoinOfDayScheduler(
	featuredRepo *repository.FeaturedCoinRepository,
	userRepo *repository.UserRepository,
	coinRepo *repository.CoinRepository,
	notifSvc *NotificationService,
	settingsSvc *SettingsService,
	logger *Logger,
) *CoinOfDayScheduler {
	return &CoinOfDayScheduler{
		featuredRepo: featuredRepo,
		userRepo:     userRepo,
		coinRepo:     coinRepo,
		notifSvc:     notifSvc,
		settingsSvc:  settingsSvc,
		logger:       logger,
		stopCh:       make(chan struct{}),
		lastPicked:   make(map[uint]string),
	}
}

// Start begins the daily loop. Call from a goroutine.
func (s *CoinOfDayScheduler) Start() {
	s.logger.Info("scheduler", "Coin of the Day scheduler started")

	// Initial delay so the app finishes startup
	select {
	case <-time.After(45 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next coin-of-the-day pick in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Coin of the Day scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *CoinOfDayScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// timeUntilNextRun returns the duration until the next daily anchor (HH:MM).
func (s *CoinOfDayScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	h, m := s.getStartTime()
	anchor := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	if anchor.After(now) {
		return anchor.Sub(now)
	}
	// Past today's anchor — schedule for tomorrow
	tomorrow := anchor.Add(24 * time.Hour)
	return tomorrow.Sub(now)
}

// getStartTime parses HH:MM, defaults to 07:00.
func (s *CoinOfDayScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingCoinOfDayStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 7, 0
	}
	return h, m
}

// runCycle is the scheduled (non-manual) entry point. Gated on the global setting.
func (s *CoinOfDayScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingCoinOfDayEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Coin of the Day disabled, skipping cycle")
		return
	}
	_, _, _, _ = s.runCycleWithTrigger("scheduled")
}

// RunNow executes an immediate coin-of-the-day pick for all opted-in users.
// Returns counts for admin manual-trigger response.
func (s *CoinOfDayScheduler) RunNow() (picked int, skipped int, errs int, err error) {
	return s.runCycleWithTrigger("manual")
}

// runCycleWithTrigger iterates opted-in users and picks/notifies each.
func (s *CoinOfDayScheduler) runCycleWithTrigger(triggerType string) (int, int, int, error) {
	s.logger.Info("scheduler", "Starting %s coin-of-the-day cycle", triggerType)
	started := time.Now()

	users, err := s.userRepo.ListCoinOfDayEnabled()
	if err != nil {
		s.logger.Error("scheduler", "Failed to list opt-in users: %v", err)
		return 0, 0, 0, err
	}

	today := time.Now().Format("2006-01-02")
	picked, skipped, errs := 0, 0, 0

	for _, user := range users {
		// In-memory idempotency to prevent duplicate picks within the same day.
		s.mu.RLock()
		lastDate := s.lastPicked[user.ID]
		s.mu.RUnlock()

		if lastDate == today {
			s.logger.Debug("scheduler", "User %d already picked today, skipping", user.ID)
			skipped++
			continue
		}

		// DB-level idempotency: don't double-feature if a record exists for today.
		alreadyFeatured, err := s.featuredRepo.HasBeenFeaturedToday(user.ID, time.Now())
		if err != nil {
			s.logger.Error("scheduler", "Failed to check today's feature for user %d: %v", user.ID, err)
			errs++
			continue
		}
		if alreadyFeatured {
			s.mu.Lock()
			s.lastPicked[user.ID] = today
			s.mu.Unlock()
			skipped++
			continue
		}

		coinID, err := s.featuredRepo.PickNextCoinID(user.ID)
		if err != nil {
			s.logger.Error("scheduler", "Failed to pick coin for user %d: %v", user.ID, err)
			errs++
			continue
		}
		if coinID == 0 {
			s.logger.Debug("scheduler", "User %d has no eligible coins", user.ID)
			skipped++
			continue
		}

		coin, err := s.coinRepo.FindByID(coinID, user.ID)
		if err != nil {
			s.logger.Error("scheduler", "Failed to load coin %d for user %d: %v", coinID, user.ID, err)
			errs++
			continue
		}

		summary := buildCoinSummary(coin)

		fc := &models.FeaturedCoin{
			UserID:     user.ID,
			CoinID:     coinID,
			Summary:    summary,
			FeaturedAt: time.Now(),
		}
		if err := s.featuredRepo.Create(fc); err != nil {
			s.logger.Error("scheduler", "Failed to persist featured coin for user %d: %v", user.ID, err)
			errs++
			continue
		}

		s.notifSvc.NotifyCoinOfDay(user.ID, fc.ID, coin.Name, summary)

		s.mu.Lock()
		s.lastPicked[user.ID] = today
		s.mu.Unlock()

		picked++
		s.logger.Info("scheduler", "Featured coin %d for user %d", coinID, user.ID)
	}

	s.logger.Info("scheduler", "%s coin-of-the-day cycle complete in %s — %d picked, %d skipped, %d errors",
		triggerType, time.Since(started), picked, skipped, errs)
	return picked, skipped, errs, nil
}

// buildCoinSummary returns a short summary suitable for the in-app modal.
// Prefers AIAnalysis or the combined Obverse/Reverse analyses; falls back to
// a structured one-liner from the coin's metadata.
func buildCoinSummary(coin *models.Coin) string {
	if coin == nil {
		return ""
	}
	if s := strings.TrimSpace(coin.AIAnalysis); s != "" {
		return s
	}
	parts := []string{}
	if strings.TrimSpace(coin.ObverseAnalysis) != "" {
		parts = append(parts, "Obverse:\n"+coin.ObverseAnalysis)
	}
	if strings.TrimSpace(coin.ReverseAnalysis) != "" {
		parts = append(parts, "Reverse:\n"+coin.ReverseAnalysis)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n\n")
	}

	// Fallback summary from structured fields
	bits := []string{}
	if coin.Denomination != "" {
		bits = append(bits, coin.Denomination)
	}
	if coin.Ruler != "" {
		bits = append(bits, "of "+coin.Ruler)
	}
	if coin.Era != "" {
		bits = append(bits, "("+string(coin.Era)+")")
	}
	if coin.Mint != "" {
		bits = append(bits, "minted at "+coin.Mint)
	}
	if len(bits) == 0 {
		return coin.Name
	}
	return strings.Join(bits, " ")
}
