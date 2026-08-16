package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestDeepIdentificationRuntimeObservabilityTracksCleanupAndJanitor(t *testing.T) {
	svc, db, uploadDir := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "observability-runtime", Email: "observability-runtime@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)

	hint, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "hint.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatalf("save hint: %v", err)
	}
	if err := svc.DeleteHintArtifacts(jobID); err != nil {
		t.Fatalf("delete hint: %v", err)
	}
	if _, err := os.Stat(hint.FilePath); !os.IsNotExist(err) {
		t.Fatalf("hint still exists after deletion: %v", err)
	}

	blockedDir := filepath.Join(uploadDir, "blocked-hint")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatalf("create blocked directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blockedDir, "child"), []byte("x"), 0o600); err != nil {
		t.Fatalf("create blocked child: %v", err)
	}
	failedArtifact := models.DeepIdentificationArtifact{
		JobID: jobID, UserID: user.ID, Role: models.DeepArtifactRoleHint,
		Origin: models.DeepArtifactOriginUploaded, FilePath: blockedDir,
		ContentHash: "operational-only", Ephemeral: true,
	}
	if err := db.Create(&failedArtifact).Error; err != nil {
		t.Fatalf("seed failing hint: %v", err)
	}
	if err := svc.DeleteHintArtifacts(jobID); err == nil {
		t.Fatal("expected non-empty directory removal to fail")
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusFailed, "active_key": "terminal"}).Error; err != nil {
		t.Fatalf("mark job terminal: %v", err)
	}
	queuedJob := models.DeepIdentificationJob{
		UserID: user.ID, Source: models.DeepJobSourceIntake,
		InputFingerprint: "runtime-observability-queued",
		Status:           models.DeepJobStatusQueued, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&queuedJob).Error; err != nil {
		t.Fatalf("seed queued job: %v", err)
	}

	svc.RecoverStaleAndSweepHintsForTest()
	summary, err := svc.GetObservabilitySummary()
	if err != nil {
		t.Fatalf("GetObservabilitySummary: %v", err)
	}
	if summary.HintDeletion.Success != 1 || summary.HintDeletion.Failure < 2 {
		t.Fatalf("hint deletion metrics = %+v", summary.HintDeletion)
	}
	if summary.Janitor.RecoverySweeps != 1 || summary.Janitor.Failures != 1 {
		t.Fatalf("janitor metrics = %+v", summary.Janitor)
	}
	if summary.QueueDepth != 1 {
		t.Fatalf("queue depth = %d, want 1", summary.QueueDepth)
	}

	_ = os.RemoveAll(blockedDir)
	_ = db.Model(&models.DeepIdentificationArtifact{}).
		Where("id = ?", failedArtifact.ID).
		Update("deleted_at", time.Now()).Error
}
