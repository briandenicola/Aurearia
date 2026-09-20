package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrDeepHandoffConflict = errors.New("handoff_idempotency_conflict")
	ErrDeepHandoffChanged  = errors.New("target_changed")
	ErrDeepHandoffState    = errors.New("deep handoff state conflict")
)

type DeepAnalysisHandoffService struct {
	copilotRepo *repository.CoinCopilotRepository
	deepRepo    *repository.DeepIdentificationRepository
	coinRepo    *repository.CoinRepository
	draftRepo   *repository.QuickCaptureRepository
	deepSvc     *DeepIdentificationService
	settingsSvc *SettingsService
	uploadDir   string
}

func NewDeepAnalysisHandoffService(
	copilotRepo *repository.CoinCopilotRepository,
	deepRepo *repository.DeepIdentificationRepository,
	coinRepo *repository.CoinRepository,
	draftRepo *repository.QuickCaptureRepository,
	deepSvc *DeepIdentificationService,
	settingsSvc *SettingsService,
	uploadDir string,
) *DeepAnalysisHandoffService {
	return &DeepAnalysisHandoffService{
		copilotRepo: copilotRepo, deepRepo: deepRepo, coinRepo: coinRepo,
		draftRepo: draftRepo, deepSvc: deepSvc, settingsSvc: settingsSvc,
		uploadDir: uploadDir,
	}
}

