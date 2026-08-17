package services

// Feature 352 Phase 6b — wishlist apply-path structured references
// (applyToWishlist calling CoinReferenceService.AppendForCoin after
// CoinService.CreateCoin). Covers US-6, FR-013, FR-030, FR-040, FR-045,
// FR-048..FR-051 and AC-020/AC-033..AC-036 from
// specs/352-deep-identification-structured-results/spec.md, and the
// plan.md Phase 6b apply-path row + R2/R8 risk items.
//
// This file deliberately mirrors the structure of
// deep_identification_proposal_phase3_test.go (the "coin" target's
// equivalent Phase 3 coverage) so the two apply targets are proven to
// behave identically wherever the spec requires it (FR-013/FR-051 "on
// every target").

import (
	"errors"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

// --- AC-020/US-6/FR-030: wishlist apply persists accepted structured
// references and preserves normalized Catalog/Volume/Number ---

func TestDeepProposalApply_WishlistCatalogReferencesPersistNormalized(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle": "Trajan Denarius",
		// Deliberately messy casing/whitespace on catalog/volume/number to
		// prove NormalizeAndValidateOne's trimming and canonical-catalog
		// substitution survive the wishlist apply path exactly as they do
		// on "coin" (Phase 3).
		"catalogReferences": []any{validCatalogRefPayload("ric", " II ", " 42 ")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	result, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"})
	if err != nil {
		t.Fatalf("apply wishlist: %v", err)
	}
	if result.CoinID == nil {
		t.Fatal("expected a coin id for wishlist apply")
	}

	var coin models.Coin
	if err := db.First(&coin, *result.CoinID).Error; err != nil {
		t.Fatal(err)
	}
	if !coin.IsWishlist {
		t.Fatal("expected IsWishlist=true")
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(coin.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly one persisted reference, got %d: %+v", len(refs), refs)
	}
	if refs[0].Catalog != "RIC" || refs[0].Volume != "II" || refs[0].Number != "42" {
		t.Fatalf("expected normalized Catalog=RIC Volume=II Number=42, got Catalog=%q Volume=%q Number=%q", refs[0].Catalog, refs[0].Volume, refs[0].Number)
	}
}

// --- FR-013/FR-051: multiple references persist additively on the
// wishlist target, and case-insensitive duplicates within the same batch
// collapse via AppendForCoin's dedupe rather than producing two rows ---

func TestDeepProposalApply_WishlistCatalogReferencesAdditiveWithCaseInsensitiveDedupe(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)
	seedDeepProposalCatalog(t, db, "NGC", false)

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle": "Trajan Denarius",
		"catalogReferences": []any{
			// Two case-variant duplicates of the same (Catalog,Volume,Number)
			// triple in the same batch - AppendForCoin's dedupeKey must
			// collapse these to one row (FR-051, mirrors
			// coin_reference_service_test.go's same-batch dedupe case).
			validCatalogRefPayload("RIC", "", "42"),
			validCatalogRefPayload("ric", "", "42"),
			// A genuinely distinct reference, proving the surviving RIC row
			// and this NGC row both persist additively rather than one
			// replacing the other.
			validCatalogRefPayload("NGC", "", "99"),
		},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	result, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"})
	if err != nil {
		t.Fatalf("apply wishlist: %v", err)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(*result.CoinID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 rows (RIC 42 collapsed from 2 case variants + NGC 99), got %d: %+v", len(refs), refs)
	}
	var foundRIC, foundNGC bool
	for _, r := range refs {
		if r.Catalog == "RIC" && r.Number == "42" {
			foundRIC = true
		}
		if r.Catalog == "NGC" && r.Number == "99" {
			foundNGC = true
		}
	}
	if !foundRIC || !foundNGC {
		t.Fatalf("expected exactly one RIC/42 row and one NGC/99 row, got %+v", refs)
	}
}

// --- FR-032: ownerEdited/ownerValue filtered array is what applies, on
// the wishlist target ---

func TestDeepProposalApply_WishlistOwnerEditedValueIsWhatApplies(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle": "Trajan Denarius",
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

	result, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"})
	if err != nil {
		t.Fatalf("apply wishlist: %v", err)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(*result.CoinID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Number != "2" {
		t.Fatalf("expected only the owner-filtered reference (#2) to apply, got %+v", refs)
	}
}

// --- Rejected/unaccepted catalogReferences are never written on the
// wishlist target ---

func TestDeepProposalApply_WishlistUnacceptedCatalogReferencesNotWritten(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"denomination":      "Denarius",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	// Accept only "denomination" - catalogReferences is proposed but never
	// marked accepted, so with a nil fieldsFilter (the "apply every field
	// marked accepted" contract) it must be silently excluded, not applied.
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"denomination": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	result, err := svc.Apply(jobID, userID, "wishlist", nil)
	if err != nil {
		t.Fatalf("apply wishlist: %v", err)
	}
	if len(result.AppliedFields) != 1 || result.AppliedFields[0] != "denomination" {
		t.Fatalf("expected only denomination to be applied, got %v", result.AppliedFields)
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	refs, err := refRepo.ListByCoin(*result.CoinID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected zero references written when catalogReferences was never accepted, got %+v", refs)
	}

	// Explicitly requesting the unaccepted field via fieldsFilter is also
	// rejected (mirrors the "coin" target's identical rule).
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.Apply(jobID2, userID, "wishlist", []string{"catalogReferences"}); !errors.Is(err, ErrDeepProposalNoAcceptedFields) {
		t.Fatalf("expected ErrDeepProposalNoAcceptedFields for an unaccepted explicit field, got %v", err)
	}
}

// --- Validation failures (unknown catalog, missing required volume, bad
// shape, >10 elements) prevent job AppliedAt and the success journal state
// on the wishlist target ---

func TestDeepProposalApply_WishlistValidationFailuresBlockAppliedAtAndJournal(t *testing.T) {
	cases := []struct {
		name         string
		seedCatalogs func(t *testing.T, db *gorm.DB)
		refs         []any
		wantErrIs    error
	}{
		{
			name: "unknown catalog",
			seedCatalogs: func(t *testing.T, db *gorm.DB) {
				// Deliberately does not seed "BOGUS".
			},
			refs:      []any{validCatalogRefPayload("BOGUS", "", "1")},
			wantErrIs: ErrReferenceUnknownCatalog,
		},
		{
			name: "missing required volume",
			seedCatalogs: func(t *testing.T, db *gorm.DB) {
				seedDeepProposalCatalog(t, db, "RIC", true)
			},
			refs:      []any{validCatalogRefPayload("RIC", "", "1")},
			wantErrIs: ErrReferenceVolumeRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, db := newDeepProposalTestDeps(t)
			userID := seedDeepProposalUser(t, db)
			tc.seedCatalogs(t, db)

			jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
				"workingTitle":      "Trajan Denarius",
				"catalogReferences": tc.refs,
			})
			if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
				"catalogReferences": {Accepted: acceptTrue()},
			}); err != nil {
				t.Fatalf("accept should not itself fail: %v", err)
			}

			var coinCountBefore int64
			if err := db.Model(&models.Coin{}).Count(&coinCountBefore).Error; err != nil {
				t.Fatal(err)
			}

			_, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"})
			if !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("expected %v, got %v", tc.wantErrIs, err)
			}

			job, err := repo.GetJob(jobID, userID)
			if err != nil {
				t.Fatal(err)
			}
			if job.AppliedAt != nil {
				t.Fatalf("job must not be marked applied when the wishlist reference write fails, AppliedAt=%v", job.AppliedAt)
			}

			var journalCount int64
			if err := db.Model(&models.CoinJournal{}).Count(&journalCount).Error; err != nil {
				t.Fatal(err)
			}
			if journalCount != 0 {
				t.Fatalf("expected no journal entry recorded when the wishlist reference write fails, got %d", journalCount)
			}

			// Registry-validation failures (unknown catalog, missing
			// required volume) are caught by
			// resolveDeepProposalCatalogReferences/NormalizeAndValidateOne
			// *inside the field-collection loop*, strictly before
			// applyToWishlist ever calls CoinService.CreateCoin - so
			// unlike a genuine AppendForCoin/CreateBatch write failure
			// (see TestDeepProposalApply_WishlistReferenceFailureLeavesCreatedCoinAndRetryDuplicatesIt),
			// these specific validation failures never create a coin row
			// at all. Confirmed identical ordering in applyToCoin.
			var coinCountAfter int64
			if err := db.Model(&models.Coin{}).Count(&coinCountAfter).Error; err != nil {
				t.Fatal(err)
			}
			if coinCountAfter != coinCountBefore {
				t.Fatalf("expected no wishlist coin created when catalogReferences fails registry validation (validated before CreateCoin), got before=%d after=%d", coinCountBefore, coinCountAfter)
			}
		})
	}
}

