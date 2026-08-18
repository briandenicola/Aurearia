package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	return NewDeepIdentificationPipelineRunner(proxy, repo, settingsSvc, tokenSvc, "http://api:8080", NewLogger(50), nil, nil)
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

func TestDeepIdentificationPipelineRunnerSettlesUnfinishedProviderRuns(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		wantStatus models.DeepProviderRunStatus
		wantError  string
	}{
		{name: "explicit cancellation", mode: "cancel", wantStatus: models.DeepProviderRunSkipped},
		{name: "hard timeout", mode: "timeout", wantStatus: models.DeepProviderRunTimedOut, wantError: "timeout"},
		{name: "agent failure", mode: "failure", wantStatus: models.DeepProviderRunFailed, wantError: "upstream"},
		{name: "partial synthesis timeout", mode: "partial", wantStatus: models.DeepProviderRunTimedOut, wantError: "timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				flusher := w.(http.Flusher)
				_, _ = w.Write([]byte("data: {\"type\":\"provider_started\",\"provider\":\"numista\"}\n\n"))
				flusher.Flush()
				close(started)
				switch tt.mode {
				case "cancel", "timeout":
					<-r.Context().Done()
				case "failure":
					_, _ = w.Write([]byte("data: {\"type\":\"error\",\"code\":\"llm_unavailable\",\"message\":\"unavailable\"}\n\n"))
					flusher.Flush()
				case "partial":
					_, _ = w.Write([]byte("data: {\"type\":\"synthesis\",\"report\":{\"partial_success\":true}}\n\n"))
					flusher.Flush()
				}
			}))
			defer server.Close()

			runner, repo, db := newDeepRunnerStreamTestDeps(t, server.URL)
			job, _ := seedDeepRunnerJob(t, db, models.DeepJobSourceIntake, nil)
			ctx := context.Background()
			cancel := func() {}
			switch tt.mode {
			case "cancel":
				var cancelCtx context.CancelFunc
				ctx, cancelCtx = context.WithCancel(ctx)
				cancel = cancelCtx
				go func() {
					<-started
					deadline := time.Now().Add(time.Second)
					for time.Now().Before(deadline) {
						var count int64
						db.Model(&models.DeepIdentificationProviderRun{}).
							Where("job_id = ? AND provider = ?", job.ID, models.DeepProviderNumista).
							Count(&count)
						if count == 1 {
							break
						}
						time.Sleep(5 * time.Millisecond)
					}
					cancelCtx()
				}()
			case "timeout":
				ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
			}
			defer cancel()

			_, _ = runner.Run(ctx, job)

			var run models.DeepIdentificationProviderRun
			if err := db.Where("job_id = ? AND provider = ?", job.ID, models.DeepProviderNumista).First(&run).Error; err != nil {
				t.Fatal(err)
			}
			if run.Status != tt.wantStatus || run.ErrorKind != tt.wantError || run.CompletedAt == nil {
				t.Fatalf("provider settlement = status %q error %q completed %v, want status %q error %q",
					run.Status, run.ErrorKind, run.CompletedAt, tt.wantStatus, tt.wantError)
			}
			metrics, err := repo.GetObservabilityMetrics()
			if err != nil {
				t.Fatal(err)
			}
			if metrics.Providers[models.DeepProviderNumista].StatusCounts[tt.wantStatus] != 1 {
				t.Fatalf("settled provider status missing from observability: %+v", metrics.Providers)
			}
		})
	}
}

