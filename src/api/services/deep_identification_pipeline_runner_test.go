package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// T077: each Python frame type translates to the correct persisted event
// type; cancellation propagates (context cancel closes the outbound HTTP
// request); EOF-without-terminal => agent_unavailable (via a non-nil error
// that deep_identification_service.go's runJob maps to that failure code).

func sseFrame(t *testing.T, frameType string, extra map[string]any) string {
	t.Helper()
	payload := map[string]any{"type": frameType}
	for k, v := range extra {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal test frame: %v", err)
	}
	return "data: " + string(raw) + "\n\n"
}

func TestStreamDeepIdentificationTranslatesFrameTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			sseFrame(t, "router_selected", map[string]any{"selected": []string{"numista"}}),
			sseFrame(t, "provider_started", map[string]any{"provider": "numista"}),
			sseFrame(t, "provider_result", map[string]any{"provider": "numista", "status": "match"}),
			sseFrame(t, "evaluation", map[string]any{"disagreements": []string{}}),
			sseFrame(t, "progress", map[string]any{"phase": "synthesis", "message": "working"}),
			sseFrame(t, "synthesis", map[string]any{"report": map[string]any{"partial_success": false, "proposed_fields": map[string]any{}}}),
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f))
			flusher.Flush()
		}
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))

	var seen []string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		seen = append(seen, frame.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"router_selected", "provider_started", "provider_result", "evaluation", "progress", "synthesis"}
	if len(seen) != len(expected) {
		t.Fatalf("expected %d frames, got %d: %#v", len(expected), len(seen), seen)
	}
	for i, ft := range expected {
		if seen[i] != ft {
			t.Fatalf("frame %d: expected type %q, got %q", i, ft, seen[i])
		}
	}

	// Verify the persisted-event-type mapping used by the pipeline runner
	// agrees with the frame types actually observed (excluding the
	// terminal `synthesis` frame, which is intentionally excluded).
	for _, ft := range []string{"router_selected", "provider_started", "provider_result", "evaluation", "progress", "synthesis_started"} {
		if _, ok := deepPipelineEventType(ft); !ok {
			t.Fatalf("expected frame type %q to map to a persisted event type", ft)
		}
	}
	for _, terminal := range []string{"synthesis", "error"} {
		if _, ok := deepPipelineEventType(terminal); ok {
			t.Fatalf("terminal frame type %q must not be mapped to a persisted event type (SettleTerminal owns it)", terminal)
		}
	}
}

func TestStreamDeepIdentificationCancellationClosesRequest(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "progress", map[string]any{"phase": "start"})))
		flusher.Flush()
		close(started)
		// Block until the client disconnects (context cancelled) or the
		// test's overall timeout fires, whichever first - simulates a
		// long-running Python pipeline.
		<-r.Context().Done()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.StreamDeepIdentification(ctx, DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
			return nil
		})
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server never received the request")
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled after cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StreamDeepIdentification did not return promptly after context cancellation")
	}
}

func TestStreamDeepIdentificationEOFWithoutTerminalFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "progress", map[string]any{"phase": "start"})))
		flusher.Flush()
		// Stream ends here without ever emitting synthesis or error.
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		return nil
	})
	if !errors.Is(err, ErrDeepStreamEndedWithoutTerminal) {
		t.Fatalf("expected ErrDeepStreamEndedWithoutTerminal, got %v", err)
	}
}

func TestStreamDeepIdentificationErrorFrameIsTerminal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte(sseFrame(t, "error", map[string]any{"code": "llm_unavailable", "message": "boom"})))
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	var gotType string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		gotType = frame.Type
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error for a well-formed terminal error frame, got %v", err)
	}
	if gotType != "error" {
		t.Fatalf("expected the error frame to reach onFrame, got %q", gotType)
	}
}

