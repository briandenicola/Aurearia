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
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	feature362CompatUserID        = uint(36201)
	feature362CompatDraftID       = uint(36202)
	feature362CompatSupportedJob  = uint(36203)
	feature362CompatDraftJob      = uint(36204)
	feature362CompatHandoffID     = uint(36205)
	feature362CompatCoinID        = uint(36206)
	feature362CompatSavedJob      = uint(36207)
	feature362CompatSavedHandoff  = uint(36208)
	feature362CompatCancelJob     = uint(36209)
	feature362CompatCancelHandoff = uint(36210)
	feature362CompatJWTSecret     = "feature362-compatibility-secret-at-least-32-characters"
)

func feature362CompatibilityDigest(db *gorm.DB, artifactPath string) (string, error) {
	var supported, draft, saved, cancelled models.DeepIdentificationJob
	var handoffs []models.CoinCopilotDeepHandoff
	var setting models.AppSetting
	if err := db.First(&supported, feature362CompatSupportedJob).Error; err != nil {
		return "", err
	}
	if err := db.First(&draft, feature362CompatDraftJob).Error; err != nil {
		return "", err
	}
	if err := db.First(&saved, feature362CompatSavedJob).Error; err != nil {
		return "", err
	}
	if err := db.First(&cancelled, feature362CompatCancelJob).Error; err != nil {
		return "", err
	}
	if err := db.Where("id IN ?", []uint{
		feature362CompatHandoffID,
		feature362CompatSavedHandoff,
		feature362CompatCancelHandoff,
	}).Order("id").Find(&handoffs).Error; err != nil {
		return "", err
	}
	if len(handoffs) != 3 {
		return "", fmt.Errorf("expected 3 Feature 362 handoffs, got %d", len(handoffs))
	}
	if err := db.Where("key = ?", "CoinCopilotAttributionEnabled").First(&setting).Error; err != nil {
		return "", err
	}
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%v|%x", []any{
		supported.ID, supported.Source, supported.Status,
		draft.ID, *draft.SourceDraftID, draft.Source, draft.Status,
		draft.AttemptCount, draft.ReportJSON,
		saved.ID, *saved.CoinID, saved.Source, saved.Status, saved.ProposalJSON,
		cancelled.ID, *cancelled.SourceDraftID, cancelled.Source, cancelled.Status,
		handoffs[0].DeepJobID, handoffs[1].DeepJobID, handoffs[2].DeepJobID,
		setting.Value,
	}, artifact)
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
	coin := models.Coin{
		ID: feature362CompatCoinID, UserID: user.ID,
		Name: "Compatibility coin", Category: models.CategoryRoman,
		Denomination: "Original denomination",
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
		Source: models.DeepJobSourceCopilotDraft, Status: models.DeepJobStatusCompleted,
		InputFingerprint: "feature362-copilot-draft", ReportJSON: `{"narrative":"preserve"}`,
		ProposalJSON: `{"schemaVersion":1,"fields":{"workingTitle":{"proposed":"Attributed draft","accepted":true}}}`,
		CompletedAt:  &now, ExpiresAt: models.DeepIdentificationNoExpirySentinel,
		ActiveKey: fmt.Sprint(feature362CompatDraftJob),
	}
	savedJob := models.DeepIdentificationJob{
		ID: feature362CompatSavedJob, UserID: user.ID, CoinID: &coin.ID,
		Source: models.DeepJobSourceSavedCoin, Status: models.DeepJobStatusCompleted,
		InputFingerprint: "feature362-saved-coin", ReportJSON: `{"narrative":"saved"}`,
		ProposalJSON: `{"schemaVersion":1,"fields":{"denomination":{"proposed":"Compat Denarius","accepted":true}}}`,
		CompletedAt:  &now, ExpiresAt: models.DeepIdentificationNoExpirySentinel,
		ActiveKey: fmt.Sprint(feature362CompatSavedJob),
	}
	cancelJob := models.DeepIdentificationJob{
		ID: feature362CompatCancelJob, UserID: user.ID, SourceDraftID: &draft.ID,
		Source: models.DeepJobSourceCopilotDraft, Status: models.DeepJobStatusQueued,
		InputFingerprint: "feature362-cancel-draft",
		ExpiresAt:        time.Now().Add(24 * time.Hour), ActiveKey: "active",
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
	savedHandoff := models.CoinCopilotDeepHandoff{
		ID: feature362CompatSavedHandoff, UserID: user.ID,
		RunID: "ccr_saved", ExecutionID: "cce_saved",
		HandoffKeyHash: "saved-a", RequestFingerprint: "saved-b", AppContextDigest: "saved-c",
		Operation:  models.CoinCopilotDeepHandoffOperationRequest,
		TargetKind: models.CoinCopilotDeepHandoffTargetCoin, TargetID: coin.ID,
		TargetSnapshotFingerprint: "saved-d", DeepJobID: savedJob.ID,
		AdmissionOutcome: models.CoinCopilotDeepHandoffOutcomeCreated,
	}
	cancelHandoff := models.CoinCopilotDeepHandoff{
		ID: feature362CompatCancelHandoff, UserID: user.ID,
		RunID: "ccr_cancel", ExecutionID: "cce_cancel",
		HandoffKeyHash: "cancel-a", RequestFingerprint: "cancel-b", AppContextDigest: "cancel-c",
		Operation:  models.CoinCopilotDeepHandoffOperationRequest,
		TargetKind: models.CoinCopilotDeepHandoffTargetDraft, TargetID: draft.ID,
		TargetSnapshotFingerprint: "cancel-d", DeepJobID: cancelJob.ID,
		AdmissionOutcome: models.CoinCopilotDeepHandoffOutcomeCreated,
	}
	for _, value := range []any{
		&user, &draft, &coin, &supported, &draftJob, &savedJob, &cancelJob,
		&handoff, &savedHandoff, &cancelHandoff,
	} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, setting := range []models.AppSetting{
		{Key: "CoinCopilotAttributionEnabled", Value: "false"},
		{Key: "DeepIdentificationEnabled", Value: "false"},
	} {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatal(err)
		}
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(user.ID),
		"role":   string(models.RoleUser),
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(feature362CompatJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath+".token", []byte(signed), 0o600); err != nil {
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
	if draftJob.Status != models.DeepJobStatusCompleted ||
		draftJob.AttemptCount != 0 || draftJob.LastSeq != 0 {
		t.Fatalf("settled Feature 362 job changed across rollback: %#v", draftJob)
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
		t.Fatalf("rollback produced events=%d provider_runs=%d", events, providerRuns)
	}
	if _, err := repository.NewDeepIdentificationRepository(db).
		GetJob(feature362CompatSupportedJob, feature362CompatUserID); err != nil {
		t.Fatalf("supported source became unreadable: %v", err)
	}
	var savedJob, cancelJob models.DeepIdentificationJob
	if err := db.First(&savedJob, feature362CompatSavedJob).Error; err != nil {
		t.Fatal(err)
	}
	if savedJob.AppliedCoinID == nil || *savedJob.AppliedCoinID != feature362CompatCoinID {
		t.Fatalf("saved-coin apply was not restored after re-upgrade: %#v", savedJob)
	}
	if err := db.First(&cancelJob, feature362CompatCancelJob).Error; err != nil {
		t.Fatal(err)
	}
	if cancelJob.Status != models.DeepJobStatusCancelled || cancelJob.CancelRequestedAt == nil {
		t.Fatalf("queued handoff was not durably cancelled: %#v", cancelJob)
	}
	var coin models.Coin
	if err := db.First(&coin, feature362CompatCoinID).Error; err != nil {
		t.Fatal(err)
	}
	if coin.Denomination != "Compat Denarius" {
		t.Fatalf("saved-coin proposal was not applied: denomination=%q", coin.Denomination)
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
	case "snapshot":
		database.Connect(dbPath)
		digest, err := feature362CompatibilityDigest(database.DB, artifactPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath+".digest", []byte(digest), 0o600); err != nil {
			t.Fatal(err)
		}
	case "verify-guard":
		database.Connect(dbPath)
		want, err := os.ReadFile(dbPath + ".digest")
		if err != nil {
			t.Fatal(err)
		}
		got, err := feature362CompatibilityDigest(database.DB, artifactPath)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("guard changed Feature 362 rows or artifact: got %s want %s", got, want)
		}
	case "verify-feature":
		verifyFeature362CompatibilityDatabase(t, dbPath, artifactPath)
	default:
		t.Fatalf("unknown FEATURE362_COMPAT_MODE %q", mode)
	}
}
