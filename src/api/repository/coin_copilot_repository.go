package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

var (
	ErrCopilotTransitionConflict  = errors.New("coin copilot transition conflict")
	ErrCopilotThreadActive        = errors.New("coin copilot thread has an active run")
	ErrCopilotOwnerCapacity       = errors.New("coin copilot owner capacity reached")
	ErrCopilotQueueCapacity       = errors.New("coin copilot queue capacity reached")
	ErrCopilotStartKeyConflict    = errors.New("coin copilot start key conflict")
	ErrCopilotDeepHandoffConflict = errors.New("coin copilot deep handoff idempotency conflict")
	ErrCopilotDeepHandoffState    = errors.New("coin copilot deep handoff state conflict")
)

type CoinCopilotRepository struct {
	db *gorm.DB
}

func NewCoinCopilotRepository(db *gorm.DB) *CoinCopilotRepository {
	return &CoinCopilotRepository{db: db}
}

// DigestCoinCopilotAppContext binds admission to the canonical app context
// stored on the run. The request body never supplies this value.
func DigestCoinCopilotAppContext(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type DeepHandoffAdmission struct {
	Handoff          *models.CoinCopilotDeepHandoff
	Job              *models.DeepIdentificationJob
	Artifacts        []models.DeepIdentificationArtifact
	Target           DeepHandoffTargetToken
	ProviderSettings map[string]string
	ProviderDefaults map[string]string
	MaxActivePerUser int
	QueueDepth       int
	AfterCommit      func(jobID uint)
}

type DeepHandoffFaceToken struct {
	ID          uint
	FilePath    string
	ContentPath string
	ContentHash string
	CreatedAt   string
}

type DeepHandoffTargetToken struct {
	Kind      models.CoinCopilotDeepHandoffTargetKind
	ID        uint
	UserID    uint
	State     string
	UpdatedAt string
	Context   string
	Obverse   DeepHandoffFaceToken
	Reverse   DeepHandoffFaceToken
}

// AdmitDeepHandoff is the one durable admission transaction. The handoff key
// lookup intentionally precedes every mutable-state check, so a changed
// binding can never be interpreted as a fresh candidate after a crash/replay.
// The production SQLite DSN uses _txlock=immediate, making this transaction a
// linearizable critical section for writers.
func (r *CoinCopilotRepository) AdmitDeepHandoff(input DeepHandoffAdmission) (*models.CoinCopilotDeepHandoff, *models.DeepIdentificationJob, bool, error) {
	if input.Handoff == nil || input.Job == nil {
		return nil, nil, false, ErrCopilotDeepHandoffState
	}
	var selected models.DeepIdentificationJob
	var resultHandoff models.CoinCopilotDeepHandoff
	replayed := false
	createdJob := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.CoinCopilotDeepHandoff
		err := tx.Where("user_id = ? AND run_id = ? AND handoff_key_hash = ?",
			input.Handoff.UserID, input.Handoff.RunID, input.Handoff.HandoffKeyHash).
			First(&existing).Error
		switch {
		case err == nil:
			if !sameDeepHandoffRequestBinding(&existing, input.Handoff) {
				return ErrCopilotDeepHandoffConflict
			}
			if err := tx.Where("id = ? AND user_id = ?", existing.DeepJobID, existing.UserID).
				First(&selected).Error; err != nil {
				return err
			}
			resultHandoff = existing
			replayed = true
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var run models.CoinCopilotRun
		if err := tx.Where("id = ? AND user_id = ?", input.Handoff.RunID, input.Handoff.UserID).
			First(&run).Error; err != nil {
			return err
		}
		if run.Status != models.CopilotRunRunning || run.CancelRequestedAt != nil ||
			run.ExecutionID != input.Handoff.ExecutionID ||
			run.CheckpointVersion != input.Handoff.ExpectedCheckpointVersion ||
			DigestCoinCopilotAppContext(run.AppContextJSON) != input.Handoff.AppContextDigest {
			return ErrCopilotDeepHandoffState
		}
		if err := validateDeepHandoffAdmissionBinding(input); err != nil {
			return err
		}
		if err := validateDeepHandoffTarget(tx, input.Target); err != nil {
			return err
		}
		for key, expected := range input.ProviderSettings {
			var setting models.AppSetting
			err := tx.Where("key = ?", key).First(&setting).Error
			if err == nil && setting.Value != expected {
				return ErrCopilotDeepHandoffState
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if defaultValue, known := input.ProviderDefaults[key]; known && defaultValue != expected {
					return ErrCopilotDeepHandoffState
				}
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if input.Job.UserID != input.Handoff.UserID ||
			!models.IsValidDeepJobSourceBinding(input.Job) {
			return ErrCopilotDeepHandoffState
		}

		if input.Handoff.Operation == models.CoinCopilotDeepHandoffOperationRerun {
			if input.Handoff.PriorJobID == nil {
				return ErrCopilotDeepHandoffState
			}
			var prior models.DeepIdentificationJob
			if err := tx.Where("id = ? AND user_id = ?", *input.Handoff.PriorJobID, input.Handoff.UserID).
				First(&prior).Error; err != nil {
				return ErrCopilotDeepHandoffState
			}
			if prior.ID == input.Job.ID || prior.Source != input.Job.Source ||
				!sameDeepTarget(&prior, input.Job) {
				return ErrCopilotDeepHandoffState
			}
		}

		// Reuse an equivalent active job first, then the newest retained
		// completed/partial report. Both remain rows in the existing engine.
		err = tx.Where("user_id = ? AND input_fingerprint = ? AND active_key = ? AND source IN ?",
			input.Job.UserID, input.Job.InputFingerprint, "active", supportedDeepJobSources).
			Order("created_at ASC").First(&selected).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = tx.Where(
				"user_id = ? AND input_fingerprint = ? AND status IN ? AND expires_at > ? AND report_json <> '' AND source IN ?",
				input.Job.UserID, input.Job.InputFingerprint,
				[]models.DeepJobStatus{models.DeepJobStatusCompleted, models.DeepJobStatusPartial},
				time.Now().UTC(), supportedDeepJobSources,
			).Order("created_at DESC, id DESC").First(&selected).Error
			if err == nil {
				input.Handoff.AdmissionOutcome = models.CoinCopilotDeepHandoffOutcomeReusedResult
			}
		} else if err == nil {
			input.Handoff.AdmissionOutcome = models.CoinCopilotDeepHandoffOutcomeReusedActive
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var active, queued int64
			if input.MaxActivePerUser > 0 {
				if err := tx.Model(&models.DeepIdentificationJob{}).
					Where("user_id = ? AND status IN ?", input.Job.UserID,
						[]models.DeepJobStatus{models.DeepJobStatusQueued, models.DeepJobStatusRunning}).
					Count(&active).Error; err != nil {
					return err
				}
				if active >= int64(input.MaxActivePerUser) {
					return ErrCopilotOwnerCapacity
				}
			}
			if input.QueueDepth > 0 {
				if err := tx.Model(&models.DeepIdentificationJob{}).
					Where("status = ?", models.DeepJobStatusQueued).Count(&queued).Error; err != nil {
					return err
				}
				if queued >= int64(input.QueueDepth) {
					return ErrCopilotQueueCapacity
				}
			}
			selected = *input.Job
			selected.ActiveKey = "active"
			if selected.Status == "" {
				selected.Status = models.DeepJobStatusQueued
			}

			if err := tx.Create(&selected).Error; err != nil {
				return err
			}
			createdJob = true
			input.Handoff.AdmissionOutcome = models.CoinCopilotDeepHandoffOutcomeCreated
			for index := range input.Artifacts {
				input.Artifacts[index].JobID = selected.ID
				input.Artifacts[index].UserID = selected.UserID
				if err := tx.Create(&input.Artifacts[index]).Error; err != nil {
					return err
				}
			}
		}
		input.Handoff.DeepJobID = selected.ID
		if !models.IsValidCoinCopilotDeepHandoff(input.Handoff) {
			return ErrCopilotDeepHandoffState
		}
		if err := tx.Create(input.Handoff).Error; err != nil {
			return err
		}
		resultHandoff = *input.Handoff
		return nil
	})
	if err != nil {
		return nil, nil, false, err
	}
	if createdJob && input.AfterCommit != nil {
		input.AfterCommit(selected.ID)
	}
	return &resultHandoff, &selected, replayed, nil
}

func sameDeepHandoffRequestBinding(existing, candidate *models.CoinCopilotDeepHandoff) bool {
	if existing == nil || candidate == nil {
		return false
	}
	return existing.RequestFingerprint == candidate.RequestFingerprint &&
		existing.ExecutionID == candidate.ExecutionID &&
		existing.ExpectedCheckpointVersion == candidate.ExpectedCheckpointVersion &&
		existing.AppContextDigest == candidate.AppContextDigest &&
		existing.Operation == candidate.Operation &&
		existing.TargetKind == candidate.TargetKind &&
		existing.TargetID == candidate.TargetID &&
		sameOptionalUint(existing.PriorJobID, candidate.PriorJobID) &&
		existing.TargetSnapshotFingerprint == candidate.TargetSnapshotFingerprint
}

func sameOptionalUint(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validateDeepHandoffAdmissionBinding(input DeepHandoffAdmission) error {
	if input.Handoff == nil || input.Job == nil ||
		input.Target.Kind != input.Handoff.TargetKind ||
		input.Target.ID != input.Handoff.TargetID ||
		input.Target.UserID != input.Handoff.UserID ||
		input.Job.UserID != input.Handoff.UserID {
		return ErrCopilotDeepHandoffState
	}
	switch input.Target.Kind {
	case models.CoinCopilotDeepHandoffTargetCoin:
		if input.Job.Source != models.DeepJobSourceSavedCoin ||
			input.Job.CoinID == nil || *input.Job.CoinID != input.Target.ID {
			return ErrCopilotDeepHandoffState
		}
	case models.CoinCopilotDeepHandoffTargetDraft:
		if input.Job.Source != models.DeepJobSourceCopilotDraft ||
			input.Job.SourceDraftID == nil || *input.Job.SourceDraftID != input.Target.ID {
			return ErrCopilotDeepHandoffState
		}
	default:
		return ErrCopilotDeepHandoffState
	}
	return nil
}

func validateDeepHandoffTarget(tx *gorm.DB, expected DeepHandoffTargetToken) error {
	if expected.ID == 0 || expected.UserID == 0 ||
		expected.Obverse.ID == 0 || expected.Reverse.ID == 0 ||
		expected.Obverse.ID == expected.Reverse.ID {
		return ErrCopilotDeepHandoffState
	}
	switch expected.Kind {
	case models.CoinCopilotDeepHandoffTargetCoin:
		var coin models.Coin
		if err := tx.Select("id", "user_id", "notes", "updated_at").
			Where("id = ? AND user_id = ?", expected.ID, expected.UserID).First(&coin).Error; err != nil {
			return ErrCopilotDeepHandoffState
		}
		if coin.UpdatedAt.UTC().Format(time.RFC3339Nano) != expected.UpdatedAt || coin.Notes != expected.Context {
			return ErrCopilotDeepHandoffState
		}
		var faces []models.CoinImage
		if err := tx.Where("coin_id = ? AND id IN ?", expected.ID, []uint{expected.Obverse.ID, expected.Reverse.ID}).
			Find(&faces).Error; err != nil {
			return err
		}
		return compareDeepHandoffFaces(faces, expected)
	case models.CoinCopilotDeepHandoffTargetDraft:
		var draft models.QuickCaptureDraft
		if err := tx.Select("id", "user_id", "status", "notes", "updated_at").
			Where("id = ? AND user_id = ? AND status = ?", expected.ID, expected.UserID, models.QuickCaptureDraftStatusActive).
			First(&draft).Error; err != nil {
			return ErrCopilotDeepHandoffState
		}
		if draft.UpdatedAt.UTC().Format(time.RFC3339Nano) != expected.UpdatedAt ||
			draft.Notes != expected.Context || string(draft.Status) != expected.State {
			return ErrCopilotDeepHandoffState
		}
		var draftFaces []models.QuickCaptureDraftImage
		if err := tx.Where("draft_id = ? AND user_id = ? AND id IN ?",
			expected.ID, expected.UserID, []uint{expected.Obverse.ID, expected.Reverse.ID}).Find(&draftFaces).Error; err != nil {
			return err
		}
		faces := make([]models.CoinImage, 0, len(draftFaces))
		for _, face := range draftFaces {
			faces = append(faces, models.CoinImage{
				ID: face.ID, FilePath: face.FilePath, ImageType: face.ImageType, CreatedAt: face.CreatedAt,
			})
		}
		return compareDeepHandoffFaces(faces, expected)
	default:
		return ErrCopilotDeepHandoffState
	}
}

func compareDeepHandoffFaces(faces []models.CoinImage, expected DeepHandoffTargetToken) error {
	if len(faces) != 2 {
		return ErrCopilotDeepHandoffState
	}
	byID := make(map[uint]models.CoinImage, len(faces))
	for _, face := range faces {
		byID[face.ID] = face
	}
	for _, item := range []struct {
		want DeepHandoffFaceToken
		role models.ImageType
	}{
		{expected.Obverse, models.ImageTypeObverse},
		{expected.Reverse, models.ImageTypeReverse},
	} {
		got, ok := byID[item.want.ID]
		if !ok || got.ImageType != item.role || got.FilePath != item.want.FilePath ||
			got.CreatedAt.UTC().Format(time.RFC3339Nano) != item.want.CreatedAt {
			return ErrCopilotDeepHandoffState
		}
		content, err := os.ReadFile(item.want.ContentPath)
		if err != nil {
			return ErrCopilotDeepHandoffState
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != item.want.ContentHash {
			return ErrCopilotDeepHandoffState
		}
	}
	return nil
}

func (r *CoinCopilotRepository) FindDeepHandoffByJob(runID string, userID, jobID uint) (*models.CoinCopilotDeepHandoff, *models.DeepIdentificationJob, error) {
	var handoff models.CoinCopilotDeepHandoff
	if err := r.db.Where("run_id = ? AND user_id = ? AND deep_job_id = ?", runID, userID, jobID).
		Order("created_at DESC, id DESC").First(&handoff).Error; err != nil {
		return nil, nil, err
	}
	var job models.DeepIdentificationJob
	if err := r.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return nil, nil, err
	}
	return &handoff, &job, nil
}

func sameDeepTarget(left, right *models.DeepIdentificationJob) bool {
	if left == nil || right == nil || left.Source != right.Source {
		return false
	}
	switch left.Source {
	case models.DeepJobSourceSavedCoin:
		return left.CoinID != nil && right.CoinID != nil && *left.CoinID == *right.CoinID
	case models.DeepJobSourceCopilotDraft:
		return left.SourceDraftID != nil && right.SourceDraftID != nil && *left.SourceDraftID == *right.SourceDraftID
	default:
		return left.CoinID == nil && right.CoinID == nil &&
			left.SourceDraftID == nil && right.SourceDraftID == nil
	}
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
		if err := requestDeepHandoffCancellation(tx, runID, userID, now); err != nil {
			return err
		}
		return tx.First(&run, "id = ?", runID).Error
	})
	return &run, immediate, err
}

func requestDeepHandoffCancellation(tx *gorm.DB, runID string, userID uint, requestedAt time.Time) error {
	deepJobIDs := tx.Model(&models.CoinCopilotDeepHandoff{}).
		Select("deep_job_id").
		Where("run_id = ? AND user_id = ?", runID, userID)
	return tx.Model(&models.DeepIdentificationJob{}).
		Where("id IN (?) AND user_id = ?", deepJobIDs, userID).
		Where("status IN ? AND cancel_requested_at IS NULL", deepJobActiveStatuses).
		Where("source IN ?", supportedDeepJobSources).
		Where(deepJobSourceBindingSQL).
		Update("cancel_requested_at", requestedAt).Error
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