// --- Bad shape (unknown property) and >10 elements also block AppliedAt,
// exercised separately since they fail decode before any registry lookup ---

func TestDeepProposalApply_WishlistBadShapeAndTooManyElementsBlockApply(t *testing.T) {
	svc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	badElement := validCatalogRefPayload("RIC", "", "1")
	badElement["unexpectedProperty"] = "should be rejected"
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"catalogReferences": []any{badElement},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"}); !errors.Is(err, ErrDeepProposalInvalidCatalogReferences) {
		t.Fatalf("expected ErrDeepProposalInvalidCatalogReferences for an unknown property, got %v", err)
	}
	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if job.AppliedAt != nil {
		t.Fatal("job must not be marked applied on a bad-shape catalogReferences element")
	}

	tooMany := make([]any, 0, 11)
	for i := 0; i < 11; i++ {
		tooMany = append(tooMany, validCatalogRefPayload("RIC", "", "1"))
	}
	jobID2 := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"catalogReferences": tooMany,
	})
	if _, err := svc.UpdateProposal(jobID2, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}
	if _, err := svc.Apply(jobID2, userID, "wishlist", []string{"catalogReferences"}); !errors.Is(err, ErrDeepProposalInvalidCatalogReferences) {
		t.Fatalf("expected ErrDeepProposalInvalidCatalogReferences for >10 elements, got %v", err)
	}
	job2, err := repo.GetJob(jobID2, userID)
	if err != nil {
		t.Fatal(err)
	}
	if job2.AppliedAt != nil {
		t.Fatal("job must not be marked applied when catalogReferences exceeds the 10-element cap")
	}
}

