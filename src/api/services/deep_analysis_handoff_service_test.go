package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupFeature362StatusService(t *testing.T) (*DeepAnalysisHandoffService, *gorm.DB, *CopilotExecutionClaims) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{},
		&models.QuickCaptureDraft{}, &models.QuickCaptureDraftImage{},
		&models.QuickCaptureDraftReference{},
		&models.DeepIdentificationJob{}, &models.CoinCopilotRun{},
		&models.CoinCopilotDeepHandoff{}, &models.AppSetting{},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{ID: 7, Username: "owner", Email: "owner@example.test", PasswordHash: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	run := models.CoinCopilotRun{
		ID: "ccr_status", ThreadID: "cct_status", UserID: 7, Status: models.CopilotRunRunning,
		Goal: "status", StartIdempotencyKeyHash: "start", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_status", CheckpointVersion: 3, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(repository.NewSettingsRepository(db))
	svc := NewDeepAnalysisHandoffService(
		repository.NewCoinCopilotRepository(db),
		repository.NewDeepIdentificationRepository(db),
		repository.NewCoinRepository(db),
		repository.NewQuickCaptureRepository(db),
		nil, settings, "",
	)
	return svc, db, &CopilotExecutionClaims{UserID: 7, RunID: run.ID, ExecutionID: run.ExecutionID}
}

func TestFeature362NewAdmissionFailsClosedAtEveryLiveFeatureGate(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]string
		reason   string
	}{
		{
			name: "Coin Copilot disabled",
			settings: map[string]string{
				SettingCoinCopilotEnabled:            "false",
				SettingCoinCopilotAttributionEnabled: "true",
				SettingDeepIdentificationEnabled:     "true",
			},
			reason: "copilot_disabled",
		},
		{
			name: "attribution disabled",
			settings: map[string]string{
				SettingCoinCopilotEnabled:            "true",
				SettingCoinCopilotAttributionEnabled: "false",
				SettingDeepIdentificationEnabled:     "true",
			},
			reason: "attribution_disabled",
		},
		{
			name: "Deep Analysis disabled",
			settings: map[string]string{
				SettingCoinCopilotEnabled:            "true",
				SettingCoinCopilotAttributionEnabled: "true",
				SettingDeepIdentificationEnabled:     "false",
			},
			reason: "deep_disabled",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, db, claims := setupFeature362StatusService(t)
			for key, value := range test.settings {
				if err := svc.settingsSvc.SetSetting(key, value); err != nil {
					t.Fatal(err)
				}
			}
			for _, operation := range []string{"request", "rerun"} {
				target := DeepAnalysisHandoffTarget{Type: "coin", ID: 999}
				request := DeepAnalysisHandoffRequest{
					ToolCallID: "call_gate_" + operation, ExpectedCheckpointVersion: 3,
					Operation: operation, Target: &target,
					HandoffIdempotencyKey: strings.Repeat("a", 32),
				}
				if operation == "rerun" {
					jobID := uint(1)
					request.JobID = &jobID
				}
				got, err := svc.Execute(claims, request)
				if err != nil {
					t.Fatal(err)
				}
				if got.Outcome != "unavailable" || got.Reason == nil || *got.Reason != test.reason {
					t.Fatalf("%s got %#v, want unavailable/%s", operation, got, test.reason)
				}
			}
			var jobs, handoffs int64
			if err := db.Model(&models.DeepIdentificationJob{}).Count(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.CoinCopilotDeepHandoff{}).Count(&handoffs).Error; err != nil {
				t.Fatal(err)
			}
			if jobs != 0 || handoffs != 0 {
				t.Fatalf("disabled admission persisted jobs=%d handoffs=%d", jobs, handoffs)
			}
		})
	}
}

func TestFeature362AcceptedStatusRemainsReadableWhenAllGatesAreDisabled(t *testing.T) {
	svc, db, claims := setupFeature362StatusService(t)
	coin := models.Coin{ID: 71, UserID: claims.UserID, Name: "Accepted coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
		UserID: claims.UserID, CoinID: &coin.ID, Source: models.DeepJobSourceSavedCoin,
		Status: models.DeepJobStatusRunning,
	})
	for _, key := range []string{
		SettingCoinCopilotEnabled,
		SettingCoinCopilotAttributionEnabled,
		SettingDeepIdentificationEnabled,
	} {
		if err := svc.settingsSvc.SetSetting(key, "false"); err != nil {
			t.Fatal(err)
		}
	}

	got, err := svc.Execute(claims, feature362StatusRequest(jobID))
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "status" || got.Job == nil || got.Job.ID != jobID {
		t.Fatalf("accepted status was stranded after disable: %#v", got)
	}
}

