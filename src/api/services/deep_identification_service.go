package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// Errors returned by DeepIdentificationService's artifact-handling methods.
// These are intentionally generic (Principle V / FR-036) - no internal
// details (file paths, DB errors) are surfaced to callers.
var (
	ErrDeepArtifactRoleInvalid   = errors.New("invalid artifact role")
	ErrDeepArtifactRoleExists    = errors.New("an artifact for this role already exists")
	ErrDeepArtifactHintLimit     = errors.New("hint image limit reached")
	ErrDeepArtifactMissingCoin   = errors.New("coin not found")
	ErrDeepArtifactMissingImage  = errors.New("coin image not found")
	ErrDeepArtifactMissingUpload = errors.New("no file provided")
)

// MaxDeepIdentificationHintArtifacts caps hint/reference images per job
// (data-model.md §5).
const MaxDeepIdentificationHintArtifacts = 3

// MaxDeepIdentificationRetryDepth caps the retry lineage depth (FR-020).
const MaxDeepIdentificationRetryDepth = 3

// Errors returned by DeepIdentificationService's job-orchestration methods
// (Phase 4/5). Also generic per Principle V / FR-036.
var (
	ErrDeepJobQueueFull      = errors.New("deep identification queue is full, try again shortly")
	ErrDeepJobAtCapacity     = errors.New("an analysis is already running for this user")
	ErrDeepJobDisabled       = errors.New("deep identification is currently disabled")
	ErrDeepJobNotFound       = errors.New("job not found")
	ErrDeepJobNotCancellable = errors.New("job is already in a terminal state")
	ErrDeepJobMissingObverse = errors.New("missing_obverse")
	ErrDeepJobMissingReverse = errors.New("missing_reverse")
	ErrDeepJobInvalidRequest = errors.New("invalid request")
	ErrDeepJobRetryDepth     = errors.New("retry depth limit reached")
	ErrDeepJobNotTerminal    = errors.New("job is not terminal")
)

// deepJobPollInterval is the fallback ticker period a worker uses to check
// for queued work when it hasn't been woken by an explicit signal. Kept
// short so tests remain fast; production correctness does not depend on
// its exact value, only that it is bounded.
const deepJobPollInterval = 25 * time.Millisecond

// DeepPipelineResult is what a pipeline run (Phase 7: the Python LangGraph
// agent, proxied via agent_proxy.go) reports back to the worker loop.
type DeepPipelineResult struct {
	ReportJSON     string
	ProposalJSON   string
	Partial        bool
	FailureCode    string
	FailureMessage string
}

// DeepPipelineRunner is the seam between the Go worker loop and the actual
// identification pipeline. Phase 4 tests inject a fake; Phase 7 wires the
// real agent-proxy-backed implementation via SetPipelineRunner.
type DeepPipelineRunner interface {
	Run(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error)
}

// noopPipelineRunner is the default runner used until Phase 7 wires the
// real pipeline. It fails fast rather than hanging, so a misconfigured
// deployment degrades safely instead of leaving jobs running forever.
type noopPipelineRunner struct{}

func (noopPipelineRunner) Run(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
	return nil, errors.New("deep identification pipeline is not configured")
}

