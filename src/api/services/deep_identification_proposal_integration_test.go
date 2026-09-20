package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

func TestFeature362ProposalScalarMatricesAreClosed(t *testing.T) {
	t.Parallel()

	wantCoin := map[string]string{
		"category": "Category", "denomination": "Denomination", "ruler": "Ruler", "era": "Era",
		"dateRange": "DateRange", "mint": "Mint", "material": "Material",
		"weightGrams": "WeightGrams", "diameterMm": "DiameterMm",
		"obverseInscription": "ObverseInscription", "reverseInscription": "ReverseInscription",
		"obverseDescription": "ObverseDescription", "reverseDescription": "ReverseDescription",
		"grade": "Grade", "rarityRating": "RarityRating",
		"coin_type": "ReferenceText",
	}
	wantDraft := map[string]string{
		"workingTitle": "WorkingTitle",
		"era":          "Era",
		"dateRange":    "DateRange",
	}
	if !reflect.DeepEqual(deepProposalCoinFieldAllowlist, wantCoin) {
		t.Fatalf("collection/wishlist scalar matrix drifted:\n got: %#v\nwant: %#v", deepProposalCoinFieldAllowlist, wantCoin)
	}
	if !reflect.DeepEqual(deepProposalDraftFieldAllowlist, wantDraft) {
		t.Fatalf("bound-draft scalar matrix drifted:\n got: %#v\nwant: %#v", deepProposalDraftFieldAllowlist, wantDraft)
	}
}

func TestFeature362AcceptedCoinScalarsReplaceOnlyThemselves(t *testing.T) {
	values := map[string]any{
		"category": string(models.CategoryGreek), "denomination": "Denarius", "ruler": "Hadrian", "era": string(models.EraAncient),
		"dateRange": "117-138", "mint": "Rome", "material": string(models.MaterialGold),
		"weightGrams": 3.25, "diameterMm": 19.5,
		"obverseInscription": "HADRIANVS", "reverseInscription": "PAX",
		"obverseDescription": "Laureate bust", "reverseDescription": "Pax standing",
		"grade": "VF", "rarityRating": "Scarce",
		"coin_type": "RIC II 42",
	}
	for _, wishlist := range []bool{false, true} {
		destination := "collection"
		if wishlist {
			destination = "wishlist"
		}
		for field, proposed := range values {
			t.Run(destination+"/"+field, func(t *testing.T) {
				svc, _, db := newDeepProposalTestDeps(t)
				userID := seedDeepProposalUser(t, db)
				weight, diameter := 1.1, 10.1
				coin := models.Coin{
					UserID: userID, Name: "Manual", IsWishlist: wishlist,
					Category: models.CategoryRoman, Grade: "Fine", RarityRating: "Common",
					Denomination: "As", Ruler: "Augustus", Era: models.EraModern,
					DateRange: "old date", Mint: "old mint", Material: models.MaterialSilver,
					WeightGrams: &weight, DiameterMm: &diameter,
					ObverseInscription: "old obv", ReverseInscription: "old rev",
					ObverseDescription: "old obv description", ReverseDescription: "old rev description",
					ReferenceText: "old type",
				}
				if err := db.Create(&coin).Error; err != nil {
					t.Fatal(err)
				}
				before := coin
				jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{field: proposed})
				if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
					field: {Accepted: acceptTrue()},
				}); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.Apply(jobID, userID, "coin", []string{field}); err != nil {
					t.Fatal(err)
				}
				var after models.Coin
				if err := db.First(&after, coin.ID).Error; err != nil {
					t.Fatal(err)
				}
				selectedGoField := deepProposalCoinFieldAllowlist[field]
				for _, goField := range deepProposalCoinFieldAllowlist {
					got := reflect.ValueOf(after).FieldByName(goField).Interface()
					old := reflect.ValueOf(before).FieldByName(goField).Interface()
					if goField == selectedGoField {
						if reflect.DeepEqual(got, old) {
							t.Fatalf("%s did not change", goField)
						}
					} else if !reflect.DeepEqual(got, old) {
						t.Fatalf("applying %s changed unrelated %s: before=%#v after=%#v", field, goField, old, got)
					}
				}
			})
		}
	}
}

