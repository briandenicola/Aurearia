package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

var (
	ErrCopilotTransitionConflict = errors.New("coin copilot transition conflict")
	ErrCopilotThreadActive       = errors.New("coin copilot thread has an active run")
	ErrCopilotOwnerCapacity      = errors.New("coin copilot owner capacity reached")
	ErrCopilotQueueCapacity      = errors.New("coin copilot queue capacity reached")
	ErrCopilotStartKeyConflict   = errors.New("coin copilot start key conflict")
)

type CoinCopilotRepository struct {
	db *gorm.DB
}

func NewCoinCopilotRepository(db *gorm.DB) *CoinCopilotRepository {
	return &CoinCopilotRepository{db: db}
}

func (r *CoinCopilotRepository) CreateRun(thread *models.CoinCopilotThread, run *models.CoinCopilotRun) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := expireCopilotStartKey(tx, run.UserID, run.StartIdempotencyKeyHash, time.Now().UTC()); err != nil {
			return err
		}
		return createCopilotRun(tx, thread, run)
	})
}

// AdmitRun relies on the production SQLite _txlock=immediate DSN so the
// capacity reads and run insert execute under one serialized write transaction.
func (r *CoinCopilotRepository) AdmitRun(thread *models.CoinCopilotThread, run *models.CoinCopilotRun, maxActivePerUser, queueDepth int) (*models.CoinCopilotRun, bool, error) {
	var admitted *models.CoinCopilotRun
	reused := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if err := expireCopilotStartKey(tx, run.UserID, run.StartIdempotencyKeyHash, now); err != nil {
			return err
		}
		var existing models.CoinCopilotRun
		err := tx.Where("user_id = ? AND start_idempotency_key_hash = ? AND created_at >= ?",
			run.UserID, run.StartIdempotencyKeyHash, now.Add(-24*time.Hour)).First(&existing).Error
		switch {
		case err == nil:
			if existing.StartRequestFingerprint != run.StartRequestFingerprint {
				return ErrCopilotStartKeyConflict
			}
			admitted = &existing
			reused = true
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var active int64
		if err := tx.Model(&models.CoinCopilotRun{}).
			Where("user_id = ? AND status IN ?", run.UserID, copilotActiveStatuses()).
			Count(&active).Error; err != nil {
			return err
		}
		if active >= int64(maxActivePerUser) {
			return ErrCopilotOwnerCapacity
		}
		var queued int64
		if err := tx.Model(&models.CoinCopilotRun{}).
			Where("status = ?", models.CopilotRunQueued).
			Count(&queued).Error; err != nil {
			return err
		}
		if queued >= int64(queueDepth) {
			return ErrCopilotQueueCapacity
		}
		if err := createCopilotRun(tx, thread, run); err != nil {
			return err
		}
		admitted = run
		return nil
	})
	return admitted, reused, err
}

func createCopilotRun(tx *gorm.DB, thread *models.CoinCopilotThread, run *models.CoinCopilotRun) error {
	if thread.CreatedAt.IsZero() {
		if err := tx.Create(thread).Error; err != nil {
			return err
		}
	} else {
		var owned models.CoinCopilotThread
		if err := tx.Where("id = ? AND user_id = ?", thread.ID, thread.UserID).First(&owned).Error; err != nil {
			return err
		}
	}
	if err := tx.Create(run).Error; err != nil {
		return err
	}
	return tx.Model(&models.CoinCopilotThread{}).
		Where("id = ? AND user_id = ?", thread.ID, thread.UserID).
		Updates(map[string]interface{}{"last_run_id": run.ID, "updated_at": time.Now().UTC()}).Error
}

func expireCopilotStartKey(tx *gorm.DB, userID uint, keyHash string, now time.Time) error {
	return tx.Model(&models.CoinCopilotRun{}).
		Where("user_id = ? AND start_idempotency_key_hash = ? AND created_at < ?", userID, keyHash, now.Add(-24*time.Hour)).
		Update("start_idempotency_key_hash", gorm.Expr("id")).Error
}

func copilotActiveStatuses() []models.CopilotRunStatus {
	return []models.CopilotRunStatus{
		models.CopilotRunQueued, models.CopilotRunRunning, models.CopilotRunPaused, models.CopilotRunCancelRequested,
	}
}

