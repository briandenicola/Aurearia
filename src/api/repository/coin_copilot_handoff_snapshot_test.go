package repository

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type handoffSnapshotFixture struct {
	db          *gorm.DB
	repo        *CoinCopilotRepository
	admission   DeepHandoffAdmission
	coin        models.Coin
	draft       models.QuickCaptureDraft
	obversePath string
	reversePath string
	wakes       int
}

func newHandoffSnapshotFixture(t *testing.T, kind models.CoinCopilotDeepHandoffTargetKind) *handoffSnapshotFixture {
	t.Helper()
	db, repo := newCopilotRepositoryTestDB(t)
	if err := db.AutoMigrate(
		&models.Coin{}, &models.CoinImage{}, &models.AppSetting{},
		&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationArtifact{},
		&models.DeepIdentificationProviderRun{}, &models.CoinCopilotDeepHandoff{},
	); err != nil {
		t.Fatal(err)
	}
	run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
	fixture := &handoffSnapshotFixture{db: db, repo: repo}
	targetID := uint(42)
	context := "before"
	state := "active"
	source := models.DeepJobSourceSavedCoin
	coinID := &targetID
	var draftID *uint

	if kind == models.CoinCopilotDeepHandoffTargetCoin {
		fixture.coin = models.Coin{ID: targetID, UserID: 7, Name: "Barrier", Notes: context}
		if err := db.Create(&fixture.coin).Error; err != nil {
			t.Fatal(err)
		}
	} else {
		fixture.draft = models.QuickCaptureDraft{
			ID: targetID, UserID: 7, WorkingTitle: "Barrier",
			Notes: context, Status: models.QuickCaptureDraftStatusActive,
		}
		if err := db.Create(&fixture.draft).Error; err != nil {
			t.Fatal(err)
		}
		source = models.DeepJobSourceCopilotDraft
		coinID = nil
		draftID = &targetID
	}

	fixture.obversePath = filepath.Join(t.TempDir(), "obverse.png")
	fixture.reversePath = filepath.Join(filepath.Dir(fixture.obversePath), "reverse.png")
	obverseContent := []byte("feature362-obverse")
	reverseContent := []byte("feature362-reverse")
	if err := os.WriteFile(fixture.obversePath, obverseContent, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.reversePath, reverseContent, 0o600); err != nil {
		t.Fatal(err)
	}
	obverse, reverse := fixture.createFaces(t, kind)
	updatedAt := fixture.coin.UpdatedAt
	if kind == models.CoinCopilotDeepHandoffTargetDraft {
		updatedAt = fixture.draft.UpdatedAt
	}
	fixture.admission = DeepHandoffAdmission{
		Handoff: &models.CoinCopilotDeepHandoff{
			UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID,
			HandoffKeyHash: "barrier-key", RequestFingerprint: "barrier-request",
			AppContextDigest: DigestCoinCopilotAppContext(run.AppContextJSON),
			Operation:        models.CoinCopilotDeepHandoffOperationRequest,
			TargetKind:       kind, TargetID: targetID,
			TargetSnapshotFingerprint: "barrier-snapshot",
		},
		Job: &models.DeepIdentificationJob{
			UserID: 7, Source: source, CoinID: coinID, SourceDraftID: draftID,
			InputFingerprint: "barrier-fingerprint",
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
		},
		Target: DeepHandoffTargetToken{
			Kind: kind, ID: targetID, UserID: 7, State: state,
			UpdatedAt: updatedAt.UTC().Format(time.RFC3339Nano), Context: context,
			Obverse: fixture.faceToken(obverse, fixture.obversePath, obverseContent),
			Reverse: fixture.faceToken(reverse, fixture.reversePath, reverseContent),
		},
		ProviderSettings: map[string]string{"DeepIdentificationOCREEnabled": "false"},
	}
	fixture.admission.AfterCommit = func(uint) { fixture.wakes++ }
	return fixture
}

func (f *handoffSnapshotFixture) createFaces(
	t *testing.T,
	kind models.CoinCopilotDeepHandoffTargetKind,
) (models.CoinImage, models.CoinImage) {
	t.Helper()
	obverse := models.CoinImage{FilePath: "obverse.png", ImageType: models.ImageTypeObverse}
	reverse := models.CoinImage{FilePath: "reverse.png", ImageType: models.ImageTypeReverse}
	if kind == models.CoinCopilotDeepHandoffTargetCoin {
		obverse.CoinID = f.coin.ID
		reverse.CoinID = f.coin.ID
		if err := f.db.Create(&obverse).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.db.Create(&reverse).Error; err != nil {
			t.Fatal(err)
		}
		return obverse, reverse
	}
	draftObverse := models.QuickCaptureDraftImage{
		DraftID: f.draft.ID, UserID: f.draft.UserID,
		FilePath: obverse.FilePath, ImageType: obverse.ImageType,
	}
	draftReverse := models.QuickCaptureDraftImage{
		DraftID: f.draft.ID, UserID: f.draft.UserID,
		FilePath: reverse.FilePath, ImageType: reverse.ImageType,
	}
	if err := f.db.Create(&draftObverse).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&draftReverse).Error; err != nil {
		t.Fatal(err)
	}
	obverse.ID, obverse.CreatedAt = draftObverse.ID, draftObverse.CreatedAt
	reverse.ID, reverse.CreatedAt = draftReverse.ID, draftReverse.CreatedAt
	return obverse, reverse
}