// DeepIdentificationService is the business-logic layer for the deep
// agentic coin identification job/event/artifact domain
// (344-deep-agentic-coin-identification). It is intentionally separate
// from AIJobService; models.AIJob is never touched by this service.
//
// This file (Phase 3) covers artifact validation/storage/reuse/deletion and
// input-fingerprint computation only. Worker-pool/pipeline orchestration is
// added in Phase 4 (deep_identification_service.go additions), REST wiring
// in Phase 5.
type DeepIdentificationService struct {
	repo        *repository.DeepIdentificationRepository
	imageRepo   *repository.ImageRepository
	settingsSvc *SettingsService
	logger      *Logger

	// artifacts is the artifact-management seam (T103): validation,
	// storage, reuse, and deletion of job artifacts. Split out because it
	// is fully self-contained (repo/imageRepo/imageSvc/uploadDir/metrics)
	// with no dependency on job-lifecycle or worker-pool state.
	artifacts *deepIdentificationArtifactStore

	// janitor is the retention/recovery seam (T103): stale-job recovery and
	// scheduled event/artifact pruning. It depends on artifacts for
	// hint/job artifact deletion but owns no state shared with the worker
	// pool or job-lifecycle seams.
	janitor *deepIdentificationJanitor

	runnerMu sync.RWMutex
	runner   DeepPipelineRunner

	// intakeMu prevents workers from claiming a queued intake job before its
	// obverse/reverse artifacts have been persisted. Shared, load-bearing
	// synchronization between job-creation (CreateJobFromIntake) and the
	// worker claim loop (workerLoop) - see
	// .squad/decisions/inbox/maximus-service-decomposition.md for why this
	// keeps job lifecycle and the worker pool on one type rather than
	// splitting them across a shared-mutex seam.
	intakeMu sync.RWMutex

	cancelMu sync.Mutex
	cancels  map[uint]context.CancelFunc

	broker  *DeepIdentificationBroker
	metrics *deepIdentificationRuntimeMetrics

	wakeMu sync.Mutex
	wake   chan struct{}

	// providerBudgets is the per-job/per-provider call-budget tracker
	// (T078). It is optional (nil in tests that don't wire it) so its
	// terminal-path Reset is always nil-guarded.
	providerBudgets *DeepProviderBudgetTracker

	// internalTokenSvc mints/verifies the job-scoped internal tokens used
	// by the Python pipeline to call back into Go's provider-tool
	// endpoints. Optional (nil-guarded) so the terminal path can revoke a
	// settled job's token (T081) without requiring every test to wire it.
	internalTokenSvc *InternalTokenService
}

// NewDeepIdentificationService constructs the service, following the
// repo -> service -> handler DI pattern used elsewhere (main.go:246-249).
func NewDeepIdentificationService(repo *repository.DeepIdentificationRepository, imageRepo *repository.ImageRepository, imageSvc *ImageService, settingsSvc *SettingsService, logger *Logger, uploadDir string) *DeepIdentificationService {
	metrics := &deepIdentificationRuntimeMetrics{}
	broker := NewDeepIdentificationBroker()
	artifacts := newDeepIdentificationArtifactStore(repo, imageRepo, imageSvc, uploadDir, metrics)
	return &DeepIdentificationService{
		repo:        repo,
		imageRepo:   imageRepo,
		settingsSvc: settingsSvc,
		logger:      logger,
		artifacts:   artifacts,
		janitor:     newDeepIdentificationJanitor(repo, settingsSvc, broker, metrics, logger, artifacts),
		runner:      noopPipelineRunner{},
		cancels:     make(map[uint]context.CancelFunc),
		broker:      broker,
		metrics:     metrics,
		wake:        make(chan struct{}, 1),
	}
}

// Broker returns the service's shared in-process SSE fan-out broker (T095)
// so the handler layer can Subscribe/SubscriberCount for the
// `GET /deep-identification/jobs/{id}/events` endpoint (T096) without the
// handler needing its own broker instance or direct access to internals.
func (s *DeepIdentificationService) Broker() *DeepIdentificationBroker {
	return s.broker
}

// SetPipelineRunner installs the pipeline implementation used by the worker
// loop (Phase 7 wires the real agent-proxy-backed runner here; tests inject
// a fake). Safe to call concurrently with running workers.
func (s *DeepIdentificationService) SetPipelineRunner(runner DeepPipelineRunner) {
	s.runnerMu.Lock()
	defer s.runnerMu.Unlock()
	if runner == nil {
		runner = noopPipelineRunner{}
	}
	s.runner = runner
}

// SetProviderBudgetTracker wires the per-job/per-provider call-budget
// tracker (constructed alongside the rest of the object graph in main.go)
// into the job service so its terminal path can release a job's tracked
// budget entries the moment the job settles (T078). Without this, the
// tracker - injected only into DeepProviderToolsHandler - has no caller of
// Reset in production and accumulates one entry per (jobID, provider)
// forever.
func (s *DeepIdentificationService) SetProviderBudgetTracker(tracker *DeepProviderBudgetTracker) {
	s.providerBudgets = tracker
	s.janitor.setProviderBudgetTracker(tracker)
}

