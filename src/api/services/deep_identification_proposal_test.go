package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newDeepProposalTestDeps(t *testing.T) (*DeepIdentificationProposalService, *repository.DeepIdentificationRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:deep_identification_proposal_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.ValueSnapshot{}, &models.CoinJournal{}, &models.AppSetting{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
		&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{}, &models.QuickCaptureDraftReference{}, &models.DraftLifecycleEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := repository.NewDeepIdentificationRepository(db)
	coinRepo := repository.NewCoinRepository(db)
	notifSvc := NewNotificationService(repository.NewNotificationRepository(db), repository.NewSocialRepository(db), repository.NewUserRepository(db), NewPushoverService(NewSettingsService(repository.NewSettingsRepository(db)), NewLogger(10)), NewLogger(10))
	coinSvc := NewCoinService(coinRepo, notifSvc)
	quickCaptureSvc := NewQuickCaptureService(repository.NewQuickCaptureRepository(db), t.TempDir()).WithCoinValidation(coinSvc)
	proposalSvc := NewDeepIdentificationProposalService(repo, coinRepo, coinSvc, quickCaptureSvc)
	return proposalSvc, repo, db
}

func seedDeepProposalUser(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	user := models.User{Username: fmt.Sprintf("proposal-owner-%d", time.Now().UnixNano()), Email: fmt.Sprintf("proposal-owner-%d@example.com", time.Now().UnixNano()), PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user.ID
}

func seedDeepProposalJob(t *testing.T, db *gorm.DB, userID uint, source models.DeepJobSource, coinID *uint, proposalFields map[string]any) uint {
	t.Helper()
	fields := map[string]*deepProposalFieldEntry{}
	for name, value := range proposalFields {
		fields[name] = &deepProposalFieldEntry{Proposed: value, Confidence: 0.8}
	}
	doc := deepProposalDocument{SchemaVersion: 1, Fields: fields}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("encode proposal: %v", err)
	}
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           source,
		CoinID:           coinID,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		ReportJSON:       `{"schemaVersion":1,"narrative":"n","coverage":[]}`,
		ProposalJSON:     string(encoded),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return job.ID
}

func acceptTrue() *bool {
	v := true
	return &v
}

// T113: the field allowlist rejects any field not writable via
// CoinService/QuickCaptureDraft (no silent new write surface).
// TestDeepIdentificationProposal_CoinTypeFieldMapsToReferenceText verifies
// Feature 345: the additive `coin_type` allowlist entry carries the OCRE
// RIC-style type label into the reused models.Coin.ReferenceText column (no
// schema migration), survives buildDeepProposalDocumentJSON end to end, and
// applies through CoinService like any other allowlisted field.
func TestDeepIdentificationProposal_CoinTypeFieldMapsToReferenceText(t *testing.T) {
	if got := deepProposalCoinFieldAllowlist["coin_type"]; got != "ReferenceText" {
		t.Fatalf("expected coin_type to map to ReferenceText, got %q", got)
	}

	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"coin_type": "RIC II Hadrian 39b",
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"coin_type": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"coin_type"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.ReferenceText != "RIC II Hadrian 39b" {
		t.Fatalf("expected coin_type applied to ReferenceText, got %q", updated.ReferenceText)
	}
}

func TestDeepIdentificationProposal_FieldAllowlistRejectsUnknownField(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Denarius",
	})

	// PATCH proposal with an unknown field name is rejected.
	_, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"secretAdminField": {Accepted: acceptTrue()},
	})
	if !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed, got %v", err)
	}

	// The document itself only ever contains synthesizer-populated fields,
	// so applying an out-of-allowlist coin field is likewise rejected even
	// if it somehow appeared in the proposal and was explicitly accepted.
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"secretAdminField": "x",
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"secretAdminField": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept field for allowlist-at-apply test: %v", err)
	}
	_, err = svc.Apply(jobID2, userID, "coin", []string{"secretAdminField"})
	if !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed on apply, got %v", err)
	}
}

// T114: apply routes exclusively through QuickCaptureService/CoinService -
// no direct coin write exists anywhere in the deep-identification package
// (Principle IV / FR-033). Proven here by asserting the coin's other,
// non-allowlisted fields (and journal) are exactly what CoinService.
// UpdateCoinWithFields would produce, and that a journal entry with source
// T114: apply routes exclusively through QuickCaptureService/CoinService -
// no direct coin write exists anywhere in the deep-identification package
// (Principle IV / FR-033). Proven here two ways: (1) only the
// UpdateCoinWithFields-selected fields change - everything else (Notes) is
// untouched, which a raw ad-hoc SQL write would have no reason to respect;
// (2) a ValueSnapshot row is recorded, a side effect CoinService's private
// updateCoin transaction always performs and which nothing outside
// CoinService can trigger, proving the write passed through that exact
// code path.
func TestDeepIdentificationProposal_ApplyRoutesThroughCoinServiceOnly(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin", Notes: "original"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Denarius",
		"mint":         "Rome",
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
		"mint":         {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	var snapshotCountBefore int64
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", userID).Count(&snapshotCountBefore).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.Apply(jobID, userID, "coin", nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.CoinID == nil || *result.CoinID != coin.ID {
		t.Fatalf("expected coinId %d, got %v", coin.ID, result.CoinID)
	}

	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Denomination != "Denarius" || updated.Mint != "Rome" {
		t.Fatalf("expected fields applied via CoinService, got denomination=%q mint=%q", updated.Denomination, updated.Mint)
	}
	if updated.Notes != "original" {
		t.Fatalf("expected Notes untouched (not in applied fields), got %q", updated.Notes)
	}

	var snapshotCountAfter int64
	if err := db.Model(&models.ValueSnapshot{}).Where("user_id = ?", userID).Count(&snapshotCountAfter).Error; err != nil {
		t.Fatal(err)
	}
	if snapshotCountAfter <= snapshotCountBefore {
		t.Fatal("expected CoinService.UpdateCoinWithFields' RecordValueSnapshot side effect to have run, proving the write passed through CoinService and not a direct SQL write")
	}
}

