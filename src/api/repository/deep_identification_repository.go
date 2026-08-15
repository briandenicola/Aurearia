package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// DeepIdentificationRepository is the owner-scoped persistence layer for
// the deep agentic coin identification job/event/provider-run domain
// (data-model.md §2-5). Every query is filtered by user_id (FR-006/FR-037).
type DeepIdentificationRepository struct {
	db *gorm.DB
}

func NewDeepIdentificationRepository(db *gorm.DB) *DeepIdentificationRepository {
	return &DeepIdentificationRepository{db: db}
}

var deepJobActiveStatuses = []models.DeepJobStatus{models.DeepJobStatusQueued, models.DeepJobStatusRunning}

// CreateJob creates a new queued job, or - if an active (queued/running) job
// already exists for the same (user_id, input_fingerprint) - returns that
// existing job instead (FR-007 idempotent duplicate-submit reuse, mirrors
// AIJobRepository.EnqueueOrFindActive). The second return value reports
// whether an existing job was reused rather than a new one created.
func (r *DeepIdentificationRepository) CreateJob(job *models.DeepIdentificationJob) (*models.DeepIdentificationJob, bool, error) {
	var result models.DeepIdentificationJob
	var reused bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.DeepIdentificationJob
		err := tx.Where("user_id = ? AND input_fingerprint = ? AND active_key = ?", job.UserID, job.InputFingerprint, "active").
			Order("created_at ASC").
			First(&existing).Error
		if err == nil {
			result = existing
			reused = true
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		job.ActiveKey = "active"
		if job.Status == "" {
			job.Status = models.DeepJobStatusQueued
		}
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		result = *job
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &result, reused, nil
}

// FindActiveByFingerprint returns the active (queued/running) job for the
// given user + fingerprint, if any.
func (r *DeepIdentificationRepository) FindActiveByFingerprint(userID uint, fingerprint string) (*models.DeepIdentificationJob, error) {
	var job models.DeepIdentificationJob
	err := r.db.Where("user_id = ? AND input_fingerprint = ? AND active_key = ?", userID, fingerprint, "active").
		Order("created_at ASC").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// ClaimNextQueuedJob dequeues the oldest queued job and stamps it running
// with the given workerID and an initial heartbeat. Returns
// (nil, false, nil) when no queued job is available.
func (r *DeepIdentificationRepository) ClaimNextQueuedJob(workerID string) (*models.DeepIdentificationJob, bool, error) {
	now := time.Now()
	var job models.DeepIdentificationJob
	var claimed bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("status = ?", models.DeepJobStatusQueued).
			Order("created_at ASC").
			First(&job).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		result := tx.Model(&models.DeepIdentificationJob{}).
			Where("id = ? AND status = ?", job.ID, models.DeepJobStatusQueued).
			Updates(map[string]interface{}{
				"status":        models.DeepJobStatusRunning,
				"worker_id":     workerID,
				"heartbeat_at":  now,
				"started_at":    now,
				"active_key":    "active",
				"attempt_count": gorm.Expr("attempt_count + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true
		return tx.First(&job, job.ID).Error
	})
	if err != nil {
		return nil, false, err
	}
	if !claimed {
		return nil, false, nil
	}
	return &job, true, nil
}

// Heartbeat updates a running job's liveness timestamp (FR-012 stale
// recovery relies on this being refreshed regularly by the owning worker).
func (r *DeepIdentificationRepository) Heartbeat(jobID uint) error {
	return r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ? AND status = ?", jobID, models.DeepJobStatusRunning).
		Update("heartbeat_at", time.Now()).Error
}

// AppendEvent assigns the next monotonic per-job sequence number and
// inserts the event in the same transaction (data-model.md §3), returning
// the assigned seq. No update/delete path is exposed for events.
func (r *DeepIdentificationRepository) AppendEvent(jobID, userID uint, eventType models.DeepIdentificationEventType, payloadJSON string) (int64, error) {
	var seq int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.DeepIdentificationJob{}).
			Where("id = ?", jobID).
			Update("last_seq", gorm.Expr("last_seq + 1")).Error; err != nil {
			return err
		}
		var job models.DeepIdentificationJob
		if err := tx.Select("last_seq").First(&job, jobID).Error; err != nil {
			return err
		}
		seq = job.LastSeq
		event := models.DeepIdentificationEvent{
			JobID:       jobID,
			UserID:      userID,
			Seq:         seq,
			Type:        eventType,
			PayloadJSON: payloadJSON,
		}
		return tx.Create(&event).Error
	})
	if err != nil {
		return 0, err
	}
	return seq, nil
}

// ListEventsSince returns events for a job with seq > since, ascending
// order, scoped to the owning user.
func (r *DeepIdentificationRepository) ListEventsSince(jobID, userID uint, since int64) ([]models.DeepIdentificationEvent, error) {
	var events []models.DeepIdentificationEvent
	err := r.db.Where("job_id = ? AND user_id = ? AND seq > ?", jobID, userID, since).
		Order("seq ASC").
		Find(&events).Error
	return events, err
}

