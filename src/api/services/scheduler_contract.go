package services

import "time"

// SchedulerStatus describes the current runtime status of a scheduler.
type SchedulerStatus struct {
	Name      string        `json:"name"`
	Enabled   bool          `json:"enabled"`
	IsRunning bool          `json:"isRunning"`
	NextRunIn time.Duration `json:"nextRunIn"`
}

// Scheduler defines the standard scheduler contract used by registry wiring.
type Scheduler interface {
	Start()
	Stop()
	RunNow() error
	timeUntilNextRun() time.Duration
	GetStatus() SchedulerStatus
}
