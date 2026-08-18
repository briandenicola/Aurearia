package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// AvailabilityCycleRepository encapsulates all AvailabilityCycle (parent) DB operations.
type AvailabilityCycleRepository struct {
	db *gorm.DB
}

// NewAvailabilityCycleRepository creates a new AvailabilityCycleRepository.
func NewAvailabilityCycleRepository(db *gorm.DB) *AvailabilityCycleRepository {
	return &AvailabilityCycleRepository{db: db}
}

// ChildCounts is the aggregation result of an AvailabilityCycle's children, grouped by status.
type ChildCounts struct {
	Total     int
	Queued    int
	Running   int
	Completed int
	Failed    int
}

// deriveCycleStatus computes the parent cycle status from its children's status counts.
// Truth table: any queued/running child keeps the cycle "running" (not yet terminal); once
// every child is terminal, the cycle is "completed" (zero failures), "failed" (all children
// failed), or "partial_failure" (a mix of completed and failed children).
func deriveCycleStatus(c ChildCounts) (status string, terminal bool) {
	if c.Queued > 0 || c.Running > 0 {
		return models.AvailabilityCycleStatusRunning, false
	}
	if c.Total == 0 {
		return models.AvailabilityCycleStatusCompleted, true
	}
	switch {
	case c.Failed == 0:
		return models.AvailabilityCycleStatusCompleted, true
	case c.Failed == c.Total:
		return models.AvailabilityCycleStatusFailed, true
	default:
		return models.AvailabilityCycleStatusPartialFailure, true
	}
}

// EnqueueCycle creates a queued cycle if no queued or running cycle already exists within
// the given since window (duplicate-cycle protection). Mirrors the existing
// AvailabilityRepository.EnqueueManualRun pattern.
func (r *AvailabilityCycleRepository) EnqueueCycle(cycle *models.AvailabilityCycle, since time.Time) (bool, error) {
	acquired := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.AvailabilityCycle{}).
			Where("status IN ? AND started_at >= ?",
				[]string{models.AvailabilityCycleStatusQueued, models.AvailabilityCycleStatusRunning},
				since,
			).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Create(cycle).Error; err != nil {
			return err
		}
		acquired = true
		return nil
	})
	return acquired, err
}

// ClaimCycle atomically transitions a queued cycle to running.
// Returns (cycle, true, nil) when claimed, (cycle, false, nil) if already claimed.
func (r *AvailabilityCycleRepository) ClaimCycle(cycleID uint) (*models.AvailabilityCycle, bool, error) {
	now := time.Now()
	var cycle models.AvailabilityCycle
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AvailabilityCycle{}).
			Where("id = ? AND status = ?", cycleID, models.AvailabilityCycleStatusQueued).
			Updates(map[string]interface{}{
				"status":     models.AvailabilityCycleStatusRunning,
				"started_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true
		return tx.First(&cycle, cycleID).Error
	})
	return &cycle, claimed, err
}

// AggregateChildCounts computes the child-status counts for a cycle by scanning
// availability_runs.cycle_id (never scans AvailabilityResult).
func (r *AvailabilityCycleRepository) AggregateChildCounts(cycleID uint) (ChildCounts, error) {
	return r.aggregateChildCountsTx(r.db, cycleID)
}

func (r *AvailabilityCycleRepository) aggregateChildCountsTx(db *gorm.DB, cycleID uint) (ChildCounts, error) {
	var counts ChildCounts
	var rows []struct {
		Status string
		Count  int
	}
	if err := db.Model(&models.AvailabilityRun{}).
		Select("status, count(*) as count").
		Where("cycle_id = ?", cycleID).
		Group("status").
		Scan(&rows).Error; err != nil {
		return counts, err
	}
	for _, row := range rows {
		counts.Total += row.Count
		switch row.Status {
		case models.AvailabilityRunStatusQueued:
			counts.Queued = row.Count
		case models.AvailabilityRunStatusRunning:
			counts.Running = row.Count
		case models.AvailabilityRunStatusCompleted:
			counts.Completed = row.Count
		case models.AvailabilityRunStatusFailed:
			counts.Failed = row.Count
		}
	}
	return counts, nil
}

// FinalizeCycle sets the cycle's terminal status, fail message, and completion timestamp.
func (r *AvailabilityCycleRepository) FinalizeCycle(cycleID uint, status, failMessage string) error {
	return r.finalizeCycleTx(r.db, cycleID, status, failMessage)
}