func (r *CoinCopilotRepository) FindRunByStartKey(userID uint, keyHash string) (*models.CoinCopilotRun, error) {
	var run models.CoinCopilotRun
	err := r.db.Where("user_id = ? AND start_idempotency_key_hash = ? AND created_at >= ?", userID, keyHash, time.Now().UTC().Add(-24*time.Hour)).
		First(&run).Error
	return &run, err
}

func (r *CoinCopilotRepository) GetRun(runID string, userID uint) (*models.CoinCopilotRun, error) {
	var run models.CoinCopilotRun
	err := r.db.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error
	return &run, err
}

func (r *CoinCopilotRepository) GetThread(threadID string, userID uint) (*models.CoinCopilotThread, []models.CoinCopilotRun, error) {
	var thread models.CoinCopilotThread
	if err := r.db.Where("id = ? AND user_id = ?", threadID, userID).First(&thread).Error; err != nil {
		return nil, nil, err
	}
	var runs []models.CoinCopilotRun
	if err := r.db.Where("thread_id = ? AND user_id = ?", threadID, userID).Order("created_at ASC").Find(&runs).Error; err != nil {
		return nil, nil, err
	}
	return &thread, runs, nil
}

func (r *CoinCopilotRepository) ListSettledRunsForHistory(threadID, currentRunID string, userID uint, limit int) ([]models.CoinCopilotRun, error) {
	if limit <= 0 {
		return []models.CoinCopilotRun{}, nil
	}
	var runs []models.CoinCopilotRun
	err := r.db.Model(&models.CoinCopilotRun{}).
		Select("id", "goal", "status", "final_answer", "created_at").
		Where("thread_id = ? AND user_id = ? AND id <> ? AND status IN ?", threadID, userID, currentRunID, []models.CopilotRunStatus{
			models.CopilotRunCompleted, models.CopilotRunFailed, models.CopilotRunCancelled,
		}).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&runs).Error
	return runs, err
}

func (r *CoinCopilotRepository) CountActiveRuns(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.CoinCopilotRun{}).
		Where("user_id = ? AND status IN ?", userID, copilotActiveStatuses()).Count(&count).Error
	return count, err
}

func (r *CoinCopilotRepository) CountQueuedRuns() (int64, error) {
	var count int64
	err := r.db.Model(&models.CoinCopilotRun{}).Where("status = ?", models.CopilotRunQueued).Count(&count).Error
	return count, err
}

func (r *CoinCopilotRepository) ClaimNextQueuedRun(workerID, executionID string) (*models.CoinCopilotRun, bool, error) {
	var run models.CoinCopilotRun
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("status = ?", models.CopilotRunQueued).Order("created_at ASC").First(&run).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		now := time.Now().UTC()
		claimedExecutionID := run.ExecutionID
		if claimedExecutionID == "" {
			claimedExecutionID = executionID
		}
		result := tx.Model(&models.CoinCopilotRun{}).
			Where("id = ? AND status = ?", run.ID, models.CopilotRunQueued).
			Updates(map[string]interface{}{
				"status":               models.CopilotRunRunning,
				"execution_id":         claimedExecutionID,
				"execution_attempt":    gorm.Expr("execution_attempt + 1"),
				"worker_id":            workerID,
				"heartbeat_at":         now,
				"started_at":           gorm.Expr("COALESCE(started_at, ?)", now),
				"execution_started_at": now,
			})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		claimed = true
		return tx.First(&run, "id = ?", run.ID).Error
	})
	return &run, claimed, err
}

func (r *CoinCopilotRepository) Heartbeat(runID, executionID string) error {
	result := r.db.Model(&models.CoinCopilotRun{}).
		Where("id = ? AND status = ? AND execution_id = ?", runID, models.CopilotRunRunning, executionID).
		Update("heartbeat_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCopilotTransitionConflict
	}
	return nil
}

func (r *CoinCopilotRepository) GetCurrentExecution(runID, executionID string, userID uint) (*models.CoinCopilotRun, error) {
	var run models.CoinCopilotRun
	err := r.db.Where("id = ? AND user_id = ? AND execution_id = ? AND status = ?", runID, userID, executionID, models.CopilotRunRunning).
		First(&run).Error
	return &run, err
}