// PruneEventsBefore deletes events created before cutoff for terminal jobs
// and stamps Job.EventsPrunedAt so subsequent replay can answer
// stream_truncated truthfully (FR-017).
func (r *DeepIdentificationRepository) PruneEventsBefore(cutoff time.Time) error {
	var jobIDs []uint
	if err := r.db.Model(&models.DeepIdentificationJob{}).
		Where("status IN ? AND completed_at IS NOT NULL AND completed_at < ? AND events_pruned_at IS NULL", deepJobTerminalStatuses(), cutoff).
		Pluck("id", &jobIDs).Error; err != nil {
		return err
	}
	if len(jobIDs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id IN ?", jobIDs).
			Delete(&models.DeepIdentificationEvent{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.DeepIdentificationJob{}).
			Where("id IN ?", jobIDs).
			Update("events_pruned_at", time.Now()).Error
	})
}

func deepJobTerminalStatuses() []models.DeepJobStatus {
	return []models.DeepJobStatus{
		models.DeepJobStatusCompleted,
		models.DeepJobStatusPartial,
		models.DeepJobStatusFailed,
		models.DeepJobStatusCancelled,
	}
}

// SettleTerminal performs the single conditional terminal UPDATE described
// in data-model.md §2.1: WHERE id = ? AND status IN (expectedStatuses).
// The update and the terminal event append happen in one transaction, so
// exactly one terminal event is ever produced per job even under a
// cancel-vs-complete race (FR-019). Returns whether this call won the race.
func (r *DeepIdentificationRepository) SettleTerminal(jobID uint, expectedStatuses []models.DeepJobStatus, newStatus models.DeepJobStatus, reportJSON, proposalJSON, failureCode, failureMessage string) (bool, error) {
	won := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		updates := map[string]interface{}{
			"status":          newStatus,
			"active_key":      gorm.Expr("id"),
			"completed_at":    now,
			"report_json":     reportJSON,
			"proposal_json":   proposalJSON,
			"failure_code":    failureCode,
			"failure_message": failureMessage,
			"partial_success": newStatus == models.DeepJobStatusPartial,
		}
		result := tx.Model(&models.DeepIdentificationJob{}).
			Where("id = ? AND status IN ?", jobID, expectedStatuses).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			won = false
			return nil
		}
		won = true
		var job models.DeepIdentificationJob
		if err := tx.Select("user_id, last_seq").First(&job, jobID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.DeepIdentificationJob{}).
			Where("id = ?", jobID).
			Update("last_seq", gorm.Expr("last_seq + 1")).Error; err != nil {
			return err
		}
		if err := tx.Select("last_seq").First(&job, jobID).Error; err != nil {
			return err
		}
		event := models.DeepIdentificationEvent{
			JobID:       jobID,
			UserID:      job.UserID,
			Seq:         job.LastSeq,
			Type:        models.DeepEventTerminal,
			PayloadJSON: `{"status":"` + string(newStatus) + `"}`,
		}
		return tx.Create(&event).Error
	})
	return won, err
}

