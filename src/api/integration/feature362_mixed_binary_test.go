package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/database"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

const (
	feature362CompatUserID       = uint(36201)
	feature362CompatDraftID      = uint(36202)
	feature362CompatSupportedJob = uint(36203)
	feature362CompatDraftJob     = uint(36204)
	feature362CompatHandoffID    = uint(36205)
)

func feature362CompatibilityDigest(db *gorm.DB, artifactPath string) (string, error) {
	var supported, draft models.DeepIdentificationJob
	var handoff models.CoinCopilotDeepHandoff
	var setting models.AppSetting
	if err := db.First(&supported, feature362CompatSupportedJob).Error; err != nil {
		return "", err
	}
	if err := db.First(&draft, feature362CompatDraftJob).Error; err != nil {
		return "", err
	}
	if err := db.First(&handoff, feature362CompatHandoffID).Error; err != nil {
		return "", err
	}
	if err := db.Where("key = ?", "CoinCopilotAttributionEnabled").First(&setting).Error; err != nil {
		return "", err
	}
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf(
		"%d|%s|%s|%d|%d|%s|%s|%d|%s|%s|%x",
		supported.ID, supported.Source, supported.Status,
		draft.ID, *draft.SourceDraftID, draft.Source, draft.Status,
		draft.AttemptCount, draft.ReportJSON, setting.Value, artifact,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}

func seedFeature362CompatibilityDatabase(t *testing.T, dbPath, artifactPath string) {
	t.Helper()
	database.Connect(dbPath)
	db := database.DB
	now := time.Now().UTC()
	user := models.User{
		ID: feature362CompatUserID, Username: "feature362-compat",
		Email: "feature362-compat@example.test", PasswordHash: "x",
	}
	draft := models.QuickCaptureDraft{
		ID: feature362CompatDraftID, UserID: user.ID,
		WorkingTitle: "Compatibility draft", Status: models.QuickCaptureDraftStatusActive,
	}
	supported := models.DeepIdentificationJob{
		ID: feature362CompatSupportedJob, UserID: user.ID,
		Source: models.DeepJobSourceIntake, Status: models.DeepJobStatusCompleted,
		InputFingerprint: "feature362-supported", ReportJSON: `{"narrative":"supported"}`,
		CompletedAt: &now, ExpiresAt: models.DeepIdentificationNoExpirySentinel,
		ActiveKey: fmt.Sprint(feature362CompatSupportedJob),
	}
	draftJob := models.DeepIdentificationJob{
		ID: feature362CompatDraftJob, UserID: user.ID, SourceDraftID: &draft.ID,
		Source: models.DeepJobSourceCopilotDraft, Status: models.DeepJobStatusQueued,
		InputFingerprint: "feature362-copilot-draft", ReportJSON: `{"narrative":"preserve"}`,
		ExpiresAt: time.Now().Add(24 * time.Hour), ActiveKey: "active",
	}
	handoff := models.CoinCopilotDeepHandoff{
		ID: feature362CompatHandoffID, UserID: user.ID,
		RunID: "ccr_compat", ExecutionID: "cce_compat",
		HandoffKeyHash: "a", RequestFingerprint: "b", AppContextDigest: "c",
		Operation:  models.CoinCopilotDeepHandoffOperationRequest,
		TargetKind: models.CoinCopilotDeepHandoffTargetDraft,
		TargetID:   draft.ID, TargetSnapshotFingerprint: "d",
		DeepJobID:        draftJob.ID,
		AdmissionOutcome: models.CoinCopilotDeepHandoffOutcomeCreated,
	}
	for _, value := range []any{&user, &draft, &supported, &draftJob, &handoff} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.AppSetting{
		Key: "CoinCopilotAttributionEnabled", Value: "false",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("feature362-compatibility-artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := feature362CompatibilityDigest(db, artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath+".digest", []byte(digest), 0o600); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}

func verifyFeature362CompatibilityDatabase(t *testing.T, dbPath, artifactPath string) {
	t.Helper()
	database.Connect(dbPath)
	db := database.DB
	want, err := os.ReadFile(dbPath + ".digest")
	if err != nil {
		t.Fatal(err)
	}
	got, err := feature362CompatibilityDigest(db, artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("Feature 362 rows or artifact changed: got %s want %s", got, want)
	}
	draftJob, err := repository.NewDeepIdentificationRepository(db).
		GetJob(feature362CompatDraftJob, feature362CompatUserID)
	if err != nil {
		t.Fatal(err)
	}
	if draftJob.Status != models.DeepJobStatusQueued ||
		draftJob.AttemptCount != 0 || draftJob.LastSeq != 0 {
		t.Fatalf("guard adopted Feature 362 job: %#v", draftJob)
	}
	var events, providerRuns int64
	if err := db.Model(&models.DeepIdentificationEvent{}).
		Where("job_id = ?", draftJob.ID).Count(&events).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationProviderRun{}).
		Where("job_id = ?", draftJob.ID).Count(&providerRuns).Error; err != nil {
		t.Fatal(err)
	}
	if events != 0 || providerRuns != 0 {
		t.Fatalf("guard produced events=%d provider_runs=%d", events, providerRuns)
	}
	if _, err := repository.NewDeepIdentificationRepository(db).
		GetJob(feature362CompatSupportedJob, feature362CompatUserID); err != nil {
		t.Fatalf("supported source became unreadable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFeature362MixedBinaryDatabaseCompatibility(t *testing.T) {
	mode := os.Getenv("FEATURE362_COMPAT_MODE")
	dbPath := os.Getenv("FEATURE362_COMPAT_DB")
	artifactPath := os.Getenv("FEATURE362_COMPAT_ARTIFACT")
	if mode == "" {
		t.Skip("run through scripts/compat/feature362-rollback.ps1")
	}
	if dbPath == "" || artifactPath == "" {
		t.Fatal("FEATURE362_COMPAT_DB and FEATURE362_COMPAT_ARTIFACT are required")
	}
	switch mode {
	case "seed":
		seedFeature362CompatibilityDatabase(t, dbPath, artifactPath)
	case "verify-guard", "verify-feature":
		verifyFeature362CompatibilityDatabase(t, dbPath, artifactPath)
	default:
		t.Fatalf("unknown FEATURE362_COMPAT_MODE %q", mode)
	}
}