func TestStreamDeepIdentificationMalformedFrameSkippedNotFabricated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {not-json\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte(sseFrame(t, "synthesis", map[string]any{"report": map[string]any{"partial_success": false}})))
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewAgentProxy(server.URL, "test-token", NewLogger(10))
	var seen []string
	err := proxy.StreamDeepIdentification(context.Background(), DeepIdentifyProxyRequest{JobID: 1}, func(frame DeepIdentifyFrame) error {
		seen = append(seen, frame.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error (malformed frame skipped, terminal frame still reached), got %v", err)
	}
	if len(seen) != 1 || seen[0] != "synthesis" {
		t.Fatalf("expected only the synthesis frame to be delivered, got %#v", seen)
	}
}

func TestBuildDeepProposalDocumentJSONProducesRichAllowlistedShape(t *testing.T) {
	// A realistic synthesis report: two allowlisted proposed fields with
	// evidence_refs, plus one non-allowlisted field that must be dropped.
	report := json.RawMessage(`{
		"proposed_fields": {
			"denomination": {"value":"Denarius","confidence":0.82,"evidence_refs":[{"provider":"numista","claim_index":0}]},
			"mint": {"value":"Rome","confidence":0.6,"evidence_refs":[{"provider":"nomisma","claim_index":0}]},
			"secretAdminField": {"value":"nope","confidence":0.99,"evidence_refs":[]}
		}
	}`)
	providerClaims := map[string][]deepProposalClaim{
		"numista": {{Field: "denomination", Value: "Denarius", Confidence: 0.8, Citation: "https://en.numista.com/catalogue/pieces12345.html", Excerpt: "Silver denarius"}},
		"nomisma": {{Field: "mint", Value: "Rome", Confidence: 0.6, Citation: "http://nomisma.org/id/roma"}},
	}
	var coinID uint = 42
	out := buildDeepProposalDocumentJSON(report, &coinID, providerClaims)

	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid rich proposal JSON, got error %v (raw: %s)", err, out)
	}
	if doc.SchemaVersion != 1 {
		t.Fatalf("expected schemaVersion 1, got %d", doc.SchemaVersion)
	}
	if doc.TargetCoinID == nil || *doc.TargetCoinID != 42 {
		t.Fatalf("expected targetCoinId 42, got %v", doc.TargetCoinID)
	}
	if _, dropped := doc.Fields["secretAdminField"]; dropped {
		t.Fatal("non-allowlisted field must be dropped from the proposal document")
	}
	den := doc.Fields["denomination"]
	if den == nil {
		t.Fatal("expected denomination field in proposal")
	}
	// Rich shape: proposed value + confidence + citation-bearing evidence,
	// and pristine owner-decision state (nothing auto-accepted).
	if den.Proposed != "Denarius" {
		t.Fatalf("expected proposed=Denarius, got %v", den.Proposed)
	}
	if den.Confidence != 0.82 {
		t.Fatalf("expected confidence 0.82 preserved, got %v", den.Confidence)
	}
	if len(den.Evidence) != 1 || den.Evidence[0].Citation != "https://en.numista.com/catalogue/pieces12345.html" {
		t.Fatalf("expected preserved citation evidence, got %#v", den.Evidence)
	}
	if den.OwnerEdited || den.OwnerValue != nil || den.Accepted != nil {
		t.Fatalf("expected pristine owner-decision state, got ownerEdited=%v ownerValue=%v accepted=%v", den.OwnerEdited, den.OwnerValue, den.Accepted)
	}

	// The old flat shape ({"fields":{"denomination":"Denarius"}}) would not
	// unmarshal into deepProposalDocument's rich per-field entries, so this
	// test necessarily fails against the pre-remediation implementation.
	if err := json.Unmarshal([]byte(`{"fields":{"denomination":"Denarius"}}`), &doc); err == nil {
		t.Fatal("sanity: the old flat proposal shape must be incompatible with the rich document")
	}
}

func TestBuildDeepProposalDocumentJSONRejectsNonAllowlistedCitationHost(t *testing.T) {
	report := json.RawMessage(`{
		"proposed_fields": {
			"denomination": {"value":"Denarius","confidence":0.8,"evidence_refs":[{"provider":"numista","claim_index":0}]}
		}
	}`)
	// A citation whose host is NOT on numista's allowlist must be dropped,
	// never persisted (no arbitrary citation injection).
	providerClaims := map[string][]deepProposalClaim{
		"numista": {{Field: "denomination", Value: "Denarius", Confidence: 0.8, Citation: "https://evil.example.com/inject"}},
	}
	var coinID uint = 42
	out := buildDeepProposalDocumentJSON(report, &coinID, providerClaims)
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}
	if den := doc.Fields["denomination"]; den == nil || len(den.Evidence) != 0 {
		t.Fatalf("expected non-allowlisted citation to be dropped, got %#v", den)
	}
}