func (r *CoinCopilotRepository) UpdateUsage(runID, executionID string, usage models.CoinCopilotRun) error {
	result := r.db.Model(&models.CoinCopilotRun{}).
		Where("id = ? AND execution_id = ? AND status = ?", runID, executionID, models.CopilotRunRunning).
		Updates(map[string]interface{}{
			"iteration_count": usage.IterationCount,
			"tool_call_count": usage.ToolCallCount,
			"input_tokens":    usage.InputTokens,
			"output_tokens":   usage.OutputTokens,
			"heartbeat_at":    time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCopilotTransitionConflict
	}
	return nil
}

func (r *CoinCopilotRepository) AppendEvent(runID string, userID uint, executionID string, eventType models.CopilotEventType, payloadJSON string) (*models.CoinCopilotEvent, error) {
	if !models.IsCopilotEventType(eventType) || isCopilotTerminalEvent(eventType) {
		return nil, ErrCopilotTransitionConflict
	}
	var event models.CoinCopilotEvent
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var run models.CoinCopilotRun
		if err := tx.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
			return err
		}
		if run.Status != models.CopilotRunRunning || run.ExecutionID != executionID {
			return ErrCopilotTransitionConflict
		}
		run.LastSeq++
		if err := tx.Model(&models.CoinCopilotRun{}).Where("id = ? AND user_id = ?", runID, userID).Update("last_seq", run.LastSeq).Error; err != nil {
			return err
		}
		event = models.CoinCopilotEvent{
			RunID: run.ID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: executionID,
			Seq: run.LastSeq, Type: eventType, PayloadJSON: payloadJSON,
		}
		return tx.Create(&event).Error
	})
	return &event, err
}

func (r *CoinCopilotRepository) CommitCheckpoint(runID string, userID uint, executionID, stateJSON, digest string) (*models.CoinCopilotCheckpoint, error) {
	var checkpoint models.CoinCopilotCheckpoint
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var run models.CoinCopilotRun
		if err := tx.Where("id = ? AND user_id = ? AND execution_id = ? AND status = ?", runID, userID, executionID, models.CopilotRunRunning).First(&run).Error; err != nil {
			return err
		}
		run.CheckpointVersion++
		checkpoint = models.CoinCopilotCheckpoint{
			RunID: run.ID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: executionID,
			Version: run.CheckpointVersion, StateJSON: stateJSON, StateDigest: digest,
		}
		if err := tx.Create(&checkpoint).Error; err != nil {
			return err
		}
		return tx.Model(&models.CoinCopilotRun{}).Where("id = ? AND execution_id = ?", runID, executionID).
			Update("checkpoint_version", run.CheckpointVersion).Error
	})
	return &checkpoint, err
}

func (r *CoinCopilotRepository) GetLatestCheckpoint(runID string, userID uint) (*models.CoinCopilotCheckpoint, error) {
	var checkpoint models.CoinCopilotCheckpoint
	err := r.db.Where("run_id = ? AND user_id = ?", runID, userID).Order("version DESC").First(&checkpoint).Error
	return &checkpoint, err
}

func (r *CoinCopilotRepository) FindResumeRequest(runID string, userID uint, keyHash string) (*models.CoinCopilotResumeRequest, error) {
	var request models.CoinCopilotResumeRequest
	err := r.db.Where("run_id = ? AND user_id = ? AND idempotency_key_hash = ?", runID, userID, keyHash).First(&request).Error
	return &request, err
}

