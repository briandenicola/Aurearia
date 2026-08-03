package services

import (
	"context"
	"fmt"
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

	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}

	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next shipment sync in %s", wait)

		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Shipment sync scheduler stopped")
			return
		}

		s.runCycle()
	}
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
	return s.settingsSvc.GetSetting(SettingShipmentSyncEnabled) == "true"
}

func (s *ShipmentScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	interval := s.getInterval()
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

func (s *ShipmentScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingShipmentSyncStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 9, 0
	}
	return h, m
}

func (s *ShipmentScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingShipmentSyncInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 60
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
