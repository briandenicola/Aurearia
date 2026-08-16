package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
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
	imageSvc    *ImageService
	settingsSvc *SettingsService
	logger      *Logger
	uploadDir   string

	runnerMu sync.RWMutex
	runner   DeepPipelineRunner

	// intakeMu prevents workers from claiming a queued intake job before its
	// obverse/reverse artifacts have been persisted.
	intakeMu sync.RWMutex

	cancelMu sync.Mutex
	cancels  map[uint]context.CancelFunc

	broker *DeepIdentificationBroker

	wakeMu sync.Mutex
	wake   chan struct{}
}

// NewDeepIdentificationService constructs the service, following the
// repo -> service -> handler DI pattern used elsewhere (main.go:246-249).
func NewDeepIdentificationService(repo *repository.DeepIdentificationRepository, imageRepo *repository.ImageRepository, imageSvc *ImageService, settingsSvc *SettingsService, logger *Logger, uploadDir string) *DeepIdentificationService {
	return &DeepIdentificationService{
		repo:        repo,
		imageRepo:   imageRepo,
		imageSvc:    imageSvc,
		settingsSvc: settingsSvc,
		logger:      logger,
		uploadDir:   uploadDir,
		runner:      noopPipelineRunner{},
		cancels:     make(map[uint]context.CancelFunc),
		broker:      NewDeepIdentificationBroker(),
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

// ValidateAndSaveArtifact validates an uploaded image (allowlisted type,
// magic-byte match, size cap - reusing services.ValidateImageData /
// NormalizeImageExt / MaxImageUploadBytes per FR-005) and, if valid, saves
// it to the job's artifact directory and creates the artifact row.
//
// Enforces: at most one obverse, at most one reverse, at most
// MaxDeepIdentificationHintArtifacts hint artifacts per job.
func (s *DeepIdentificationService) ValidateAndSaveArtifact(jobID, userID uint, role models.DeepArtifactRole, filename string, fileData []byte) (*models.DeepIdentificationArtifact, error) {
	switch role {
	case models.DeepArtifactRoleObverse, models.DeepArtifactRoleReverse, models.DeepArtifactRoleHint:
	default:
		return nil, ErrDeepArtifactRoleInvalid
	}

	if len(fileData) == 0 {
		return nil, ErrDeepArtifactMissingUpload
	}
	if err := ValidateImageData(fileData); err != nil {
		return nil, err
	}
	ext, err := NormalizeImageExt(filepath.Ext(filename))
	if err != nil {
		return nil, err
	}

	existing, err := s.listArtifacts(jobID)
	if err != nil {
		return nil, err
	}
	obverseCount, reverseCount, hintCount := countArtifactRoles(existing)
	switch role {
	case models.DeepArtifactRoleObverse:
		if obverseCount > 0 {
			return nil, ErrDeepArtifactRoleExists
		}
	case models.DeepArtifactRoleReverse:
		if reverseCount > 0 {
			return nil, ErrDeepArtifactRoleExists
		}
	case models.DeepArtifactRoleHint:
		if hintCount >= MaxDeepIdentificationHintArtifacts {
			return nil, ErrDeepArtifactHintLimit
		}
	}

	dir := s.imageSvc.DeepJobArtifactDir(jobID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to prepare artifact directory: %w", err)
	}
	seq := len(existing) + 1
	filePath := filepath.Join(dir, fmt.Sprintf("%d-%s%s", seq, role, ext))
	if err := os.WriteFile(filePath, fileData, 0o644); err != nil {
		return nil, fmt.Errorf("failed to save artifact: %w", err)
	}

	hash := sha256.Sum256(fileData)
	artifact := &models.DeepIdentificationArtifact{
		JobID:       jobID,
		UserID:      userID,
		Role:        role,
		Origin:      models.DeepArtifactOriginUploaded,
		FilePath:    filePath,
		ContentHash: hex.EncodeToString(hash[:]),
		ByteSize:    int64(len(fileData)),
		MimeType:    detectMimeType(fileData),
		Ephemeral:   role == models.DeepArtifactRoleHint,
	}
	if err := s.createArtifact(artifact); err != nil {
		os.Remove(filePath)
		return nil, err
	}
	return artifact, nil
}

// ReuseSavedCoinImage creates an artifact row referencing an existing saved
// coin image without copying bytes (FR-003). The fingerprint input is
// derived from the stored file's path, size, and mtime rather than
// re-reading the (potentially large) file - if the underlying file changes
// later, a subsequent job attempt sees a different fingerprint (retry-after-
// change edge case, data-model.md §2.3).
func (s *DeepIdentificationService) ReuseSavedCoinImage(jobID, userID, coinID uint, role models.DeepArtifactRole, sourceCoinImageID uint) (*models.DeepIdentificationArtifact, error) {
	switch role {
	case models.DeepArtifactRoleObverse, models.DeepArtifactRoleReverse:
	default:
		return nil, ErrDeepArtifactRoleInvalid
	}

	if _, err := s.imageRepo.FindCoinByOwner(coinID, userID); err != nil {
		return nil, ErrDeepArtifactMissingCoin
	}
	image, err := s.imageRepo.FindImage(sourceCoinImageID, coinID)
	if err != nil {
		return nil, ErrDeepArtifactMissingImage
	}

	existing, err := s.listArtifacts(jobID)
	if err != nil {
		return nil, err
	}
	obverseCount, reverseCount, _ := countArtifactRoles(existing)
	if role == models.DeepArtifactRoleObverse && obverseCount > 0 {
		return nil, ErrDeepArtifactRoleExists
	}
	if role == models.DeepArtifactRoleReverse && reverseCount > 0 {
		return nil, ErrDeepArtifactRoleExists
	}

	hashHex, size, err := s.savedImageFingerprintHash(image)
	if err != nil {
		return nil, err
	}

	artifact := &models.DeepIdentificationArtifact{
		JobID:             jobID,
		UserID:            userID,
		Role:              role,
		Origin:            models.DeepArtifactOriginSavedCoinImage,
		SourceCoinImageID: &sourceCoinImageID,
		FilePath:          "",
		ContentHash:       hashHex,
		ByteSize:          size,
		Ephemeral:         false,
	}
	if err := s.createArtifact(artifact); err != nil {
		return nil, err
	}
	return artifact, nil
}

// savedImageFingerprintHash computes the fingerprint-input hash for an
// existing saved coin image from its stored file's path, size, and mtime
// (data-model.md §2.3) - shared by ReuseSavedCoinImage and by the job-create
// orchestration's up-front fingerprint computation, so both agree on the
// exact same hash for the same image.
func (s *DeepIdentificationService) savedImageFingerprintHash(image *models.CoinImage) (hashHex string, size int64, err error) {
	fullPath := filepath.Join(s.uploadDir, image.FilePath)
	var mtime time.Time
	if info, statErr := os.Stat(fullPath); statErr == nil {
		size = info.Size()
		mtime = info.ModTime()
	}
	fingerprintSource := fmt.Sprintf("%s|%d|%d", image.FilePath, size, mtime.UnixNano())
	hash := sha256.Sum256([]byte(fingerprintSource))
	return hex.EncodeToString(hash[:]), size, nil
}

// DeleteHintArtifacts deletes (files + DeletedAt stamp) every not-yet-deleted
// hint artifact for a job. Tolerant of an already-missing file. Called from
// every terminal path (completed/partial/failed/cancelled) and the
// janitor's restart sweep (FR-030, SC-004).
func (s *DeepIdentificationService) DeleteHintArtifacts(jobID uint) error {
	return s.deleteArtifacts(jobID, true)
}

// DeleteJobArtifacts deletes every not-yet-deleted artifact (hint and
// coin-face) for a job, used by the result-retention janitor sweep
// (FR-034, data-model.md §9). Idempotent - calling it twice is a no-op the
// second time.
func (s *DeepIdentificationService) DeleteJobArtifacts(jobID uint) error {
	return s.deleteArtifacts(jobID, false)
}

func (s *DeepIdentificationService) deleteArtifacts(jobID uint, hintOnly bool) error {
	artifacts, err := s.listArtifacts(jobID)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range artifacts {
		a := artifacts[i]
		if a.DeletedAt != nil {
			continue
		}
		if hintOnly && a.Role != models.DeepArtifactRoleHint {
			continue
		}
		if a.FilePath != "" {
			// Tolerant of an already-missing file (crash/retry edge case).
			if err := os.Remove(a.FilePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete artifact file: %w", err)
			}
		}
		if err := s.markArtifactDeleted(a.ID, now); err != nil {
			return err
		}
	}
	return nil
}

// --- Phase 4: job orchestration (worker pool, cancel, timeout, janitor) ---

// StartJob enqueues a new job or, per FR-007, returns the existing
// active (queued/running) job for the same (user, fingerprint) pair, or the
// user's existing active job if they are already at their concurrency
// limit. Returns ErrDeepJobQueueFull if the global queue depth is exceeded.
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
		// Different fingerprint than the user's existing active job(s):
		// still bounded by the per-user limit (FR-007), so surface the
		// most recent active job rather than silently enqueueing another.
		jobs, listErr := s.repo.ListJobs(job.UserID, repository.DeepJobListFilters{})
		if listErr == nil {
			for i := range jobs {
				if isDeepJobActive(jobs[i].Status) {
					return &jobs[i], true, nil
				}
			}
		}
		return nil, false, ErrDeepJobQueueFull
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
	hashHex, _, err := s.savedImageFingerprintHash(image)
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

	artifacts, err := s.listArtifacts(sourceJobID)
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
				if s.logger != nil {
					s.logger.Error("deep-identification", "worker %s failed to claim job: %v", workerID, err)
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

	if _, err := s.repo.SettleTerminal(job.ID, []models.DeepJobStatus{models.DeepJobStatusRunning}, newStatus, reportJSON, proposalJSON, failureCode, failureMessage); err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to settle job %d: %v", job.ID, err)
		}
	}
	s.broker.Publish(job.ID)
	if err := s.DeleteHintArtifacts(job.ID); err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to clean up hint artifacts for job %d: %v", job.ID, err)
		}
	}
}

