package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// deepTestDBCounter guarantees a unique in-memory SQLite DSN per test even
// when tests run back-to-back fast enough that time.Now().UnixNano() would
// otherwise collide (observed on this Windows runtime's clock resolution).
// A shared-cache, same-name in-memory DB collision silently mixes state
// between unrelated tests, producing hard-to-diagnose intermittent failures.
var deepTestDBCounter int64

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 200, G: 30, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode tiny png fixture: %v", err)
	}
	return buf.Bytes()
}

func newDeepIdentificationServiceTestDeps(t *testing.T) (*DeepIdentificationService, *gorm.DB, string) {
	t.Helper()
	dsn := fmt.Sprintf("file:deep_identification_svc_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)

	uploadDir := t.TempDir()
	repo := repository.NewDeepIdentificationRepository(db)
	imageRepo := repository.NewImageRepository(db)
	imageSvc := NewImageService(imageRepo, uploadDir)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}
	svc := NewDeepIdentificationService(repo, imageRepo, imageSvc, settingsSvc, NewLogger(100), uploadDir)
	return svc, db, uploadDir
}

func seedDeepTestJob(t *testing.T, db *gorm.DB, userID uint) uint {
	t.Helper()
	job := &models.DeepIdentificationJob{
		UserID: userID, Source: models.DeepJobSourceIntake,
		InputFingerprint: fmt.Sprintf("fp-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("failed to seed job: %v", err)
	}
	return job.ID
}

func TestDeepIdentificationService_ValidateAndSaveArtifact_HappyPath(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "artifact-owner", Email: "artifact-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)

	png := tinyPNGBytes(t)
	artifact, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", png)
	if err != nil {
		t.Fatalf("ValidateAndSaveArtifact failed: %v", err)
	}
	if artifact.Ephemeral {
		t.Fatal("obverse artifact must not be ephemeral")
	}
	if _, err := os.Stat(artifact.FilePath); err != nil {
		t.Fatalf("expected artifact file to exist on disk: %v", err)
	}
}

func TestDeepIdentificationService_ValidateAndSaveArtifact_RejectsDisallowedType(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "bad-type-owner", Email: "bad-type-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)

	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "notes.txt", []byte("not an image")); err == nil {
		t.Fatal("expected non-image bytes to be rejected")
	}
}

func TestDeepIdentificationService_ValidateAndSaveArtifact_RejectsOversize(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "oversize-owner", Email: "oversize-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)

	oversized := make([]byte, MaxImageUploadBytes+1)
	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", oversized); err == nil {
		t.Fatal("expected oversized upload to be rejected")
	}
}

func TestDeepIdentificationService_ValidateAndSaveArtifact_EnforcesSingleObverseAndReverse(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "dup-role-owner", Email: "dup-role-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	png := tinyPNGBytes(t)

	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "o.png", png); err != nil {
		t.Fatalf("first obverse save failed: %v", err)
	}
	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "o2.png", png); err != ErrDeepArtifactRoleExists {
		t.Fatalf("expected ErrDeepArtifactRoleExists for a second obverse, got %v", err)
	}
}

func TestDeepIdentificationService_ValidateAndSaveArtifact_EnforcesHintCap(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "hint-owner", Email: "hint-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	png := tinyPNGBytes(t)

	for i := 0; i < MaxDeepIdentificationHintArtifacts; i++ {
		if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, fmt.Sprintf("hint%d.png", i), png); err != nil {
			t.Fatalf("hint %d save failed: %v", i, err)
		}
	}
	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "hint-over.png", png); err != ErrDeepArtifactHintLimit {
		t.Fatalf("expected ErrDeepArtifactHintLimit for the 4th hint, got %v", err)
	}
}

func TestDeepIdentificationService_ReuseSavedCoinImage(t *testing.T) {
	svc, db, uploadDir := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "reuse-owner", Email: "reuse-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: user.ID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coin.ID), "obverse.png"))
	fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, tinyPNGBytes(t), 0644); err != nil {
		t.Fatal(err)
	}
	image := models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: models.ImageTypeObverse}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)

	artifact, err := svc.ReuseSavedCoinImage(jobID, user.ID, coin.ID, models.DeepArtifactRoleObverse, image.ID)
	if err != nil {
		t.Fatalf("ReuseSavedCoinImage failed: %v", err)
	}
	if artifact.Origin != models.DeepArtifactOriginSavedCoinImage {
		t.Fatalf("expected origin saved_coin_image, got %s", artifact.Origin)
	}
	if artifact.FilePath != "" {
		t.Fatalf("expected empty FilePath for a reused artifact, got %q", artifact.FilePath)
	}
	if artifact.SourceCoinImageID == nil || *artifact.SourceCoinImageID != image.ID {
		t.Fatal("expected SourceCoinImageID to be set to the reused image id")
	}
}

