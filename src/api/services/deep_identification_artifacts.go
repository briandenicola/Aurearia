package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// deepIdentificationArtifactStore owns artifact validation/storage/reuse/
// deletion for deep identification jobs (data-model.md §2/§5). It is the
// artifact-management seam split out of DeepIdentificationService (T103):
// every method here is self-contained (repo + imageRepo + imageSvc +
// uploadDir + shared runtime metrics), with no dependency on job-lifecycle
// or worker-pool state, so it can be constructed and tested in isolation.
type deepIdentificationArtifactStore struct {
	repo      *repository.DeepIdentificationRepository
	imageRepo *repository.ImageRepository
	imageSvc  *ImageService
	uploadDir string
	metrics   *deepIdentificationRuntimeMetrics
}

// newDeepIdentificationArtifactStore constructs the artifact store,
// following the repo -> service -> handler DI pattern (Principle I).
func newDeepIdentificationArtifactStore(
	repo *repository.DeepIdentificationRepository,
	imageRepo *repository.ImageRepository,
	imageSvc *ImageService,
	uploadDir string,
	metrics *deepIdentificationRuntimeMetrics,
) *deepIdentificationArtifactStore {
	return &deepIdentificationArtifactStore{
		repo:      repo,
		imageRepo: imageRepo,
		imageSvc:  imageSvc,
		uploadDir: uploadDir,
		metrics:   metrics,
	}
}

// ValidateAndSaveArtifact validates an uploaded image (allowlisted type,
// magic-byte match, size cap - reusing services.ValidateImageData /
// NormalizeImageExt / MaxImageUploadBytes per FR-005) and, if valid, saves
// it to the job's artifact directory and creates the artifact row.
//
// Enforces: at most one obverse, at most one reverse, at most
// MaxDeepIdentificationHintArtifacts hint artifacts per job.
func (a *deepIdentificationArtifactStore) ValidateAndSaveArtifact(jobID, userID uint, role models.DeepArtifactRole, filename string, fileData []byte) (*models.DeepIdentificationArtifact, error) {
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

	existing, err := a.listArtifacts(jobID)
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

	dir := a.imageSvc.DeepJobArtifactDir(jobID)
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
	if err := a.createArtifact(artifact); err != nil {
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
func (a *deepIdentificationArtifactStore) ReuseSavedCoinImage(jobID, userID, coinID uint, role models.DeepArtifactRole, sourceCoinImageID uint) (*models.DeepIdentificationArtifact, error) {
	switch role {
	case models.DeepArtifactRoleObverse, models.DeepArtifactRoleReverse:
	default:
		return nil, ErrDeepArtifactRoleInvalid
	}

	if _, err := a.imageRepo.FindCoinByOwner(coinID, userID); err != nil {
		return nil, ErrDeepArtifactMissingCoin
	}
	image, err := a.imageRepo.FindImage(sourceCoinImageID, coinID)
	if err != nil {
		return nil, ErrDeepArtifactMissingImage
	}

	existing, err := a.listArtifacts(jobID)
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

	hashHex, size, err := a.savedImageFingerprintHash(image)
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
	if err := a.createArtifact(artifact); err != nil {
		return nil, err
	}
	return artifact, nil
}

// savedImageFingerprintHash computes the fingerprint-input hash for an
// existing saved coin image from its stored file's path, size, and mtime
// (data-model.md §2.3) - shared by ReuseSavedCoinImage and by the job-create
// orchestration's up-front fingerprint computation, so both agree on the
// exact same hash for the same image.
func (a *deepIdentificationArtifactStore) savedImageFingerprintHash(image *models.CoinImage) (hashHex string, size int64, err error) {
	fullPath := filepath.Join(a.uploadDir, image.FilePath)
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
func (a *deepIdentificationArtifactStore) DeleteHintArtifacts(jobID uint) error {
	err := a.deleteArtifacts(jobID, true)
	if err != nil {
		a.metrics.hintDeletionFailure.Add(1)
	} else {
		a.metrics.hintDeletionSuccess.Add(1)
	}
	return err
}

// DeleteJobArtifacts deletes every not-yet-deleted artifact (hint and
// coin-face) for a job, used by the result-retention janitor sweep
// (FR-034, data-model.md §9). Idempotent - calling it twice is a no-op the
// second time.
func (a *deepIdentificationArtifactStore) DeleteJobArtifacts(jobID uint) error {
	return a.deleteArtifacts(jobID, false)
}

func (a *deepIdentificationArtifactStore) deleteArtifacts(jobID uint, hintOnly bool) error {
	artifacts, err := a.listArtifacts(jobID)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range artifacts {
		art := artifacts[i]
		if art.DeletedAt != nil {
			continue
		}
		if hintOnly && art.Role != models.DeepArtifactRoleHint {
			continue
		}
		if art.FilePath != "" {
			// Tolerant of an already-missing file (crash/retry edge case).
			if err := os.Remove(art.FilePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete artifact file: %w", err)
			}
		}
		if err := a.markArtifactDeleted(art.ID, now); err != nil {
			return err
		}
	}
	return nil
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

func (a *deepIdentificationArtifactStore) listArtifacts(jobID uint) ([]models.DeepIdentificationArtifact, error) {
	return a.repo.ListArtifacts(jobID)
}

func (a *deepIdentificationArtifactStore) createArtifact(artifact *models.DeepIdentificationArtifact) error {
	return a.repo.CreateArtifact(artifact)
}

func (a *deepIdentificationArtifactStore) markArtifactDeleted(id uint, when time.Time) error {
	return a.repo.MarkArtifactDeleted(id, when)
}