func (r *CoinCopilotRepository) Resume(runID string, userID uint, keyHash, fingerprint, executionID, answer string, expectedVersion int64, stateJSON, digest string) (*models.CoinCopilotRun, error) {
	var run models.CoinCopilotRun
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
			return err
		}
		if run.Status != models.CopilotRunPaused || run.CheckpointVersion != expectedVersion || run.ResumeDeadline == nil || run.ResumeDeadline.Before(time.Now().UTC()) {
			return ErrCopilotTransitionConflict
		}
		nextVersion := expectedVersion + 1
		checkpoint := models.CoinCopilotCheckpoint{
			RunID: run.ID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: executionID,
			Version: nextVersion, StateJSON: stateJSON, StateDigest: digest,
		}
		if err := tx.Create(&checkpoint).Error; err != nil {
			return err
		}
		resume := models.CoinCopilotResumeRequest{
			RunID: run.ID, UserID: userID, IdempotencyKeyHash: keyHash, RequestFingerprint: fingerprint,
			AcceptedCheckpointVersion: nextVersion, ExecutionID: executionID,
		}
		if err := tx.Create(&resume).Error; err != nil {
			return err
		}
		result := tx.Model(&models.CoinCopilotRun{}).
			Where("id = ? AND user_id = ? AND status = ? AND checkpoint_version = ?", runID, userID, models.CopilotRunPaused, expectedVersion).
			Updates(map[string]interface{}{
				"status": models.CopilotRunQueued, "checkpoint_version": nextVersion, "execution_id": executionID,
				"paused_at": nil, "resume_deadline": nil, "worker_id": "", "heartbeat_at": nil,
				"execution_started_at": nil,
				"last_seq":             gorm.Expr("last_seq + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrCopilotTransitionConflict
		}
		if err := tx.First(&run, "id = ?", runID).Error; err != nil {
			return err
		}
		return tx.Create(&models.CoinCopilotEvent{
			RunID: run.ID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: executionID,
			Seq: run.LastSeq, Type: models.CopilotEventRunResumed,
			PayloadJSON: fmt.Sprintf(`{"executionId":%q,"attempt":%d,"checkpointVersion":%d}`, executionID, run.ExecutionAttempt+1, run.CheckpointVersion),
		}).Error
	})
	return &run, err
}

func (r *CoinCopilotRepository) TransitionWithEvent(runID string, userID uint, executionID string, expected []models.CopilotRunStatus, next models.CopilotRunStatus, updates map[string]interface{}, eventType models.CopilotEventType, payloadJSON string) (bool, *models.CoinCopilotEvent, error) {
	if !models.IsCopilotEventType(eventType) || !copilotEventMatchesStatus(eventType, next) {
		return false, nil, ErrCopilotTransitionConflict
	}
	var event models.CoinCopilotEvent
	won := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var run models.CoinCopilotRun
		if err := tx.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
			return err
		}
		if executionID != "" && run.ExecutionID != executionID {
			return ErrCopilotTransitionConflict
		}
		if containsCopilotStatus(expected, run.Status) && !isValidCopilotTransition(run.Status, next) {
			return ErrCopilotTransitionConflict
		}
		updates["status"] = next
		updates["last_seq"] = gorm.Expr("last_seq + 1")
		if models.IsCopilotRunTerminal(next) {
			updates["completed_at"] = time.Now().UTC()
		}

		query := tx.Model(&models.CoinCopilotRun{}).Where("id = ? AND user_id = ? AND status IN ?", runID, userID, expected)
		if executionID != "" {
			query = query.Where("execution_id = ?", executionID)
		}
		result := query.Updates(updates)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		won = true
		if err := tx.Select("thread_id, last_seq, execution_id").First(&run, "id = ?", runID).Error; err != nil {
			return err
		}
		event = models.CoinCopilotEvent{
			RunID: runID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: run.ExecutionID,
			Seq: run.LastSeq, Type: eventType, PayloadJSON: payloadJSON,
		}
		return tx.Create(&event).Error
	})
	return won, &event, err
}

func containsCopilotStatus(statuses []models.CopilotRunStatus, target models.CopilotRunStatus) bool {
	for _, status := range statuses {
		if status == target {
			return true
		}
	}
	return false
}

func isCopilotTerminalEvent(eventType models.CopilotEventType) bool {
	return eventType == models.CopilotEventRunCompleted ||
		eventType == models.CopilotEventRunFailed ||
		eventType == models.CopilotEventRunCancelled
}

func copilotEventMatchesStatus(eventType models.CopilotEventType, status models.CopilotRunStatus) bool {
	switch status {
	case models.CopilotRunCompleted:
		return eventType == models.CopilotEventRunCompleted
	case models.CopilotRunFailed:
		return eventType == models.CopilotEventRunFailed
	case models.CopilotRunCancelled:
		return eventType == models.CopilotEventRunCancelled
	case models.CopilotRunPaused:
		return eventType == models.CopilotEventRunPaused
	default:
		return false
	}
}

func isValidCopilotTransition(current, next models.CopilotRunStatus) bool {
	switch current {
	case models.CopilotRunRunning:
		return next == models.CopilotRunPaused || next == models.CopilotRunCompleted ||
			next == models.CopilotRunFailed || next == models.CopilotRunCancelled
	case models.CopilotRunPaused:
		return next == models.CopilotRunFailed || next == models.CopilotRunCancelled
	case models.CopilotRunCancelRequested:
		return next == models.CopilotRunCancelled
	default:
		return false
	}
}