// --- FR-030/FR-040: journal text names the "catalogReferences" field but
// never leaks catalog/cert/reference values, on the wishlist target ---

func TestDeepProposalApply_WishlistJournalNamesFieldNeverValues(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	const secretNumber = "XJ-77-SECRET-CERT-9001"
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", secretNumber)},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	result, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"})
	if err != nil {
		t.Fatalf("apply wishlist: %v", err)
	}

	var journal models.CoinJournal
	if err := db.Where("coin_id = ?", *result.CoinID).First(&journal).Error; err != nil {
		t.Fatalf("expected a journal entry, got: %v", err)
	}
	if !strings.Contains(journal.Entry, "catalogReferences") {
		t.Fatalf("expected journal entry to name the catalogReferences field, got %q", journal.Entry)
	}
	if strings.Contains(journal.Entry, "RIC") || strings.Contains(journal.Entry, secretNumber) || strings.Contains(journal.Entry, "numista") {
		t.Fatalf("journal entry must never contain catalog/cert/reference values, got %q", journal.Entry)
	}
}

// --- Draft target behavior is unchanged: catalogReferences is not in
// deepProposalDraftFieldAllowlist, so requesting it on "draft" is rejected
// exactly like any other unknown field, and no DraftLifecycleEvent/draft
// row is affected by catalogReferences content ---