func (s *DeepAnalysisHandoffService) Execute(
	claims *CopilotExecutionClaims,
	request DeepAnalysisHandoffRequest,
) (DeepAnalysisHandoffResult, error) {
	if claims == nil || ValidateDeepAnalysisHandoffRequest(request) != nil {
		return DeepAnalysisHandoffResult{}, ErrCopilotInvalidRequest
	}
	run, err := s.copilotRepo.GetCurrentExecution(claims.RunID, claims.ExecutionID, claims.UserID)
	if err != nil || run.CheckpointVersion != request.ExpectedCheckpointVersion {
		return DeepAnalysisHandoffResult{}, ErrDeepHandoffState
	}
	if request.Operation == "status" {
		return s.status(claims, request, run)
	}
	copilotSettings := s.settingsSvc.GetCoinCopilotSettings()
	deepSettings := s.settingsSvc.GetDeepIdentificationSettings()
	if !copilotSettings.Enabled {
		return unavailableHandoff(request.Operation, "copilot_disabled"), nil
	}
	if !copilotSettings.AttributionEnabled {
		return unavailableHandoff(request.Operation, "attribution_disabled"), nil
	}
	if !deepSettings.Enabled {
		return unavailableHandoff(request.Operation, "deep_disabled"), nil
	}

	resolved, outcome, err := s.resolveTarget(claims.UserID, request.Operation, *request.Target)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
		}
		return DeepAnalysisHandoffResult{}, err
	}
	if outcome != nil {
		return *outcome, nil
	}

	providers, generation := s.settingsSvc.DeepProviderConfiguration()
	contextDigest, contextVersion := deepAnalysisBoundedContext(resolved.context)
	snapshot := DeepAnalysisTargetSnapshotV2{
		SchemaVersion: 2, OwnerID: claims.UserID,
		TargetKind: request.Target.Type, TargetID: request.Target.ID,
		TargetState: resolved.state, TargetVersion: resolved.targetVersion,
		Obverse:              resolved.obverse.snapshotFace,
		Reverse:              resolved.reverse.snapshotFace,
		BoundedContextSHA256: contextDigest, BoundedContextVersion: contextVersion,
		EffectiveProviders: providers, ProviderConfigurationGeneration: generation,
	}
	snapshotDigest, err := ComputeDeepAnalysisTargetSnapshot(snapshot)
	if err != nil {
		return DeepAnalysisHandoffResult{}, err
	}

	var priorID *uint
	if request.Operation == "rerun" {
		prior, err := s.deepRepo.GetJob(*request.JobID, claims.UserID)
		if err != nil || !deepJobMatchesTarget(prior, request.Target) {
			return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
		}
		value := prior.ID
		priorID = &value
	}

	inputFingerprint := snapshotDigest
	if priorID != nil {
		inputFingerprint = handoffSHA256Hex(fmt.Sprintf("rerun|%s|%d", snapshotDigest, *priorID))
	}
	appDigest := repository.DigestCoinCopilotAppContext(run.AppContextJSON)
	requestFingerprint := handoffSHA256Hex(strings.Join([]string{
		request.Operation, request.Target.Type, fmt.Sprint(request.Target.ID),
		fmt.Sprint(valueOrZero(priorID)), claims.ExecutionID,
		fmt.Sprint(request.ExpectedCheckpointVersion), appDigest, snapshotDigest,
	}, "|"))
	handoff := &models.CoinCopilotDeepHandoff{
		UserID: claims.UserID, RunID: claims.RunID, ExecutionID: claims.ExecutionID,
		HandoffKeyHash:            handoffSHA256Hex(request.HandoffIdempotencyKey),
		RequestFingerprint:        requestFingerprint,
		ExpectedCheckpointVersion: request.ExpectedCheckpointVersion,
		AppContextDigest:          appDigest,
		Operation:                 models.CoinCopilotDeepHandoffOperation(request.Operation),
		TargetKind:                models.CoinCopilotDeepHandoffTargetKind(request.Target.Type),
		TargetID:                  request.Target.ID, PriorJobID: priorID,
		TargetSnapshotFingerprint: snapshotDigest,
	}
	job := &models.DeepIdentificationJob{
		UserID: claims.UserID, CoinID: resolved.coinID, SourceDraftID: resolved.draftID,
		Source: resolved.source, Status: models.DeepJobStatusQueued,
		InputFingerprint: inputFingerprint, Notes: resolved.context,
		RequestedProviders: strings.Join(providers, ","),
		ExpiresAt:          time.Now().UTC().Add(deepSettings.ResultRetention),
	}

	artifacts, cleanup, err := s.stageFaces(claims.UserID, resolved)
	if err != nil {
		return DeepAnalysisHandoffResult{}, err
	}
	keepFiles := false
	defer func() {
		if !keepFiles {
			cleanup()
		}
	}()
	admitted, selected, replayed, err := s.copilotRepo.AdmitDeepHandoff(repository.DeepHandoffAdmission{
		Handoff: handoff, Job: job, Artifacts: artifacts,
		Target:           resolved.token,
		ProviderSettings: s.settingsSvc.DeepProviderSettingSnapshot(),
		ProviderDefaults: s.settingsSvc.GetSettingDefaults(),
		MaxActivePerUser: deepSettings.MaxActivePerUser, QueueDepth: deepSettings.QueueDepth,
		AfterCommit: s.deepSvc.NotifyHandoffCommitted,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrCopilotDeepHandoffConflict):
			return DeepAnalysisHandoffResult{}, ErrDeepHandoffConflict
		case errors.Is(err, repository.ErrCopilotOwnerCapacity):
			return unavailableHandoff(request.Operation, "job_at_capacity"), nil
		case errors.Is(err, repository.ErrCopilotQueueCapacity):
			return unavailableHandoff(request.Operation, "queue_full"), nil
		default:
			return DeepAnalysisHandoffResult{}, ErrDeepHandoffState
		}
	}
	keepFiles = admitted.AdmissionOutcome == models.CoinCopilotDeepHandoffOutcomeCreated && !replayed
	return handoffResult(request.Operation, resolved.label, request.Target, selected, admitted, replayed), nil
}

