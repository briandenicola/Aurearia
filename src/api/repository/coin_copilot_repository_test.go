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
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newCopilotRepositoryTestDB(t *testing.T) (*gorm.DB, *CoinCopilotRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CoinCopilotThread{}, &models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{}, &models.CoinCopilotEvent{}, &models.CoinCopilotResumeRequest{}); err != nil {
		t.Fatal(err)
	}
	return db, NewCoinCopilotRepository(db)
}

func seedCopilotRun(t *testing.T, repo *CoinCopilotRepository, userID uint, status models.CopilotRunStatus) *models.CoinCopilotRun {
	t.Helper()
	thread := &models.CoinCopilotThread{ID: "cct_test", UserID: userID, Title: "Test"}
	run := &models.CoinCopilotRun{
		ID: "ccr_test", ThreadID: thread.ID, UserID: userID, Status: status, Goal: "test",
		StartIdempotencyKeyHash: "key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_test", MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1,
		HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestCoinCopilotRepositoryOwnerScopeHidesForeignIDs(t *testing.T) {
	_, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunQueued)
	if _, err := repo.GetRun(run.ID, 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign run lookup error = %v, want record not found", err)
	}
	if _, _, err := repo.GetThread(run.ThreadID, 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign thread lookup error = %v, want record not found", err)
	}
}

func TestCoinCopilotRepositoryHistoryIsOwnerScopedAndBounded(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	thread := &models.CoinCopilotThread{ID: "cct_history", UserID: 7, Title: "History"}
	if err := db.Create(thread).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		run := &models.CoinCopilotRun{
			ID: fmt.Sprintf("ccr_history_%d", i), ThreadID: thread.ID, UserID: 7, Status: models.CopilotRunCompleted,
			Goal: fmt.Sprintf("goal-%d", i), FinalAnswer: fmt.Sprintf("answer-%d", i),
			StartIdempotencyKeyHash: fmt.Sprintf("key-%d", i), StartRequestFingerprint: "fingerprint",
			MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
			MaxPersistedToolResultBytes: 32768, CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := db.Create(run).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.CoinCopilotRun{
		ID: "ccr_history_foreign", ThreadID: thread.ID, UserID: 8, Status: models.CopilotRunCompleted,
		Goal: "foreign-private-goal", FinalAnswer: "foreign-private-answer",
		StartIdempotencyKeyHash: "foreign-key", StartRequestFingerprint: "fingerprint",
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
		MaxPersistedToolResultBytes: 32768, CreatedAt: base.Add(10 * time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}
	runs, err := repo.ListSettledRunsForHistory(thread.ID, "ccr_current", 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || runs[0].Goal != "goal-4" || runs[2].Goal != "goal-2" {
		t.Fatalf("history = %#v", runs)
	}
	for _, run := range runs {
		if run.UserID != 0 || run.AppContextJSON != "" || run.Goal == "foreign-private-goal" {
			t.Fatalf("history leaked private or foreign fields: %#v", run)
		}
	}
}

func TestCoinCopilotRepositoryClaimPreservesResumeExecutionID(t *testing.T) {
	_, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunQueued)
	run.ExecutionID = "cce_resume"
	if err := repo.db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).
		Update("execution_id", run.ExecutionID).Error; err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := repo.ClaimNextQueuedRun("worker", "cce_worker_generated")
	if err != nil || !ok {
		t.Fatalf("claim ok=%v err=%v", ok, err)
	}
	if claimed.ExecutionID != "cce_resume" {
		t.Fatalf("claimed execution ID = %q, want resume identity", claimed.ExecutionID)
	}
}

func TestCoinCopilotRepositoryClaimAssignsInitialExecutionID(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunQueued)
	if err := db.Model(&models.CoinCopilotRun{}).Where("id = ?", run.ID).
		Update("execution_id", "").Error; err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := repo.ClaimNextQueuedRun("worker", "cce_initial")
	if err != nil || !ok {
		t.Fatalf("claim ok=%v err=%v", ok, err)
	}
	if claimed.ExecutionID != "cce_initial" {
		t.Fatalf("claimed execution ID = %q, want generated initial identity", claimed.ExecutionID)
	}
}

func TestCoinCopilotRepositoryEventSequenceAndTerminalUniqueness(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
	first, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventRunStarted, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventPlanUpdated, `{"plan":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, "cce_stale", models.CopilotEventPlanUpdated, `{"plan":[]}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("stale execution append error = %v, want transition conflict", err)
	}
	if first.Seq != 1 || second.Seq != 2 {
		t.Fatalf("sequences = %d,%d, want 1,2", first.Seq, second.Seq)
	}
	won, _, err := repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunCompleted,
		map[string]interface{}{"final_answer": "done"}, models.CopilotEventRunCompleted, `{"answer":"done"}`)
	if err != nil || !won {
		t.Fatalf("first terminal transition won=%v err=%v", won, err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventPlanUpdated, `{"plan":[]}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("post-terminal append error = %v, want transition conflict", err)
	}
	won, _, err = repo.TransitionWithEvent(run.ID, run.UserID, run.ExecutionID,
		[]models.CopilotRunStatus{models.CopilotRunRunning}, models.CopilotRunFailed,
		map[string]interface{}{"failure_code": "internal"}, models.CopilotEventRunFailed, `{"code":"internal"}`)
	if err != nil || won {
		t.Fatalf("second terminal transition won=%v err=%v", won, err)
	}
	var terminalCount int64
	if err := db.Model(&models.CoinCopilotEvent{}).Where("run_id = ? AND type IN ?", run.ID,
		[]models.CopilotEventType{models.CopilotEventRunCompleted, models.CopilotEventRunFailed, models.CopilotEventRunCancelled}).Count(&terminalCount).Error; err != nil {
		t.Fatal(err)
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminalCount)
	}
}

func TestCoinCopilotRepositoryRejectsInvalidEventAndTransitionPairs(t *testing.T) {
	_, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventType("invented"), `{}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("unknown event error = %v", err)
	}
	if _, err := repo.AppendEvent(run.ID, run.UserID, run.ExecutionID, models.CopilotEventRunCompleted, `{}`); !errors.Is(err, ErrCopilotTransitionConflict) {
		t.Fatalf("terminal append error = %v", err)
	}
	if won, _, err := repo.TransitionWithEvent(
		run.ID, run.UserID, run.ExecutionID, []models.CopilotRunStatus{models.CopilotRunRunning},
		models.CopilotRunCompleted, map[string]any{}, models.CopilotEventRunFailed, `{}`,
	); !errors.Is(err, ErrCopilotTransitionConflict) || won {
		t.Fatalf("mismatched terminal transition won=%v err=%v", won, err)
	}
}