func TestFeature362AcceptedDraftScalarsReplaceOnlyThemselves(t *testing.T) {
	values := map[string]any{
		"workingTitle": "Hadrian denarius",
		"era":          string(models.EraAncient),
		"dateRange":    "117-138",
	}
	for field, proposed := range values {
		t.Run(field, func(t *testing.T) {
			svc, _, db := newDeepProposalTestDeps(t)
			userID := seedDeepProposalUser(t, db)
			draft := &models.QuickCaptureDraft{
				UserID: userID, WorkingTitle: "Manual title", Era: string(models.EraModern),
				DateRange: "old date", AcquisitionSource: "manual source",
				Notes: "manual notes", Status: models.QuickCaptureDraftStatusActive,
			}
			jobID := seedFeature362DraftJob(t, db, userID, draft, map[string]any{field: proposed})
			if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
				field: {Accepted: acceptTrue()},
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Apply(jobID, userID, "draft", []string{field}); err != nil {
				t.Fatal(err)
			}
			var after models.QuickCaptureDraft
			if err := db.First(&after, draft.ID).Error; err != nil {
				t.Fatal(err)
			}
			fields := map[string]string{"workingTitle": "WorkingTitle", "era": "Era", "dateRange": "DateRange"}
			for proposalField, goField := range fields {
				got := reflect.ValueOf(after).FieldByName(goField).Interface()
				old := reflect.ValueOf(*draft).FieldByName(goField).Interface()
				if proposalField == field {
					if reflect.DeepEqual(got, old) {
						t.Fatalf("%s did not change", goField)
					}
				} else if !reflect.DeepEqual(got, old) {
					t.Fatalf("applying %s changed unrelated %s: before=%#v after=%#v", field, goField, old, got)
				}
			}
			if after.AcquisitionSource != draft.AcquisitionSource || after.Notes != draft.Notes {
				t.Fatalf("draft manual data changed: before=%#v after=%#v", draft, after)
			}
		})
	}
}

func TestFeature362ProposalApplyRejectsUnsupportedSelectionWithoutPartialWrites(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{
		UserID: userID, Name: "Manual coin", Denomination: "Manual denomination",
		Notes: "manual notes", PurchaseLocation: "manual source", IsPrivate: true,
	}

	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination":  "Denarius",
		"purchasePrice": 999,
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination":  {Accepted: acceptTrue()},
		"purchasePrice": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Apply(jobID, userID, "coin", []string{"denomination", "purchasePrice"})
	if err == nil || !strings.Contains(err.Error(), "re_review_required") {
		t.Fatalf("expected re_review_required, got %v", err)
	}
	if !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("re-review error must preserve unsupported-field classification, got %v", err)
	}

	var after models.Coin
	if err := db.First(&after, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Denomination != coin.Denomination || after.Notes != coin.Notes ||
		after.PurchaseLocation != coin.PurchaseLocation || after.IsPrivate != coin.IsPrivate {
		t.Fatalf("failed apply partially mutated manual data: before=%#v after=%#v", coin, after)
	}
}