func (s *DeepAnalysisHandoffService) status(
	claims *CopilotExecutionClaims,
	request DeepAnalysisHandoffRequest,
	run *models.CoinCopilotRun,
) (DeepAnalysisHandoffResult, error) {
	handoff, job, bindingErr := s.copilotRepo.FindDeepHandoffByJob(
		claims.RunID, claims.UserID, *request.JobID,
	)
	if bindingErr == nil {
		if !models.IsValidCoinCopilotDeepHandoff(handoff) ||
			handoff.ExecutionID != claims.ExecutionID ||
			handoff.ExpectedCheckpointVersion > run.CheckpointVersion ||
			handoff.AppContextDigest != repository.DigestCoinCopilotAppContext(run.AppContextJSON) ||
			!deepJobMatchesTarget(job, &DeepAnalysisHandoffTarget{
				Type: string(handoff.TargetKind), ID: handoff.TargetID,
			}) {
			return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
		}
		target := &DeepAnalysisHandoffTarget{Type: string(handoff.TargetKind), ID: handoff.TargetID}
		label, available := s.statusTargetLabel(claims.UserID, *target)
		if !available {
			return DeepAnalysisHandoffResult{Outcome: "target_unavailable"}, nil
		}
		result, err := BuildDeepAnalysisHandoffResultFromJob("status", target, label, job)
		if err != nil {
			return retryAvailableStatus(target, label, job), nil
		}
		result.InputDigest = handoff.TargetSnapshotFingerprint
		return result, nil
	}

	job, err := s.deepRepo.GetJob(*request.JobID, claims.UserID)
	if err != nil || !models.IsValidDeepJobSourceBinding(job) {
		return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
	}
	var target DeepAnalysisHandoffTarget
	switch job.Source {
	case models.DeepJobSourceSavedCoin:
		target = DeepAnalysisHandoffTarget{Type: "coin", ID: *job.CoinID}
	case models.DeepJobSourceCopilotDraft:
		target = DeepAnalysisHandoffTarget{Type: "draft", ID: *job.SourceDraftID}
	default:
		return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
	}
	label, available := s.statusTargetLabel(claims.UserID, target)
	if !available {
		return DeepAnalysisHandoffResult{Outcome: "not_eligible"}, nil
	}
	result, err := BuildDeepAnalysisHandoffResultFromJob("status", &target, label, job)
	if err != nil {
		return retryAvailableStatus(&target, label, job), nil
	}
	return result, nil
}

func (s *DeepAnalysisHandoffService) statusTargetLabel(
	userID uint,
	target DeepAnalysisHandoffTarget,
) (string, bool) {
	switch target.Type {
	case "coin":
		coin, err := s.coinRepo.FindByID(target.ID, userID)
		if err != nil {
			return "", false
		}
		return coin.Name, true
	case "draft":
		draft, err := s.draftRepo.GetDraftForOwner(target.ID, userID)
		if err != nil || draft.Status != models.QuickCaptureDraftStatusActive {
			return "", false
		}
		return draft.WorkingTitle, true
	default:
		return "", false
	}
}

func retryAvailableStatus(
	target *DeepAnalysisHandoffTarget,
	label string,
	job *models.DeepIdentificationJob,
) DeepAnalysisHandoffResult {
	reason := "result_missing"
	switch job.Status {
	case models.DeepJobStatusFailed:
		reason = "result_missing"
	case models.DeepJobStatusCancelled:
		reason = "cancelled"
	}
	result := DeepAnalysisHandoffResult{
		SchemaVersion: 1, Operation: "status", Outcome: "retry_available",
		Reason: handoffStringPointer(reason), InputDigest: job.InputFingerprint,
		ReviewURL: fmt.Sprintf("/deep-analysis/%d", job.ID),
		Job: &DeepAnalysisHandoffJob{
			ID: job.ID, Source: string(job.Source), Status: string(job.Status),
			Reused: true, CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339),
		},
	}
	result.Target = &struct {
		Type         string `json:"type"`
		ID           uint   `json:"id"`
		DisplayLabel string `json:"display_label"`
	}{Type: target.Type, ID: target.ID, DisplayLabel: label}
	if job.CompletedAt != nil {
		value := job.CompletedAt.UTC().Format(time.RFC3339)
		result.Job.CompletedAt = &value
	}
	return result
}

type resolvedHandoffFace struct {
	snapshotFace DeepAnalysisSnapshotFace
	filePath     string
	bytes        []byte
}

type resolvedHandoffTarget struct {
	label, state, targetVersion, context string
	source                               models.DeepJobSource
	coinID, draftID                      *uint
	obverse, reverse                     resolvedHandoffFace
	token                                repository.DeepHandoffTargetToken
}