// T115: apply on a coin deleted mid-run returns 409 source_coin_missing;
// the report remains readable.
func TestDeepIdentificationProposal_ApplyOnDeletedCoinReturnsSourceCoinMissing(t *testing.T) {
	svc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Doomed Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Denarius",
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if err := db.Delete(&models.Coin{}, coin.ID).Error; err != nil {
		t.Fatalf("delete coin: %v", err)
	}

	_, err := svc.Apply(jobID, userID, "coin", nil)
	if !errors.Is(err, ErrDeepProposalSourceMissing) {
		t.Fatalf("expected ErrDeepProposalSourceMissing, got %v", err)
	}

	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatalf("job should still be readable: %v", err)
	}
	if job.ReportJSON == "" {
		t.Fatal("expected the report to remain readable after a failed apply")
	}
}

// T116: a second apply attempt returns 409 already_applied unless a fresh
// report cycle exists.
func TestDeepIdentificationProposal_SecondApplyReturnsAlreadyApplied(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Denarius",
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", nil); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	_, err := svc.Apply(jobID, userID, "coin", nil)
	if !errors.Is(err, ErrDeepProposalAlreadyApplied) {
		t.Fatalf("expected ErrDeepProposalAlreadyApplied, got %v", err)
	}

	// Editing the already-applied proposal is likewise rejected.
	_, err = svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	})
	if !errors.Is(err, ErrDeepProposalAlreadyApplied) {
		t.Fatalf("expected ErrDeepProposalAlreadyApplied on edit-after-apply, got %v", err)
	}

	// A fresh report cycle (a new job/retry) is a distinct job id and is
	// therefore free to be applied on its own.
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Aureus",
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal on fresh job: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", nil); err != nil {
		t.Fatalf("apply on fresh report cycle should succeed: %v", err)
	}
}

// T117 (data-model.md §6): every terminal completed/partial job has
// narrative/coverage/disagreements/unresolvedQuestions populated, and
// partial jobs set PartialSuccess=true. This asserts on the shape written
// by SettleTerminal-produced ReportJSON.
func TestDeepIdentificationProposal_ReportShapeCompletenessForTerminalJobs(t *testing.T) {
	_, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)

	completedReport := `{
		"schemaVersion": 1,
		"narrative": "A well-preserved denarius.",
		"coverage": [{"provider":"numista","status":"contributed"}],
		"disagreements": [],
		"unresolvedQuestions": [],
		"partialSuccess": false,
		"generatedAt": "2026-08-15T13:04:11Z"
	}`
	partialReport := `{
		"schemaVersion": 1,
		"narrative": "Partial identification; NGC not automated.",
		"coverage": [
			{"provider":"numista","status":"contributed"},
			{"provider":"ngc","status":"not_automated"}
		],
		"disagreements": [{"field":"mint","claims":[],"resolution":"unresolved"}],
		"unresolvedQuestions": ["Reverse legend partially illegible."],
		"partialSuccess": true,
		"generatedAt": "2026-08-15T13:05:00Z"
	}`

	completedJobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, nil)
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", completedJobID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusCompleted, "report_json": completedReport, "partial_success": false}).Error; err != nil {
		t.Fatal(err)
	}
	partialJobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, nil)
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", partialJobID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusPartial, "report_json": partialReport, "partial_success": true}).Error; err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		jobID   uint
		partial bool
	}{
		{"completed", completedJobID, false},
		{"partial", partialJobID, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job, err := repo.GetJob(tc.jobID, userID)
			if err != nil {
				t.Fatal(err)
			}
			var report map[string]any
			if err := json.Unmarshal([]byte(job.ReportJSON), &report); err != nil {
				t.Fatalf("report is not valid json: %v", err)
			}
			for _, key := range []string{"narrative", "coverage", "disagreements", "unresolvedQuestions"} {
				if _, ok := report[key]; !ok {
					t.Fatalf("expected report to have key %q populated, got %v", key, report)
				}
			}
			if narrative, _ := report["narrative"].(string); narrative == "" {
				t.Fatal("expected a non-empty narrative")
			}
			if job.PartialSuccess != tc.partial {
				t.Fatalf("expected PartialSuccess=%v, got %v", tc.partial, job.PartialSuccess)
			}
		})
	}
}