func (r *CoinCopilotRepository) RequestCancel(runID string, userID uint) (*models.CoinCopilotRun, bool, error) {
	var run models.CoinCopilotRun
	immediate := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		switch run.Status {
		case models.CopilotRunQueued, models.CopilotRunPaused:
			result := tx.Model(&models.CoinCopilotRun{}).Where("id = ? AND user_id = ? AND status = ?", runID, userID, run.Status).
				Updates(map[string]interface{}{"status": models.CopilotRunCancelled, "cancel_requested_at": now, "completed_at": now, "last_seq": gorm.Expr("last_seq + 1")})
			if result.Error != nil || result.RowsAffected == 0 {
				return result.Error
			}
			immediate = true
			if err := tx.First(&run, "id = ?", runID).Error; err != nil {
				return err
			}
			return tx.Create(&models.CoinCopilotEvent{
				RunID: run.ID, ThreadID: run.ThreadID, UserID: userID, ExecutionID: run.ExecutionID,
				Seq: run.LastSeq, Type: models.CopilotEventRunCancelled, PayloadJSON: `{"reason":"owner_cancelled"}`,
			}).Error
		case models.CopilotRunRunning:
			result := tx.Model(&models.CoinCopilotRun{}).Where("id = ? AND user_id = ? AND status = ?", runID, userID, models.CopilotRunRunning).
				Updates(map[string]interface{}{"status": models.CopilotRunCancelRequested, "cancel_requested_at": now})
			if result.Error != nil || result.RowsAffected == 0 {
				return result.Error
			}
		case models.CopilotRunCancelRequested, models.CopilotRunCancelled:
			immediate = run.Status == models.CopilotRunCancelled
		default:
			return ErrCopilotTransitionConflict
		}
		return tx.First(&run, "id = ?", runID).Error
	})
	return &run, immediate, err
}

func (r *CoinCopilotRepository) ListEventsSince(runID string, userID uint, since int64) ([]models.CoinCopilotEvent, error) {
	var events []models.CoinCopilotEvent
	err := r.db.Where("run_id = ? AND user_id = ? AND seq > ?", runID, userID, since).Order("seq ASC").Find(&events).Error
	return events, err
}

func (r *CoinCopilotRepository) DeleteThread(threadID string, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var thread models.CoinCopilotThread
		if err := tx.Where("id = ? AND user_id = ?", threadID, userID).First(&thread).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&models.CoinCopilotRun{}).Where("thread_id = ? AND user_id = ? AND status IN ?", threadID, userID, []models.CopilotRunStatus{
			models.CopilotRunQueued, models.CopilotRunRunning, models.CopilotRunPaused, models.CopilotRunCancelRequested,
		}).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return ErrCopilotThreadActive
		}
		var runIDs []string
		if err := tx.Model(&models.CoinCopilotRun{}).Where("thread_id = ? AND user_id = ?", threadID, userID).Pluck("id", &runIDs).Error; err != nil {
			return err
		}
		if len(runIDs) > 0 {
			if err := tx.Where("run_id IN ?", runIDs).Delete(&models.CoinCopilotResumeRequest{}).Error; err != nil {
				return err
			}
			if err := tx.Where("run_id IN ?", runIDs).Delete(&models.CoinCopilotCheckpoint{}).Error; err != nil {
				return err
			}
			if err := tx.Where("run_id IN ?", runIDs).Delete(&models.CoinCopilotEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ? AND user_id = ?", runIDs, userID).Delete(&models.CoinCopilotRun{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ? AND user_id = ?", threadID, userID).Delete(&models.CoinCopilotThread{}).Error
	})
}