func (s *DeepAnalysisHandoffService) resolveTarget(userID uint, operation string, target DeepAnalysisHandoffTarget) (resolvedHandoffTarget, *DeepAnalysisHandoffResult, error) {
	var resolved resolvedHandoffTarget
	var images []models.CoinImage
	if target.Type == "coin" {
		coin, err := s.coinRepo.FindByID(target.ID, userID)
		if err != nil {
			return resolved, nil, err
		}
		resolved.label, resolved.state, resolved.context = coin.Name, "active", coin.Notes
		resolved.targetVersion = coin.UpdatedAt.UTC().Format(time.RFC3339Nano)
		resolved.source = models.DeepJobSourceSavedCoin
		resolved.coinID = &coin.ID
		resolved.token = repository.DeepHandoffTargetToken{
			Kind: models.CoinCopilotDeepHandoffTargetCoin, ID: coin.ID, UserID: userID,
			State: "active", UpdatedAt: resolved.targetVersion, Context: coin.Notes,
		}
		images = coin.Images
	} else {
		draft, err := s.draftRepo.GetDraftForOwner(target.ID, userID)
		if err != nil {
			return resolved, nil, err
		}
		if draft.Status != models.QuickCaptureDraftStatusActive {
			return resolved, &DeepAnalysisHandoffResult{
				SchemaVersion: 1, Operation: operation, Outcome: "unavailable",
				Reason: handoffStringPointer("draft_inactive"),
			}, nil
		}
		resolved.label, resolved.state, resolved.context = draft.WorkingTitle, string(draft.Status), draft.Notes
		resolved.targetVersion = draft.UpdatedAt.UTC().Format(time.RFC3339Nano)
		resolved.source = models.DeepJobSourceCopilotDraft
		resolved.draftID = &draft.ID
		resolved.token = repository.DeepHandoffTargetToken{
			Kind: models.CoinCopilotDeepHandoffTargetDraft, ID: draft.ID, UserID: userID,
			State: string(draft.Status), UpdatedAt: resolved.targetVersion, Context: draft.Notes,
		}
		for _, image := range draft.Images {
			images = append(images, models.CoinImage{
				ID: image.ID, CoinID: draft.ID, FilePath: image.FilePath,
				ImageType: image.ImageType, IsPrimary: image.IsPrimary, CreatedAt: image.CreatedAt,
			})
		}
	}
	var obverse, reverse *models.CoinImage
	for index := range images {
		image := &images[index]
		if image.ImageType == models.ImageTypeObverse && (obverse == nil || image.ID > obverse.ID) {
			obverse = image
		}
		if image.ImageType == models.ImageTypeReverse && (reverse == nil || image.ID > reverse.ID) {
			reverse = image
		}
	}
	if obverse == nil || reverse == nil {
		reason := "missing_both"
		if obverse != nil {
			reason = "missing_reverse"
		} else if reverse != nil {
			reason = "missing_obverse"
		}
		return resolved, &DeepAnalysisHandoffResult{
			SchemaVersion: 1, Operation: operation, Outcome: "missing_images",
			Reason: handoffStringPointer(reason),
		}, nil
	}
	var err error
	resolved.obverse, err = s.readFace(*obverse)
	if err != nil {
		return resolved, nil, err
	}
	resolved.reverse, err = s.readFace(*reverse)
	if err != nil {
		return resolved, nil, err
	}
	if resolved.obverse.snapshotFace.ContentHash == resolved.reverse.snapshotFace.ContentHash {
		return resolved, &DeepAnalysisHandoffResult{
			SchemaVersion: 1, Operation: operation, Outcome: "missing_images",
			Reason: handoffStringPointer("duplicate_faces"),
		}, nil
	}
	resolved.token.Obverse = repository.DeepHandoffFaceToken{
		ID: obverse.ID, FilePath: obverse.FilePath,
		ContentPath: resolved.obverse.filePath, ContentHash: resolved.obverse.snapshotFace.ContentHash,
		CreatedAt: obverse.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	resolved.token.Reverse = repository.DeepHandoffFaceToken{
		ID: reverse.ID, FilePath: reverse.FilePath,
		ContentPath: resolved.reverse.filePath, ContentHash: resolved.reverse.snapshotFace.ContentHash,
		CreatedAt: reverse.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	return resolved, nil, nil
}

func (s *DeepAnalysisHandoffService) readFace(image models.CoinImage) (resolvedHandoffFace, error) {
	path := image.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.uploadDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return resolvedHandoffFace{}, err
	}
	if err := ValidateImageData(data); err != nil {
		return resolvedHandoffFace{}, ErrDeepInvalidInput
	}
	hash := handoffSHA256HexBytes(data)
	return resolvedHandoffFace{
		filePath: path, bytes: data,
		snapshotFace: DeepAnalysisSnapshotFace{
			RowID:       image.ID,
			Version:     image.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + image.FilePath,
			ContentHash: hash,
		},
	}, nil
}

func (s *DeepAnalysisHandoffService) stageFaces(userID uint, target resolvedHandoffTarget) ([]models.DeepIdentificationArtifact, func(), error) {
	base := filepath.Join(s.uploadDir, "deep-identification")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, func() {}, err
	}
	dir, err := os.MkdirTemp(base, "handoff-")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	faces := []struct {
		role models.DeepArtifactRole
		face resolvedHandoffFace
	}{
		{models.DeepArtifactRoleObverse, target.obverse},
		{models.DeepArtifactRoleReverse, target.reverse},
	}
	artifacts := make([]models.DeepIdentificationArtifact, 0, 2)
	for _, item := range faces {
		path := filepath.Join(dir, string(item.role)+filepath.Ext(item.face.filePath))
		if err := os.WriteFile(path, item.face.bytes, 0o600); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		artifacts = append(artifacts, models.DeepIdentificationArtifact{
			UserID: userID, Role: item.role, Origin: models.DeepArtifactOriginUploaded,
			FilePath: path, ContentHash: item.face.snapshotFace.ContentHash,
			ByteSize: int64(len(item.face.bytes)), MimeType: http.DetectContentType(item.face.bytes),
		})
	}
	return artifacts, cleanup, nil
}

