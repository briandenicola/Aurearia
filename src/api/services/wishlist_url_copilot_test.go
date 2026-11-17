package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

type failingWishlistURLExtractor struct{ calls int }

func (f *failingWishlistURLExtractor) ExtractWishlistURL(context.Context, WishlistURLExtractionRequest) (*WishlistURLExtractionResponse, error) {
	f.calls++
	return nil, errors.New("agent returned 503")
}

func htmlListingTransport(status int, body string) http.RoundTripper {
	return wishlistURLRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
}

func TestProposeForCopilotReportsUnusableListingsAsOutcomes(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewCoinRepository(db)

	invalid, err := NewWishlistURLService(repo, &wishlistURLExtractorStub{}, nil).
		ProposeForCopilot(context.Background(), 3, "http://127.0.0.1/private")
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Outcome != WishlistURLOutcomeInvalidURL || invalid.SourceURL != "" || invalid.Hypothesis != nil {
		t.Fatalf("invalid URL proposal = %+v", invalid)
	}

	unreadableService := NewWishlistURLService(repo, &wishlistURLExtractorStub{}, nil)
	unreadableService.client = &http.Client{Transport: htmlListingTransport(http.StatusForbidden, "")}
	unreadable, err := unreadableService.ProposeForCopilot(context.Background(), 3, "https://dealer.example/lot/42?utm_source=x")
	if err != nil {
		t.Fatal(err)
	}
	if unreadable.Outcome != WishlistURLOutcomeUnreadable || unreadable.SourceURL != "https://dealer.example/lot/42" ||
		len(unreadable.Warnings) != 1 {
		t.Fatalf("unreadable proposal = %+v", unreadable)
	}

	extractor := &failingWishlistURLExtractor{}
	unavailableService := NewWishlistURLService(repo, extractor, nil)
	unavailableService.client = &http.Client{Transport: htmlListingTransport(http.StatusOK, "<title>Coin</title><main>Coin</main>")}
	unavailable, err := unavailableService.ProposeForCopilot(context.Background(), 3, "https://dealer.example/lot/42")
	if err != nil {
		t.Fatal(err)
	}
	if unavailable.Outcome != WishlistURLOutcomeUnavailable || extractor.calls != 1 {
		t.Fatalf("unavailable proposal = %+v", unavailable)
	}
}

func TestProposeForCopilotKeepsOneShortEvidenceQuotePerField(t *testing.T) {
	db := setupTestDB(t)
	long := strings.Repeat("Θ", copilotWishlistEvidenceMaxChars+50)
	extractor := &wishlistURLExtractorStub{resp: WishlistURLExtractionResponse{Hypothesis: WishlistURLHypothesis{
		Name:  &WishlistURLHypothesisField{Value: "Theodosius I. AE 3", Confidence: 0.9, Evidence: []string{"Theodosius I. AE 3", "second"}},
		Notes: &WishlistURLHypothesisField{Value: "Ex Clay", Confidence: 0.6, Evidence: []string{long}},
	}}}
	service := NewWishlistURLService(repository.NewCoinRepository(db), extractor, nil)
	service.client = &http.Client{Transport: htmlListingTransport(http.StatusOK, "<title>Theodosius I. AE 3</title><main>Coin</main>")}

	proposal, err := service.ProposeForCopilot(context.Background(), 3, "https://dealer.example/lot/42")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Outcome != "ready" || len(proposal.Hypothesis.Name.Evidence) != 1 ||
		len([]rune(proposal.Hypothesis.Notes.Evidence[0])) != copilotWishlistEvidenceMaxChars {
		t.Fatalf("proposal = %+v", proposal.Hypothesis)
	}
}

func TestProposeForCopilotReportsOwnersDuplicate(t *testing.T) {
	db := setupTestDB(t)
	coin := models.Coin{Name: "Owned", IsWishlist: true, UserID: 3, ReferenceURL: "https://dealer.example/lot/42"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	service := NewWishlistURLService(repository.NewCoinRepository(db), &wishlistURLExtractorStub{}, nil)
	service.client = &http.Client{Transport: wishlistURLRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("duplicate should not be fetched")
		return nil, nil
	})}

	proposal, err := service.ProposeForCopilot(context.Background(), 3, "https://dealer.example/lot/42")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Outcome != "duplicate" || proposal.ExistingCoinID == nil || *proposal.ExistingCoinID != coin.ID {
		t.Fatalf("proposal = %+v", proposal)
	}
	if _, err := json.Marshal(proposal); err != nil {
		t.Fatal(err)
	}
	service.client = &http.Client{Transport: htmlListingTransport(http.StatusForbidden, "")}
	other, err := service.ProposeForCopilot(context.Background(), 4, "https://dealer.example/lot/42?fbclid=x")
	if err != nil {
		t.Fatal(err)
	}
	if other.Outcome != WishlistURLOutcomeUnreadable || other.ExistingCoinID != nil {
		t.Fatalf("another owner's wishlist coin leaked: %+v", other)
	}
}