// SetInternalTokenService wires the internal token service into the job
// service so its terminal path can revoke a settled job's job-scoped token
// (T081), preventing a long-TTL token from being replayed after the job it
// was minted for has already completed, failed, or been cancelled.
func (s *DeepIdentificationService) SetInternalTokenService(tokenSvc *InternalTokenService) {
	s.internalTokenSvc = tokenSvc
	s.janitor.setInternalTokenService(tokenSvc)
}

func (s *DeepIdentificationService) pipelineRunner() DeepPipelineRunner {
	s.runnerMu.RLock()
	defer s.runnerMu.RUnlock()
	return s.runner
}

// notifyWorkers wakes a single idle worker (if any) without blocking. A
// missed signal is harmless: the poll-interval ticker fallback in the
// worker loop will pick the job up shortly after.
func (s *DeepIdentificationService) notifyWorkers() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// ValidateAndSaveArtifact delegates to the artifact-management seam
// (T103); see deepIdentificationArtifactStore.ValidateAndSaveArtifact for
// the full behavior contract.
func (s *DeepIdentificationService) ValidateAndSaveArtifact(jobID, userID uint, role models.DeepArtifactRole, filename string, fileData []byte) (*models.DeepIdentificationArtifact, error) {
	return s.artifacts.ValidateAndSaveArtifact(jobID, userID, role, filename, fileData)
}

// ReuseSavedCoinImage delegates to the artifact-management seam (T103); see
// deepIdentificationArtifactStore.ReuseSavedCoinImage for the full behavior
// contract.
func (s *DeepIdentificationService) ReuseSavedCoinImage(jobID, userID, coinID uint, role models.DeepArtifactRole, sourceCoinImageID uint) (*models.DeepIdentificationArtifact, error) {
	return s.artifacts.ReuseSavedCoinImage(jobID, userID, coinID, role, sourceCoinImageID)
}

// DeleteHintArtifacts delegates to the artifact-management seam (T103); see
// deepIdentificationArtifactStore.DeleteHintArtifacts for the full behavior
// contract.
func (s *DeepIdentificationService) DeleteHintArtifacts(jobID uint) error {
	return s.artifacts.DeleteHintArtifacts(jobID)
}

// DeleteJobArtifacts delegates to the artifact-management seam (T103); see
// deepIdentificationArtifactStore.DeleteJobArtifacts for the full behavior
// contract.
func (s *DeepIdentificationService) DeleteJobArtifacts(jobID uint) error {
	return s.artifacts.DeleteJobArtifacts(jobID)
}

// --- Phase 4: job orchestration (worker pool, cancel, timeout, janitor) ---

// StartJob enqueues a new job or, per FR-007, returns the existing
// active (queued/running) job for the same (user, fingerprint) pair. If the
// user is already at their per-user concurrency limit with a *different*
// in-flight job, StartJob refuses the new submission with
// ErrDeepJobAtCapacity rather than substituting an unrelated job's result.
// Returns ErrDeepJobQueueFull if the global queue depth is exceeded.
func (s *DeepIdentificationService) StartJob(job *models.DeepIdentificationJob) (*models.DeepIdentificationJob, bool, error) {
	settings := s.settingsSvc.GetDeepIdentificationSettings()
	if !settings.Enabled {
		return nil, false, ErrDeepJobDisabled
	}

	activeCount, err := s.repo.CountActiveJobsForUser(job.UserID)
	if err != nil {
		return nil, false, err
	}
	if activeCount >= int64(settings.MaxActivePerUser) {
		existing, err := s.repo.FindActiveByFingerprint(job.UserID, job.InputFingerprint)
		if err == nil && existing != nil {
			return existing, true, nil
		}
		// Different fingerprint than the user's existing active job(s): this
		// is not a duplicate submission, it is a genuinely new request that
		// the per-user concurrency limit has no room for. Refuse it rather
		// than handing back an unrelated job's (eventual) result under
		// someone else's coin - see ErrDeepJobAtCapacity.
		return nil, false, ErrDeepJobAtCapacity
	}

	queuedCount, err := s.repo.CountQueuedJobs()
	if err != nil {
		return nil, false, err
	}
	if queuedCount >= int64(settings.QueueDepth) {
		return nil, false, ErrDeepJobQueueFull
	}

	created, reused, err := s.repo.CreateJob(job)
	if err != nil {
		return nil, false, err
	}
	if !reused {
		s.notifyWorkers()
	}
	return created, reused, nil
}

