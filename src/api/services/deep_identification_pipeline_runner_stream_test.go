package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// B1 cross-boundary regression (blocker 344): these tests drive the ACTUAL
// DeepIdentificationPipelineRunner.Run over a fake Python agent that emits
// the exact Python-shaped SSE frames run_deep_identification_stream now
// produces — full ProviderEvidence (with claims) on `provider_result`. They
// prove the runner accumulates those streamed claims (not hand-injected
// providerClaims) into the persisted rich ProposalJSON with citation +
// confidence, and that the persisted user-visible event log carries only the
// bounded public payload (no claims/citations leaked). A runner that dropped
// claims — or a Python side that omitted them — fails these.

func newDeepRunnerStreamTestDeps(t *testing.T, agentURL string) (*DeepIdentificationPipelineRunner, *repository.DeepIdentificationRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:deep_runner_stream_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)

	repo := repository.NewDeepIdentificationRepository(db)
	runner := newDeepRunnerOnDB(t, db, agentURL)
	return runner, repo, db
}

// newDeepRunnerOnDB builds a real pipeline runner bound to an existing DB
// (so proposal/apply tests can drive the same job the runner produced). It
// configures the minimal settings the runner reads (AI provider + key) and
// mints tokens with a throwaway secret; no live provider or LLM is contacted
// because the agent at agentURL is a fake httptest server.
func newDeepRunnerOnDB(t *testing.T, db *gorm.DB, agentURL string) *DeepIdentificationPipelineRunner {
	t.Helper()
	repo := repository.NewDeepIdentificationRepository(db)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(SettingAIProvider, "anthropic"); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if err := settingsSvc.SetSetting(SettingAnthropicAPIKey, "test-key"); err != nil {
		t.Fatalf("set key: %v", err)
	}
	proxy := NewAgentProxy(agentURL, "svc-token", NewLogger(50))
	tokenSvc := NewInternalTokenService("internal-secret")
	return NewDeepIdentificationPipelineRunner(proxy, repo, settingsSvc, tokenSvc, "http://api:8080", NewLogger(50), nil)
}

// pythonShapedDeepStreamFrames returns the SSE stream a real
// run_deep_identification_stream emits for a single contributing numista
// provider: a full ProviderEvidence provider_result (claims included) plus a
// terminal synthesis whose proposed_fields reference those claims by index.
func pythonShapedDeepStreamFrames() []string {
	providerResult := map[string]any{
		"type":        "provider_result",
		"provider":    "numista",
		"status":      "contributed",
		"automatable": true,
		"confidence":  0.7,
		"call_count":  1,
		"error_kind":  nil,
		"link_out":    "",
		"attribution": "Source: Numista",
		"claims": []map[string]any{
			{
				"field":      "denomination",
				"value":      "Denarius",
				"confidence": 0.8,
				"citation":   "https://en.numista.com/catalogue/pieces12345.html",
				"excerpt":    "Silver denarius, Rome mint",
			},
		},
	}
	synthesis := map[string]any{
		"type": "synthesis",
		"report": map[string]any{
			"narrative": "A silver denarius.",
			"proposed_fields": map[string]any{
				"denomination": map[string]any{
					"value":      "Denarius",
					"confidence": 0.82,
					"evidence_refs": []map[string]any{
						{"provider": "numista", "claim_index": 0},
					},
				},
			},
			"partial_success": false,
		},
	}
	frames := []map[string]any{
		{"type": "progress", "stage": "image_evidence_ready"},
		{"type": "router_selected", "selected": []string{"numista"}, "rationale": "auto"},
		{"type": "provider_started", "provider": "numista"},
		providerResult,
		{"type": "evaluation", "disagreement_count": 0, "resolved_count": 0},
		{"type": "synthesis_started"},
		synthesis,
	}
	out := make([]string, 0, len(frames))
	for _, f := range frames {
		raw, _ := json.Marshal(f)
		out = append(out, "data: "+string(raw)+"\n\n")
	}
	return out
}

func newPythonShapedDeepAgent(t *testing.T, frames []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deep-identify/stream" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		for _, f := range frames {
			_, _ = w.Write([]byte(f))
			flusher.Flush()
		}
	}))
}

