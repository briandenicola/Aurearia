package services

// Feature 352 Phase 3 — collection-valued proposal write surface
// (catalogReferences). Covers FR-001..FR-005, FR-012..FR-015, FR-045 and
// AC-001..003/AC-008..010 from
// specs/352-deep-identification-structured-results/spec.md.

import (
	"errors"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

// seedDeepProposalCatalog inserts a CatalogRegistry row into the same DB
// newDeepProposalTestDeps wires, mirroring seedCatalog in
// coin_reference_service_test.go (kept separate because that helper lives
// in a different test file/DB-bootstrap family with its own setupTestDB).
func seedDeepProposalCatalog(t *testing.T, db *gorm.DB, catalog string, volumeRequired bool) {
	t.Helper()
	if err := db.Create(&models.CatalogRegistry{
		Catalog:        catalog,
		DisplayName:    catalog,
		Era:            models.EraAncient,
		VolumeRequired: volumeRequired,
	}).Error; err != nil {
		t.Fatalf("seed catalog %s: %v", catalog, err)
	}
}

// validCatalogRefPayload builds one well-formed catalogReferences[] element
// as the wire `any` shape (map[string]any), the same shape json.Unmarshal
// into `any` would produce for a proposal field's Proposed/OwnerValue.
func validCatalogRefPayload(catalog, volume, number string) map[string]any {
	return map[string]any{
		"catalog":        catalog,
		"volume":         volume,
		"number":         number,
		"uri":            "https://example.org/" + catalog + "/" + number,
		"sourceProvider": "numista",
		"confidence":     0.9,
		"rawText":        catalog + " " + volume + " " + number,
		"needsVolume":    false,
	}
}

// --- AC-001/FR-013: accepted catalogReferences append without replacing existing refs ---

func TestDeepProposalApply_CatalogReferencesAppendWithoutReplacingExisting(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	// Pre-existing reference the apply must not disturb.
	existingRef := models.CoinReference{CoinID: coin.ID, Catalog: "RIC", Number: "1"}
	if err := db.Create(&existingRef).Error; err != nil {
		t.Fatal(err)
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "42")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(coin.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected existing ref preserved + new ref appended (2 total), got %d: %+v", len(refs), refs)
	}
	foundExisting, foundNew := false, false
	for _, r := range refs {
		if r.Number == "1" {
			foundExisting = true
		}
		if r.Number == "42" {
			foundNew = true
		}
	}
	if !foundExisting || !foundNew {
		t.Fatalf("expected both existing (#1) and new (#42) references present, got %+v", refs)
	}
}

// --- FR-032: ownerEdited/ownerValue filtered array is what applies ---

func TestDeepProposalApply_OwnerEditedValueIsWhatApplies(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	// AI proposed two references; owner edits the field down to only one
	// (a filtered array), which per resolveDeepProposalFieldValue must be
	// exactly what gets applied - not the original AI proposal.
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{
			validCatalogRefPayload("RIC", "", "1"),
			validCatalogRefPayload("RIC", "", "2"),
		},
	})
	ownerFiltered := []any{validCatalogRefPayload("RIC", "", "2")}
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {OwnerValue: ownerFiltered, OwnerValueSet: true, Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(coin.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Number != "2" {
		t.Fatalf("expected only the owner-filtered reference (#2) to apply, got %+v", refs)
	}
}

// --- FR-004: unknown properties on a catalogReferences element are rejected ---

func TestDeepProposalCatalogReferences_UnknownPropertyRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	badElement := validCatalogRefPayload("RIC", "", "1")
	badElement["unexpectedProperty"] = "should be rejected"

	// PATCH validation.
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	_, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {OwnerValue: []any{badElement}, OwnerValueSet: true},
	})
	if err == nil {
		t.Fatal("expected PATCH to reject an unknown catalogReferences property, got nil error")
	}

	// Apply validation - the AI-proposed value itself carries the unknown
	// property (no owner edit at all).
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{badElement},
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept without owner edit should not itself fail PATCH: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject an unknown catalogReferences property, got nil error")
	}
}