func (f *handoffSnapshotFixture) faceToken(
	image models.CoinImage,
	contentPath string,
	content []byte,
) DeepHandoffFaceToken {
	return DeepHandoffFaceToken{
		ID: image.ID, FilePath: image.FilePath, ContentPath: contentPath,
		ContentHash: fmt.Sprintf("%x", sha256.Sum256(content)),
		CreatedAt:   image.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (f *handoffSnapshotFixture) admitAfterSnapshotChange(change func()) error {
	snapshotReady := make(chan struct{})
	releaseAdmission := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(snapshotReady)
		<-releaseAdmission
		_, _, _, err := f.repo.AdmitDeepHandoff(f.admission)
		result <- err
	}()
	<-snapshotReady
	change()
	close(releaseAdmission)
	return <-result
}

func (f *handoffSnapshotFixture) cloneAdmission() DeepHandoffAdmission {
	admission := f.admission
	handoff := *f.admission.Handoff
	job := *f.admission.Job
	admission.Handoff = &handoff
	admission.Job = &job
	admission.Artifacts = append([]models.DeepIdentificationArtifact(nil), f.admission.Artifacts...)
	return admission
}

func (f *handoffSnapshotFixture) durableCounts(t *testing.T) (handoffs, jobs, providers int64) {
	t.Helper()
	for name, target := range map[string]struct {
		model any
		count *int64
	}{
		"handoffs":  {&models.CoinCopilotDeepHandoff{}, &handoffs},
		"jobs":      {&models.DeepIdentificationJob{}, &jobs},
		"providers": {&models.DeepIdentificationProviderRun{}, &providers},
	} {
		if err := f.db.Model(target.model).Count(target.count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
	}
	return handoffs, jobs, providers
}

func (f *handoffSnapshotFixture) assertNoWork(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrCopilotDeepHandoffState) {
		t.Fatalf("admission error=%v, want state conflict", err)
	}
	for name, model := range map[string]any{
		"handoffs":      &models.CoinCopilotDeepHandoff{},
		"jobs":          &models.DeepIdentificationJob{},
		"provider runs": &models.DeepIdentificationProviderRun{},
	} {
		var count int64
		if countErr := f.db.Model(model).Count(&count).Error; countErr != nil {
			t.Fatal(countErr)
		}
		if count != 0 {
			t.Errorf("%s=%d, want zero", name, count)
		}
	}
	if f.wakes != 0 {
		t.Errorf("worker wakes=%d, want zero", f.wakes)
	}
}

func TestFeature362DurableHandoffExactReplayCreatesNoWork(t *testing.T) {
	fixture := newHandoffSnapshotFixture(t, models.CoinCopilotDeepHandoffTargetCoin)
	_, firstJob, replayed, err := fixture.repo.AdmitDeepHandoff(fixture.cloneAdmission())
	if err != nil || replayed {
		t.Fatalf("initial admission job=%+v replayed=%v err=%v", firstJob, replayed, err)
	}
	_, replayJob, replayed, err := fixture.repo.AdmitDeepHandoff(fixture.cloneAdmission())
	if err != nil || !replayed || replayJob.ID != firstJob.ID {
		t.Fatalf("replay job=%+v replayed=%v err=%v", replayJob, replayed, err)
	}
	handoffs, jobs, providers := fixture.durableCounts(t)
	if handoffs != 1 || jobs != 1 || providers != 0 || fixture.wakes != 1 {
		t.Fatalf(
			"replay work: handoffs=%d jobs=%d providers=%d wakes=%d",
			handoffs, jobs, providers, fixture.wakes,
		)
	}
}

func TestFeature362DurableHandoffChangedBindingConflictsWithoutWork(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.CoinCopilotDeepHandoff)
	}{
		{"request fingerprint", func(h *models.CoinCopilotDeepHandoff) { h.RequestFingerprint = "changed" }},
		{"operation", func(h *models.CoinCopilotDeepHandoff) { h.Operation = models.CoinCopilotDeepHandoffOperationRerun }},
		{"target kind", func(h *models.CoinCopilotDeepHandoff) { h.TargetKind = models.CoinCopilotDeepHandoffTargetDraft }},
		{"target id", func(h *models.CoinCopilotDeepHandoff) { h.TargetID++ }},
		{"prior rerun job id", func(h *models.CoinCopilotDeepHandoff) {
			value := uint(314)
			h.PriorJobID = &value
		}},
		{"execution", func(h *models.CoinCopilotDeepHandoff) { h.ExecutionID = "cce_stale" }},
		{"expected checkpoint", func(h *models.CoinCopilotDeepHandoff) { h.ExpectedCheckpointVersion++ }},
		{"stored app context", func(h *models.CoinCopilotDeepHandoff) { h.AppContextDigest = "changed" }},
		{"server snapshot", func(h *models.CoinCopilotDeepHandoff) {
			h.TargetSnapshotFingerprint = "changed"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newHandoffSnapshotFixture(t, models.CoinCopilotDeepHandoffTargetCoin)
			if _, _, replayed, err := fixture.repo.AdmitDeepHandoff(fixture.cloneAdmission()); err != nil || replayed {
				t.Fatalf("initial admission replayed=%v err=%v", replayed, err)
			}
			beforeHandoffs, beforeJobs, beforeProviders := fixture.durableCounts(t)
			beforeWakes := fixture.wakes
			changed := fixture.cloneAdmission()
			test.mutate(changed.Handoff)
			if _, _, _, err := fixture.repo.AdmitDeepHandoff(changed); !errors.Is(err, ErrCopilotDeepHandoffConflict) {
				t.Fatalf("changed binding error=%v, want idempotency conflict", err)
			}
			handoffs, jobs, providers := fixture.durableCounts(t)
			if handoffs != beforeHandoffs || jobs != beforeJobs ||
				providers != beforeProviders || fixture.wakes != beforeWakes {
				t.Fatalf(
					"changed binding added work: handoffs=%d/%d jobs=%d/%d providers=%d/%d wakes=%d/%d",
					handoffs, beforeHandoffs, jobs, beforeJobs,
					providers, beforeProviders, fixture.wakes, beforeWakes,
				)
			}
		})
	}
}