// StartJanitor runs the retention/recovery sweep loop (FR-012/FR-017/
// FR-034): on boot and every 60s it recovers stale running jobs; hourly it
// prunes events past the retention window and expires job/artifact rows
// past ExpiresAt. It also sweeps hint artifacts left un-deleted by a crash
// (T040 defensive backstop), independent of the terminal-hook path.
func (s *DeepIdentificationService) StartJanitor(ctx context.Context) {
	s.recoverStaleAndSweepHints()

	staleTicker := time.NewTicker(60 * time.Second)
	retentionTicker := time.NewTicker(time.Hour)
	go func() {
		defer staleTicker.Stop()
		defer retentionTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-staleTicker.C:
				s.recoverStaleAndSweepHints()
			case <-retentionTicker.C:
				s.runRetentionSweep()
			}
		}
	}()
}

// RecoverStaleAndSweepHintsForTest exposes the janitor's stale-job
// recovery + hint-sweep sweep for handler-package tests (T105) that need
// to trigger a restart-recovery pass without waiting for StartJanitor's
// ticker, since deep_identification_sse_test.go lives outside this
// package and recoverStaleAndSweepHints is unexported.
func (s *DeepIdentificationService) RecoverStaleAndSweepHintsForTest() {
	s.recoverStaleAndSweepHints()
}