// RecoverStaleJobs flips running jobs whose heartbeat is older than
// staleAfter to failed:stale_restart, appending exactly one terminal event
// per job (FR-012). It never leaves a job running forever across a process
// restart.
func (r *DeepIdentificationRepository) RecoverStaleJobs(staleAfter time.Duration) ([]uint, error) {
	cutoff := time.Now().Add(-staleAfter)
	var jobIDs []uint
	if err := r.db.Model(&models.DeepIdentificationJob{}).
		Where("status = ? AND (heartbeat_at IS NULL OR heartbeat_at < ?)", models.DeepJobStatusRunning, cutoff).
		Pluck("id", &jobIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range jobIDs {
		if _, err := r.SettleTerminal(id, []models.DeepJobStatus{models.DeepJobStatusRunning}, models.DeepJobStatusFailed, "", "", "stale_restart", "The process restarted before this job finished."); err != nil {
			return nil, err
		}
	}
	return jobIDs, nil
}

// GetJob returns a job scoped to its owner; a non-owner lookup behaves
// identically to a missing job (gorm.ErrRecordNotFound).
func (r *DeepIdentificationRepository) GetJob(id, userID uint) (*models.DeepIdentificationJob, error) {
	var job models.DeepIdentificationJob
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// DeepJobListFilters narrows ListJobs results.
type DeepJobListFilters struct {
	CoinID *uint
	Status models.DeepJobStatus
	// ActiveOnly restricts results to queued/running jobs.
	ActiveOnly bool
	// BeforeID, when set, returns only jobs with id < BeforeID (cursor
	// pagination: the caller passes the id of the last row seen so far).
	BeforeID *uint
	// Limit caps the number of rows returned. Zero means unlimited (used by
	// internal callers such as the per-user active-job scan).
	Limit int
}

// ListJobs returns a user's jobs, most recent (highest id) first, optionally
// filtered by coin id, status, active-only, and a cursor (BeforeID).
// Ordering by id rather than created_at avoids ambiguity when multiple jobs
// are created within the same timestamp tick.
func (r *DeepIdentificationRepository) ListJobs(userID uint, filters DeepJobListFilters) ([]models.DeepIdentificationJob, error) {
	q := r.db.Where("user_id = ?", userID)
	if filters.CoinID != nil {
		q = q.Where("coin_id = ?", *filters.CoinID)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	if filters.ActiveOnly {
		q = q.Where("status IN ?", []models.DeepJobStatus{models.DeepJobStatusQueued, models.DeepJobStatusRunning})
	}
	if filters.BeforeID != nil {
		q = q.Where("id < ?", *filters.BeforeID)
	}
	q = q.Order("id DESC")
	if filters.Limit > 0 {
		q = q.Limit(filters.Limit)
	}
	var jobs []models.DeepIdentificationJob
	err := q.Find(&jobs).Error
	return jobs, err
}

// RequestCancel records a cancel request against an owner-scoped job. It
// only sets CancelRequestedAt; the actual settle-to-cancelled transition is
// performed by whichever path (handler for queued jobs, worker for running
// jobs) observes it first via SettleTerminal.
func (r *DeepIdentificationRepository) RequestCancel(jobID, userID uint) error {
	result := r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ? AND user_id = ?", jobID, userID).
		Update("cancel_requested_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListArtifacts returns every artifact row for a job (including
// already-deleted ones, so callers can distinguish idempotent re-deletes).
func (r *DeepIdentificationRepository) ListArtifacts(jobID uint) ([]models.DeepIdentificationArtifact, error) {
	var artifacts []models.DeepIdentificationArtifact
	err := r.db.Where("job_id = ?", jobID).Order("id ASC").Find(&artifacts).Error
	return artifacts, err
}

// CreateArtifact inserts a new artifact row.
func (r *DeepIdentificationRepository) CreateArtifact(a *models.DeepIdentificationArtifact) error {
	return r.db.Create(a).Error
}

// MarkArtifactDeleted stamps DeletedAt on an artifact row (the file itself
// is removed by the caller). Idempotent: calling it again on an
// already-deleted row is a harmless no-op.
func (r *DeepIdentificationRepository) MarkArtifactDeleted(id uint, when time.Time) error {
	return r.db.Model(&models.DeepIdentificationArtifact{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", when).Error
}

// SetRetryLineage stamps a newly-created job's RetryOfJobID/RetryDepth
// after StartJob's idempotent-reuse check has already resolved it as a
// genuinely new job (T045/FR-020).
func (r *DeepIdentificationRepository) SetRetryLineage(jobID, retryOfJobID uint, retryDepth int) error {
	return r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{"retry_of_job_id": retryOfJobID, "retry_depth": retryDepth}).Error
}

// CountActiveJobsForUser returns how many queued/running jobs a user
// currently owns (FR-007 per-user active-job limit, T030).
func (r *DeepIdentificationRepository) CountActiveJobsForUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.DeepIdentificationJob{}).
		Where("user_id = ? AND status IN ?", userID, deepJobActiveStatuses).
		Count(&count).Error
	return count, err
}

// CountQueuedJobs returns the total number of queued (not yet claimed) jobs
// across all users, used for global queue-depth backpressure (T031).
func (r *DeepIdentificationRepository) CountQueuedJobs() (int64, error) {
	var count int64
	err := r.db.Model(&models.DeepIdentificationJob{}).
		Where("status = ?", models.DeepJobStatusQueued).
		Count(&count).Error
	return count, err
}

// ListExpiredJobIDs returns terminal job ids whose ExpiresAt has passed
// (data-model.md §9 retention, FR-034). Used by the result-retention
// janitor sweep to remove artifacts (rows/files) past their retention
// window; the job row itself is left in place for audit history.
func (r *DeepIdentificationRepository) ListExpiredJobIDs(now time.Time) ([]uint, error) {
	var jobIDs []uint
	err := r.db.Model(&models.DeepIdentificationJob{}).
		Where("status IN ? AND expires_at IS NOT NULL AND expires_at < ?", deepJobTerminalStatuses(), now).
		Pluck("id", &jobIDs).Error
	return jobIDs, err
}

// ListJobIDsWithUndeletedHintArtifacts returns the ids of terminal jobs that
// still have one or more hint artifacts lacking DeletedAt - i.e. a crash
// happened between SettleTerminal and the hint-cleanup hook. Used by the
// janitor's startup sweep (T034/T040) as a defensive backstop to the
// terminal-hook path.
func (r *DeepIdentificationRepository) ListJobIDsWithUndeletedHintArtifacts() ([]uint, error) {
	var jobIDs []uint
	err := r.db.Model(&models.DeepIdentificationArtifact{}).
		Distinct("deep_identification_artifacts.job_id").
		Joins("JOIN deep_identification_jobs ON deep_identification_jobs.id = deep_identification_artifacts.job_id").
		Where("deep_identification_artifacts.role = ? AND deep_identification_artifacts.deleted_at IS NULL AND deep_identification_jobs.status IN ?",
			models.DeepArtifactRoleHint, deepJobTerminalStatuses()).
		Pluck("deep_identification_artifacts.job_id", &jobIDs).Error
	return jobIDs, err
}