func TestFeature362AdmissionRejectsChangedCoinSnapshot(t *testing.T) {
	tests := []struct {
		name   string
		change func(*handoffSnapshotFixture)
	}{
		{"coin deletion", func(f *handoffSnapshotFixture) { _ = f.db.Delete(&f.coin).Error }},
		{"obverse row replacement", func(f *handoffSnapshotFixture) {
			_ = f.db.Delete(&models.CoinImage{}, f.admission.Target.Obverse.ID).Error
		}},
		{"obverse content replacement", func(f *handoffSnapshotFixture) {
			_ = os.WriteFile(f.obversePath, []byte("changed"), 0o600)
		}},
		{"reverse row replacement", func(f *handoffSnapshotFixture) {
			_ = f.db.Delete(&models.CoinImage{}, f.admission.Target.Reverse.ID).Error
		}},
		{"reverse content replacement", func(f *handoffSnapshotFixture) {
			_ = os.WriteFile(f.reversePath, []byte("changed"), 0o600)
		}},
		{"notes value", func(f *handoffSnapshotFixture) {
			_ = f.db.Model(&f.coin).Update("notes", "changed").Error
		}},
		{"context version", func(f *handoffSnapshotFixture) {
			_ = f.db.Model(&f.coin).Update("updated_at", time.Now().UTC().Add(time.Second)).Error
		}},
		{"provider generation", func(f *handoffSnapshotFixture) {
			_ = f.db.Create(&models.AppSetting{Key: "DeepIdentificationOCREEnabled", Value: "true"}).Error
		}},
		{"owner", func(f *handoffSnapshotFixture) {
			_ = f.db.Model(&f.coin).Update("user_id", 8).Error
		}},
		{"target kind", func(f *handoffSnapshotFixture) {
			f.admission.Handoff.TargetKind = models.CoinCopilotDeepHandoffTargetDraft
		}},
		{"target id", func(f *handoffSnapshotFixture) { f.admission.Handoff.TargetID++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newHandoffSnapshotFixture(t, models.CoinCopilotDeepHandoffTargetCoin)
			fixture.assertNoWork(t, fixture.admitAfterSnapshotChange(func() { test.change(fixture) }))
		})
	}
}

func TestFeature362AdmissionRejectsChangedDraftLifecycle(t *testing.T) {
	tests := []struct {
		name   string
		change func(*handoffSnapshotFixture)
	}{
		{"promotion", func(f *handoffSnapshotFixture) {
			_ = f.db.Model(&f.draft).Update("status", models.QuickCaptureDraftStatusPromoted).Error
		}},
		{"discard", func(f *handoffSnapshotFixture) {
			_ = f.db.Model(&f.draft).Update("status", models.QuickCaptureDraftStatusDiscarded).Error
		}},
		{"deletion", func(f *handoffSnapshotFixture) { _ = f.db.Delete(&f.draft).Error }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newHandoffSnapshotFixture(t, models.CoinCopilotDeepHandoffTargetDraft)
			fixture.assertNoWork(t, fixture.admitAfterSnapshotChange(func() { test.change(fixture) }))
		})
	}
}
