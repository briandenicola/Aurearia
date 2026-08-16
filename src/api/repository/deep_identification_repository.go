package repository

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// RecordRouterSelection persists the router's bounded provider decision so
// create/get responses remain transparent after the event stream is pruned.
func (r *DeepIdentificationRepository) RecordRouterSelection(jobID, userID uint, selected []string, rationale string) error {
	return r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ? AND user_id = ?", jobID, userID).
		Updates(map[string]interface{}{
			"selected_providers": strings.Join(selected, ","),
			"router_rationale":   rationale,
		}).Error
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

// UpdateProposalJSON persists an owner-edited proposal document (T110). It
// is guarded by applied_at IS NULL: editing a proposal after it has already
// been applied is meaningless (the AI-vs-owner distinction it carries no
// longer has anywhere left to flow), so the caller should treat 0 rows
// affected as "already applied" rather than silently succeeding.
func (r *DeepIdentificationRepository) UpdateProposalJSON(jobID, userID uint, proposalJSON string) (bool, error) {
	result := r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ? AND user_id = ? AND applied_at IS NULL", jobID, userID).
		Update("proposal_json", proposalJSON)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ApplyJob performs the single conditional apply UPDATE (T111): WHERE
// applied_at IS NULL, mirroring SettleTerminal's conditional-UPDATE race
// guarantee (FR-019) for the sibling apply-once guarantee (FR-033). Only
// one concurrent Apply call for a given job can ever win; the loser gets
// RowsAffected == 0 and must report 409 already_applied.
func (r *DeepIdentificationRepository) ApplyJob(jobID, userID uint, appliedCoinID, appliedDraftID *uint, appliedAt time.Time) (bool, error) {
	result := r.db.Model(&models.DeepIdentificationJob{}).
		Where("id = ? AND user_id = ? AND applied_at IS NULL", jobID, userID).
		Updates(map[string]interface{}{
			"applied_coin_id":  appliedCoinID,
			"applied_draft_id": appliedDraftID,
			"applied_at":       appliedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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

// RecordProviderStarted creates or resets the bounded operational row for a
// provider attempt. ClaimsJSON is always cleared: provider content belongs in
// the transient pipeline flow, never in observability storage.
func (r *DeepIdentificationRepository) RecordProviderStarted(
	jobID, userID uint,
	provider models.DeepProviderName,
	automatable bool,
	startedAt time.Time,
) error {
	run := models.DeepIdentificationProviderRun{
		JobID:       jobID,
		UserID:      userID,
		Provider:    provider,
		Status:      models.DeepProviderRunRunning,
		Automatable: automatable,
		ClaimsJSON:  "",
		StartedAt:   &startedAt,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "job_id"}, {Name: "provider"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"user_id":      userID,
			"status":       models.DeepProviderRunRunning,
			"automatable":  automatable,
			"claims_json":  "",
			"confidence":   0,
			"call_count":   0,
			"latency_ms":   0,
			"error_kind":   "",
			"started_at":   startedAt,
			"completed_at": nil,
		}),
	}).Create(&run).Error
}

// RecordProviderResult settles a provider attempt with operational fields
// only. It intentionally never accepts or persists claims, citations, links,
// queries, or any other user/provider content.
func (r *DeepIdentificationRepository) RecordProviderResult(
	jobID, userID uint,
	provider models.DeepProviderName,
	status models.DeepProviderRunStatus,
	automatable bool,
	confidence float64,
	callCount, latencyMS int,
	errorKind string,
	startedAt, completedAt time.Time,
) error {
	run := models.DeepIdentificationProviderRun{
		JobID:       jobID,
		UserID:      userID,
		Provider:    provider,
		Status:      status,
		Automatable: automatable,
		ClaimsJSON:  "",
		Confidence:  confidence,
		CallCount:   callCount,
		LatencyMS:   latencyMS,
		ErrorKind:   errorKind,
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "job_id"}, {Name: "provider"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"user_id":      userID,
			"status":       status,
			"automatable":  automatable,
			"claims_json":  "",
			"confidence":   confidence,
			"call_count":   callCount,
			"latency_ms":   latencyMS,
			"error_kind":   errorKind,
			"started_at":   startedAt,
			"completed_at": completedAt,
		}),
	}).Create(&run).Error
}