func TestDeepIdentificationService_FingerprintStableAndChangesOnMtime(t *testing.T) {
	svc, db, uploadDir := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "fp-owner", Email: "fp-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: user.ID, Name: "FP Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coin.ID), "obverse.png"))
	fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, tinyPNGBytes(t), 0644); err != nil {
		t.Fatal(err)
	}
	image := models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: models.ImageTypeObverse}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}

	jobID1 := seedDeepTestJob(t, db, user.ID)
	a1, err := svc.ReuseSavedCoinImage(jobID1, user.ID, coin.ID, models.DeepArtifactRoleObverse, image.ID)
	if err != nil {
		t.Fatalf("first ReuseSavedCoinImage failed: %v", err)
	}

	jobID2 := seedDeepTestJob(t, db, user.ID)
	a2, err := svc.ReuseSavedCoinImage(jobID2, user.ID, coin.ID, models.DeepArtifactRoleObverse, image.ID)
	if err != nil {
		t.Fatalf("second ReuseSavedCoinImage failed: %v", err)
	}
	if a1.ContentHash != a2.ContentHash {
		t.Fatal("expected stable fingerprint input for an unchanged saved image")
	}

	// Simulate the saved image changing on disk (retry-after-change edge case).
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(fullPath, append(tinyPNGBytes(t), 0x00), 0644); err != nil {
		t.Fatal(err)
	}
	jobID3 := seedDeepTestJob(t, db, user.ID)
	a3, err := svc.ReuseSavedCoinImage(jobID3, user.ID, coin.ID, models.DeepArtifactRoleObverse, image.ID)
	if err != nil {
		t.Fatalf("third ReuseSavedCoinImage failed: %v", err)
	}
	if a3.ContentHash == a1.ContentHash {
		t.Fatal("expected a changed saved image to yield a different fingerprint input")
	}
}

func TestComputeInputFingerprint_StableAndSensitiveToInputs(t *testing.T) {
	base := FingerprintInput{
		UserID: 1, CoinID: 0,
		ObverseHash: "aaa", ReverseHash: "bbb",
		HintHashes: []string{"h2", "h1"}, Notes: "  hello  ",
		RequestedProviders: []string{"numista", "nomisma"},
	}
	fp1 := ComputeInputFingerprint(base)
	fp2 := ComputeInputFingerprint(FingerprintInput{
		UserID: 1, CoinID: 0,
		ObverseHash: "aaa", ReverseHash: "bbb",
		// Same hints/providers, different input order - must hash identically.
		HintHashes: []string{"h1", "h2"}, Notes: "hello",
		RequestedProviders: []string{"nomisma", "numista"},
	})
	if fp1 != fp2 {
		t.Fatal("expected fingerprint to be stable/order-independent for equivalent inputs")
	}

	changed := base
	changed.Notes = "different notes"
	if ComputeInputFingerprint(changed) == fp1 {
		t.Fatal("expected fingerprint to change when notes change")
	}
}

func TestDeepIdentificationService_DeleteJobArtifacts_Idempotent(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "delete-owner", Email: "delete-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	png := tinyPNGBytes(t)
	artifact, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "hint.png", png)
	if err != nil {
		t.Fatalf("ValidateAndSaveArtifact failed: %v", err)
	}

	if err := svc.DeleteJobArtifacts(jobID); err != nil {
		t.Fatalf("first DeleteJobArtifacts failed: %v", err)
	}
	if _, err := os.Stat(artifact.FilePath); !os.IsNotExist(err) {
		t.Fatal("expected artifact file to be removed")
	}

	// Second call must not error even though the file is already gone and
	// the row already has DeletedAt set (simulates a crash/retry).
	if err := svc.DeleteJobArtifacts(jobID); err != nil {
		t.Fatalf("second (idempotent) DeleteJobArtifacts failed: %v", err)
	}
}

func TestDeepIdentificationService_DeleteHintArtifacts_LeavesFacesIntact(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "hint-only-owner", Email: "hint-only-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	png := tinyPNGBytes(t)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "o.png", png)
	if err != nil {
		t.Fatalf("obverse save failed: %v", err)
	}
	hint, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "h.png", png)
	if err != nil {
		t.Fatalf("hint save failed: %v", err)
	}

	if err := svc.DeleteHintArtifacts(jobID); err != nil {
		t.Fatalf("DeleteHintArtifacts failed: %v", err)
	}
	if _, err := os.Stat(hint.FilePath); !os.IsNotExist(err) {
		t.Fatal("expected hint artifact file to be removed")
	}
	if _, err := os.Stat(obverse.FilePath); err != nil {
		t.Fatalf("expected obverse artifact file to remain: %v", err)
	}
}