func TestPublicCopilotWishlistProposalAcceptsSharedFixtures(t *testing.T) {
	for _, name := range []string{"wishlist_url_proposal_ready.json", "wishlist_url_proposal_unreadable.json"} {
		raw, err := loadCoinCopilotFixture[json.RawMessage](t, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PublicCopilotWishlistProposal(raw); err != nil {
			t.Fatalf("%s rejected: %v", name, err)
		}
	}
}

func TestPublicCopilotWishlistProposalRejectsUnsafeOrInconsistentResults(t *testing.T) {
	ready, err := loadCoinCopilotFixture[map[string]any](t, "wishlist_url_proposal_ready.json")
	if err != nil {
		t.Fatal(err)
	}
	mutate := func(change func(map[string]any)) json.RawMessage {
		encoded, _ := json.Marshal(ready)
		var copy map[string]any
		_ = json.Unmarshal(encoded, &copy)
		change(copy)
		out, _ := json.Marshal(copy)
		return out
	}
	for name, raw := range map[string]json.RawMessage{
		"unknown field":         mutate(func(m map[string]any) { m["script"] = "x" }),
		"javascript source":     mutate(func(m map[string]any) { m["sourceUrl"] = "javascript:alert(1)" }),
		"private source":        mutate(func(m map[string]any) { m["sourceUrl"] = "http://127.0.0.1/lot" }),
		"uncanonical source":    mutate(func(m map[string]any) { m["sourceUrl"] = "https://Dealer.example/lot?utm_source=x" }),
		"private image":         mutate(func(m map[string]any) { m["imageUrl"] = "http://localhost/x.jpg" }),
		"private reverse image": mutate(func(m map[string]any) { m["reverseImageUrl"] = "http://10.0.0.1/x.jpg" }),
		"unknown outcome":       mutate(func(m map[string]any) { m["outcome"] = "added" }),
		"ready without fields":  mutate(func(m map[string]any) { delete(m, "hypothesis") }),
		"failure with fields":   mutate(func(m map[string]any) { m["outcome"] = "unreadable" }),
		"duplicate with fields": mutate(func(m map[string]any) { m["outcome"] = "duplicate"; m["existingCoinId"] = 4 }),
		"wrong schema":          mutate(func(m map[string]any) { m["schemaVersion"] = 2 }),
		"trailing data":         append(mutate(func(map[string]any) {}), []byte(`{}`)...),
	} {
		if _, err := PublicCopilotWishlistProposal(raw); err == nil {
			t.Errorf("%s: proposal unexpectedly accepted", name)
		}
	}
}

func TestCoinCopilotWorkerPublishesWishlistProposal(t *testing.T) {
	db, service := newCopilotServiceTest(t)
	repo := repository.NewCoinCopilotRepository(db)
	thread := &models.CoinCopilotThread{ID: "cct_wishlist", UserID: 7, Title: "Wishlist"}
	run := &models.CoinCopilotRun{
		ID: "ccr_wishlist", ThreadID: thread.ID, UserID: thread.UserID, Status: models.CopilotRunRunning,
		Goal: "Add this coin to my wishlist", StartIdempotencyKeyHash: "wishlist-key", StartRequestFingerprint: "fingerprint",
		ExecutionID: "cce_wishlist", ExecutionAttempt: 1, MaxIterations: 8, MaxToolCalls: 12,
		MaxConcurrentTools: 1, HardTimeoutSeconds: 120, MaxPersistedToolResultBytes: 32768,
	}
	if err := repo.CreateRun(thread, run); err != nil {
		t.Fatal(err)
	}
	result, err := loadCoinCopilotFixture[json.RawMessage](t, "wishlist_url_proposal_ready.json")
	if err != nil {
		t.Fatal(err)
	}
	apply := func(frameID, callID string, result json.RawMessage) error {
		payload, _ := json.Marshal(map[string]any{
			"tool_call_id": callID, "tool_name": "wishlist_url_proposal", "step_id": "step_" + callID,
			"status": "succeeded", "duration_ms": 25, "result_summary": "Proposal ready.", "result": result,
		})
		return service.applyFrame(run, CopilotAgentFrame{
			SchemaVersion: 1, RunID: run.ID, ExecutionID: run.ExecutionID,
			FrameID: frameID, Type: "tool_completed", Payload: payload,
		})
	}
	if err := apply("frm_wishlist", "call_wishlist", result); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != models.CopilotEventToolCompleted {
		t.Fatalf("events = %#v", events)
	}
	var public struct {
		WishlistProposal *WishlistURLAnalysis `json:"wishlistProposal"`
	}
	if err := json.Unmarshal([]byte(events[0].PayloadJSON), &public); err != nil {
		t.Fatal(err)
	}
	if public.WishlistProposal == nil || public.WishlistProposal.Outcome != "ready" ||
		public.WishlistProposal.Hypothesis == nil || public.WishlistProposal.Hypothesis.Name.Value != "Theodosius I. AE 3" {
		t.Fatalf("public proposal = %#v", public.WishlistProposal)
	}

	if err := apply("frm_bad", "call_bad", json.RawMessage(`{"schemaVersion":1,"outcome":"added"}`)); err != nil {
		t.Fatalf("invalid proposal should not fail the run: %v", err)
	}
	events, err = repo.ListEventsSince(run.ID, run.UserID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || strings.Contains(events[1].PayloadJSON, "wishlistProposal") {
		t.Fatalf("invalid proposal was published: %#v", events)
	}
}
