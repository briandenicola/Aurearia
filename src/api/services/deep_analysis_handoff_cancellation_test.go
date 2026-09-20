package services

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

type handoffCancellationFixture struct {
	db        *gorm.DB
	service   *CoinCopilotService
	deepRepo  *repository.DeepIdentificationRepository
	admission repository.DeepHandoffAdmission
	wakes     atomic.Int32
}

func newHandoffCancellationFixture(t *testing.T) *handoffCancellationFixture {
	t.Helper()
	db, service := newCopilotServiceTest(t)
	if err := db.AutoMigrate(
		&models.Coin{}, &models.CoinImage{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationArtifact{},
		&models.DeepIdentificationEvent{}, &models.CoinCopilotDeepHandoff{},
	); err != nil {
		t.Fatal(err)
	}
	thread := &models.CoinCopilotThread{ID: "cct_handoff_cancel", UserID: 7, Title: "Cancellation"}
	run := &models.CoinCopilotRun{
		ID: "ccr_handoff_cancel", ThreadID: thread.ID, UserID: 7,
		Status: models.CopilotRunRunning, Goal: "attribute coin",
		StartIdempotencyKeyHash: "start", StartRequestFingerprint: "request",
		ExecutionID: "cce_handoff_cancel", MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := service.repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{ID: 42, UserID: 7, Name: "Cancellation coin", Notes: "manual"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	obversePath := filepath.Join(base, "obverse.png")
	reversePath := filepath.Join(base, "reverse.png")
	obverseContent := []byte("cancellation-obverse")
	reverseContent := []byte("cancellation-reverse")
	if err := os.WriteFile(obversePath, obverseContent, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reversePath, reverseContent, 0o600); err != nil {
		t.Fatal(err)
	}
	obverse := models.CoinImage{CoinID: coin.ID, FilePath: "obverse.png", ImageType: models.ImageTypeObverse}
	reverse := models.CoinImage{CoinID: coin.ID, FilePath: "reverse.png", ImageType: models.ImageTypeReverse}
	if err := db.Create(&obverse).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&reverse).Error; err != nil {
		t.Fatal(err)
	}
	fixture := &handoffCancellationFixture{
		db: db, service: service,
		deepRepo: repository.NewDeepIdentificationRepository(db),
	}
	fixture.admission = repository.DeepHandoffAdmission{
		Handoff: &models.CoinCopilotDeepHandoff{
			UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID,
			HandoffKeyHash: "handoff-key", RequestFingerprint: "handoff-request",
			AppContextDigest: repository.DigestCoinCopilotAppContext(run.AppContextJSON),
			Operation:        models.CoinCopilotDeepHandoffOperationRequest,
			TargetKind:       models.CoinCopilotDeepHandoffTargetCoin, TargetID: coin.ID,
			TargetSnapshotFingerprint: "snapshot",
		},
		Job: &models.DeepIdentificationJob{
			UserID: 7, Source: models.DeepJobSourceSavedCoin, CoinID: &coin.ID,
			Status: models.DeepJobStatusQueued, InputFingerprint: "cancellation-fingerprint",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
		Target: repository.DeepHandoffTargetToken{
			Kind: models.CoinCopilotDeepHandoffTargetCoin,
			ID:   coin.ID, UserID: coin.UserID, State: "active",
			UpdatedAt: coin.UpdatedAt.UTC().Format(time.RFC3339Nano), Context: coin.Notes,
			Obverse: cancellationFaceToken(obverse, obversePath, obverseContent),
			Reverse: cancellationFaceToken(reverse, reversePath, reverseContent),
		},
		Artifacts: []models.DeepIdentificationArtifact{
			{Role: models.DeepArtifactRoleObverse, Origin: models.DeepArtifactOriginSavedCoinImage, FilePath: obversePath},
			{Role: models.DeepArtifactRoleReverse, Origin: models.DeepArtifactOriginSavedCoinImage, FilePath: reversePath},
		},
	}
	fixture.admission.AfterCommit = func(uint) { fixture.wakes.Add(1) }
	return fixture
}

func cancellationFaceToken(
	image models.CoinImage,
	contentPath string,
	content []byte,
) repository.DeepHandoffFaceToken {
	return repository.DeepHandoffFaceToken{
		ID: image.ID, FilePath: image.FilePath, ContentPath: contentPath,
		ContentHash: fmt.Sprintf("%x", sha256.Sum256(content)),
		CreatedAt:   image.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (f *handoffCancellationFixture) cloneAdmission() repository.DeepHandoffAdmission {
	admission := f.admission
	handoff := *f.admission.Handoff
	job := *f.admission.Job
	admission.Handoff = &handoff
	admission.Job = &job
	admission.Artifacts = append([]models.DeepIdentificationArtifact(nil), f.admission.Artifacts...)
	return admission
}

func (f *handoffCancellationFixture) counts(t *testing.T) (handoffs, jobs int64) {
	t.Helper()
	if err := f.db.Model(&models.CoinCopilotDeepHandoff{}).Count(&handoffs).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&models.DeepIdentificationJob{}).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	return handoffs, jobs
}

func TestFeature362CancelFirstPreventsAdmission(t *testing.T) {
	fixture := newHandoffCancellationFixture(t)
	if _, immediate, err := fixture.service.Cancel(7, fixture.admission.Handoff.RunID); err != nil || immediate {
		t.Fatalf("cancel immediate=%v err=%v", immediate, err)
	}
	if _, _, _, err := fixture.service.repo.AdmitDeepHandoff(fixture.cloneAdmission()); !errors.Is(err, repository.ErrCopilotDeepHandoffState) {
		t.Fatalf("admission error=%v, want state conflict", err)
	}
	handoffs, jobs := fixture.counts(t)
	if handoffs != 0 || jobs != 0 || fixture.wakes.Load() != 0 {
		t.Fatalf("cancel-first work: handoffs=%d jobs=%d wakes=%d", handoffs, jobs, fixture.wakes.Load())
	}
}

func TestFeature362AdmissionFirstRequestsDeepCancellation(t *testing.T) {
	fixture := newHandoffCancellationFixture(t)
	_, job, replayed, err := fixture.service.repo.AdmitDeepHandoff(fixture.cloneAdmission())
	if err != nil || replayed {
		t.Fatalf("admission job=%+v replayed=%v err=%v", job, replayed, err)
	}
	if _, immediate, err := fixture.service.Cancel(7, fixture.admission.Handoff.RunID); err != nil || immediate {
		t.Fatalf("cancel immediate=%v err=%v", immediate, err)
	}
	var stored models.DeepIdentificationJob
	if err := fixture.db.First(&stored, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	handoffs, jobs := fixture.counts(t)
	if handoffs != 1 || jobs != 1 || fixture.wakes.Load() != 1 || stored.CancelRequestedAt == nil {
		t.Fatalf(
			"admission-first state: handoffs=%d jobs=%d wakes=%d cancel=%v",
			handoffs, jobs, fixture.wakes.Load(), stored.CancelRequestedAt,
		)
	}
}

func TestFeature362SimultaneousDuplicateAdmissionCreatesOneJob(t *testing.T) {
	fixture := newHandoffCancellationFixture(t)
	start := make(chan struct{})
	results := make(chan struct {
		replayed bool
		err      error
	}, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, _, replayed, err := fixture.service.repo.AdmitDeepHandoff(fixture.cloneAdmission())
			results <- struct {
				replayed bool
				err      error
			}{replayed: replayed, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	replays := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.replayed {
			replays++
		}
	}
	handoffs, jobs := fixture.counts(t)
	if replays != 1 || handoffs != 1 || jobs != 1 || fixture.wakes.Load() != 1 {
		t.Fatalf(
			"duplicate state: replays=%d handoffs=%d jobs=%d wakes=%d",
			replays, handoffs, jobs, fixture.wakes.Load(),
		)
	}
}

func TestFeature362CancellationRejectsLateSettlements(t *testing.T) {
	fixture := newHandoffCancellationFixture(t)
	_, job, _, err := fixture.service.repo.AdmitDeepHandoff(fixture.cloneAdmission())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := fixture.service.Cancel(7, fixture.admission.Handoff.RunID); err != nil {
		t.Fatal(err)
	}
	won, err := fixture.deepRepo.SettleTerminal(
		job.ID, []models.DeepJobStatus{models.DeepJobStatusQueued, models.DeepJobStatusRunning},
		models.DeepJobStatusCompleted, `{"late":true}`, `{"late":true}`, "", "",
	)
	if err != nil || won {
		t.Fatalf("late report won=%v err=%v", won, err)
	}
	if updated, err := fixture.deepRepo.UpdateProposalJSON(job.ID, 7, `{"late":true}`); err != nil || updated {
		t.Fatalf("late proposal updated=%v err=%v", updated, err)
	}
	if _, err := fixture.deepRepo.AppendEvent(job.ID, 7, models.DeepEventProgress, `{"late":true}`); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("late event error=%v, want record not found", err)
	}
	won, _, err = fixture.service.repo.TransitionWithEvent(
		fixture.admission.Handoff.RunID, 7, fixture.admission.Handoff.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning},
		models.CopilotRunCompleted, map[string]any{"final_answer": "late"},
		models.CopilotEventRunCompleted, `{"late":true}`,
	)
	if err != nil || won {
		t.Fatalf("late Copilot settlement won=%v err=%v", won, err)
	}
	var stored models.DeepIdentificationJob
	if err := fixture.db.First(&stored, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	var deepEvents, copilotEvents int64
	_ = fixture.db.Model(&models.DeepIdentificationEvent{}).Where("job_id = ?", job.ID).Count(&deepEvents).Error
	_ = fixture.db.Model(&models.CoinCopilotEvent{}).Where("run_id = ?", fixture.admission.Handoff.RunID).Count(&copilotEvents).Error
	if stored.ReportJSON != "" || stored.ProposalJSON != "" || deepEvents != 0 || copilotEvents != 0 {
		t.Fatalf(
			"late settlement persisted: report=%q proposal=%q deepEvents=%d copilotEvents=%d",
			stored.ReportJSON, stored.ProposalJSON, deepEvents, copilotEvents,
		)
	}
}