func TestDeepProposalApply_DraftTargetDoesNotApplyCatalogReferences(t *testing.T) {
	svc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"notes":             "Some notes.",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", "1")},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"notes":             {Accepted: acceptTrue()},
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	// Explicitly requesting catalogReferences on the draft target is
	// rejected - it is not a key in deepProposalDraftFieldAllowlist.
	if _, err := svc.Apply(jobID, userID, "draft", []string{"catalogReferences"}); !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed for catalogReferences on draft target, got %v", err)
	}

	// The nil-fieldsFilter path (apply every accepted field) also never
	// silently drops into a catalogReferences write for drafts: since both
	// "notes" and "catalogReferences" are accepted, this must still fail
	// the same way, proving there is no draft code path that reaches
	// AppendForCoin/CoinReferenceService at all.
	if _, err := svc.Apply(jobID, userID, "draft", nil); !errors.Is(err, ErrDeepProposalFieldNotAllowed) {
		t.Fatalf("expected ErrDeepProposalFieldNotAllowed for draft apply with catalogReferences accepted, got %v", err)
	}

	var draftCount int64
	if err := db.Model(&models.QuickCaptureDraft{}).Count(&draftCount).Error; err != nil {
		t.Fatal(err)
	}
	if draftCount != 0 {
		t.Fatal("expected no draft row created when the draft apply is rejected for an unsupported field")
	}

	// With catalogReferences excluded from the fieldsFilter, the draft
	// target still works normally on its own scalar fields - unchanged
	// behavior.
	result, err := svc.Apply(jobID, userID, "draft", []string{"notes"})
	if err != nil {
		t.Fatalf("apply draft with only supported fields: %v", err)
	}
	if result.DraftID == nil {
		t.Fatal("expected a draft id")
	}

	refRepo := repository.NewCoinReferenceRepository(db)
	var refCount int64
	if err := db.Model(&models.CoinReference{}).Count(&refCount).Error; err != nil {
		t.Fatal(err)
	}
	if refCount != 0 {
		t.Fatalf("expected zero coin_references rows to ever exist - draft apply has no coin to attach references to, got %d", refCount)
	}
	_ = refRepo
}