func feature362StatusRequest(jobID uint) DeepAnalysisHandoffRequest {
	return DeepAnalysisHandoffRequest{
		ToolCallID: "call_status", ExpectedCheckpointVersion: 3,
		Operation: "status", JobID: &jobID,
	}
}

func TestFeature362ResolveTargetPreservesRerunOperation(t *testing.T) {
	svc, db, claims := setupFeature362StatusService(t)
	tests := []struct {
		name   string
		status models.QuickCaptureDraftStatus
		reason string
	}{
		{"inactive draft", models.QuickCaptureDraftStatusDiscarded, "draft_inactive"},
		{"missing images", models.QuickCaptureDraftStatusActive, "missing_both"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := models.QuickCaptureDraft{
				ID: uint(900 + index), UserID: claims.UserID,
				WorkingTitle: test.name, Status: test.status,
			}
			if err := db.Create(&draft).Error; err != nil {
				t.Fatal(err)
			}
			_, outcome, err := svc.resolveTarget(
				claims.UserID,
				"rerun",
				DeepAnalysisHandoffTarget{Type: "draft", ID: draft.ID},
			)
			if err != nil {
				t.Fatal(err)
			}
			if outcome == nil || outcome.Operation != "rerun" ||
				outcome.Reason == nil || *outcome.Reason != test.reason {
				t.Fatalf("outcome=%+v, want rerun/%s", outcome, test.reason)
			}
		})
	}
}

