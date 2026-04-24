package services

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// AvailabilityScheduler runs periodic wishlist availability checks.
type AvailabilityScheduler struct {
	svc      *AvailabilityService
	coinRepo *repository.CoinRepository
	logger   *Logger
	stopCh   chan struct{}
	once     sync.Once
}

// NewAvailabilityScheduler creates a new scheduler.
func NewAvailabilityScheduler(
	svc *AvailabilityService,
	coinRepo *repository.CoinRepository,
) *AvailabilityScheduler {
	return &AvailabilityScheduler{
		svc:      svc,
		coinRepo: coinRepo,
		logger:   AppLogger,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic check loop. Call from a goroutine.
func (s *AvailabilityScheduler) Start() {
	s.logger.Info("scheduler", "Wishlist availability scheduler started")

	// Initial delay to let the app finish startup
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		// Wait until the next scheduled time before running
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next availability check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *AvailabilityScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// timeUntilNextRun calculates the delay until the next scheduled run.
// Uses WishlistCheckStartTime (HH:MM) as the daily anchor and
// WishlistCheckInterval (minutes) as the repeat cadence.
func (s *AvailabilityScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	startHour, startMin := s.getStartTime()
	interval := s.getInterval()

	// Build today's anchor from the start time
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())

	// If anchor is in the future, that's the next run
	if anchor.After(now) {
		return anchor.Sub(now)
	}

	// Find the next occurrence: anchor + N*interval that is still in the future
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

// getStartTime parses HH:MM from settings, defaults to 02:00.
func (s *AvailabilityScheduler) getStartTime() (int, int) {
	raw := GetSetting(SettingWishlistCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 2, 0
	}
	return h, m
}

// getInterval returns the configured check interval.
func (s *AvailabilityScheduler) getInterval() time.Duration {
	minStr := GetSetting(SettingWishlistCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 120
	}
	return time.Duration(mins) * time.Minute
}

// runCycle executes one full availability check for all users.
func (s *AvailabilityScheduler) runCycle() {
	enabled := GetSetting(SettingWishlistCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Wishlist checking disabled, skipping cycle")
		return
	}

	s.logger.Info("scheduler", "Starting scheduled availability check cycle")

	coins, err := s.coinRepo.GetAllWishlistWithURLs()
	if err != nil {
		s.logger.Error("scheduler", "Failed to fetch all wishlist coins: %s", err)
		return
	}

	if len(coins) == 0 {
		s.logger.Info("scheduler", "No wishlist coins with URLs found")
		return
	}

	// Group coins by user
	userCoins := make(map[uint]bool)
	for _, coin := range coins {
		userCoins[coin.UserID] = true
	}

	s.logger.Info("scheduler", "Found %d coins across %d users", len(coins), len(userCoins))

	for userID := range userCoins {
		_, err := s.svc.CheckWishlistForUser(userID, "scheduled", nil)
		if err != nil {
			s.logger.Error("scheduler", "Scheduled check failed for user %d: %s", userID, err)
		}
	}

	s.logger.Info("scheduler", "Scheduled availability check cycle complete")
}
