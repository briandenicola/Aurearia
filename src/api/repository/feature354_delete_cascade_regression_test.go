package repository

// Independent QA regression coverage for spec 354 DELETE-job behavior
// (FR-014/FR-015/FR-016, D3), owned by Brutus (Tester/QA).
//
// Cassius's own deep_identification_repository_test.go already proves the
// job row is hard-deleted and that a non-terminal job is rejected with
// ErrDeepJobNotTerminal. This file is deliberately additive and dedicated
// (new file, to avoid merge collisions) — it proves two things his test
// does not:
//
//  1. DeleteJob is a genuine CASCADE of the job's own child rows (provider
//     runs, events, artifacts) — not just the parent job row — so no
//     orphaned rows are left behind in the database.
//  2. DeleteJob NEVER cascades to the linked Coin (FR-015: deleting a run
//     history entry must not delete or modify the coin it was applied to).

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestFeature354_DeleteJob_CascadesProviderRunsEventsAndArtifacts(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "f354-delete-cascade")

	job := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: "fp-f354-delete-cascade",
		ExpiresAt:        models.DeepIdentificationNoExpirySentinel,
	}
	if _, _, err := repo.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationProviderRun{
		JobID: job.ID, UserID: owner.ID, Provider: "numista", Status: models.DeepProviderRunContributed,
	}).Error; err != nil {
		t.Fatalf("seed provider run: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationEvent{
		JobID: job.ID, UserID: owner.ID, Seq: 1, Type: models.DeepEventProgress, PayloadJSON: "{}",
	}).Error; err != nil {
		t.Fatalf("seed event: %v", err)
	}
	if err := db.Create(&models.DeepIdentificationArtifact{
		JobID: job.ID, UserID: owner.ID, Role: models.DeepArtifactRoleHint,
		Origin: models.DeepArtifactOriginUploaded, FilePath: "tmp/f354.png", ContentHash: "hash-f354",
	}).Error; err != nil {
		t.Fatalf("seed artifact: %v", err)
	}

	if err := repo.DeleteJob(owner.ID, job.ID); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	var runCount, eventCount, artifactCount int64
	if err := db.Model(&models.DeepIdentificationProviderRun{}).Where("job_id = ?", job.ID).Count(&runCount).Error; err != nil {
		t.Fatalf("count provider runs: %v", err)
	}
	if err := db.Model(&models.DeepIdentificationEvent{}).Where("job_id = ?", job.ID).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if err := db.Unscoped().Model(&models.DeepIdentificationArtifact{}).Where("job_id = ?", job.ID).Count(&artifactCount).Error; err != nil {
		t.Fatalf("count artifacts: %v", err)
	}

	if runCount != 0 {
		t.Errorf("expected 0 orphaned provider runs after DeleteJob, got %d", runCount)
	}
	if eventCount != 0 {
		t.Errorf("expected 0 orphaned events after DeleteJob, got %d", eventCount)
	}
	if artifactCount != 0 {
		t.Errorf("expected artifacts hard-deleted (or at minimum fully soft-deleted+unscoped-gone) after DeleteJob, got %d rows still present unscoped", artifactCount)
	}
}

func TestFeature354_DeleteJob_DoesNotCascadeToLinkedCoin(t *testing.T) {
	db := newDeepIdentificationTestDB(t)
	repo := NewDeepIdentificationRepository(db)
	owner := createDeepTestUser(t, db, "f354-delete-nocascade")

	coin := models.Coin{UserID: owner.ID, Name: "Trajan Denarius"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}

	job := &models.DeepIdentificationJob{
		UserID:           owner.ID,
		Source:           models.DeepJobSourceIntake,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: "fp-f354-delete-nocascade",
		ExpiresAt:        models.DeepIdentificationNoExpirySentinel,
		AppliedCoinID:    &coin.ID,
	}
	if _, _, err := repo.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if err := repo.DeleteJob(owner.ID, job.ID); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	var reloaded models.Coin
	if err := db.First(&reloaded, coin.ID).Error; err != nil {
		t.Fatalf("expected linked coin to survive job deletion (FR-015), but reload failed: %v", err)
	}
	if reloaded.Name != "Trajan Denarius" {
		t.Fatalf("expected linked coin fields untouched by job deletion, got name=%q", reloaded.Name)
	}
}
