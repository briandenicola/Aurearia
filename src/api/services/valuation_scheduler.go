package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// ValuationScheduler runs periodic collection valuation checks.
type ValuationScheduler struct {
	svc      *ValuationService
	coinRepo *repository.CoinRepository
	logger   *Logger
	stopCh   chan struct{}
}

// NewValuationScheduler creates a new scheduler.
func NewValuationScheduler(
	svc *ValuationService,
	coinRepo *repository.CoinRepository,
) *ValuationScheduler {
	return &ValuationScheduler{
		svc:      svc,
		coinRepo: coinRepo,
		logger:   AppLogger,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic valuation loop. Call from a goroutine.
func (s *ValuationScheduler) Start() {
	s.logger.Info("valuation-scheduler", "Collection valuation scheduler started")

	// Initial delay to let the app finish startup
	select {
	case <-time.After(60 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("valuation-scheduler", "Next valuation check in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("valuation-scheduler", "Scheduler stopped")
			return
		}

		s.runCycle()
	}
}

// Stop signals the scheduler to shut down.
func (s *ValuationScheduler) Stop() {
	close(s.stopCh)
}

// timeUntilNextRun calculates delay until the next scheduled run.
// Uses ValuationCheckStartTime (HH:MM) as the daily anchor and
// ValuationCheckIntervalDays as the repeat cadence.
func (s *ValuationScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	startHour, startMin := s.getStartTime()
	intervalDays := s.getIntervalDays()

	// Build today's anchor from the start time
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())

	// If anchor is in the future today, that's the next run
	if anchor.After(now) {
		return anchor.Sub(now)
	}

	// Find the next occurrence: anchor + N*interval that is still in the future
	interval := time.Duration(intervalDays) * 24 * time.Hour
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

// getStartTime parses HH:MM from settings, defaults to 03:00.
func (s *ValuationScheduler) getStartTime() (int, int) {
	raw := GetSetting(SettingValuationCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 3, 0
	}
	return h, m
}

// getIntervalDays returns the configured check interval in days.
func (s *ValuationScheduler) getIntervalDays() int {
	dayStr := GetSetting(SettingValuationCheckInterval)
	days, err := strconv.Atoi(dayStr)
	if err != nil || days < 1 {
		days = 7
	}
	return days
}

// runCycle executes one full valuation check for all users with owned coins.
func (s *ValuationScheduler) runCycle() {
	enabled := GetSetting(SettingValuationCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("valuation-scheduler", "Collection valuation disabled, skipping cycle")
		return
	}

	s.logger.Info("valuation-scheduler", "Starting scheduled valuation cycle")

	// Get distinct user IDs that have owned coins
	userIDs, err := s.svc.valRepo.GetUsersWithOwnedCoins()
	if err != nil {
		s.logger.Error("valuation-scheduler", "Failed to fetch users: %s", err)
		return
	}

	if len(userIDs) == 0 {
		s.logger.Info("valuation-scheduler", "No users with owned coins found")
		return
	}

	s.logger.Info("valuation-scheduler", "Found %d users with owned coins", len(userIDs))

	for _, userID := range userIDs {
		_, err := s.svc.ValuateCollectionForUser(userID, "scheduled", nil)
		if err != nil {
			s.logger.Error("valuation-scheduler", "Scheduled valuation failed for user %d: %s", userID, err)
		}
	}

	s.logger.Info("valuation-scheduler", "Scheduled valuation cycle complete")
}