func TestCoinCopilotRepositoryResumeAndDeleteCascade(t *testing.T) {
	db, repo := newCopilotRepositoryTestDB(t)
	run := seedCopilotRun(t, repo, 7, models.CopilotRunPaused)
	deadline := time.Now().UTC().Add(time.Hour)
	if err := db.Model(run).Updates(map[string]interface{}{"checkpoint_version": 1, "resume_deadline": deadline}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoinCopilotCheckpoint{
		RunID: run.ID, ThreadID: run.ThreadID, UserID: run.UserID, ExecutionID: run.ExecutionID,
		Version: 1, StateJSON: `{"schema_version":1}`, StateDigest: "digest",
	}).Error; err != nil {
		t.Fatal(err)
	}
	resumed, err := repo.Resume(run.ID, run.UserID, "resume-key", "body", "cce_next", "answer", 1, `{"schema_version":1}`, "digest2")
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != models.CopilotRunQueued || resumed.CheckpointVersion != 2 {
		t.Fatalf("resumed run = %#v", resumed)
	}
	events, err := repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != models.CopilotEventRunResumed || events[0].Seq != 1 {
		t.Fatalf("resume events = %#v", events)
	}
	if err := db.Model(resumed).Updates(map[string]interface{}{"status": models.CopilotRunCompleted, "completed_at": time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteThread(run.ThreadID, run.UserID); err != nil {
		t.Fatal(err)
	}
	for _, model := range []any{&models.CoinCopilotRun{}, &models.CoinCopilotCheckpoint{}, &models.CoinCopilotResumeRequest{}, &models.CoinCopilotEvent{}} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%T count=%d err=%v", model, count, err)
		}
	}
}