func TestDeepIdentificationService_AllTerminalStatesDeleteHints(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus models.DeepJobStatus
		run        func(context.Context, *models.DeepIdentificationJob) (*DeepPipelineResult, error)
		cancel     bool
	}{
		{
			name: "completed", wantStatus: models.DeepJobStatusCompleted,
			run: func(context.Context, *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
				return &DeepPipelineResult{ReportJSON: `{"narrative":"complete"}`, ProposalJSON: `{"fields":{}}`}, nil
			},
		},
		{
			name: "partial", wantStatus: models.DeepJobStatusPartial,
			run: func(context.Context, *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
				return &DeepPipelineResult{ReportJSON: `{"narrative":"partial"}`, Partial: true}, nil
			},
		},
		{
			name: "failed", wantStatus: models.DeepJobStatusFailed,
			run: func(context.Context, *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
				return nil, errors.New("provider unavailable")
			},
		},
		{
			name: "cancelled", wantStatus: models.DeepJobStatusCancelled, cancel: true,
			run: func(ctx context.Context, _ *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, db, _ := newDeepIdentificationServiceTestDeps(t)
			user := models.User{
				Username:     "terminal-" + tc.name,
				Email:        "terminal-" + tc.name + "@example.com",
				PasswordHash: "x",
			}
			if err := db.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			enableDeepIdentification(t, svc, nil)
			jobID := seedDeepTestJob(t, db, user.ID)
			hint, err := svc.ValidateAndSaveArtifact(
				jobID, user.ID, models.DeepArtifactRoleHint, "hint.png", tinyPNGBytes(t),
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
				Updates(map[string]any{"status": models.DeepJobStatusRunning, "started_at": time.Now()}).Error; err != nil {
				t.Fatal(err)
			}
			var job models.DeepIdentificationJob
			if err := db.First(&job, jobID).Error; err != nil {
				t.Fatal(err)
			}
			svc.SetPipelineRunner(&fakeRunner{run: tc.run})
			ctx := context.Background()
			if tc.cancel {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}

			svc.runJob(ctx, &job)

			var final models.DeepIdentificationJob
			if err := db.First(&final, jobID).Error; err != nil {
				t.Fatal(err)
			}
			if final.Status != tc.wantStatus {
				t.Fatalf("expected %s, got %s", tc.wantStatus, final.Status)
			}
			var artifact models.DeepIdentificationArtifact
			if err := db.First(&artifact, hint.ID).Error; err != nil {
				t.Fatal(err)
			}
			if artifact.DeletedAt == nil {
				t.Fatal("expected hint DeletedAt to be stamped")
			}
			if _, err := os.Stat(hint.FilePath); !os.IsNotExist(err) {
				t.Fatal("expected hint file to be deleted")
			}
			if strings.Contains(final.ReportJSON, hint.FilePath) || strings.Contains(final.ProposalJSON, hint.FilePath) {
				t.Fatal("terminal report/proposal must never contain a hint artifact path")
			}
		})
	}
}

func TestDeepIdentificationService_RetentionSweepDeletesAllArtifacts(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "expired-artifacts", Email: "expired-artifacts@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	hint, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "hint.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.DeepJobStatusCompleted,
			"completed_at": time.Now().Add(-48 * time.Hour),
			"expires_at":   time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.runRetentionSweep()

	for _, artifact := range []*models.DeepIdentificationArtifact{obverse, hint} {
		var reloaded models.DeepIdentificationArtifact
		if err := db.First(&reloaded, artifact.ID).Error; err != nil {
			t.Fatal(err)
		}
		if reloaded.DeletedAt == nil {
			t.Fatalf("expected artifact %d DeletedAt to be stamped", artifact.ID)
		}
		if _, err := os.Stat(artifact.FilePath); !os.IsNotExist(err) {
			t.Fatalf("expected artifact %d file to be deleted", artifact.ID)
		}
	}
}

// --- Phase 4: worker pool / cancel / timeout / janitor tests ---

// fakeRunner is an injectable DeepPipelineRunner used by Phase 4 tests. It
// tracks the peak number of concurrently-running jobs and lets each test
// decide, per job, how the "pipeline" behaves.
type fakeRunner struct {
	mu            sync.Mutex
	current       int
	maxConcurrent int
	run           func(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error)
}

func (f *fakeRunner) Run(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
	f.mu.Lock()
	f.current++
	if f.current > f.maxConcurrent {
		f.maxConcurrent = f.current
	}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.current--
		f.mu.Unlock()
	}()
	return f.run(ctx, job)
}