// --- Phase 5: job-create orchestration (multipart intake / saved-coin) ---

// CreateJobInput is the resolved, validated set of inputs for creating a
// deep identification job, independent of how the handler parsed them out
// of multipart form data.
type CreateJobInput struct {
	UserID             uint
	CoinID             *uint // saved-coin source; nil for new-coin intake
	Notes              string
	RequestedProviders []string
	ObverseBytes       []byte // nil if reusing the saved coin's existing obverse
	ObverseFilename    string
	ReverseBytes       []byte // nil if reusing the saved coin's existing reverse
	ReverseFilename    string
	Hints              []CreateJobHintInput
}

// CreateJobHintInput is one ephemeral hint/reference image supplied at
// job-create time.
type CreateJobHintInput struct {
	Filename string
	Bytes    []byte
}

// CreateJobFromIntake resolves obverse/reverse inputs (uploaded bytes, or a
// saved coin's existing image when coinID is set and no upload is
// supplied), computes the input fingerprint up front so idempotent
// duplicate-submit reuse (FR-007) works without ever creating artifacts for
// a job that turns out to be a dupe, then creates the job and persists its
// artifacts before workers may claim it. If artifact persistence fails after
// the job row was created, the job is settled failed immediately rather than
// left queued with missing evidence.
func (s *DeepIdentificationService) CreateJobFromIntake(in CreateJobInput) (*models.DeepIdentificationJob, bool, error) {
	source := models.DeepJobSourceIntake
	var obverseImage, reverseImage *models.CoinImage
	if in.CoinID != nil {
		if _, err := s.imageRepo.FindCoinByOwner(*in.CoinID, in.UserID); err != nil {
			return nil, false, ErrDeepArtifactMissingCoin
		}
		source = models.DeepJobSourceSavedCoin
	}

	obverseHash, err := s.resolveRoleHash(in.ObverseBytes, in.CoinID, models.ImageTypeObverse, &obverseImage)
	if err != nil {
		return nil, false, err
	}
	reverseHash, err := s.resolveRoleHash(in.ReverseBytes, in.CoinID, models.ImageTypeReverse, &reverseImage)
	if err != nil {
		return nil, false, err
	}

	hintHashes := make([]string, len(in.Hints))
	for i, h := range in.Hints {
		sum := sha256.Sum256(h.Bytes)
		hintHashes[i] = hex.EncodeToString(sum[:])
	}

	var coinID uint
	if in.CoinID != nil {
		coinID = *in.CoinID
	}
	fingerprint := ComputeInputFingerprint(FingerprintInput{
		UserID:             in.UserID,
		CoinID:             coinID,
		ObverseHash:        obverseHash,
		ReverseHash:        reverseHash,
		HintHashes:         hintHashes,
		Notes:              in.Notes,
		RequestedProviders: in.RequestedProviders,
	})

	job := &models.DeepIdentificationJob{
		UserID:             in.UserID,
		CoinID:             in.CoinID,
		Source:             source,
		InputFingerprint:   fingerprint,
		Notes:              in.Notes,
		RequestedProviders: strings.Join(in.RequestedProviders, ","),
		ExpiresAt:          time.Now().Add(90 * 24 * time.Hour),
	}

	// StartJob makes the row visible as queued and wakes workers. Hold the
	// intake write lock across row creation and artifact persistence so neither
	// the wake signal nor the polling fallback can claim incomplete evidence.
	s.intakeMu.Lock()
	defer s.intakeMu.Unlock()

	created, reused, err := s.StartJob(job)
	if err != nil {
		return nil, false, err
	}
	if reused {
		return created, true, nil
	}

	if err := s.persistIntakeArtifacts(created.ID, in, obverseImage, reverseImage); err != nil {
		_, _ = s.repo.SettleTerminal(created.ID, []models.DeepJobStatus{models.DeepJobStatusQueued}, models.DeepJobStatusFailed, "", "", "invalid_input", "The submitted images could not be saved.")
		_ = s.DeleteJobArtifacts(created.ID)
		return nil, false, err
	}
	return created, false, nil
}