// --- FR-005: more than 10 catalogReferences entries are rejected ---

func TestDeepProposalCatalogReferences_MoreThanTenEntriesRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	tooMany := make([]any, 0, 11)
	for i := 0; i < 11; i++ {
		tooMany = append(tooMany, validCatalogRefPayload("RIC", "", string(rune('A'+i))))
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {OwnerValue: tooMany, OwnerValueSet: true},
	}); err == nil {
		t.Fatal("expected PATCH to reject more than 10 catalogReferences entries, got nil error")
	}

	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": tooMany,
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept without owner edit should not itself fail PATCH: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject more than 10 catalogReferences entries, got nil error")
	}
}

// --- FR-045: unknown catalog is rejected ---

func TestDeepProposalCatalogReferences_UnknownCatalogRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	// Deliberately do not seed the "BOGUS" catalog.
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("BOGUS", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	_, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"})
	if !errors.Is(err, ErrReferenceUnknownCatalog) {
		t.Fatalf("expected ErrReferenceUnknownCatalog, got %v", err)
	}

	// PATCH-side owner edit with an unknown catalog must also be rejected.
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{},
	})
	_, err = svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {OwnerValue: []any{validCatalogRefPayload("BOGUS", "", "1")}, OwnerValueSet: true},
	})
	if !errors.Is(err, ErrReferenceUnknownCatalog) {
		t.Fatalf("expected ErrReferenceUnknownCatalog on PATCH, got %v", err)
	}
}

// --- FR-045 (volume-required): empty volume for a volume-required catalog is rejected ---

func TestDeepProposalCatalogReferences_VolumeRequiredEmptyVolumeRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", true)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	_, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"})
	if !errors.Is(err, ErrReferenceVolumeRequired) {
		t.Fatalf("expected ErrReferenceVolumeRequired, got %v", err)
	}

	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{},
	})
	_, err = svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {OwnerValue: []any{validCatalogRefPayload("RIC", "", "1")}, OwnerValueSet: true},
	})
	if !errors.Is(err, ErrReferenceVolumeRequired) {
		t.Fatalf("expected ErrReferenceVolumeRequired on PATCH, got %v", err)
	}
}

// --- FR-004: invalid confidence and invalid sourceProvider are rejected ---

func TestDeepProposalCatalogReferences_InvalidConfidenceRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	tooHigh := validCatalogRefPayload("RIC", "", "1")
	tooHigh["confidence"] = 1.5
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{tooHigh},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject confidence > 1, got nil error")
	}

	tooLow := validCatalogRefPayload("RIC", "", "2")
	tooLow["confidence"] = -0.1
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{tooLow},
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject confidence < 0, got nil error")
	}
}

func TestDeepProposalCatalogReferences_InvalidSourceProviderRejected(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	badProvider := validCatalogRefPayload("RIC", "", "1")
	badProvider["sourceProvider"] = "wikipedia"
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{badProvider},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject an unrecognised sourceProvider, got nil error")
	}

	// Empty sourceProvider is likewise rejected (required field).
	emptyProvider := validCatalogRefPayload("RIC", "", "2")
	emptyProvider["sourceProvider"] = ""
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{emptyProvider},
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to reject an empty sourceProvider, got nil error")
	}
}

// --- FR-002/FR-003: catalogReferences (a collection key) can never reach
// the scalar setter; unknown keys remain ErrDeepProposalFieldNotAllowed ---