func (r *AvailabilityCycleRepository) finalizeCycleTx(db *gorm.DB, cycleID uint, status, failMessage string) error {
	now := time.Now()
	return db.Model(&models.AvailabilityCycle{}).Where("id = ?", cycleID).Updates(map[string]interface{}{
		"status":       status,
		"fail_message": failMessage,
		"completed_at": &now,
	}).Error
}

// ListCycles returns paginated availability cycles, newest first.
func (r *AvailabilityCycleRepository) ListCycles(page, limit int) ([]models.AvailabilityCycle, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := r.db.Model(&models.AvailabilityCycle{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var cycles []models.AvailabilityCycle
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&cycles).Error
	return cycles, total, err
}

// GetCycleWithChildren returns a single cycle with its child run summaries (no per-coin
// results are preloaded — cycle detail is roll-up only).
func (r *AvailabilityCycleRepository) GetCycleWithChildren(id uint) (*models.AvailabilityCycle, error) {
	var cycle models.AvailabilityCycle
	err := r.db.Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Preload("User", func(db2 *gorm.DB) *gorm.DB {
			return db2.Select("id, username")
		}).Order("started_at ASC")
	}).First(&cycle, id).Error
	if err != nil {
		return nil, err
	}
	for i := range cycle.Children {
		cycle.Children[i].UserName = cycle.Children[i].User.Username
	}
	return &cycle, nil
}

var terminalCycleStatuses = []string{
	models.AvailabilityCycleStatusCompleted,
	models.AvailabilityCycleStatusFailed,
	models.AvailabilityCycleStatusPartialFailure,
}

// PruneOldCycles keeps only the most recent `keep` terminal cycles. Surviving children's
// cycle_id is nulled out before the parent cycle rows are deleted so per-owner run history
// is never accidentally cascaded away (D4).
func (r *AvailabilityCycleRepository) PruneOldCycles(keep int) {
	var count int64
	r.db.Model(&models.AvailabilityCycle{}).
		Where("status IN ?", terminalCycleStatuses).
		Count(&count)
	if count <= int64(keep) {
		return
	}

	var cutoffCycle models.AvailabilityCycle
	if err := r.db.Where("status IN ?", terminalCycleStatuses).
		Order("completed_at DESC").Offset(keep).Limit(1).First(&cutoffCycle).Error; err != nil {
		return
	}
	if cutoffCycle.CompletedAt == nil {
		return
	}

	r.db.Transaction(func(tx *gorm.DB) error {
		idsSubquery := tx.Model(&models.AvailabilityCycle{}).Select("id").
			Where("status IN ? AND completed_at <= ?", terminalCycleStatuses, cutoffCycle.CompletedAt)
		if err := tx.Model(&models.AvailabilityRun{}).
			Where("cycle_id IN (?)", idsSubquery).
			Update("cycle_id", nil).Error; err != nil {
			return err
		}
		return tx.Where("status IN ? AND completed_at <= ?", terminalCycleStatuses, cutoffCycle.CompletedAt).
			Delete(&models.AvailabilityCycle{}).Error
	})
}

// RecoverStaleCycles finalizes any "running" cycle whose children are all terminal
// (recovering from a crash mid-cycle), leaving cycles with still-active children alone.
// Returns the IDs of cycles still "queued" (enqueued but never claimed before a restart)
// so the caller can re-enqueue them into the in-memory worker queue.
func (r *AvailabilityCycleRepository) RecoverStaleCycles(timeout time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-timeout)
	var runningCycles []models.AvailabilityCycle
	if err := r.db.Where("status = ? AND started_at < ?", models.AvailabilityCycleStatusRunning, cutoff).
		Find(&runningCycles).Error; err != nil {
		return nil, err
	}
	for _, cycle := range runningCycles {
		counts, err := r.AggregateChildCounts(cycle.ID)
		if err != nil {
			continue
		}
		if status, terminal := deriveCycleStatus(counts); terminal {
			failMsg := ""
			if status != models.AvailabilityCycleStatusCompleted {
				failMsg = models.GenericAvailabilityFailureMessage
			}
			_ = r.FinalizeCycle(cycle.ID, status, failMsg)
		}
	}

	var queuedIDs []uint
	err := r.db.Model(&models.AvailabilityCycle{}).
		Where("status = ?", models.AvailabilityCycleStatusQueued).
		Order("started_at ASC").
		Pluck("id", &queuedIDs).Error
	return queuedIDs, err
}