func (s *DeepIdentificationService) recoverStaleAndSweepHints() {
	settings := s.settingsSvc.GetDeepIdentificationSettings()
	staleAfter := settings.HardTimeout + 2*time.Minute
	recoveredIDs, err := s.repo.RecoverStaleJobs(staleAfter)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to recover stale jobs: %v", err)
		}
	}
	for _, id := range recoveredIDs {
		s.broker.Publish(id)
		if err := s.DeleteHintArtifacts(id); err != nil && s.logger != nil {
			s.logger.Error("deep-identification", "failed to clean up hints for recovered job %d: %v", id, err)
		}
	}

	// Defensive backstop: sweep hint artifacts for any terminal job that a
	// crash left un-cleaned (independent of the two hooks above).
	jobIDs, err := s.repo.ListJobIDsWithUndeletedHintArtifacts()
	if err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to list jobs with undeleted hints: %v", err)
		}
		return
	}
	for _, id := range jobIDs {
		if err := s.DeleteHintArtifacts(id); err != nil && s.logger != nil {
			s.logger.Error("deep-identification", "failed to sweep hints for job %d: %v", id, err)
		}
	}
}

func (s *DeepIdentificationService) runRetentionSweep() {
	settings := s.settingsSvc.GetDeepIdentificationSettings()
	cutoff := time.Now().Add(-settings.EventRetention)
	if err := s.repo.PruneEventsBefore(cutoff); err != nil && s.logger != nil {
		s.logger.Error("deep-identification", "failed to prune events: %v", err)
	}
	expiredIDs, err := s.repo.ListExpiredJobIDs(time.Now())
	if err != nil {
		if s.logger != nil {
			s.logger.Error("deep-identification", "failed to list expired jobs: %v", err)
		}
		return
	}
	for _, id := range expiredIDs {
		if err := s.DeleteJobArtifacts(id); err != nil && s.logger != nil {
			s.logger.Error("deep-identification", "failed to delete artifacts for expired job %d: %v", id, err)
		}
	}
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

func detectMimeType(data []byte) string {
	// Mirrors ValidateImageData's http.DetectContentType-based allowlist so
	// the stored MimeType always matches what was actually validated.
	if len(data) == 0 {
		return ""
	}
	return http.DetectContentType(data)
}

// --- thin persistence helpers (kept private; repository has no direct
// artifact CRUD yet beyond what the service needs) ---

func (s *DeepIdentificationService) listArtifacts(jobID uint) ([]models.DeepIdentificationArtifact, error) {
	return s.repo.ListArtifacts(jobID)
}

func (s *DeepIdentificationService) createArtifact(a *models.DeepIdentificationArtifact) error {
	return s.repo.CreateArtifact(a)
}

func (s *DeepIdentificationService) markArtifactDeleted(id uint, when time.Time) error {
	return s.repo.MarkArtifactDeleted(id, when)
}