func seedFeature362StatusJob(t *testing.T, db *gorm.DB, job models.DeepIdentificationJob) uint {
	t.Helper()
	if job.InputFingerprint == "" {
		job.InputFingerprint = handoffSHA256Hex(time.Now().String() + string(job.Source))
	}
	if job.ExpiresAt.IsZero() {
		job.ExpiresAt = models.DeepIdentificationNoExpirySentinel
	}
	if job.ActiveKey == "" {
		job.ActiveKey = time.Now().Format("150405.000000000")
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	return job.ID
}

func TestFeature362StatusClosedEligibilityAndPersistedResultVariants(t *testing.T) {
	svc, db, claims := setupFeature362StatusService(t)
	coin := models.Coin{ID: 41, UserID: claims.UserID, Name: "Owned denarius"}
	draft := models.QuickCaptureDraft{
		ID: 42, UserID: claims.UserID, WorkingTitle: "Active draft",
		Status: models.QuickCaptureDraftStatusActive,
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	completeReport := `{
		"narrative":"Persisted complete result.",
		"proposed_fields":{"ruler":{"value":"Maximinus I","confidence":0.44,"evidence_refs":[{"provider":"image"}]}},
		"disagreements":[],
		"unresolved_questions":["Reverse legend remains uncertain."],
		"coverage":[{"provider":"numista","status":"no_match"}],
		"attributions":[],
		"partial_success":false
	}`
	tests := []struct {
		name       string
		job        models.DeepIdentificationJob
		wantState  string
		wantResult bool
	}{
		{
			name: "current owned saved coin queued lifecycle only",
			job: models.DeepIdentificationJob{
				UserID: claims.UserID, CoinID: &coin.ID, Source: models.DeepJobSourceSavedCoin,
				Status: models.DeepJobStatusQueued,
			},
			wantResult: false,
		},
		{
			name: "active owned copilot draft running lifecycle only",
			job: models.DeepIdentificationJob{
				UserID: claims.UserID, SourceDraftID: &draft.ID, Source: models.DeepJobSourceCopilotDraft,
				Status: models.DeepJobStatusRunning,
			},
			wantResult: false,
		},
		{
			name: "retained completed report survives pruned events",
			job: models.DeepIdentificationJob{
				UserID: claims.UserID, CoinID: &coin.ID, Source: models.DeepJobSourceSavedCoin,
				Status: models.DeepJobStatusCompleted, ReportJSON: completeReport,
				ProposalJSON: `{"schemaVersion":1,"fields":{"ruler":{"proposed":"Maximinus I","confidence":0.44,"ownerEdited":false,"ownerValue":null,"accepted":null}}}`,
				CompletedAt:  &now, EventsPrunedAt: &now,
			},
			wantState: "complete", wantResult: true,
		},
		{
			name: "retained partial report",
			job: models.DeepIdentificationJob{
				UserID: claims.UserID, SourceDraftID: &draft.ID, Source: models.DeepJobSourceCopilotDraft,
				Status: models.DeepJobStatusPartial, ReportJSON: completeReport, PartialSuccess: true,
				ProposalJSON: `{"schemaVersion":1,"fields":{"ruler":{"proposed":"Maximinus I","confidence":0.44,"ownerEdited":false,"ownerValue":null,"accepted":null}}}`,
				CompletedAt:  &now,
			},
			wantState: "partial", wantResult: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jobID := seedFeature362StatusJob(t, db, test.job)
			got, err := svc.Execute(claims, feature362StatusRequest(jobID))
			if err != nil {
				t.Fatal(err)
			}
			if got.Outcome != "status" || got.Job == nil || got.Job.ID != jobID {
				t.Fatalf("unexpected eligible status: %#v", got)
			}
			wantURL := "/deep-analysis/" + fmt.Sprint(jobID)
			if got.ReviewURL != wantURL {
				t.Fatalf("review_url=%q want %q", got.ReviewURL, wantURL)
			}
			if (got.Result != nil) != test.wantResult {
				t.Fatalf("result presence=%v want %v: %#v", got.Result != nil, test.wantResult, got)
			}
			if got.Result != nil && got.Result.State != test.wantState {
				t.Fatalf("result state=%q want %q", got.Result.State, test.wantState)
			}
		})
	}
	var handoffRows int64
	if err := db.Model(&models.CoinCopilotDeepHandoff{}).Count(&handoffRows).Error; err != nil {
		t.Fatal(err)
	}
	if handoffRows != 0 {
		t.Fatalf("read-only status created %d handoff rows", handoffRows)
	}
}

func TestFeature362StatusNondisclosureAndBoundOnlyTargetUnavailable(t *testing.T) {
	svc, db, claims := setupFeature362StatusService(t)
	if err := db.Create(&models.User{ID: 8, Username: "other", Email: "other@example.test", PasswordHash: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	discarded := models.QuickCaptureDraft{
		ID: 81, UserID: claims.UserID, WorkingTitle: "Discarded",
		Status: models.QuickCaptureDraftStatusDiscarded,
	}
	if err := db.Create(&discarded).Error; err != nil {
		t.Fatal(err)
	}
	missingCoinID := uint(404)
	cases := []struct {
		name  string
		jobID uint
	}{
		{"unknown id", 999999},
		{"foreign job", seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
			UserID: 8, CoinID: &missingCoinID, Source: models.DeepJobSourceSavedCoin,
			Status: models.DeepJobStatusQueued,
		})},
		{"legacy unbound intake", seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
			UserID: claims.UserID, Source: models.DeepJobSourceIntake, Status: models.DeepJobStatusQueued,
		})},
		{"unknown source", seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
			UserID: claims.UserID, Source: models.DeepJobSource("future_source"), Status: models.DeepJobStatusQueued,
		})},
		{"deleted unbound coin", seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
			UserID: claims.UserID, CoinID: &missingCoinID, Source: models.DeepJobSourceSavedCoin,
			Status: models.DeepJobStatusQueued,
		})},
		{"discarded unbound draft", seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
			UserID: claims.UserID, SourceDraftID: &discarded.ID, Source: models.DeepJobSourceCopilotDraft,
			Status: models.DeepJobStatusQueued,
		})},
	}
	const canonicalNotEligible = `{"outcome":"not_eligible","reason":null}`
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := svc.Execute(claims, feature362StatusRequest(test.jobID))
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != canonicalNotEligible {
				t.Fatalf("body=%s want exact %s", encoded, canonicalNotEligible)
			}
		})
	}

	boundJobID := seedFeature362StatusJob(t, db, models.DeepIdentificationJob{
		UserID: claims.UserID, CoinID: &missingCoinID, Source: models.DeepJobSourceSavedCoin,
		Status: models.DeepJobStatusQueued,
	})
	handoff := models.CoinCopilotDeepHandoff{
		UserID: claims.UserID, RunID: claims.RunID, ExecutionID: claims.ExecutionID,
		HandoffKeyHash: "key", RequestFingerprint: "request", ExpectedCheckpointVersion: 3,
		AppContextDigest: repository.DigestCoinCopilotAppContext(""),
		Operation:        models.CoinCopilotDeepHandoffOperationRequest,
		TargetKind:       models.CoinCopilotDeepHandoffTargetCoin, TargetID: missingCoinID,
		TargetSnapshotFingerprint: strings.Repeat("b", 64), DeepJobID: boundJobID,
		AdmissionOutcome: models.CoinCopilotDeepHandoffOutcomeCreated,
	}
	if err := db.Create(&handoff).Error; err != nil {
		t.Fatal(err)
	}
	got, err := svc.Execute(claims, feature362StatusRequest(boundJobID))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	const canonicalUnavailable = `{"outcome":"target_unavailable","reason":null}`
	if string(encoded) != canonicalUnavailable {
		t.Fatalf("bound missing target body=%s want exact %s", encoded, canonicalUnavailable)
	}
}