// resolveRoleHash computes the fingerprint-input hash for one coin-face
// role: from uploaded bytes if present, otherwise from the saved coin's
// existing image of that type (populating *outImage so the caller can
// persist a ReuseSavedCoinImage artifact without a second lookup). Returns
// ErrDeepJobMissingObverse/ErrDeepJobMissingReverse if neither is available.
func (s *DeepIdentificationService) resolveRoleHash(uploaded []byte, coinID *uint, imageType models.ImageType, outImage **models.CoinImage) (string, error) {
	missingErr := ErrDeepJobMissingObverse
	if imageType == models.ImageTypeReverse {
		missingErr = ErrDeepJobMissingReverse
	}
	if len(uploaded) > 0 {
		sum := sha256.Sum256(uploaded)
		return hex.EncodeToString(sum[:]), nil
	}
	if coinID == nil {
		return "", missingErr
	}
	image, err := s.imageRepo.FindCoinImageByType(*coinID, imageType)
	if err != nil {
		return "", missingErr
	}
	hashHex, _, err := s.artifacts.savedImageFingerprintHash(image)
	if err != nil {
		return "", err
	}
	*outImage = image
	return hashHex, nil
}

func (s *DeepIdentificationService) persistIntakeArtifacts(jobID uint, in CreateJobInput, obverseImage, reverseImage *models.CoinImage) error {
	if len(in.ObverseBytes) > 0 {
		if _, err := s.ValidateAndSaveArtifact(jobID, in.UserID, models.DeepArtifactRoleObverse, in.ObverseFilename, in.ObverseBytes); err != nil {
			return err
		}
	} else if obverseImage != nil {
		if _, err := s.ReuseSavedCoinImage(jobID, in.UserID, *in.CoinID, models.DeepArtifactRoleObverse, obverseImage.ID); err != nil {
			return err
		}
	}
	if len(in.ReverseBytes) > 0 {
		if _, err := s.ValidateAndSaveArtifact(jobID, in.UserID, models.DeepArtifactRoleReverse, in.ReverseFilename, in.ReverseBytes); err != nil {
			return err
		}
	} else if reverseImage != nil {
		if _, err := s.ReuseSavedCoinImage(jobID, in.UserID, *in.CoinID, models.DeepArtifactRoleReverse, reverseImage.ID); err != nil {
			return err
		}
	}
	for _, h := range in.Hints {
		if _, err := s.ValidateAndSaveArtifact(jobID, in.UserID, models.DeepArtifactRoleHint, h.Filename, h.Bytes); err != nil {
			return err
		}
	}
	return nil
}