// SettleRunningProviderRuns closes provider attempts that never emitted a
// provider_result because the pipeline was cancelled, timed out, or ended.
func (r *DeepIdentificationRepository) SettleRunningProviderRuns(
	jobID, userID uint,
	status models.DeepProviderRunStatus,
	errorKind string,
	completedAt time.Time,
) error {
	var runs []models.DeepIdentificationProviderRun
	if err := r.db.
		Select("id", "started_at").
		Where("job_id = ? AND user_id = ? AND status = ?", jobID, userID, models.DeepProviderRunRunning).
		Find(&runs).Error; err != nil {
		return err
	}
	for _, run := range runs {
		latencyMS := 0
		if run.StartedAt != nil && !completedAt.Before(*run.StartedAt) {
			latencyMS = int(completedAt.Sub(*run.StartedAt).Milliseconds())
		}
		if err := r.db.Model(&models.DeepIdentificationProviderRun{}).
			Where("id = ? AND status = ?", run.ID, models.DeepProviderRunRunning).
			Updates(map[string]interface{}{
				"status":       status,
				"error_kind":   errorKind,
				"latency_ms":   latencyMS,
				"completed_at": completedAt,
				"claims_json":  "",
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeepIdentificationDurableMetrics contains only aggregate operational data
// selected from durable rows.
type DeepIdentificationDurableMetrics struct {
	JobsByTerminalStatus map[models.DeepJobStatus]int64
	PartialSuccessRate   float64
	Duration             models.DeepIdentificationLatencySummary
	Providers            map[models.DeepProviderName]models.DeepIdentificationProviderMetrics
	QueueDepth           int64
}

// GetObservabilityMetrics aggregates the durable portion of the admin
// observability surface. Queries select only status/timestamp/count columns.
func (r *DeepIdentificationRepository) GetObservabilityMetrics() (*DeepIdentificationDurableMetrics, error) {
	metrics := &DeepIdentificationDurableMetrics{
		JobsByTerminalStatus: make(map[models.DeepJobStatus]int64),
		Providers:            make(map[models.DeepProviderName]models.DeepIdentificationProviderMetrics),
	}
	for _, status := range deepJobTerminalStatuses() {
		metrics.JobsByTerminalStatus[status] = 0
	}

	type statusCount struct {
		Status models.DeepJobStatus
		Count  int64
	}
	var jobCounts []statusCount
	if err := r.db.Model(&models.DeepIdentificationJob{}).
		Select("status, COUNT(*) AS count").
		Where("status IN ?", deepJobTerminalStatuses()).
		Group("status").
		Scan(&jobCounts).Error; err != nil {
		return nil, err
	}
	var terminalCount int64
	for _, row := range jobCounts {
		metrics.JobsByTerminalStatus[row.Status] = row.Count
		terminalCount += row.Count
	}
	if terminalCount > 0 {
		metrics.PartialSuccessRate = float64(metrics.JobsByTerminalStatus[models.DeepJobStatusPartial]) / float64(terminalCount)
	}

	type jobTiming struct {
		StartedAt   *time.Time
		CompletedAt *time.Time
	}
	var timings []jobTiming
	if err := r.db.Model(&models.DeepIdentificationJob{}).
		Select("started_at, completed_at").
		Where("status IN ? AND started_at IS NOT NULL AND completed_at IS NOT NULL", deepJobTerminalStatuses()).
		Find(&timings).Error; err != nil {
		return nil, err
	}
	durations := make([]int64, 0, len(timings))
	for _, timing := range timings {
		if timing.StartedAt != nil && timing.CompletedAt != nil && !timing.CompletedAt.Before(*timing.StartedAt) {
			durations = append(durations, timing.CompletedAt.Sub(*timing.StartedAt).Milliseconds())
		}
	}
	metrics.Duration = latencyPercentiles(durations)

	type providerStatusCount struct {
		Provider models.DeepProviderName
		Status   models.DeepProviderRunStatus
		Count    int64
	}
	var providerCounts []providerStatusCount
	if err := r.db.Model(&models.DeepIdentificationProviderRun{}).
		Select("provider, status, COUNT(*) AS count").
		Group("provider, status").
		Scan(&providerCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range providerCounts {
		provider := metrics.Providers[row.Provider]
		if provider.StatusCounts == nil {
			provider.StatusCounts = make(map[models.DeepProviderRunStatus]int64)
		}
		provider.StatusCounts[row.Status] = row.Count
		metrics.Providers[row.Provider] = provider
	}

	type providerTiming struct {
		Provider  models.DeepProviderName
		LatencyMS int
	}
	var providerTimings []providerTiming
	if err := r.db.Model(&models.DeepIdentificationProviderRun{}).
		Select("provider, latency_ms").
		Where("completed_at IS NOT NULL AND latency_ms >= 0").
		Find(&providerTimings).Error; err != nil {
		return nil, err
	}
	latencies := make(map[models.DeepProviderName][]int64)
	for _, timing := range providerTimings {
		latencies[timing.Provider] = append(latencies[timing.Provider], int64(timing.LatencyMS))
	}
	for providerName, values := range latencies {
		provider := metrics.Providers[providerName]
		if provider.StatusCounts == nil {
			provider.StatusCounts = make(map[models.DeepProviderRunStatus]int64)
		}
		provider.Latency = latencyPercentiles(values)
		metrics.Providers[providerName] = provider
	}

	queueDepth, err := r.CountQueuedJobs()
	if err != nil {
		return nil, err
	}
	metrics.QueueDepth = queueDepth
	return metrics, nil
}

func latencyPercentiles(values []int64) models.DeepIdentificationLatencySummary {
	if len(values) == 0 {
		return models.DeepIdentificationLatencySummary{}
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	at := func(percentile float64) int64 {
		index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
		if index < 0 {
			index = 0
		}
		return sorted[index]
	}
	return models.DeepIdentificationLatencySummary{P50MS: at(0.50), P95MS: at(0.95)}
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

// GetLatestProviderStatus returns the most recent provider-run row for the
// given provider across all jobs, ordered by created_at desc (Feature 345
// US4 admin health, FR-034). This is a deliberately NON-user-scoped read:
// it powers the admin-only OCRE health surface and therefore selects only
// the bounded operational columns (status/timing/counts/error-kind) — it
// never joins or returns per-job user content (notes, legends, claims). It
// returns (nil, nil) when no attempt has ever been recorded for the
// provider, which the caller renders as a null lastOutcome.
func (r *DeepIdentificationRepository) GetLatestProviderStatus(provider models.DeepProviderName) (*models.DeepIdentificationProviderRun, error) {
	var run models.DeepIdentificationProviderRun
	err := r.db.
		Select("id", "provider", "status", "confidence", "call_count", "latency_ms", "error_kind", "started_at", "completed_at", "created_at", "updated_at").
		Where("provider = ? AND completed_at IS NOT NULL", provider).
		Order("completed_at DESC").
		Limit(1).
		Take(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}