func TestFeature362ProposalApplyRollsBackScalarWhenReferenceWriteFails(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Manual coin", Denomination: "Manual"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination":      "Denarius",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "42")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination":      {Accepted: acceptTrue()},
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("feature362_reference_failure", func(tx *gorm.DB) {
		switch tx.Statement.Dest.(type) {
		case *[]models.CoinReference, *models.CoinReference:
			tx.AddError(errors.New("forced reference insert failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Apply(jobID, userID, "coin", []string{"denomination", "catalogReferences"}); err == nil {
		t.Fatal("expected forced reference failure")
	}
	var after models.Coin
	if err := db.First(&after, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Denomination != "Manual" {
		t.Fatalf("scalar escaped failed transaction: %q", after.Denomination)
	}
	var refCount int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", coin.ID).Count(&refCount).Error; err != nil {
		t.Fatal(err)
	}
	if refCount != 0 {
		t.Fatalf("reference escaped failed transaction: %d", refCount)
	}
	var job models.DeepIdentificationJob
	if err := db.First(&job, jobID).Error; err != nil {
		t.Fatal(err)
	}
	if job.AppliedAt != nil {
		t.Fatal("failed transaction marked proposal applied")
	}
}

func TestFeature362ProposalApplyRejectsTargetChangedAfterAnalysis(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Before analysis", Denomination: "Manual"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"denomination": "Denarius",
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("name", "Manual edit after analysis").Error; err != nil {
		t.Fatal(err)
	}

	_, err := svc.Apply(jobID, userID, "coin", []string{"denomination"})
	if !errors.Is(err, ErrDeepProposalReReviewRequired) {
		t.Fatalf("expected stale target to require re-review, got %v", err)
	}
	var after models.Coin
	if err := db.First(&after, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Name != "Manual edit after analysis" || after.Denomination != "Manual" {
		t.Fatalf("stale apply changed manual data: %#v", after)
	}
}

// realisticRunnerStreamFrames returns the exact Python-shaped SSE frames a
// real run_deep_identification_stream emits for a multi-provider run: full
// ProviderEvidence provider_result frames (claims included) for numista
// (denomination, era) and nomisma (mint), plus a terminal synthesis whose
// proposed_fields reference those claims by index and add an image-only
// `notes` field and one non-allowlisted key that must be dropped Go-side.
// The proposal these frames produce through the real runner is the input the
// Get/PATCH/Apply integration below exercises — no hand-built providerClaims.
func realisticRunnerStreamFrames() []string {
	numistaResult := map[string]any{
		"type": "provider_result", "provider": "numista", "status": "contributed",
		"automatable": true, "confidence": 0.7, "call_count": 1, "link_out": "",
		"attribution": "Source: Numista",
		"claims": []map[string]any{
			{"field": "denomination", "value": "Denarius", "confidence": 0.8, "citation": "https://en.numista.com/catalogue/pieces12345.html", "excerpt": "Silver denarius, Rome mint"},
			{"field": "era", "value": "roman-imperial", "confidence": 0.7, "citation": "https://en.numista.com/catalogue/pieces12345.html"},
		},
	}
	nomismaResult := map[string]any{
		"type": "provider_result", "provider": "nomisma", "status": "contributed",
		"automatable": true, "confidence": 0.6, "call_count": 1, "link_out": "",
		"attribution": "Data: Nomisma.org (CC BY)",
		"claims": []map[string]any{
			{"field": "mint", "value": "Rome", "confidence": 0.6, "citation": "http://nomisma.org/id/rome"},
		},
	}
	synthesis := map[string]any{
		"type": "synthesis",
		"report": map[string]any{
			"narrative": "A silver denarius of Septimius Severus.",
			"proposed_fields": map[string]any{
				"denomination":     map[string]any{"value": "Denarius", "confidence": 0.82, "evidence_refs": []map[string]any{{"provider": "numista", "claim_index": 0}}},
				"mint":             map[string]any{"value": "Rome", "confidence": 0.61, "evidence_refs": []map[string]any{{"provider": "nomisma", "claim_index": 0}}},
				"era":              map[string]any{"value": "roman-imperial", "confidence": 0.7, "evidence_refs": []map[string]any{{"provider": "numista", "claim_index": 1}}},
				"notes":            map[string]any{"value": "Bought at a show; dealer said Severan.", "confidence": 0.5, "evidence_refs": []map[string]any{{"provider": "image"}}},
				"secretAdminField": map[string]any{"value": "nope", "confidence": 0.99, "evidence_refs": []map[string]any{}},
			},
			"partial_success": false,
		},
	}
	frames := []map[string]any{
		{"type": "progress", "stage": "image_evidence_ready"},
		{"type": "router_selected", "selected": []string{"numista", "nomisma"}, "rationale": "auto"},
		{"type": "provider_started", "provider": "numista"},
		numistaResult,
		{"type": "provider_started", "provider": "nomisma"},
		nomismaResult,
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

// seedDeepJobWithRunnerProposal creates a job, drives the ACTUAL pipeline
// runner over a fake Python agent emitting realisticRunnerStreamFrames, and
// persists the runner-produced report+proposal onto the job. The proposal is
// therefore genuine runner output (streamed claims accumulated across the
// provider_result frames), not a hand-built providerClaims map — so it fails
// if production ever stops carrying claims across the stream boundary.
func seedDeepJobWithRunnerProposal(t *testing.T, db *gorm.DB, userID uint, source models.DeepJobSource, coinID *uint) uint {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		for _, f := range realisticRunnerStreamFrames() {
			_, _ = w.Write([]byte(f))
			flusher.Flush()
		}
	}))
	t.Cleanup(server.Close)

	runner := newDeepRunnerOnDB(t, db, server.URL)
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           source,
		CoinID:           coinID,
		Status:           models.DeepJobStatusRunning,
		InputFingerprint: fmt.Sprintf("fp-int-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}

	result, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if result == nil || result.ProposalJSON == "" {
		t.Fatal("expected a non-empty rich proposal JSON from the actual runner")
	}
	// Persist the terminal result onto the job exactly as SettleTerminal
	// would, then mark it completed so the proposal service can load/apply.
	if err := db.Model(job).Updates(map[string]any{
		"report_json":   result.ReportJSON,
		"proposal_json": result.ProposalJSON,
		"status":        models.DeepJobStatusCompleted,
	}).Error; err != nil {
		t.Fatalf("persist terminal result: %v", err)
	}
	return job.ID
}

// TestDeepIdentificationProposal_RunnerSynthesisThroughApply_SavedCoin is
// the B1 integration regression for the saved-coin path: a realistic runner
// synthesis + provider_result frames are translated to ProposalJSON,
// persisted, parsed/loaded by the proposal service, edited/accepted via
// PATCH, and applied to a saved coin. It fails under the old flat
// map[string]string proposal shape (which cannot be parsed into the rich
// document), asserts no coin write happens before apply, and asserts
// citation/confidence are preserved through persistence.
func TestDeepIdentificationProposal_RunnerSynthesisThroughApply_SavedCoin(t *testing.T) {
	svc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin", Notes: "original"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepJobWithRunnerProposal(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID)

	// Parse/load must succeed on the rich document and preserve citation +
	// confidence (this is the assertion that fails under the old flat shape).
	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(job.ProposalJSON), &doc); err != nil {
		t.Fatalf("proposal did not load as the rich document (old flat shape regression): %v", err)
	}
	if _, dropped := doc.Fields["secretAdminField"]; dropped {
		t.Fatal("non-allowlisted field must never appear in the persisted proposal")
	}
	den := doc.Fields["denomination"]
	if den == nil || den.Confidence != 0.82 {
		t.Fatalf("expected denomination confidence 0.82 preserved, got %#v", den)
	}
	if len(den.Evidence) != 1 || den.Evidence[0].Citation != "https://en.numista.com/catalogue/pieces12345.html" {
		t.Fatalf("expected citation preserved through persistence, got %#v", den.Evidence)
	}

	// No coin write before an explicit confirm: the coin is still pristine
	// after loading + editing the proposal.
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
		"mint":         {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	var beforeApply models.Coin
	if err := db.First(&beforeApply, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if beforeApply.Denomination != "" || beforeApply.Mint != "" {
		t.Fatalf("coin must not be written before apply, got denomination=%q mint=%q", beforeApply.Denomination, beforeApply.Mint)
	}

	// Owner edit is preserved distinctly from the AI value.
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"mint": {OwnerValue: "Lugdunum", OwnerValueSet: true, Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("owner-edit proposal: %v", err)
	}

	result, err := svc.Apply(jobID, userID, "coin", nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.CoinID == nil || *result.CoinID != coin.ID {
		t.Fatalf("expected coinId %d, got %v", coin.ID, result.CoinID)
	}
	var applied models.Coin
	if err := db.First(&applied, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if applied.Denomination != "Denarius" {
		t.Fatalf("expected AI denomination applied, got %q", applied.Denomination)
	}
	if applied.Mint != "Lugdunum" {
		t.Fatalf("expected owner-edited mint applied over AI value, got %q", applied.Mint)
	}
	if applied.Notes != "original" {
		t.Fatalf("expected untouched Notes, got %q", applied.Notes)
	}
}

// TestDeepIdentificationProposal_RunnerSynthesisThroughApply_NewIntake is
// the B1 integration regression for the new-intake path: the same runner
// synthesis flows through persistence, parse/load, PATCH accept, and Apply
// to a QuickCaptureDraft (never a direct coin write).
func TestDeepIdentificationProposal_RunnerSynthesisThroughApply_NewIntake(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	jobID := seedDeepJobWithRunnerProposal(t, db, userID, models.DeepJobSourceIntake, nil)

	// Only draft-allowlisted fields (era, notes) are applicable to a draft
	// target; coin-only fields (denomination/mint) remain proposable but not
	// draft-writable. Notes also satisfies the draft's minimum identity.
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"era":   {Accepted: acceptTrue()},
		"notes": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	var draftCountBefore int64
	if err := db.Model(&models.QuickCaptureDraft{}).Where("user_id = ?", userID).Count(&draftCountBefore).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.Apply(jobID, userID, "draft", []string{"era", "notes"})
	if err != nil {
		t.Fatalf("apply to draft: %v", err)
	}
	if result.DraftID == nil {
		t.Fatal("expected a draft id from the intake apply path")
	}
	if result.CoinID != nil {
		t.Fatal("intake apply must not write a coin directly")
	}

	var draftCountAfter int64
	if err := db.Model(&models.QuickCaptureDraft{}).Where("user_id = ?", userID).Count(&draftCountAfter).Error; err != nil {
		t.Fatal(err)
	}
	if draftCountAfter != draftCountBefore+1 {
		t.Fatalf("expected exactly one new draft, before=%d after=%d", draftCountBefore, draftCountAfter)
	}

	var draft models.QuickCaptureDraft
	if err := db.First(&draft, *result.DraftID).Error; err != nil {
		t.Fatal(err)
	}
	if string(draft.Era) != "roman-imperial" {
		t.Fatalf("expected era seeded onto draft, got %q", draft.Era)
	}
}

// TestDeepIdentificationProposal_ImageOnlyFieldRetainedWithEmptyEvidence is
// the T070 regression for Brian's exact bug reappearing in the proposal
// layer: a synthesis `proposed_fields` entry whose only evidence_ref is
// `{"provider": "image"}` (contract §5 — the image is not a provider and
// carries no citation) must still survive translation into the rich
// deepProposalDocument. buildDeepProposalDocumentJSON correctly skips
// *refs* whose provider is "image" (FR-025: "image" must never appear as a
// provider on any provider-facing surface), but that ref-level skip must
// never cascade into dropping the *field* itself — an image-only
// identification is still a real, owner-reviewable proposal.
//
// realisticRunnerStreamFrames' "notes" field has exactly one evidence_ref,
// `{"provider": "image"}`, and no other claims — it is the image-only case.
func TestDeepIdentificationProposal_ImageOnlyFieldRetainedWithEmptyEvidence(t *testing.T) {
	_, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepJobWithRunnerProposal(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID)

	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(job.ProposalJSON), &doc); err != nil {
		t.Fatalf("proposal did not parse as the rich document: %v", err)
	}

	notes, ok := doc.Fields["notes"]
	if !ok || notes == nil {
		t.Fatal("image-only field must not be dropped from the proposal document — it must be present for owner review")
	}
	if notes.Proposed != "A silver denarius of Septimius Severus.\n\nBought at a show; dealer said Severan." {
		t.Fatalf("expected the narrative and AI-proposed note preserved even without provider evidence, got %#v", notes.Proposed)
	}
	if len(notes.Evidence) != 0 {
		t.Fatalf("expected an empty evidence array for an image-only field (no provider citation exists), got %#v", notes.Evidence)
	}
	if notes.Accepted != nil {
		t.Fatalf("expected pristine accepted=nil before any owner decision, got %#v", notes.Accepted)
	}

	// The whole document must be non-empty: an image-only proposal is still
	// a real proposal, not silently discarded down to "{}".
	if len(doc.Fields) == 0 {
		t.Fatal("proposal document must not be empty when the synthesis proposed at least one image-only field")
	}
}