// RetryJob creates a new job linked to an existing terminal job via
// RetryOfJobID (FR-020). Inputs are re-resolved at retry time: uploaded
// coin-face artifacts from the source job are re-used byte-for-byte
// (copied), saved-coin images are re-resolved as they exist now, and hint
// images (deleted at the source job's terminal state) are NOT copied - the
// caller must re-supply them via a fresh CreateJobFromIntake if still
// wanted. The source job's events/report are never modified.
func (s *DeepIdentificationService) RetryJob(sourceJobID, userID uint, notesOverride *string, providersOverride []string) (*models.DeepIdentificationJob, bool, error) {
	source, err := s.repo.GetJob(sourceJobID, userID)
	if err != nil {
		return nil, false, ErrDeepJobNotFound
	}
	if !models.IsDeepJobTerminal(source.Status) {
		return nil, false, ErrDeepJobNotTerminal
	}
	if source.RetryDepth >= MaxDeepIdentificationRetryDepth {
		return nil, false, ErrDeepJobRetryDepth
	}

	artifacts, err := s.artifacts.listArtifacts(sourceJobID)
	if err != nil {
		return nil, false, err
	}

	in := CreateJobInput{UserID: userID, CoinID: source.CoinID}
	if notesOverride != nil {
		in.Notes = *notesOverride
	} else {
		in.Notes = source.Notes
	}
	if providersOverride != nil {
		in.RequestedProviders = providersOverride
	} else if source.RequestedProviders != "" {
		in.RequestedProviders = strings.Split(source.RequestedProviders, ",")
	}

	for i := range artifacts {
		a := artifacts[i]
		if a.DeletedAt != nil || a.Role == models.DeepArtifactRoleHint {
			continue // hints are ephemeral and were deleted at terminal; never retried
		}
		var bytes []byte
		if a.Origin == models.DeepArtifactOriginUploaded && a.FilePath != "" {
			bytes, err = os.ReadFile(a.FilePath)
			if err != nil {
				continue // source file gone; fall back to re-resolving from the saved coin below
			}
		}
		switch a.Role {
		case models.DeepArtifactRoleObverse:
			if len(bytes) > 0 {
				in.ObverseBytes = bytes
				in.ObverseFilename = "obverse-retry" + filepath.Ext(a.FilePath)
			}
		case models.DeepArtifactRoleReverse:
			if len(bytes) > 0 {
				in.ReverseBytes = bytes
				in.ReverseFilename = "reverse-retry" + filepath.Ext(a.FilePath)
			}
		}
	}

	created, reused, err := s.CreateJobFromIntake(in)
	if err != nil {
		return nil, false, err
	}
	if !reused {
		retryDepth := source.RetryDepth + 1
		if err := s.repo.SetRetryLineage(created.ID, sourceJobID, retryDepth); err != nil {
			return nil, false, err
		}
		created.RetryOfJobID = &sourceJobID
		created.RetryDepth = retryDepth
	}
	return created, reused, nil
}

func isDeepJobActive(status models.DeepJobStatus) bool {
	return status == models.DeepJobStatusQueued || status == models.DeepJobStatusRunning
}

// GetJob returns an owner-scoped job by id (repository.IsRecordNotFound
// distinguishes "not found or not owned" for callers).
func (s *DeepIdentificationService) GetJob(jobID, userID uint) (*models.DeepIdentificationJob, error) {
	return s.repo.GetJob(jobID, userID)
}

// ListEventsSince is a thin pass-through to the repository (T096), scoped
// to the owning user exactly like GetJob, for the SSE handler's replay and
// live-tail re-reads.
func (s *DeepIdentificationService) ListEventsSince(jobID, userID uint, since int64) ([]models.DeepIdentificationEvent, error) {
	return s.repo.ListEventsSince(jobID, userID, since)
}

// ListJobs returns an owner-scoped, filtered/paginated job list (Phase 5
// REST layer). See repository.DeepJobListFilters for the supported filters.
func (s *DeepIdentificationService) ListJobs(userID uint, filters repository.DeepJobListFilters) ([]models.DeepIdentificationJob, error) {
	return s.repo.ListJobs(userID, filters)
}

// RequestCancel records a cancel request (FR-019). If the job is still
// queued, the settle-to-cancelled transition happens immediately here since
// no worker is watching it yet. If it is running, the in-memory cancel
// registry's context is cancelled so the worker loop observes it and
// performs the (single, race-safe) settle itself.
func (s *DeepIdentificationService) RequestCancel(jobID, userID uint) error {
	job, err := s.repo.GetJob(jobID, userID)
	if err != nil {
		return ErrDeepJobNotFound
	}
	switch job.Status {
	case models.DeepJobStatusQueued:
		if err := s.repo.RequestCancel(jobID, userID); err != nil {
			return err
		}
		_, err := s.repo.SettleTerminal(jobID, []models.DeepJobStatus{models.DeepJobStatusQueued}, models.DeepJobStatusCancelled, "", "", "", "")
		if err != nil {
			return err
		}
		s.broker.Publish(jobID)
		return s.DeleteHintArtifacts(jobID)
	case models.DeepJobStatusRunning:
		if err := s.repo.RequestCancel(jobID, userID); err != nil {
			return err
		}
		s.cancelMu.Lock()
		cancel, ok := s.cancels[jobID]
		s.cancelMu.Unlock()
		if ok {
			cancel()
		}
		return nil
	default:
		return ErrDeepJobNotCancellable
	}
}