// --- Phase 3/6 retry/partial-write assessment (task item 8): CreateCoin
// necessarily precedes AppendForCoin in applyToWishlist (plan.md Phase 6b
// apply-path row). This documents, rather than blocks, the consequence: a
// reference-write failure after a successful CreateCoin leaves a created
// wishlist coin behind, and a client retry of the same job (still
// unapplied, since AppliedAt was correctly withheld) calls applyToWishlist
// again and creates a SECOND wishlist coin rather than reusing the first.
//
// IMPORTANT (finding worth recording): registry-validation failures
// (unknown catalog / missing required volume - see
// TestDeepProposalApply_WishlistValidationFailuresBlockAppliedAtAndJournal)
// do NOT reach this window at all: resolveDeepProposalCatalogReferences ->
// decodeDeepProposalCatalogReferences -> NormalizeAndValidateOne runs
// entirely inside applyToWishlist's field-collection loop, strictly
// *before* CreateCoin is ever called (confirmed identical in applyToCoin,
// before UpdateCoinWithFields). Also notable: CoinRepository.Create itself
// unconditionally does `Preload("References")` immediately after every
// insert (coin_repository.go), so a *schema*-level fault on the
// coin_references table (e.g. a dropped table) actually fails CreateCoin
// itself first - it can never produce the "CreateCoin succeeded, then the
// reference write silently failed" split. The real remaining window is
// narrower than the Apply() doc comment's generic phrasing suggests: only
// a genuine per-statement failure isolated to AppendForCoin's own
// CreateBatch insert (not a schema/table-level fault) can land here - e.g.
// a transient write error on that specific statement. Reproduced below
// with a targeted GORM create-callback that fails only the
// coin_references INSERT (not the SELECT/Preload CreateCoin also issues
// against the same table), isolating exactly that window.
//
// This is NOT a novel defect introduced by Phase 6b. It is the exact
// scenario the existing Apply() doc comment already names for the
// journal-write-failure case ("...letting a client retry re-run
// applyToWishlist/applyToCoin again, creating a *second* wishlist coin...
// Do not turn this back into a hard error without first making the whole
// apply transactional") generalized to any failure between CreateCoin and
// ApplyJob - which necessarily includes the reference write Phase 6b adds,
// since it explicitly runs before ApplyJob (plan.md Phase 6b risk R8:
// "Partial apply between scalar write and reference write ... Medium ...
// Both writes before ApplyJob; documented ordering; explicit review item").
// Verdict: documented, accepted, non-blocking. Making applyToWishlist
// transactional (or de-duplicating retries) is a distinct, larger design
// change out of scope for a targeted Phase 6b apply-path change, and
// Principle IV (simplest complete proportional change) argues against
// bundling it here uninvited.
func TestDeepProposalApply_WishlistReferenceFailureLeavesCreatedCoinAndRetryDuplicatesIt(t *testing.T) {
	svc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	seedDeepProposalCatalog(t, db, "RIC", false)

	const poisonNumber = "FORCE-APPEND-FAILURE"
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"workingTitle":      "Trajan Denarius",
		"catalogReferences": []any{validCatalogRefPayload("RIC", "", poisonNumber)},
	})
	if _, err := svc.UpdateProposal(jobID, userID, map[string]DeepProposalFieldEdit{
		"catalogReferences": {Accepted: acceptTrue()},
	}); err != nil {
		t.Fatalf("accept should not itself fail: %v", err)
	}

	// Fail only the coin_references INSERT statement carrying the poison
	// number - never any SELECT (so CreateCoin's own Preload("References")
	// and AppendForCoin's ListByCoin/registry lookups are unaffected) -
	// isolating exactly the "validation already passed, the append insert
	// itself fails" window.
	callbackName := "test:fail_poison_coin_reference_insert"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if refs, ok := tx.Statement.Dest.(*[]models.CoinReference); ok {
			for _, r := range *refs {
				if r.Number == poisonNumber {
					_ = tx.AddError(errors.New("injected failure: coin_references insert rejected"))
					return
				}
			}
		}
	}); err != nil {
		t.Fatalf("register test callback: %v", err)
	}

	if _, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"}); err == nil {
		t.Fatal("expected the apply to fail when the coin_references insert is rejected")
	}
	// Remove the injected fault before retrying - the retry call below is
	// the "same job, now with a healthy write path" case, not a repeat of
	// the fault.
	if err := db.Callback().Create().Remove(callbackName); err != nil {
		t.Fatalf("remove test callback: %v", err)
	}

	var wishlistCoinsAfterFailure int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ?", userID, true).Count(&wishlistCoinsAfterFailure).Error; err != nil {
		t.Fatal(err)
	}
	if wishlistCoinsAfterFailure != 1 {
		t.Fatalf("expected the failed attempt to still leave exactly one created wishlist coin behind (CreateCoin precedes AppendForCoin), got %d", wishlistCoinsAfterFailure)
	}

	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if job.AppliedAt != nil {
		t.Fatal("job must remain unapplied so a retry is possible")
	}

	// Retry the same still-unapplied job now that the write path is
	// healthy again.
	if _, err := svc.Apply(jobID, userID, "wishlist", []string{"catalogReferences"}); err != nil {
		t.Fatalf("retry once the write path is healthy again should succeed: %v", err)
	}

	var wishlistCoinsAfterRetry int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ?", userID, true).Count(&wishlistCoinsAfterRetry).Error; err != nil {
		t.Fatal(err)
	}
	if wishlistCoinsAfterRetry != 2 {
		t.Fatalf("documented R8 partial-write behavior: expected the retry to create a SECOND wishlist coin (not reuse the first), got %d total wishlist coins", wishlistCoinsAfterRetry)
	}
}