func (r *CoinCopilotRepository) RecoverStale(staleBefore time.Time, resumeWindow time.Duration) ([]string, error) {
	var runs []models.CoinCopilotRun
	if err := r.db.Where("status IN ? AND (heartbeat_at IS NULL OR heartbeat_at < ?)",
		[]models.CopilotRunStatus{models.CopilotRunRunning, models.CopilotRunCancelRequested}, staleBefore).Find(&runs).Error; err != nil {
		return nil, err
	}
	recovered := make([]string, 0, len(runs))
	for _, run := range runs {
		if run.Status == models.CopilotRunCancelRequested {
			won, _, err := r.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
				[]models.CopilotRunStatus{models.CopilotRunCancelRequested}, models.CopilotRunCancelled,
				map[string]interface{}{"worker_id": "", "heartbeat_at": nil},
				models.CopilotEventRunCancelled, `{"reason":"owner_cancelled"}`)
			if err != nil {
				return nil, err
			}
			if won {
				recovered = append(recovered, run.ID)
			}
			continue
		}
		if run.CheckpointVersion > 0 {
			now := time.Now().UTC()
			deadline := now.Add(resumeWindow)
			won, _, err := r.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID, []models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunPaused,
				map[string]interface{}{"paused_at": now, "resume_deadline": deadline, "worker_id": "", "heartbeat_at": nil},
				models.CopilotEventRunPaused, fmt.Sprintf(`{"reason":"execution_lost","checkpointVersion":%d,"resumeDeadline":%q}`, run.CheckpointVersion, deadline.Format(time.RFC3339)))
			if err != nil {
				return nil, err
			}
			if won {
				recovered = append(recovered, run.ID)
			}
		} else {
			won, _, err := r.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID, []models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunFailed,
				map[string]interface{}{"failure_code": "execution_lost", "failure_message": "Coin Copilot execution was interrupted."},
				models.CopilotEventRunFailed, `{"code":"execution_lost","message":"Coin Copilot execution was interrupted.","retryable":true}`)
			if err != nil {
				return nil, err
			}
			if won {
				recovered = append(recovered, run.ID)
			}
		}
	}
	return recovered, nil
}

func (r *CoinCopilotRepository) ExpirePaused(now time.Time) ([]string, error) {
	var runs []models.CoinCopilotRun
	if err := r.db.Where("status = ? AND resume_deadline IS NOT NULL AND resume_deadline < ?", models.CopilotRunPaused, now).Find(&runs).Error; err != nil {
		return nil, err
	}
	var expired []string
	for _, run := range runs {
		won, _, err := r.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID, []models.CopilotRunStatus{models.CopilotRunPaused}, models.CopilotRunFailed,
			map[string]interface{}{"failure_code": "resume_window_expired", "failure_message": "The resume window expired."},
			models.CopilotEventRunFailed, `{"code":"resume_window_expired","message":"The resume window expired.","retryable":false}`)
		if err != nil {
			return nil, err
		}
		if won {
			expired = append(expired, run.ID)
		}
	}
	return expired, nil
}

func (r *CoinCopilotRepository) Prune(eventCutoff, checkpointCutoff time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var eventRuns []string
		if err := tx.Model(&models.CoinCopilotRun{}).
			Where("status IN ? AND completed_at < ? AND events_pruned_at IS NULL", []models.CopilotRunStatus{models.CopilotRunCompleted, models.CopilotRunFailed, models.CopilotRunCancelled}, eventCutoff).
			Pluck("id", &eventRuns).Error; err != nil {
			return err
		}
		if len(eventRuns) > 0 {
			if err := tx.Where("run_id IN ?", eventRuns).Delete(&models.CoinCopilotEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.CoinCopilotRun{}).Where("id IN ?", eventRuns).Update("events_pruned_at", time.Now().UTC()).Error; err != nil {
				return err
			}
		}
		var checkpointRuns []string
		if err := tx.Model(&models.CoinCopilotRun{}).
			Where("status IN ? AND completed_at < ? AND checkpoints_pruned_at IS NULL", []models.CopilotRunStatus{models.CopilotRunCompleted, models.CopilotRunFailed, models.CopilotRunCancelled}, checkpointCutoff).
			Pluck("id", &checkpointRuns).Error; err != nil {
			return err
		}
		if len(checkpointRuns) > 0 {
			if err := tx.Where("run_id IN ?", checkpointRuns).Delete(&models.CoinCopilotCheckpoint{}).Error; err != nil {
				return err
			}
			if err := tx.Where("run_id IN ?", checkpointRuns).Delete(&models.CoinCopilotResumeRequest{}).Error; err != nil {
				return err
			}
			return tx.Model(&models.CoinCopilotRun{}).Where("id IN ?", checkpointRuns).Update("checkpoints_pruned_at", time.Now().UTC()).Error
		}
		return nil
	})
}