func TestBuildDeepProposalDocumentJSONCreatesReportOnlyIntakeProposal(t *testing.T) {
	out := buildDeepProposalDocumentJSON(json.RawMessage(`{"narrative":"No structured match was available."}`), nil, nil)
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid proposal JSON, got %v", err)
	}
	if got := doc.Fields["notes"]; got == nil || got.Proposed != "No structured match was available." {
		t.Fatalf("expected report narrative to remain saveable as draft notes, got %#v", got)
	}
}

// TestDeepIdentificationBackwardCompatibility_PreAndPostImageHypothesisFixtures
// is the T071 backward-compatibility regression: Brian has existing
// deep-identification jobs already persisted from before this feature
// shipped, so a schema/shape assumption that only holds for new rows must
// not break his history on deploy. It drives two report shapes through the
// full Get -> parse -> PATCH accept -> Apply round trip and asserts zero
// errors at every step:
//
//  1. a pre-351 report/proposal with no `image_hypothesis` concept at all
//     (every evidence_ref points at a real, automatable provider, exactly
//     the shape every job persisted before Phase 3-8 has), and
//  2. a post-351 proposal containing an image-only field (evidence_refs:
//     [{"provider":"image"}], contract §5).
func TestDeepIdentificationBackwardCompatibility_PreAndPostImageHypothesisFixtures(t *testing.T) {
	t.Run("pre-351 report without image_hypothesis loads renders and applies", func(t *testing.T) {
		svc, db := newDeepBackfillProposalHarness(t)
		userID := seedDeepProposalUser(t, db)
		coin := models.Coin{UserID: userID, Name: "Pre-351 Coin"}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatalf("seed coin: %v", err)
		}

		// Pre-351 shape: proposed_fields exist, every evidence_ref names a
		// real provider claim - no image_hypothesis, no "image" provider
		// anywhere in the document.
		report := json.RawMessage(`{
			"narrative": "A silver denarius of Trajan.",
			"proposed_fields": {
				"denomination": {"value":"Denarius","confidence":0.85,"evidence_refs":[{"provider":"numista","claim_index":0}]}
			}
		}`)
		providerClaims := map[string][]deepProposalClaim{
			"numista": {{Field: "denomination", Value: "Denarius", Confidence: 0.85, Citation: "https://en.numista.com/catalogue/pieces1.html"}},
		}
		proposalJSON := buildDeepProposalDocumentJSON(report, &coin.ID, providerClaims)
		if proposalJSON == "" {
			t.Fatal("expected non-empty proposal for pre-351 shape")
		}
		jobID := createDeepBackfillJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, report, proposalJSON)

		if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
			"denomination": {Accepted: acceptTrue()},
		}); err != nil {
			t.Fatalf("expected zero-error PATCH accept on pre-351 shape, got %v", err)
		}
		result, err := svc.Apply(jobID, userID, "coin", nil)
		if err != nil {
			t.Fatalf("expected zero-error apply on pre-351 shape, got %v", err)
		}
		if result.CoinID == nil || *result.CoinID != coin.ID {
			t.Fatalf("expected applied coinId %d, got %v", coin.ID, result.CoinID)
		}
		var applied models.Coin
		if err := db.First(&applied, coin.ID).Error; err != nil {
			t.Fatal(err)
		}
		if applied.Denomination != "Denarius" {
			t.Fatalf("expected Denarius applied from pre-351 report, got %q", applied.Denomination)
		}
	})

	t.Run("post-351 proposal with image-only fields loads renders and applies", func(t *testing.T) {
		svc, db := newDeepBackfillProposalHarness(t)
		userID := seedDeepProposalUser(t, db)
		coin := models.Coin{UserID: userID, Name: "Post-351 Coin"}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatalf("seed coin: %v", err)
		}

		// Post-351 shape: an image-only field (no automatable provider
		// matched anything - exactly Brian's Maximinus I scenario).
		report := json.RawMessage(`{
			"narrative": "A silver denarius identified from the coin's own images.",
			"proposed_fields": {
				"ruler": {"value":"Maximinus I","confidence":0.75,"evidence_refs":[{"provider":"image"}]}
			}
		}`)
		proposalJSON := buildDeepProposalDocumentJSON(report, &coin.ID, nil)
		if proposalJSON == "" {
			t.Fatal("expected non-empty proposal for image-only post-351 shape")
		}
		jobID := createDeepBackfillJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, report, proposalJSON)

		if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
			"ruler": {Accepted: acceptTrue()},
		}); err != nil {
			t.Fatalf("expected zero-error PATCH accept on image-only field, got %v", err)
		}
		result, err := svc.Apply(jobID, userID, "coin", nil)
		if err != nil {
			t.Fatalf("expected zero-error apply on image-only field, got %v", err)
		}
		if result.CoinID == nil || *result.CoinID != coin.ID {
			t.Fatalf("expected applied coinId %d, got %v", coin.ID, result.CoinID)
		}
		var applied models.Coin
		if err := db.First(&applied, coin.ID).Error; err != nil {
			t.Fatal(err)
		}
		if applied.Ruler != "Maximinus I" {
			t.Fatalf("expected image-only ruler applied with zero errors, got %q", applied.Ruler)
		}
	})
}