func seedDeepRunnerJob(t *testing.T, db *gorm.DB, source models.DeepJobSource, coinID *uint) (*models.DeepIdentificationJob, uint) {
	t.Helper()
	user := models.User{Username: fmt.Sprintf("runner-owner-%d", atomic.AddInt64(&deepTestDBCounter, 1)), Email: fmt.Sprintf("ro%d@example.com", time.Now().UnixNano()), PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	job := &models.DeepIdentificationJob{
		UserID:           user.ID,
		Source:           source,
		CoinID:           coinID,
		Status:           models.DeepJobStatusRunning,
		InputFingerprint: fmt.Sprintf("fp-runner-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return job, user.ID
}

// TestRunnerAccumulatesStreamedClaimsIntoProposal is the core B1 regression:
// it feeds the exact Python-shaped SSE frames (provider_result carrying full
// claims) through the real Run, and asserts the resulting ProposalJSON
// preserves the streamed citation + confidence. Under the pre-remediation
// Python (which emitted `{type, provider, status}` only) the provider_result
// frame carries no claims, providerClaims stays empty, and denomination's
// Evidence would be empty — failing this test.
func TestRunnerAccumulatesStreamedClaimsIntoProposal(t *testing.T) {
	server := newPythonShapedDeepAgent(t, pythonShapedDeepStreamFrames())
	defer server.Close()

	runner, repo, db := newDeepRunnerStreamTestDeps(t, server.URL)
	var coinID uint = 77
	job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceSavedCoin, &coinID)

	result, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if result == nil || result.ProposalJSON == "" {
		t.Fatalf("expected a non-empty proposal from the streamed synthesis, got %#v", result)
	}

	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(result.ProposalJSON), &doc); err != nil {
		t.Fatalf("proposal did not parse as the rich document: %v (raw=%s)", err, result.ProposalJSON)
	}
	den := doc.Fields["denomination"]
	if den == nil {
		t.Fatalf("expected denomination field in proposal, got fields=%v", doc.Fields)
	}
	if den.Confidence != 0.82 {
		t.Fatalf("expected proposed-field confidence 0.82, got %v", den.Confidence)
	}
	if len(den.Evidence) != 1 {
		t.Fatalf("expected exactly one citation-bearing evidence claim accumulated from the stream, got %#v", den.Evidence)
	}
	if den.Evidence[0].Citation != "https://en.numista.com/catalogue/pieces12345.html" {
		t.Fatalf("expected streamed citation preserved, got %q", den.Evidence[0].Citation)
	}
	if den.Evidence[0].Confidence != 0.8 {
		t.Fatalf("expected streamed claim confidence 0.8 preserved, got %v", den.Evidence[0].Confidence)
	}

	// The persisted, user-visible provider_result event must carry only the
	// bounded public payload — claimCount, never the raw claims/citation.
	events, err := repo.ListEventsSince(job.ID, userID, 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	var providerEvent *models.DeepIdentificationEvent
	for i := range events {
		if events[i].Type == models.DeepEventProviderResult {
			providerEvent = &events[i]
		}
	}
	if providerEvent == nil {
		t.Fatal("expected a persisted provider_result event")
	}
	if strings.Contains(providerEvent.PayloadJSON, "en.numista.com") ||
		strings.Contains(providerEvent.PayloadJSON, "excerpt") ||
		strings.Contains(providerEvent.PayloadJSON, "Silver denarius") {
		t.Fatalf("persisted provider_result must not leak claims/citations, got %s", providerEvent.PayloadJSON)
	}
	var publicPayload struct {
		Provider   string  `json:"provider"`
		Status     string  `json:"status"`
		Confidence float64 `json:"confidence"`
		ClaimCount int     `json:"claimCount"`
	}
	if err := json.Unmarshal([]byte(providerEvent.PayloadJSON), &publicPayload); err != nil {
		t.Fatalf("public provider_result payload did not parse: %v (raw=%s)", err, providerEvent.PayloadJSON)
	}
	if publicPayload.Provider != "numista" || publicPayload.Status != "contributed" {
		t.Fatalf("unexpected public payload identity: %#v", publicPayload)
	}
	if publicPayload.ClaimCount != 1 {
		t.Fatalf("expected claimCount 1 in the bounded public payload, got %d", publicPayload.ClaimCount)
	}
}

// TestRunnerRejectsNonAllowlistedStreamedCitationHost proves that a
// provider_result frame whose claim citation is off the provider's canonical
// host allowlist never reaches the persisted proposal (SC-006): the runner
// re-checks the host Go-side even though Python is supposed to have dropped
// it, so a compromised/buggy agent cannot inject an arbitrary citation URL.
func TestRunnerRejectsNonAllowlistedStreamedCitationHost(t *testing.T) {
	frames := pythonShapedDeepStreamFrames()
	// Replace the provider_result frame with one carrying an off-allowlist
	// citation host; keep everything else identical.
	badResult := map[string]any{
		"type":        "provider_result",
		"provider":    "numista",
		"status":      "contributed",
		"automatable": true,
		"confidence":  0.7,
		"call_count":  1,
		"link_out":    "",
		"attribution": "Source: Numista",
		"claims": []map[string]any{
			{
				"field":      "denomination",
				"value":      "Denarius",
				"confidence": 0.8,
				"citation":   "https://evil.example.com/inject",
			},
		},
	}
	raw, _ := json.Marshal(badResult)
	frames[3] = "data: " + string(raw) + "\n\n"

	server := newPythonShapedDeepAgent(t, frames)
	defer server.Close()

	runner, repo, db := newDeepRunnerStreamTestDeps(t, server.URL)
	var coinID uint = 88
	job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceSavedCoin, &coinID)

	result, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	// denomination's only evidence_ref pointed at an off-allowlist citation,
	// so it is dropped; with no surviving allowlisted field the proposal is
	// intentionally empty (no arbitrary-citation injection path).
	if result.ProposalJSON != "" {
		var doc deepProposalDocument
		if err := json.Unmarshal([]byte(result.ProposalJSON), &doc); err != nil {
			t.Fatalf("parse proposal: %v", err)
		}
		if den := doc.Fields["denomination"]; den != nil && len(den.Evidence) != 0 {
			t.Fatalf("off-allowlist citation must be dropped, got %#v", den.Evidence)
		}
	}

	// And the persisted public payload's claimCount excludes the rejected claim.
	events, err := repo.ListEventsSince(job.ID, userID, 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	for _, ev := range events {
		if ev.Type != models.DeepEventProviderResult {
			continue
		}
		if strings.Contains(ev.PayloadJSON, "evil.example.com") {
			t.Fatalf("off-allowlist citation must never be persisted, got %s", ev.PayloadJSON)
		}
		var pp struct {
			ClaimCount int `json:"claimCount"`
		}
		_ = json.Unmarshal([]byte(ev.PayloadJSON), &pp)
		if pp.ClaimCount != 0 {
			t.Fatalf("expected claimCount 0 for a rejected off-allowlist claim, got %d", pp.ClaimCount)
		}
	}
}

// TestRunnerMalformedProviderResultFrameSkippedNotFabricated proves a
// malformed provider_result claim block does not crash the run or fabricate
// evidence: the runner ignores the unparyable frame's claims and still
// produces a valid (claim-free) proposal from the terminal synthesis.
func TestRunnerMalformedProviderResultFrameSkippedNotFabricated(t *testing.T) {
	frames := pythonShapedDeepStreamFrames()
	// Claims is the wrong JSON type (object instead of array) — unmarshal of
	// the evidence struct fails, so no claims are accumulated for numista.
	frames[3] = "data: " + `{"type":"provider_result","provider":"numista","status":"contributed","claims":{"not":"an-array"}}` + "\n\n"

	server := newPythonShapedDeepAgent(t, frames)
	defer server.Close()

	runner, _, db := newDeepRunnerStreamTestDeps(t, server.URL)
	var coinID uint = 99
	job, _ := seedDeepRunnerJob(t, db, models.DeepJobSourceSavedCoin, &coinID)

	result, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatalf("runner.Run should tolerate a malformed provider_result frame, got %v", err)
	}
	// The synthesis references numista claim_index 0, but no claims were
	// accumulated, so the field has no citation evidence and is dropped —
	// leaving an empty (never fabricated) proposal.
	if result.ProposalJSON != "" {
		var doc deepProposalDocument
		if err := json.Unmarshal([]byte(result.ProposalJSON), &doc); err != nil {
			t.Fatalf("parse proposal: %v", err)
		}
		if den := doc.Fields["denomination"]; den != nil && len(den.Evidence) != 0 {
			t.Fatalf("no evidence should be fabricated from a malformed frame, got %#v", den.Evidence)
		}
	}
}