func (s *DeepIdentificationService) registerCancel(jobID uint, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	s.cancels[jobID] = cancel
}

func (s *DeepIdentificationService) unregisterCancel(jobID uint) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	delete(s.cancels, jobID)
}

// StartWorkers launches a bounded pool of worker goroutines
// (SettingDeepIdentificationWorkerCount) that each loop:
// ClaimNextQueuedJob -> heartbeat ticker -> run pipeline -> SettleTerminal.
// Workers exit when ctx is cancelled.
func (s *DeepIdentificationService) StartWorkers(ctx context.Context) {
	settings := s.settingsSvc.GetDeepIdentificationSettings()
	workerCount := settings.WorkerCount
	if workerCount < 1 {
		workerCount = 1
	}
	for i := 0; i < workerCount; i++ {
		workerID := fmt.Sprintf("worker-%d", i)
		go s.workerLoop(ctx, workerID)
	}
}

func (s *DeepIdentificationService) workerLoop(ctx context.Context, workerID string) {
	ticker := time.NewTicker(deepJobPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-ticker.C:
		}
		for {
			s.intakeMu.RLock()
			job, claimed, err := s.repo.ClaimNextQueuedJob(workerID)
			s.intakeMu.RUnlock()
			if err != nil {
				// A failed claim never mutates job state: ClaimNextQueuedJob
				// runs its SELECT+UPDATE inside a single transaction, so on
				// any error (including SQLITE_BUSY from a competing writer)
				// the whole transaction rolls back and the job is left
				// exactly as it was - still status=queued. Nothing is lost:
				// this worker (or another) retries it on the next wake/tick
				// (deepJobPollInterval, 25ms) without janitor involvement.
				// With busy_timeout now set (database.Connect), SQLite waits
				// out a competing writer instead of failing immediately, so
				// this branch should be rare; treat it as a transient,
				// self-healing condition rather than an operator page.
				if s.logger != nil {
					s.logger.Warn("deep-identification", "worker %s failed to claim job (will retry): %v", workerID, err)
				}
				break
			}
			if !claimed {
				break
			}
			s.runJob(ctx, job)
			if ctx.Err() != nil {
				return
			}
		}
	}
}

// runJob executes a single claimed job's pipeline run under a hard timeout,
// heartbeats while it runs, and performs exactly one SettleTerminal call
// regardless of how it ends (success/partial/failure/timeout/cancel).
func (s *DeepIdentificationService) runJob(parent context.Context, job *models.DeepIdentificationJob) {
	settings := s.settingsSvc.GetDeepIdentificationSettings()

	jobCtx, cancel := context.WithCancel(parent)
	s.registerCancel(job.ID, cancel)
	defer func() {
		s.unregisterCancel(job.ID)
		cancel()
	}()

	// Close the narrow race where a cancel request lands between
	// ClaimNextQueuedJob and registerCancel above: if it's already recorded,
	// honor it immediately rather than starting the pipeline.
	if fresh, err := s.repo.GetJob(job.ID, job.UserID); err == nil && fresh.CancelRequestedAt != nil {
		cancel()
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(jobCtx, settings.HardTimeout)
	defer cancelTimeout()

	heartbeatCtx, stopHeartbeat := context.WithCancel(jobCtx)
	defer stopHeartbeat()
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				_ = s.repo.Heartbeat(job.ID)
			}
		}
	}()

	result, runErr := s.pipelineRunner().Run(timeoutCtx, job)
	stopHeartbeat()
	<-heartbeatDone

	var newStatus models.DeepJobStatus
	var reportJSON, proposalJSON, failureCode, failureMessage string
	switch {
	case jobCtx.Err() == context.Canceled:
		newStatus = models.DeepJobStatusCancelled
	case timeoutCtx.Err() == context.DeadlineExceeded:
		newStatus = models.DeepJobStatusFailed
		failureCode = "timeout"
		failureMessage = "The identification pipeline did not finish within the allotted time."
	case runErr != nil:
		newStatus = models.DeepJobStatusFailed
		failureCode = "agent_unavailable"
		failureMessage = "The identification pipeline could not complete."
	case result != nil && result.Partial:
		newStatus = models.DeepJobStatusPartial
		reportJSON = result.ReportJSON
		proposalJSON = result.ProposalJSON
		failureCode = result.FailureCode
		failureMessage = result.FailureMessage
	default:
		newStatus = models.DeepJobStatusCompleted
		if result != nil {
			reportJSON = result.ReportJSON
			proposalJSON = result.ProposalJSON
		}
	}

	won, err := s.repo.SettleTerminal(job.ID, []models.DeepJobStatus{models.DeepJobStatusRunning}, newStatus, reportJSON, proposalJSON, failureCode, failureMessage)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to settle job %d: %v", job.ID, err)
		}
	} else if won && s.logger != nil {
		durationMS := int64(0)
		if job.StartedAt != nil {
			durationMS = time.Since(*job.StartedAt).Milliseconds()
		}
		s.logger.Info("deep-identification", "job %d user %d settled status=%s duration_ms=%d", job.ID, job.UserID, newStatus, durationMS)
	}
	s.broker.Publish(job.ID)
	if s.providerBudgets != nil {
		s.providerBudgets.Reset(job.ID)
	}
	if s.internalTokenSvc != nil {
		s.internalTokenSvc.RevokeJob(job.ID)
	}
	if err := s.DeleteHintArtifacts(job.ID); err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to clean up hint artifacts for job %d: %v", job.ID, err)
		}
	}
}