func TestFeature362RepositoryAdmissionCommitsArtifactsBeforeWorkerWake(t *testing.T) {
	t.Run("revalidates target and provider generation inside admission", func(t *testing.T) {
		for _, test := range []struct {
			name        string
			mutate      func(*gorm.DB, *models.Coin)
			replaceFace bool
		}{
			{
				name: "target context changes",
				mutate: func(db *gorm.DB, coin *models.Coin) {
					_ = db.Model(coin).Updates(map[string]any{
						"notes": "changed", "updated_at": time.Now().UTC().Add(time.Second),
					}).Error
				},
			},
			{
				name: "provider generation changes",
				mutate: func(db *gorm.DB, _ *models.Coin) {
					_ = db.Create(&models.AppSetting{Key: "DeepIdentificationOCREEnabled", Value: "true"}).Error
				},
			},
			{
				name:        "face content changes",
				mutate:      func(*gorm.DB, *models.Coin) {},
				replaceFace: true,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				db, repo := newCopilotRepositoryTestDB(t)
				if err := db.AutoMigrate(
					&models.Coin{}, &models.CoinImage{}, &models.AppSetting{},
					&models.DeepIdentificationJob{}, &models.DeepIdentificationArtifact{},
					&models.CoinCopilotDeepHandoff{},
				); err != nil {
					t.Fatal(err)
				}
				run := seedCopilotRun(t, repo, 7, models.CopilotRunRunning)
				coin := models.Coin{ID: 42, UserID: 7, Name: "Barrier", Notes: "before"}
				if err := db.Create(&coin).Error; err != nil {
					t.Fatal(err)
				}
				contentDir := t.TempDir()
				obverseContent := []byte("feature362-obverse")
				reverseContent := []byte("feature362-reverse")
				obversePath := filepath.Join(contentDir, "obverse.png")
				reversePath := filepath.Join(contentDir, "reverse.png")
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
				appDigest := DigestCoinCopilotAppContext(run.AppContextJSON)
				handoff := &models.CoinCopilotDeepHandoff{
					UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID,
					HandoffKeyHash: "barrier-key", RequestFingerprint: "barrier-request",
					ExpectedCheckpointVersion: 0, AppContextDigest: appDigest,
					Operation:  models.CoinCopilotDeepHandoffOperationRequest,
					TargetKind: models.CoinCopilotDeepHandoffTargetCoin, TargetID: coin.ID,
					TargetSnapshotFingerprint: "barrier-snapshot",
				}
				job := &models.DeepIdentificationJob{
					UserID: 7, Source: models.DeepJobSourceSavedCoin, CoinID: &coin.ID,
					InputFingerprint: "barrier-fingerprint", ExpiresAt: time.Now().UTC().Add(time.Hour),
				}
				target := DeepHandoffTargetToken{
					Kind: models.CoinCopilotDeepHandoffTargetCoin, ID: coin.ID, UserID: 7,
					State: "active", UpdatedAt: coin.UpdatedAt.UTC().Format(time.RFC3339Nano), Context: coin.Notes,
					Obverse: DeepHandoffFaceToken{
						ID: obverse.ID, FilePath: obverse.FilePath,
						ContentPath: obversePath, ContentHash: fmt.Sprintf("%x", sha256.Sum256(obverseContent)),
						CreatedAt: obverse.CreatedAt.UTC().Format(time.RFC3339Nano),
					},
					Reverse: DeepHandoffFaceToken{
						ID: reverse.ID, FilePath: reverse.FilePath,
						ContentPath: reversePath, ContentHash: fmt.Sprintf("%x", sha256.Sum256(reverseContent)),
						CreatedAt: reverse.CreatedAt.UTC().Format(time.RFC3339Nano),
					},
				}
				if test.replaceFace {
					if err := os.WriteFile(obversePath, []byte("changed-content"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				test.mutate(db, &coin)
				_, _, _, err := repo.AdmitDeepHandoff(DeepHandoffAdmission{
					Handoff: handoff, Job: job, Target: target,
					ProviderSettings: map[string]string{"DeepIdentificationOCREEnabled": "false"},
				})
				if !errors.Is(err, ErrCopilotDeepHandoffState) {
					t.Fatalf("admission error=%v want state conflict", err)
				}
				var jobs, handoffs int64
				_ = db.Model(&models.DeepIdentificationJob{}).Count(&jobs).Error
				_ = db.Model(&models.CoinCopilotDeepHandoff{}).Count(&handoffs).Error
				if jobs != 0 || handoffs != 0 {
					t.Fatalf("stale admission created jobs=%d handoffs=%d", jobs, handoffs)
				}
			})
		}
	})
	fixture := newHandoffSnapshotFixture(t, models.CoinCopilotDeepHandoffTargetCoin)
	db, repo := fixture.db, fixture.repo
	handoff, job := fixture.admission.Handoff, fixture.admission.Job
	handoff.HandoffKeyHash = "hashed-key"
	handoff.RequestFingerprint = "request-fingerprint"
	artifacts := []models.DeepIdentificationArtifact{
		{UserID: 7, Role: models.DeepArtifactRoleObverse, Origin: models.DeepArtifactOriginSavedCoinImage, FilePath: "obverse.png", ContentHash: "obverse"},
		{UserID: 7, Role: models.DeepArtifactRoleReverse, Origin: models.DeepArtifactOriginSavedCoinImage, FilePath: "reverse.png", ContentHash: "reverse"},
	}
	wakes := 0
	admission := fixture.admission
	admission.Artifacts = artifacts
	admission.AfterCommit = func(jobID uint) {
		var artifactCount, handoffCount int64
		_ = db.Model(&models.DeepIdentificationArtifact{}).Where("job_id = ?", jobID).Count(&artifactCount).Error
		_ = db.Model(&models.CoinCopilotDeepHandoff{}).Where("deep_job_id = ?", jobID).Count(&handoffCount).Error
		if artifactCount != 2 || handoffCount != 1 {
			t.Errorf("worker woke before durable readiness: artifacts=%d handoffs=%d", artifactCount, handoffCount)
		}
		wakes++
	}
	gotHandoff, gotJob, reused, err := repo.AdmitDeepHandoff(admission)
	if err != nil || reused || gotHandoff.ID == 0 || gotJob.ID == 0 {
		t.Fatalf("admission handoff=%+v job=%+v reused=%v err=%v", gotHandoff, gotJob, reused, err)
	}
	if wakes != 1 {
		t.Fatalf("worker wakes=%d want 1", wakes)
	}

	_, replayJob, replayed, err := repo.AdmitDeepHandoff(DeepHandoffAdmission{
		Handoff: handoff, Job: job, AfterCommit: func(uint) { wakes++ },
	})
	if err != nil || !replayed || replayJob.ID != gotJob.ID {
		t.Fatalf("replay job=%+v replayed=%v err=%v", replayJob, replayed, err)
	}
	if wakes != 1 {
		t.Fatalf("replay woke worker; wakes=%d", wakes)
	}

	changed := *handoff
	changed.RequestFingerprint = "changed-binding"
	if _, _, _, err := repo.AdmitDeepHandoff(DeepHandoffAdmission{Handoff: &changed, Job: job}); !errors.Is(err, ErrCopilotDeepHandoffConflict) {
		t.Fatalf("changed binding error=%v want ErrCopilotDeepHandoffConflict", err)
	}
	var jobs, handoffs, artifactRows int64
	_ = db.Model(&models.DeepIdentificationJob{}).Count(&jobs).Error
	_ = db.Model(&models.CoinCopilotDeepHandoff{}).Count(&handoffs).Error
	_ = db.Model(&models.DeepIdentificationArtifact{}).Count(&artifactRows).Error
	if jobs != 1 || handoffs != 1 || artifactRows != 2 {
		t.Fatalf("conflict created rows: jobs=%d handoffs=%d artifacts=%d", jobs, handoffs, artifactRows)
	}
}