func (f *fakeRunner) peak() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxConcurrent
}

func enableDeepIdentification(t *testing.T, svc *DeepIdentificationService, overrides map[string]string) {
	t.Helper()
	if err := svc.settingsSvc.SetSetting(SettingDeepIdentificationEnabled, "true"); err != nil {
		t.Fatalf("failed to enable deep identification: %v", err)
	}
	for k, v := range overrides {
		if err := svc.settingsSvc.SetSetting(k, v); err != nil {
			t.Fatalf("failed to set %s: %v", k, err)
		}
	}
}

func newDeepStartJob(t *testing.T, userID uint, notes string) *models.DeepIdentificationJob {
	t.Helper()
	return &models.DeepIdentificationJob{
		UserID: userID,
		Source: models.DeepJobSourceIntake,
		InputFingerprint: ComputeInputFingerprint(FingerprintInput{
			UserID: userID, ObverseHash: "o", ReverseHash: "r", Notes: notes,
		}),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}
}

func TestDeepIdentificationService_StartJob_QueueDepthAndPerUserLimit(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "worker-owner", Email: "worker-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	other := models.User{Username: "worker-owner-2", Email: "worker-owner-2@example.com", PasswordHash: "x"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{
		SettingDeepIdentificationMaxActivePerUser: "1",
		SettingDeepIdentificationQueueDepth:       "2",
	})

	job1, reused1, err := svc.StartJob(newDeepStartJob(t, user.ID, "first"))
	if err != nil || reused1 {
		t.Fatalf("expected first job to be newly created, got reused=%v err=%v", reused1, err)
	}

	// Same user, different fingerprint: blocked by the per-user active limit
	// and should surface the existing active job rather than a new one.
	job2, reused2, err := svc.StartJob(newDeepStartJob(t, user.ID, "second"))
	if err != nil {
		t.Fatalf("expected per-user limit to return the existing job, got err=%v", err)
	}
	if !reused2 || job2.ID != job1.ID {
		t.Fatalf("expected the existing active job to be returned, got reused=%v id=%d (want %d)", reused2, job2.ID, job1.ID)
	}

	// Different user: allowed (fills the queue depth of 2).
	if _, reused3, err := svc.StartJob(newDeepStartJob(t, other.ID, "other-first")); err != nil || reused3 {
		t.Fatalf("expected second user's first job to be created, got reused=%v err=%v", reused3, err)
	}

	// A third distinct user now exceeds the global queue depth of 2 (two
	// jobs already queued) and should get ErrDeepJobQueueFull.
	third := models.User{Username: "worker-owner-3", Email: "worker-owner-3@example.com", PasswordHash: "x"}
	if err := db.Create(&third).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.StartJob(newDeepStartJob(t, third.ID, "third")); !errors.Is(err, ErrDeepJobQueueFull) {
		t.Fatalf("expected ErrDeepJobQueueFull, got %v", err)
	}
}

func TestDeepIdentificationService_StartJob_DisabledByDefault(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "disabled-owner", Email: "disabled-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.StartJob(newDeepStartJob(t, user.ID, "n")); !errors.Is(err, ErrDeepJobDisabled) {
		t.Fatalf("expected ErrDeepJobDisabled, got %v", err)
	}
}