// StartJanitor delegates to the janitor/retention seam (T103); see
// deepIdentificationJanitor.StartJanitor for the full behavior contract
// (FR-012/FR-017/FR-034).
func (s *DeepIdentificationService) StartJanitor(ctx context.Context) {
	s.janitor.StartJanitor(ctx)
}

// RecoverStaleAndSweepHintsForTest exposes the janitor's stale-job
// recovery + hint-sweep sweep for handler-package tests (T105) that need
// to trigger a restart-recovery pass without waiting for StartJanitor's
// ticker, since deep_identification_sse_test.go lives outside this
// package and the janitor's recovery method is unexported.
func (s *DeepIdentificationService) RecoverStaleAndSweepHintsForTest() {
	s.janitor.recoverStaleAndSweepHints()
}

// FingerprintInput is the normalized set of inputs hashed into a job's
// InputFingerprint (data-model.md §2.3).
type FingerprintInput struct {
	UserID             uint
	CoinID             uint // 0 for new-intake jobs
	ObverseHash        string
	ReverseHash        string
	HintHashes         []string
	Notes              string
	RequestedProviders []string
}

// ComputeInputFingerprint implements the sha256 formula from
// data-model.md §2.3:
//
//	sha256("v1" | user_id | coin_id_or_0 | obverse_hash | reverse_hash |
//	       sorted(hint_hashes) | sha256(normalized_notes) | sorted(providers))
func ComputeInputFingerprint(in FingerprintInput) string {
	hints := append([]string(nil), in.HintHashes...)
	sort.Strings(hints)
	providers := append([]string(nil), in.RequestedProviders...)
	sort.Strings(providers)

	normalizedNotes := strings.TrimSpace(in.Notes)
	notesHash := sha256.Sum256([]byte(normalizedNotes))

	var b strings.Builder
	b.WriteString("v1|")
	b.WriteString(strconv.FormatUint(uint64(in.UserID), 10))
	b.WriteString("|")
	b.WriteString(strconv.FormatUint(uint64(in.CoinID), 10))
	b.WriteString("|")
	b.WriteString(in.ObverseHash)
	b.WriteString("|")
	b.WriteString(in.ReverseHash)
	b.WriteString("|")
	b.WriteString(strings.Join(hints, ","))
	b.WriteString("|")
	b.WriteString(hex.EncodeToString(notesHash[:]))
	b.WriteString("|")
	b.WriteString(strings.Join(providers, ","))

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func countArtifactRoles(artifacts []models.DeepIdentificationArtifact) (obverse, reverse, hint int) {
	for _, a := range artifacts {
		switch a.Role {
		case models.DeepArtifactRoleObverse:
			obverse++
		case models.DeepArtifactRoleReverse:
			reverse++
		case models.DeepArtifactRoleHint:
			hint++
		}
	}
	return
}