func TestDeepIdentificationPipelineRunnerPassesQuickLookupEvidence(t *testing.T) {
	requests := make(chan DeepIdentifyProxyRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/analyze":
			analysis, _ := json.Marshal(map[string]any{
				"ngcCert": "8232252-186", "ngcGrade": strings.Repeat("G", 40),
				"ngcDescription": "Maximinus I denarius",
				"labelText":      "8232252-186 " + strings.Repeat("L", 2100),
				"name":           "Maximinus I AR Denarius",
				"ruler":          "Maximinus I",
				"denomination":   "Denarius",
				"category":       "Roman",
				"confidence":     "high",
			})
			_ = json.NewEncoder(w).Encode(map[string]string{"analysis": string(analysis)})
		case "/api/deep-identify/stream":
			var request DeepIdentifyProxyRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			requests <- request
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)
			_, _ = w.Write([]byte("data: {\"type\":\"provider_result\",\"provider\":\"ngc\",\"status\":\"not_automated\",\"automatable\":false,\"link_out\":\"https://www.ngccoin.com/certlookup/8232252186/NGCAncients/\"}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"synthesis\",\"report\":{\"partial_success\":false}}\n\n"))
			flusher.Flush()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	runner, _, db := newDeepRunnerStreamTestDeps(t, server.URL)
	runner.WithQuickEvidence(NewCoinLookupService(runner.proxy, runner.settingsSvc, NewLogger(20)))
	job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceIntake, nil)
	imagePath := filepath.Join(t.TempDir(), "obverse.png")
	if err := os.WriteFile(imagePath, []byte("valid-enough-for-proxy-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.DeepIdentificationArtifact{
		JobID: job.ID, UserID: userID, Role: models.DeepArtifactRoleObverse,
		Origin: models.DeepArtifactOriginUploaded, FilePath: imagePath, MimeType: "image/png",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	request := <-requests
	if request.QuickEvidence == nil || request.QuickEvidence.NGC == nil {
		t.Fatalf("quick NGC evidence was not passed to Deep Analysis: %#v", request.QuickEvidence)
	}
	if request.QuickEvidence.NGC.CertNumber != "8232252-186" ||
		len(request.QuickEvidence.NGC.Grade) != 32 {
		t.Fatalf("unexpected NGC evidence: %#v", request.QuickEvidence.NGC)
	}
	if request.QuickEvidence.CoinFields["ruler"] != "Maximinus I" ||
		request.QuickEvidence.CoinFields["denomination"] != "Denarius" {
		t.Fatalf("quick coin fields were not preserved: %#v", request.QuickEvidence.CoinFields)
	}
	if !strings.Contains(request.QuickEvidence.LabelText, "8232252-186") {
		t.Fatalf("label text missing certificate: %q", request.QuickEvidence.LabelText)
	}
	if len([]rune(request.QuickEvidence.LabelText)) != 2000 {
		t.Fatalf("label text length = %d, want 2000", len([]rune(request.QuickEvidence.LabelText)))
	}
}

// TestQuickLookupOutcomeTypedAndJobStillCompletes is the T017 regression:
// forced Lookup error yields `unavailable` and the job still completes,
// empty result yields `no_data`, success yields `ok`. It also asserts no
// user content (cert numbers, label text, ruler names) ever appears in the
// emitted progress event payload — only the fixed outcome-class message
// (FR-030, 344 FR-036).
func TestQuickLookupOutcomeTypedAndJobStillCompletes(t *testing.T) {
	tests := []struct {
		name            string
		analyzeStatus   int
		analyzeBody     map[string]any
		wantOutcome     quickLookupOutcome
		wantMessage     string
		forbiddenSubstr []string
	}{
		{
			name:          "forced lookup error yields unavailable",
			analyzeStatus: http.StatusInternalServerError,
			wantOutcome:   quickLookupOutcomeUnavailable,
			wantMessage:   "Quick lookup did not complete",
		},
		{
			name:          "empty result yields no_data",
			analyzeStatus: http.StatusOK,
			analyzeBody: map[string]any{
				"ngcCert": nil, "ngcGrade": nil, "ngcDescription": nil,
				"labelText": "", "name": nil, "ruler": nil, "denomination": nil,
				"category": nil, "confidence": "low",
			},
			wantOutcome: quickLookupOutcomeNoData,
			wantMessage: "Quick lookup completed with no supporting data",
		},
		{
			name:          "success yields ok",
			analyzeStatus: http.StatusOK,
			analyzeBody: map[string]any{
				"ngcCert": "8232252-186", "ngcGrade": "Ch AU", "ngcDescription": "Maximinus I denarius",
				"labelText": "8232252-186 Maximinus I", "name": "Maximinus I AR Denarius",
				"ruler": "Maximinus I", "denomination": "Denarius", "category": "Roman", "confidence": "high",
			},
			wantOutcome:     quickLookupOutcomeOK,
			wantMessage:     "Quick lookup found supporting data",
			forbiddenSubstr: []string{"8232252", "Maximinus", "Denarius", "Ch AU"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/analyze":
					if tt.analyzeStatus != http.StatusOK {
						http.Error(w, "analysis unavailable", tt.analyzeStatus)
						return
					}
					analysis, _ := json.Marshal(tt.analyzeBody)
					_ = json.NewEncoder(w).Encode(map[string]string{"analysis": string(analysis)})
				case "/api/deep-identify/stream":
					w.Header().Set("Content-Type", "text/event-stream")
					flusher := w.(http.Flusher)
					_, _ = w.Write([]byte("data: {\"type\":\"synthesis\",\"report\":{\"narrative\":\"stub\",\"partial_success\":false}}\n\n"))
					flusher.Flush()
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			runner, repo, db := newDeepRunnerStreamTestDeps(t, server.URL)
			runner.WithQuickEvidence(NewCoinLookupService(runner.proxy, runner.settingsSvc, NewLogger(20)))
			job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceIntake, nil)
			imagePath := filepath.Join(t.TempDir(), "obverse.png")
			if err := os.WriteFile(imagePath, []byte("valid-enough-for-proxy-test"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&models.DeepIdentificationArtifact{
				JobID: job.ID, UserID: userID, Role: models.DeepArtifactRoleObverse,
				Origin: models.DeepArtifactOriginUploaded, FilePath: imagePath, MimeType: "image/png",
			}).Error; err != nil {
				t.Fatal(err)
			}

			result, err := runner.Run(context.Background(), job)
			// The job must still complete: a quick-lookup failure/no-data
			// outcome must never surface as a pipeline error (T017).
			if err != nil {
				t.Fatalf("runner.Run returned an error; job did not complete: %v", err)
			}
			if result == nil || result.ReportJSON == "" {
				t.Fatal("expected a synthesized report even when quick lookup did not contribute")
			}

			var report map[string]any
			if jsonErr := json.Unmarshal([]byte(result.ReportJSON), &report); jsonErr != nil {
				t.Fatalf("unmarshal report: %v", jsonErr)
			}
			if got := report["quickLookupOutcome"]; got != string(tt.wantOutcome) {
				t.Fatalf("report quickLookupOutcome = %v, want %q", got, tt.wantOutcome)
			}

			events, err := repo.ListEventsSince(job.ID, userID, 0)
			if err != nil {
				t.Fatalf("ListEventsSince: %v", err)
			}
			var found bool
			for _, event := range events {
				if event.Type != models.DeepEventProgress {
					continue
				}
				if !strings.Contains(event.PayloadJSON, `"phase":"quick_lookup"`) {
					continue
				}
				found = true
				if !strings.Contains(event.PayloadJSON, tt.wantMessage) {
					t.Fatalf("progress payload = %s, want message %q", event.PayloadJSON, tt.wantMessage)
				}
				for _, forbidden := range tt.forbiddenSubstr {
					if strings.Contains(event.PayloadJSON, forbidden) {
						t.Fatalf("progress payload leaked user content %q: %s", forbidden, event.PayloadJSON)
					}
				}
			}
			if !found {
				t.Fatal("expected a quick_lookup progress event to be recorded")
			}
		})
	}
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

	var providerRun models.DeepIdentificationProviderRun
	if err := db.Where("job_id = ? AND provider = ?", job.ID, models.DeepProviderNumista).First(&providerRun).Error; err != nil {
		t.Fatalf("load persisted provider run: %v", err)
	}
	if providerRun.Status != models.DeepProviderRunContributed ||
		providerRun.CallCount != 1 || providerRun.LatencyMS < 1 {
		t.Fatalf("provider run operational fields = %+v", providerRun)
	}
	if providerRun.ClaimsJSON != "" {
		t.Fatalf("provider run must not persist claims: %q", providerRun.ClaimsJSON)
	}
}

// TestRunnerRouterSelectedSkippedSurvivesTranslation is the T049/B4
// regression: the deterministic router (Phase 6) now emits a populated
// `skipped[]` on the `router_selected` frame. This proves the Go translator
// (`deepRouterSelectedPublicPayloadJSON`) — which has always parsed
// `skipped` but never received a non-empty one — carries it through
// end-to-end into the persisted, user-visible event payload instead of
// silently dropping it.
func TestRunnerRouterSelectedSkippedSurvivesTranslation(t *testing.T) {
	frames := []map[string]any{
		{"type": "progress", "stage": "image_evidence_ready"},
		{
			"type":      "router_selected",
			"selected":  []string{"numista"},
			"rationale": "inclusion by default: selected 1 of 2 automatable providers; evidence-driven skip: ocre (non-Roman-Imperial era signal: greek)",
			"skipped": []map[string]any{
				{"provider": "ocre", "reason": "non-Roman-Imperial era signal: greek"},
			},
		},
		{"type": "provider_started", "provider": "numista"},
		{
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
		},
		{"type": "evaluation", "disagreement_count": 0, "resolved_count": 0},
		{"type": "synthesis_started"},
		{
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
		},
	}
	sseLines := make([]string, 0, len(frames))
	for _, f := range frames {
		raw, err := json.Marshal(f)
		if err != nil {
			t.Fatalf("marshal fixture frame: %v", err)
		}
		sseLines = append(sseLines, "data: "+string(raw)+"\n\n")
	}

	server := newPythonShapedDeepAgent(t, sseLines)
	defer server.Close()

	runner, repo, db := newDeepRunnerStreamTestDeps(t, server.URL)
	var coinID uint = 78
	job, userID := seedDeepRunnerJob(t, db, models.DeepJobSourceSavedCoin, &coinID)

	if _, err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("runner.Run: %v", err)
	}

	events, err := repo.ListEventsSince(job.ID, userID, 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	var routerEvent *models.DeepIdentificationEvent
	for i := range events {
		if events[i].Type == models.DeepEventRouterSelected {
			routerEvent = &events[i]
		}
	}
	if routerEvent == nil {
		t.Fatal("expected a persisted router_selected event")
	}

	var publicPayload struct {
		SelectedProviders []string `json:"selectedProviders"`
		Rationale         string   `json:"rationale"`
		Skipped           []struct {
			Provider string `json:"provider"`
			Reason   string `json:"reason"`
		} `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(routerEvent.PayloadJSON), &publicPayload); err != nil {
		t.Fatalf("public router_selected payload did not parse: %v (raw=%s)", err, routerEvent.PayloadJSON)
	}
	if len(publicPayload.Skipped) != 1 {
		t.Fatalf("expected exactly one persisted skip, got %#v (raw=%s)", publicPayload.Skipped, routerEvent.PayloadJSON)
	}
	if publicPayload.Skipped[0].Provider != "ocre" {
		t.Fatalf("expected skipped provider ocre, got %q", publicPayload.Skipped[0].Provider)
	}
	if publicPayload.Skipped[0].Reason != "non-Roman-Imperial era signal: greek" {
		t.Fatalf("expected the stated skip reason to survive translation, got %q", publicPayload.Skipped[0].Reason)
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

// T037 (US3/FR-022): the OCRE provider_result public payload persisted into
// the owner-visible, replayable event log must carry only status/timing/
// counts/error-kind/link-out — never the user's notes or the full legend/
// inscription text that seeded the query, and never raw claim citations.
func TestOCREProviderResultPublicPayloadOmitsUserAndLegendText(t *testing.T) {
	secretLegend := "IMP CAESAR AVGVSTVS DIVI F"
	secretNote := "my private grading note and provenance"
	claims := []deepProposalClaim{
		{
			Field:      "coin_type",
			Value:      "RIC I Augustus 1",
			Confidence: 0.82,
			Citation:   "https://numismatics.org/ocre/id/ric.1(2).aug.1",
			Excerpt:    "matched authority; legend seed " + secretLegend + "; note " + secretNote,
		},
	}

	payload := deepProviderResultPublicPayloadJSON("ocre", "contributed", 0.82, "", "", claims)

	for _, forbidden := range []string{secretLegend, secretNote, "matched authority", "ric.1(2).aug.1", "numismatics.org"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("public payload leaked %q: %s", forbidden, payload)
		}
	}

	var pp deepProviderResultPublicPayload
	if err := json.Unmarshal([]byte(payload), &pp); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if pp.Provider != "ocre" || pp.Status != "contributed" || pp.ClaimCount != 1 {
		t.Fatalf("unexpected bounded payload: %+v", pp)
	}
}
