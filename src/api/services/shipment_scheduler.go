package services

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// ShipmentScheduler polls carrier APIs for shipment status updates.
type ShipmentScheduler struct {
	shipmentSvc *ShipmentService
	settingsSvc *SettingsService
	logger      *Logger

	stopCh    chan struct{}
	once      sync.Once
	statusMu  sync.RWMutex
	isRunning bool
}

const (
	shipmentSyncTimeout = 2 * time.Minute
)

func NewShipmentScheduler(shipmentSvc *ShipmentService, settingsSvc *SettingsService, logger *Logger) *ShipmentScheduler {
	return &ShipmentScheduler{
		shipmentSvc: shipmentSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

func (s *ShipmentScheduler) Start() {
	s.logger.Info("scheduler", "Shipment sync scheduler started")
	scheduleLoop{
		category:     "scheduler",
		name:         "shipment sync",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 30 * time.Second,
		next:         func(now time.Time, _ bool) time.Time { return s.nextRun(now) },
		run:          s.runCycle,
	}.Run()
}

func (s *ShipmentScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

func (s *ShipmentScheduler) RunNow() error {
	return s.runCycleWithTrigger("manual")
}

func (s *ShipmentScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()

	return SchedulerStatus{
		Name:      "shipment-sync",
		Enabled:   s.isEnabled(),
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

func (s *ShipmentScheduler) runCycle() {
	if !s.isEnabled() {
		s.logger.Debug("scheduler", "Shipment sync disabled, skipping cycle")
		return
	}
	_ = s.runCycleWithTrigger("scheduled")
}

func (s *ShipmentScheduler) runCycleWithTrigger(triggerType string) error {
	s.statusMu.Lock()
	s.isRunning = true
	s.statusMu.Unlock()
	defer func() {
		s.statusMu.Lock()
		s.isRunning = false
		s.statusMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), shipmentSyncTimeout)
	defer cancel()

	summary, err := s.shipmentSvc.SyncCandidates(ctx, nil, s.getBatchSize())
	if err != nil {
		s.logger.Error("scheduler", "%s shipment sync failed: %v", triggerType, err)
		return err
	}

	s.logger.Info(
		"scheduler",
		"%s shipment sync complete — %d checked, %d updated, %d failed",
		triggerType,
		summary.Checked,
		summary.Updated,
		summary.Failed,
	)
	return nil
}

func (s *ShipmentScheduler) isEnabled() bool {
	return s.settingsSvc.GetSetting(SettingParcelAppEnabled) == "true"
}

func (s *ShipmentScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

func (s *ShipmentScheduler) nextRun(now time.Time) time.Time {
	h, m := s.getStartTime()
	schedule := dailySchedule{Hour: h, Minute: m, Interval: s.getInterval(), Location: scheduleLocation(s.settingsSvc, s.logger)}
	return schedule.slotAfter(now)
}

func (s *ShipmentScheduler) getStartTime() (int, int) {
	return parseStartTime(s.settingsSvc.GetSetting(SettingShipmentSyncStartTime), 9, 0)
}

func (s *ShipmentScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingShipmentSyncInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 20 {
		mins = 20
	}
	return time.Duration(mins) * time.Minute
}

func (s *ShipmentScheduler) getBatchSize() int {
	raw := s.settingsSvc.GetSetting(SettingShipmentSyncBatchSize)
	size, err := strconv.Atoi(raw)
	if err != nil || size < 1 {
		size = 100
	}
	if size > 1000 {
		size = 1000
	}
	return size
}