func TestDeepProposalCatalogReferences_NeverReachesScalarSetter(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	coin := models.Coin{UserID: userID, Name: "Test Coin", ReferenceText: "original text"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	// If catalogReferences ever reached setCoinFieldFromProposalValue as a
	// scalar it would either error attempting to stringify a slice, or
	// silently clobber an unrelated column. Prove applying catalogReferences
	// alone leaves ReferenceText untouched.
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.ReferenceText != "original text" {
		t.Fatalf("catalogReferences must never touch ReferenceText (scalar setter), got %q", updated.ReferenceText)
	}

	// A genuinely unknown field name (neither scalar nor collection
	// allowlist, and absent from this job's own proposal document) is
	// still rejected the same way it always was (mirrors
	// TestDeepIdentificationProposal_FieldAllowlistRejectsUnknownField).
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "2")},
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"totallyUnknownKey": {Accepted: acceptTrue()},
	}); !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed for an unknown key at PATCH, got %v", err)
	}
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept catalogReferences on jobID2: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "coin", []string{"catalogReferences", "totallyUnknownKey"}); !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed for an unknown key at Apply, got %v", err)
	}
}

// --- Reference failure does not mark job applied ---

func TestDeepProposalApply_ReferenceFailureDoesNotMarkJobApplied(t *testing.T) {
	svc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	// No catalog seeded: reference validation (reached through
	// decodeDeepProposalCatalogReferences -> NormalizeAndValidateOne) will
	// fail with ErrReferenceUnknownCatalog, exercising the "reference
	// write failure" branch of applyToCoin.
	coin := models.Coin{UserID: userID, Name: "Test Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("NOPE", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected Apply to fail on an unresolvable reference")
	}

	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if job.AppliedAt != nil {
		t.Fatalf("job must not be marked applied when the reference write fails, AppliedAt=%v", job.AppliedAt)
	}

	// The job remains applicable once retried with a valid catalog - proof
	// the failed attempt left no partial commitment (ApplyJob was never
	// reached, so a retry is a normal first apply, not a conflict).
	seedDeepProposalCatalog(t, db, "NOPE", false)
	if _, err := svc.Apply(jobID, userID, "coin", []string{"catalogReferences"}); err != nil {
		t.Fatalf("retry with a now-valid catalog should succeed: %v", err)
	}
}

// --- Non-owner cannot apply refs to another user's coin ---

func TestDeepProposalApply_NonOwnerCannotApplyReferencesToAnothersCoin(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	ownerID := seedDeepProposalUser(t, db)
	// seedDeepProposalUser derives its username from time.Now().UnixNano();
	// on a coarse-resolution clock two back-to-back calls can collide, so
	// seed the second user directly with an explicitly distinct username.
	other := models.User{Username: "proposal-other-user", Email: "proposal-other-user@example.com", PasswordHash: "x"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	otherID := other.ID
	seedDeepProposalCatalog(t, db, "RIC", false)

	coin := models.Coin{UserID: ownerID, Name: "Owner's Coin"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	// The job is seeded as belonging to ownerID; otherID's attempt to
	// load/apply/edit it must be treated as not-found (job lookup is
	// user-scoped via repo.GetJob(jobID, userID)), never leaking a write to
	// the owner's coin.
	jobID := seedDeepProposalJob(t, db, ownerID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, ownerID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("owner accept: %v", err)
	}

	if _, err := svc.Apply(jobID, otherID, "coin", []string{"catalogReferences"}); !errors.Is(err, ErrDeepProposalNotFound) {
		t.Fatalf("expected ErrDeepProposalNotFound for a non-owner apply attempt, got %v", err)
	}
	if _, err := svc.UpdateProposal(jobID, otherID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); !errors.Is(err, ErrDeepProposalNotFound) {
		t.Fatalf("expected ErrDeepProposalNotFound for a non-owner PATCH attempt, got %v", err)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(coin.ID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("non-owner attempt must never write a reference to the owner's coin, got %+v", refs)
	}

	// Owner's own apply still succeeds afterward.
	if _, err := svc.Apply(jobID, ownerID, "coin", []string{"catalogReferences"}); err != nil {
		t.Fatalf("owner apply after non-owner attempt: %v", err)
	}
}