func TestDeepIdentificationService_WorkerPool_BoundsConcurrency(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "pool-owner", Email: "pool-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{
		SettingDeepIdentificationWorkerCount:      "2",
		SettingDeepIdentificationMaxActivePerUser: "10",
		SettingDeepIdentificationQueueDepth:       "10",
	})

	runner := &fakeRunner{run: func(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
		time.Sleep(60 * time.Millisecond)
		return &DeepPipelineResult{ReportJSON: "{}"}, nil
	}}
	svc.SetPipelineRunner(runner)

	const jobCount = 6
	for i := 0; i < jobCount; i++ {
		if _, _, err := svc.StartJob(newDeepStartJob(t, user.ID, fmt.Sprintf("job-%d", i))); err != nil {
			t.Fatalf("StartJob %d failed: %v", i, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartWorkers(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for {
		var remaining int64
		db.Model(&models.DeepIdentificationJob{}).
			Where("status IN ?", []models.DeepJobStatus{models.DeepJobStatusQueued, models.DeepJobStatusRunning}).
			Count(&remaining)
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			var jobs []models.DeepIdentificationJob
			db.Find(&jobs)
			for _, j := range jobs {
				t.Logf("job %d status=%s worker=%s heartbeat=%v", j.ID, j.Status, j.WorkerID, j.HeartbeatAt)
			}
			t.Fatalf("jobs did not drain in time, %d remaining", remaining)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if runner.peak() > 2 {
		t.Fatalf("expected at most 2 concurrent jobs, saw peak %d", runner.peak())
	}
	var completedCount int64
	db.Model(&models.DeepIdentificationJob{}).Where("status = ?", models.DeepJobStatusCompleted).Count(&completedCount)
	if completedCount != jobCount {
		t.Fatalf("expected %d completed jobs, got %d", jobCount, completedCount)
	}
}

func TestDeepIdentificationService_HardTimeout_SettlesExactlyOnce(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "timeout-owner", Email: "timeout-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{
		SettingDeepIdentificationWorkerCount:        "1",
		SettingDeepIdentificationHardTimeoutSeconds: "1",
	})
	runner := &fakeRunner{run: func(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
		<-ctx.Done() // never returns on its own
		return nil, ctx.Err()
	}}
	svc.SetPipelineRunner(runner)

	job, _, err := svc.StartJob(newDeepStartJob(t, user.ID, "hangs"))
	if err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartWorkers(ctx)

	deadline := time.Now().Add(5 * time.Second)
	var final models.DeepIdentificationJob
	for {
		if err := db.First(&final, job.ID).Error; err != nil {
			t.Fatal(err)
		}
		if models.IsDeepJobTerminal(final.Status) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job never settled, status=%s", final.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if final.Status != models.DeepJobStatusFailed || final.FailureCode != "timeout" {
		t.Fatalf("expected failed/timeout, got status=%s code=%s", final.Status, final.FailureCode)
	}
	var terminalEvents int64
	db.Model(&models.DeepIdentificationEvent{}).Where("job_id = ? AND type = ?", job.ID, models.DeepEventTerminal).Count(&terminalEvents)
	if terminalEvents != 1 {
		t.Fatalf("expected exactly one terminal event, got %d", terminalEvents)
	}
}

func TestDeepIdentificationService_CancelVsComplete_ExactlyOneTerminalState(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "race-owner", Email: "race-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{
		SettingDeepIdentificationWorkerCount: "4",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartWorkers(ctx)

	const iterations = 40
	for i := 0; i < iterations; i++ {
		complete := make(chan struct{})
		runner := &fakeRunner{run: func(ctx context.Context, job *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-complete:
				return &DeepPipelineResult{ReportJSON: "{}"}, nil
			}
		}}
		svc.SetPipelineRunner(runner)

		job := newDeepStartJob(t, user.ID, fmt.Sprintf("race-%d", i))
		created, _, err := svc.StartJob(job)
		if err != nil {
			t.Fatalf("iter %d: StartJob failed: %v", i, err)
		}

		// Wait for a worker to actually claim the job (status running)
		// before racing cancel against completion, matching the real
		// window this test exercises.
		waitDeadline := time.Now().Add(2 * time.Second)
		for {
			var current models.DeepIdentificationJob
			if err := db.First(&current, created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.Status == models.DeepJobStatusRunning {
				break
			}
			if time.Now().After(waitDeadline) {
				t.Fatalf("iter %d: job never reached running", i)
			}
			time.Sleep(time.Millisecond)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = svc.RequestCancel(created.ID, user.ID)
		}()
		go func() {
			defer wg.Done()
			close(complete)
		}()
		wg.Wait()

		settleDeadline := time.Now().Add(2 * time.Second)
		var final models.DeepIdentificationJob
		for {
			if err := db.First(&final, created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if models.IsDeepJobTerminal(final.Status) {
				break
			}
			if time.Now().After(settleDeadline) {
				t.Fatalf("iter %d: job never settled, status=%s", i, final.Status)
			}
			time.Sleep(time.Millisecond)
		}
		if final.Status != models.DeepJobStatusCancelled && final.Status != models.DeepJobStatusCompleted {
			t.Fatalf("iter %d: unexpected terminal status %s", i, final.Status)
		}
		var terminalEvents int64
		db.Model(&models.DeepIdentificationEvent{}).Where("job_id = ? AND type = ?", created.ID, models.DeepEventTerminal).Count(&terminalEvents)
		if terminalEvents != 1 {
			t.Fatalf("iter %d: expected exactly one terminal event, got %d (status=%s)", i, terminalEvents, final.Status)
		}
	}
}

func TestDeepIdentificationService_RestartRecovery_StaleJobsSettleToFailed(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "stale-owner", Email: "stale-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{
		SettingDeepIdentificationHardTimeoutSeconds: "1",
	})

	// Simulate a job left "running" by a prior process instance: no
	// worker is alive to heartbeat it, and its heartbeat is already old.
	job := &models.DeepIdentificationJob{
		UserID: user.ID, Source: models.DeepJobSourceIntake,
		InputFingerprint: fmt.Sprintf("stale-fp-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		Status:           models.DeepJobStatusRunning,
		ActiveKey:        "active",
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	staleHeartbeat := time.Now().Add(-1 * time.Hour)
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).Update("heartbeat_at", staleHeartbeat).Error; err != nil {
		t.Fatal(err)
	}

	svc.recoverStaleAndSweepHints()

	var final models.DeepIdentificationJob
	if err := db.First(&final, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if final.Status != models.DeepJobStatusFailed || final.FailureCode != "stale_restart" {
		t.Fatalf("expected failed/stale_restart, got status=%s code=%s", final.Status, final.FailureCode)
	}
}

func TestDeepIdentificationService_JanitorSweepsOrphanedHintsFromCrash(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "crash-owner", Email: "crash-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	// Mark the job terminal directly (simulating that SettleTerminal ran
	// before the crash) but leave a hint artifact with no DeletedAt
	// (simulating the crash happening before the terminal-hook's
	// DeleteHintArtifacts call completed).
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusCompleted, "completed_at": time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	hint, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleHint, "h.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatalf("failed to seed hint artifact: %v", err)
	}

	svc.recoverStaleAndSweepHints()

	var reloaded models.DeepIdentificationArtifact
	if err := db.First(&reloaded, hint.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.DeletedAt == nil {
		t.Fatal("expected orphaned hint artifact to be swept (DeletedAt set) by the janitor's startup sweep")
	}
	if _, err := os.Stat(hint.FilePath); !os.IsNotExist(err) {
		t.Fatal("expected orphaned hint artifact file to be removed")
	}
}

// --- Phase 5: CreateJobFromIntake / RetryJob orchestration tests ---

func TestDeepIdentificationService_CreateJobFromIntake_IntakeHappyPath(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "intake-owner", Email: "intake-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	png := tinyPNGBytes(t)
	job, reused, err := svc.CreateJobFromIntake(CreateJobInput{
		UserID: user.ID, Notes: "a nice denarius",
		ObverseBytes: png, ObverseFilename: "o.png",
		ReverseBytes: append([]byte(nil), png...), ReverseFilename: "r.png",
	})
	if err != nil {
		t.Fatalf("CreateJobFromIntake failed: %v", err)
	}
	if reused {
		t.Fatal("expected a new job, not a reuse")
	}
	if job.Source != models.DeepJobSourceIntake || job.Status != models.DeepJobStatusQueued {
		t.Fatalf("unexpected job source=%s status=%s", job.Source, job.Status)
	}
	artifacts, err := svc.repo.ListArtifacts(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	obverseCount, reverseCount, hintCount := countArtifactRoles(artifacts)
	if obverseCount != 1 || reverseCount != 1 || hintCount != 0 {
		t.Fatalf("expected 1 obverse + 1 reverse artifact, got obverse=%d reverse=%d hint=%d", obverseCount, reverseCount, hintCount)
	}
}

func TestDeepIdentificationService_WorkerCannotClaimUntilIntakeArtifactsAreReady(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "intake-race-owner", Email: "intake-race-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	started := make(chan struct{}, 1)
	svc.SetPipelineRunner(&fakeRunner{run: func(context.Context, *models.DeepIdentificationJob) (*DeepPipelineResult, error) {
		started <- struct{}{}
		return &DeepPipelineResult{ReportJSON: "{}"}, nil
	}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartWorkers(ctx)

	// Reproduce the production critical section deterministically: StartJob
	// publishes a queued row and wakes the worker while intake persistence is
	// still holding the exclusive lock.
	svc.intakeMu.Lock()
	job, reused, err := svc.StartJob(newDeepStartJob(t, user.ID, "intake race"))
	if err != nil || reused {
		svc.intakeMu.Unlock()
		t.Fatalf("expected a new queued job, got reused=%v err=%v", reused, err)
	}

	select {
	case <-started:
		svc.intakeMu.Unlock()
		t.Fatal("worker claimed the job before intake artifacts were ready")
	case <-time.After(100 * time.Millisecond):
	}
	svc.intakeMu.Unlock()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("worker did not claim job %d after intake became ready", job.ID)
	}
}

func TestDeepIdentificationService_CreateJobFromIntake_MissingRoleRejected(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "missing-role-owner", Email: "missing-role-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	png := tinyPNGBytes(t)
	if _, _, err := svc.CreateJobFromIntake(CreateJobInput{UserID: user.ID, ReverseBytes: png, ReverseFilename: "r.png"}); !errors.Is(err, ErrDeepJobMissingObverse) {
		t.Fatalf("expected ErrDeepJobMissingObverse, got %v", err)
	}
	if _, _, err := svc.CreateJobFromIntake(CreateJobInput{UserID: user.ID, ObverseBytes: png, ObverseFilename: "o.png"}); !errors.Is(err, ErrDeepJobMissingReverse) {
		t.Fatalf("expected ErrDeepJobMissingReverse, got %v", err)
	}
	var jobCount int64
	db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 0 {
		t.Fatalf("expected no job rows created by rejected requests, got %d", jobCount)
	}
}

func TestDeepIdentificationService_CreateJobFromIntake_SavedCoinReusesExistingImages(t *testing.T) {
	svc, db, uploadDir := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "saved-coin-owner", Email: "saved-coin-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: user.ID, Name: "Saved Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	for _, role := range []models.ImageType{models.ImageTypeObverse, models.ImageTypeReverse} {
		relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coin.ID), string(role)+".png"))
		fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, tinyPNGBytes(t), 0644); err != nil {
			t.Fatal(err)
		}
		image := models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: role}
		if err := db.Create(&image).Error; err != nil {
			t.Fatal(err)
		}
	}
	enableDeepIdentification(t, svc, nil)

	job, reused, err := svc.CreateJobFromIntake(CreateJobInput{UserID: user.ID, CoinID: &coin.ID})
	if err != nil {
		t.Fatalf("CreateJobFromIntake failed: %v", err)
	}
	if reused {
		t.Fatal("expected a new job")
	}
	if job.Source != models.DeepJobSourceSavedCoin {
		t.Fatalf("expected saved_coin source, got %s", job.Source)
	}
	artifacts, err := svc.repo.ListArtifacts(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	obverseCount, reverseCount, _ := countArtifactRoles(artifacts)
	if obverseCount != 1 || reverseCount != 1 {
		t.Fatalf("expected reused obverse+reverse artifacts, got obverse=%d reverse=%d", obverseCount, reverseCount)
	}
	for _, a := range artifacts {
		if a.Origin != models.DeepArtifactOriginSavedCoinImage {
			t.Fatalf("expected saved_coin_image origin, got %s", a.Origin)
		}
		if a.FilePath != "" {
			t.Fatalf("expected no copied file path for a reused saved-coin image, got %q", a.FilePath)
		}
	}
}

func TestDeepIdentificationService_CreateJobFromIntake_SavedCoinPartialImageMissingRoleRejected(t *testing.T) {
	// T092: a saved coin with only ONE of the two roles already photographed
	// must still be rejected with the specific missing-role error (422 at
	// the handler layer) when the caller uploads neither image for the
	// still-missing role - the presence of coinID/one existing image must
	// not silently waive the requirement for the other role.
	svc, db, uploadDir := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "partial-saved-coin-owner", Email: "partial-saved-coin-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: user.ID, Name: "Partial Saved Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	// Only the obverse image already exists on the saved coin; reverse is
	// intentionally absent, and the request supplies no reverse upload.
	relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coin.ID), "obverse.png"))
	fullPath := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, tinyPNGBytes(t), 0644); err != nil {
		t.Fatal(err)
	}
	image := models.CoinImage{CoinID: coin.ID, FilePath: relPath, ImageType: models.ImageTypeObverse}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	if _, _, err := svc.CreateJobFromIntake(CreateJobInput{UserID: user.ID, CoinID: &coin.ID}); !errors.Is(err, ErrDeepJobMissingReverse) {
		t.Fatalf("expected ErrDeepJobMissingReverse for a saved coin missing only the reverse photo, got %v", err)
	}
	var jobCount int64
	db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 0 {
		t.Fatalf("expected no job row created by the rejected partial-image saved-coin request, got %d", jobCount)
	}

	// Supplying the missing reverse upload alongside the reused obverse must
	// succeed, proving the rejection above was role-specific and not a
	// broader saved-coin regression.
	png := tinyPNGBytes(t)
	job, reused, err := svc.CreateJobFromIntake(CreateJobInput{UserID: user.ID, CoinID: &coin.ID, ReverseBytes: png, ReverseFilename: "r.png"})
	if err != nil {
		t.Fatalf("expected success once the missing reverse upload is supplied, got %v", err)
	}
	if reused {
		t.Fatal("expected a new job")
	}
	artifacts, err := svc.repo.ListArtifacts(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	obverseCount, reverseCount, _ := countArtifactRoles(artifacts)
	if obverseCount != 1 || reverseCount != 1 {
		t.Fatalf("expected one reused obverse + one uploaded reverse artifact, got obverse=%d reverse=%d", obverseCount, reverseCount)
	}
}