// newDeepBackfillProposalHarness is a thin wrapper over the shared
// newDeepProposalTestDeps helper (deep_identification_proposal_test.go,
// same package) that drops the repository handle this file's tests do not
// need, keeping call sites focused on the service + db.
func newDeepBackfillProposalHarness(t *testing.T) (*DeepIdentificationProposalService, *gorm.DB) {
	t.Helper()
	svc, _, db := newDeepProposalTestDeps(t)
	return svc, db
}

// createDeepBackfillJob persists a completed job carrying a caller-supplied
// report/proposal shape exactly as SettleTerminal would, so the T071
// fixtures exercise the real Get/PATCH/Apply path rather than a synthetic
// shortcut.
func createDeepBackfillJob(t *testing.T, db *gorm.DB, userID uint, source models.DeepJobSource, coinID *uint, report json.RawMessage, proposalJSON string) uint {
	t.Helper()
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           source,
		CoinID:           coinID,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-backfill-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		ReportJSON:       string(report),
		ProposalJSON:     proposalJSON,
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed backfill job: %v", err)
	}
	return job.ID
}

func TestBuildDeepProposalDocumentJSONMapsIntakeFindingsToDraftFields(t *testing.T) {
	report := json.RawMessage(`{
		"narrative":"The evidence supports a Roman denarius.",
		"proposed_fields":{
			"ruler":{"value":"Trajan","confidence":0.8},
			"denomination":{"value":"Denarius","confidence":0.9},
			"mint":{"value":"Rome","confidence":0.7}
		}
	}`)
	out := buildDeepProposalDocumentJSON(report, nil, nil)
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid proposal JSON, got %v", err)
	}
	if got := doc.Fields["workingTitle"]; got == nil || got.Proposed != "Trajan Denarius" {
		t.Fatalf("expected draft working title, got %#v", got)
	}
	notes := doc.Fields["notes"]
	if notes == nil || !strings.Contains(notes.Proposed.(string), "mint: Rome") {
		t.Fatalf("expected structured findings in draft notes, got %#v", notes)
	}
	for name := range doc.Fields {
		if _, allowed := deepProposalDraftFieldAllowlist[name]; !allowed {
			t.Fatalf("intake proposal contains non-draft field %q", name)
		}
	}
}