func handoffResult(operation, label string, target *DeepAnalysisHandoffTarget, job *models.DeepIdentificationJob, handoff *models.CoinCopilotDeepHandoff, replayed bool) DeepAnalysisHandoffResult {
	outcome := "accepted"
	switch handoff.AdmissionOutcome {
	case models.CoinCopilotDeepHandoffOutcomeReusedActive:
		outcome = "reused_active"
	case models.CoinCopilotDeepHandoffOutcomeReusedResult:
		outcome = "reused_result"
	}
	if operation == "status" {
		outcome = "status"
	}
	result := DeepAnalysisHandoffResult{
		SchemaVersion: 1, Operation: operation, Outcome: outcome,
		InputDigest: handoff.TargetSnapshotFingerprint,
		ReviewURL:   fmt.Sprintf("/deep-analysis/%d", job.ID),
		Job: &DeepAnalysisHandoffJob{
			ID: job.ID, Source: string(job.Source), Status: string(job.Status),
			Reused:    replayed || outcome != "accepted",
			CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339),
		},
	}
	result.Target = &struct {
		Type         string `json:"type"`
		ID           uint   `json:"id"`
		DisplayLabel string `json:"display_label"`
	}{Type: target.Type, ID: target.ID, DisplayLabel: label}
	if job.CompletedAt != nil {
		value := job.CompletedAt.UTC().Format(time.RFC3339)
		result.Job.CompletedAt = &value
	}
	return result
}

func unavailableHandoff(operation, reason string) DeepAnalysisHandoffResult {
	return DeepAnalysisHandoffResult{
		SchemaVersion: 1, Operation: operation, Outcome: "unavailable",
		Reason: handoffStringPointer(reason),
	}
}

func deepJobMatchesTarget(job *models.DeepIdentificationJob, target *DeepAnalysisHandoffTarget) bool {
	if job == nil || target == nil {
		return false
	}
	if target.Type == "coin" {
		return job.Source == models.DeepJobSourceSavedCoin && job.CoinID != nil && *job.CoinID == target.ID
	}
	return job.Source == models.DeepJobSourceCopilotDraft && job.SourceDraftID != nil && *job.SourceDraftID == target.ID
}

func handoffSHA256Hex(value string) string { return handoffSHA256HexBytes([]byte(value)) }

func handoffSHA256HexBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func valueOrZero(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func handoffStringPointer(value string) *string { return &value }