func TestDeepIdentificationService_CreateJobFromIntake_ForeignOwnedCoinRejected(t *testing.T) {
	// T091: creating a job with a coinId owned by a different user must be
	// rejected the same way as a nonexistent coin, never leaking whether the
	// coin exists under another account.
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	owner := models.User{Username: "coin-owner", Email: "coin-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	requester := models.User{Username: "not-the-owner", Email: "not-the-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: owner.ID, Name: "Someone Else's Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	png := tinyPNGBytes(t)
	if _, _, err := svc.CreateJobFromIntake(CreateJobInput{
		UserID: requester.ID, CoinID: &coin.ID,
		ObverseBytes: png, ObverseFilename: "o.png", ReverseBytes: png, ReverseFilename: "r.png",
	}); !errors.Is(err, ErrDeepArtifactMissingCoin) {
		t.Fatalf("expected ErrDeepArtifactMissingCoin for a foreign-owned coinId, got %v", err)
	}
	var jobCount int64
	db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 0 {
		t.Fatalf("expected no job row created for a foreign-owned coinId, got %d", jobCount)
	}
}

func TestDeepIdentificationService_CreateJobFromIntake_DuplicateSubmitIsIdempotent(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "dup-owner", Email: "dup-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, nil)

	png := tinyPNGBytes(t)
	in := CreateJobInput{UserID: user.ID, ObverseBytes: png, ObverseFilename: "o.png", ReverseBytes: png, ReverseFilename: "r.png"}
	first, reused1, err := svc.CreateJobFromIntake(in)
	if err != nil || reused1 {
		t.Fatalf("expected first submit to create a new job, got reused=%v err=%v", reused1, err)
	}
	second, reused2, err := svc.CreateJobFromIntake(in)
	if err != nil {
		t.Fatalf("second submit failed: %v", err)
	}
	if !reused2 || second.ID != first.ID {
		t.Fatalf("expected the duplicate submit to reuse job %d, got reused=%v id=%d", first.ID, reused2, second.ID)
	}
	var jobCount int64
	db.Model(&models.DeepIdentificationJob{}).Count(&jobCount)
	if jobCount != 1 {
		t.Fatalf("expected exactly one job row, got %d", jobCount)
	}
}

func TestDeepIdentificationService_RetryJob_LineageAndDepthCap(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "retry-owner", Email: "retry-owner@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	enableDeepIdentification(t, svc, map[string]string{SettingDeepIdentificationMaxActivePerUser: "10"})

	png := tinyPNGBytes(t)
	original, _, err := svc.CreateJobFromIntake(CreateJobInput{
		UserID: user.ID, Notes: "original",
		ObverseBytes: png, ObverseFilename: "o.png", ReverseBytes: png, ReverseFilename: "r.png",
	})
	if err != nil {
		t.Fatalf("failed to create original job: %v", err)
	}

	// Retry before the source job is terminal is rejected.
	if _, _, err := svc.RetryJob(original.ID, user.ID, nil, nil); !errors.Is(err, ErrDeepJobNotTerminal) {
		t.Fatalf("expected ErrDeepJobNotTerminal, got %v", err)
	}

	current := original
	for depth := 1; depth <= MaxDeepIdentificationRetryDepth; depth++ {
		if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", current.ID).
			Updates(map[string]interface{}{"status": models.DeepJobStatusCompleted, "completed_at": time.Now()}).Error; err != nil {
			t.Fatal(err)
		}
		retryNotes := fmt.Sprintf("retry depth %d", depth)
		next, reused, err := svc.RetryJob(current.ID, user.ID, &retryNotes, nil)
		if err != nil {
			t.Fatalf("retry at depth %d failed: %v", depth, err)
		}
		if reused {
			t.Fatalf("retry at depth %d unexpectedly reused an existing job", depth)
		}
		if next.RetryOfJobID == nil || *next.RetryOfJobID != current.ID {
			t.Fatalf("retry at depth %d: expected RetryOfJobID=%d, got %v", depth, current.ID, next.RetryOfJobID)
		}
		if next.RetryDepth != depth {
			t.Fatalf("retry at depth %d: expected RetryDepth=%d, got %d", depth, depth, next.RetryDepth)
		}
		current = next
	}

	// One more retry beyond the cap is rejected.
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", current.ID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusCompleted, "completed_at": time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RetryJob(current.ID, user.ID, nil, nil); !errors.Is(err, ErrDeepJobRetryDepth) {
		t.Fatalf("expected ErrDeepJobRetryDepth, got %v", err)
	}

	// The original job's events/report must remain untouched by all this.
	var reloadedOriginal models.DeepIdentificationJob
	if err := db.First(&reloadedOriginal, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloadedOriginal.Notes != "original" {
		t.Fatalf("expected original job notes untouched, got %q", reloadedOriginal.Notes)
	}
}