func TestDeepPipelineProviderCatalogNeverAutomatesRPCorNGC(t *testing.T) {
	settings := DeepIdentificationSettings{MaxProviders: 4, NumistaCallBudget: 4}
	catalog := deepPipelineProviderCatalog(settings)
	found := map[string]DeepProviderCatalogEntryProxy{}
	for _, entry := range catalog {
		found[entry.Provider] = entry
	}
	for _, provider := range []string{"rpc", "ngc"} {
		entry, ok := found[provider]
		if !ok {
			t.Fatalf("expected a catalog entry for %q", provider)
		}
		if entry.Automatable {
			t.Fatalf("provider %q must never be automatable (deferred T156 or terms-prohibited): %#v", provider, entry)
		}
	}
	for _, provider := range []string{"numista", "nomisma"} {
		entry, ok := found[provider]
		if !ok || !entry.Automatable {
			t.Fatalf("expected provider %q to be automatable, got %#v", provider, entry)
		}
	}
}

// TestDeepPipelineProviderCatalogOCREConditional verifies Feature 345: the
// OCRE catalog entry is automatable with its own call budget iff the
// DeepIdentificationOCREEnabled flag is on, and stays a typed not_automated
// entry when off — while RPC's entry is untouched either way.
func TestDeepPipelineProviderCatalogOCREConditional(t *testing.T) {
	find := func(catalog []DeepProviderCatalogEntryProxy, provider string) DeepProviderCatalogEntryProxy {
		for _, entry := range catalog {
			if entry.Provider == provider {
				return entry
			}
		}
		t.Fatalf("no catalog entry for %q", provider)
		return DeepProviderCatalogEntryProxy{}
	}

	// Flag ON: OCRE becomes automatable with its configured call budget.
	on := deepPipelineProviderCatalog(DeepIdentificationSettings{MaxProviders: 4, NumistaCallBudget: 4, OCREEnabled: true, OCRECallBudget: 7})
	ocreOn := find(on, "ocre")
	if !ocreOn.Automatable {
		t.Fatalf("flag on: expected OCRE automatable, got %#v", ocreOn)
	}
	if ocreOn.CallBudget != 7 {
		t.Fatalf("flag on: expected OCRE call budget 7, got %#v", ocreOn)
	}

	// Flag OFF: OCRE is a typed not-automated entry, no budget.
	off := deepPipelineProviderCatalog(DeepIdentificationSettings{MaxProviders: 4, NumistaCallBudget: 4, OCREEnabled: false, OCRECallBudget: 7})
	ocreOff := find(off, "ocre")
	if ocreOff.Automatable {
		t.Fatalf("flag off: expected OCRE not automatable, got %#v", ocreOff)
	}
	if ocreOff.CallBudget != 0 {
		t.Fatalf("flag off: expected no OCRE call budget, got %#v", ocreOff)
	}

	// RPC is untouched by the OCRE flag in both cases.
	if find(on, "rpc").Automatable || find(off, "rpc").Automatable {
		t.Fatalf("RPC catalog entry must remain not automatable regardless of the OCRE flag")
	}
}

func TestDeepRouterSelectedPublicPayloadUsesContractShape(t *testing.T) {
	skipped := []struct {
		Provider string `json:"provider"`
		Reason   string `json:"reason"`
	}{
		{Provider: "ocre", Reason: "not selected"},
	}
	raw := deepRouterSelectedPublicPayloadJSON(
		[]string{"nomisma", "numista"},
		"selected from image evidence",
		skipped,
	)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if _, internalNameLeaked := payload["selected"]; internalNameLeaked {
		t.Fatal("internal selected field must not leak into the public event")
	}
	selected, ok := payload["selectedProviders"].([]any)
	if !ok || len(selected) != 2 {
		t.Fatalf("expected selectedProviders contract field, got %#v", payload)
	}
	if payload["rationale"] != "selected from image evidence" {
		t.Fatalf("expected evidence-based rationale, got %#v", payload["rationale"])
	}
}

func TestDeepPipelineBoundsClampToContractLimits(t *testing.T) {
	bounds := deepPipelineBounds(DeepIdentificationSettings{HardTimeout: 300 * time.Second, MaxProviders: 4})
	if bounds.TotalTimeoutS <= 0 || bounds.TotalTimeoutS > 900 {
		t.Fatalf("expected total_timeout_s within (0,900], got %d", bounds.TotalTimeoutS)
	}
	if bounds.MaxConcurrency < 1 || bounds.MaxConcurrency > 10 {
		t.Fatalf("expected max_concurrency within [1,10], got %d", bounds.MaxConcurrency)
	}
	if bounds.ProviderTimeoutS < 1 || bounds.ProviderTimeoutS > 120 {
		t.Fatalf("expected provider_timeout_s within [1,120], got %d", bounds.ProviderTimeoutS)
	}
	if bounds.RecursionLimit < 1 || bounds.RecursionLimit > 50 {
		t.Fatalf("expected recursion_limit within [1,50], got %d", bounds.RecursionLimit)
	}
}

