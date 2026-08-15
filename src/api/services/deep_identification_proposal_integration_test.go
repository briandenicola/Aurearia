package services

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// realisticDeepSynthesisReport returns a synthesis `report` payload shaped
// exactly like the Python DeepSynthesis dump the pipeline runner receives
// on the terminal `synthesis` frame (contract §5), plus the matching
// provider_result claims the runner accumulates while streaming. It mixes
// allowlisted coin fields (denomination, mint, era), citation-bearing
// evidence, and one non-allowlisted key that must be dropped.
func realisticDeepSynthesisReport() (json.RawMessage, map[string][]deepProposalClaim) {
	report := json.RawMessage(`{
		"narrative": "A silver denarius of Septimius Severus.",
		"proposed_fields": {
			"denomination": {"value":"Denarius","confidence":0.82,"evidence_refs":[{"provider":"numista","claim_index":0}]},
			"mint": {"value":"Rome","confidence":0.61,"evidence_refs":[{"provider":"nomisma","claim_index":0}]},
			"era": {"value":"roman-imperial","confidence":0.7,"evidence_refs":[{"provider":"numista","claim_index":1}]},
			"notes": {"value":"Bought at a show; dealer said Severan.","confidence":0.5,"evidence_refs":[]},
			"secretAdminField": {"value":"nope","confidence":0.99,"evidence_refs":[]}
		},
		"partial_success": false
	}`)
	claims := map[string][]deepProposalClaim{
		"numista": {
			{Field: "denomination", Value: "Denarius", Confidence: 0.8, Citation: "https://en.numista.com/catalogue/pieces12345.html", Excerpt: "Silver denarius, Rome mint"},
			{Field: "era", Value: "roman-imperial", Confidence: 0.7, Citation: "https://en.numista.com/catalogue/pieces12345.html"},
		},
		"nomisma": {
			{Field: "mint", Value: "Rome", Confidence: 0.6, Citation: "http://nomisma.org/id/rome"},
		},
	}
	return report, claims
}

func seedDeepJobWithBuiltProposal(t *testing.T, db *gorm.DB, userID uint, source models.DeepJobSource, coinID *uint) uint {
	t.Helper()
	report, claims := realisticDeepSynthesisReport()
	proposalJSON := buildDeepProposalDocumentJSON(report, coinID, claims)
	if proposalJSON == "" {
		t.Fatal("expected a non-empty rich proposal JSON from the runner builder")
	}
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           source,
		CoinID:           coinID,
		Status:           models.DeepJobStatusCompleted,
		InputFingerprint: fmt.Sprintf("fp-int-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&deepTestDBCounter, 1)),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
		ReportJSON:       string(report),
		ProposalJSON:     proposalJSON,
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
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
	jobID := seedDeepJobWithBuiltProposal(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID)

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
	jobID := seedDeepJobWithBuiltProposal(t, db, userID, models.DeepJobSourceIntake, nil)

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