// T013(a): the quick-lookup context deadline equals the configured setting
// (351 T011/T013). This is the direct regression for the pre-351 magic
// literal `15*time.Second` at deep_identification_pipeline_runner.go:112 -
// it fails immediately if that literal, or any other fixed duration, is
// reintroduced instead of settings.QuickLookupTimeout.
func TestDeepQuickLookupContextDeadlineMatchesConfiguredSetting(t *testing.T) {
	settings := DeepIdentificationSettings{QuickLookupTimeout: 90 * time.Second}
	before := time.Now()
	ctx, cancel := deepQuickLookupContext(context.Background(), settings)
	defer cancel()
	after := time.Now()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected quick-lookup context to carry a deadline")
	}
	minExpected := before.Add(settings.QuickLookupTimeout)
	maxExpected := after.Add(settings.QuickLookupTimeout)
	if deadline.Before(minExpected) || deadline.After(maxExpected) {
		t.Fatalf("quick-lookup deadline = %v, want within [%v, %v] (i.e. exactly QuickLookupTimeout from now)", deadline, minExpected, maxExpected)
	}
}

// T013(b): bounds.TotalTimeoutS after the quick-lookup pass is still >=
// deepPipelineMinTotalTimeoutS for the default settings combination decided
// in T012 (HardTimeoutSeconds=420, QuickLookupTimeoutSeconds=90). This is
// the regression for the T012 finding: raising the quick-lookup budget from
// 15s to 90s without also raising HardTimeout would have shrunk the
// pipeline's own budget to ~190s; the paired default of 420s keeps it at
// ~310s, comfortably above both deepPipelineMinTotalTimeoutS and the
// pre-change ~265s baseline (SC-013).
func TestDeepPipelineRemainingBudgetAfterQuickLookupMeetsMinimumForDefaults(t *testing.T) {
	settings := DeepIdentificationSettings{
		HardTimeout:        420 * time.Second,
		QuickLookupTimeout: 90 * time.Second,
		MaxProviders:       4,
	}
	bounds := deepPipelineBounds(settings)

	// Simulate the ctx deadline as it stands immediately after quick lookup
	// has consumed its full budget from the shared job ctx (worst case).
	jobDeadline := time.Now().Add(settings.HardTimeout - settings.QuickLookupTimeout)
	ctx, cancel := context.WithDeadline(context.Background(), jobDeadline)
	defer cancel()

	adjusted, err := deepPipelineApplyRemainingBudget(ctx, bounds)
	if err != nil {
		t.Fatalf("deepPipelineApplyRemainingBudget: %v", err)
	}
	if adjusted.TotalTimeoutS < deepPipelineMinTotalTimeoutS {
		t.Fatalf("post-quick-lookup TotalTimeoutS = %d, want >= deepPipelineMinTotalTimeoutS (%d)", adjusted.TotalTimeoutS, deepPipelineMinTotalTimeoutS)
	}
	// Cross-check against the ~310s T012 derivation (420-90-20), with a
	// small tolerance for wall-clock scheduling jitter in the test itself.
	const wantApprox = 310
	if adjusted.TotalTimeoutS < wantApprox-2 || adjusted.TotalTimeoutS > wantApprox+2 {
		t.Fatalf("post-quick-lookup TotalTimeoutS = %d, want ~%d (420-90-20 per T012)", adjusted.TotalTimeoutS, wantApprox)
	}
	const preChangeBaseline = 265
	if adjusted.TotalTimeoutS < preChangeBaseline {
		t.Fatalf("post-quick-lookup TotalTimeoutS = %d regresses below the pre-change ~265s baseline (SC-013)", adjusted.TotalTimeoutS)
	}
}
