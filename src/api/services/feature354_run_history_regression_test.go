package services

// Independent QA regression coverage for spec 354 (Deep-Identification Run
// History & Wishlist-Eligible Coin of the Day), owned by Brutus (Tester/QA).
//
// This file is deliberately separate from deep_identification_service_test.go
// and deep_identification_proposal_test.go (Cassius's tests-first files for
// the same phases) to avoid merge collisions while the team works
// concurrently. It expresses the FROZEN CONTRACT recorded in
// .squad/decisions/inbox/maximus-analysis-history-wishlist-featured.md
// (D1, D2, D6, D8, D13) and specs/354-.../spec.md (FR-001, FR-002,
// FR-007..011, FR-021..028).
//
// These tests intentionally exercise BEHAVIOR (does the janitor still
// delete artifacts? does re-apply return the same/new coin id? is the
// wishlist coin eligible?) rather than asserting on internal storage
// representation (nullable column vs. sentinel timestamp), because plan.md
// explicitly defers that implementation choice to Cassius. That keeps this
// suite valid regardless of which of the two documented D1 strategies
// lands.
//
// Until Cassius lands the Phase 2-6 implementation, most of these are
// expected to FAIL (red) against today's shipping behavior - that is the
// intended tests-first state per constitution §17, not a defect in this
// file. Re-run after implementation lands and report genuine failures
// accurately (do not weaken assertions to make them pass).

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// --- US6 / FR-001 / FR-002 / FR-013: retention regression -----------------

func TestFeature354_RetentionSweep_NeverDeletesCompletedJobArtifacts(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "f354-completed", Email: "f354-completed@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}

	// Mark the job completed and force its expiry deep into the past - the
	// same setup TestDeepIdentificationService_RetentionSweepDeletesAllArtifacts
	// (today's baseline behavior, which FR-001 supersedes for completed/partial).
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.DeepJobStatusCompleted,
			"completed_at": time.Now().Add(-200 * 24 * time.Hour),
			"expires_at":   time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.janitor.runRetentionSweep()

	var reloaded models.DeepIdentificationArtifact
	if err := db.First(&reloaded, obverse.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.DeletedAt != nil {
		t.Fatalf("FR-001 violation: completed job's obverse artifact %d was deleted by the retention sweep despite being past its old ExpiresAt", obverse.ID)
	}
	if _, err := os.Stat(obverse.FilePath); os.IsNotExist(err) {
		t.Fatalf("FR-001 violation: completed job's obverse artifact file was unlinked from disk")
	}

	var job models.DeepIdentificationJob
	if err := db.First(&job, jobID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != models.DeepJobStatusCompleted {
		t.Fatalf("completed job status must be unaffected by the retention sweep, got %s", job.Status)
	}
}

func TestFeature354_RetentionSweep_NeverDeletesPartialJobArtifacts(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "f354-partial", Email: "f354-partial@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":          models.DeepJobStatusPartial,
			"partial_success": true,
			"completed_at":    time.Now().Add(-200 * 24 * time.Hour),
			"expires_at":      time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.janitor.runRetentionSweep()

	var reloaded models.DeepIdentificationArtifact
	if err := db.First(&reloaded, obverse.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.DeletedAt != nil {
		t.Fatalf("FR-001 violation: partial job's obverse artifact %d was deleted by the retention sweep", obverse.ID)
	}
}

func TestFeature354_RetentionSweep_StillExpiresFailedJobArtifacts(t *testing.T) {
	// FR-002: failed/cancelled runs keep the existing 90-day behavior
	// unchanged - this proves the feature does NOT accidentally widen
	// retention to every terminal status.
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "f354-failed", Email: "f354-failed@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.DeepJobStatusFailed,
			"completed_at": time.Now().Add(-200 * 24 * time.Hour),
			"expires_at":   time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.janitor.runRetentionSweep()

	var reloaded models.DeepIdentificationArtifact
	if err := db.First(&reloaded, obverse.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.DeletedAt == nil {
		t.Fatalf("FR-002 regression: failed job's artifact should still expire on the existing 90-day window")
	}
}

func TestFeature354_RetentionSweep_StillExpiresCancelledJobArtifacts(t *testing.T) {
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "f354-cancelled", Email: "f354-cancelled@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	obverse, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.DeepJobStatusCancelled,
			"completed_at": time.Now().Add(-200 * 24 * time.Hour),
			"expires_at":   time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.janitor.runRetentionSweep()

	var reloaded models.DeepIdentificationArtifact
	if err := db.First(&reloaded, obverse.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.DeletedAt == nil {
		t.Fatalf("FR-002 regression: cancelled job's artifact should still expire on the existing 90-day window")
	}
}

func TestFeature354_RetentionSweep_IsIdempotentAcrossRestarts(t *testing.T) {
	// US6 "idempotent restart": running the sweep twice in a row (as would
	// happen if the process restarted mid-hour, re-triggering the janitor)
	// must not error, must not double-delete, and must leave the same
	// terminal state both times.
	svc, db, _ := newDeepIdentificationServiceTestDeps(t)
	user := models.User{Username: "f354-restart", Email: "f354-restart@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepTestJob(t, db, user.ID)
	if _, err := svc.ValidateAndSaveArtifact(jobID, user.ID, models.DeepArtifactRoleObverse, "obverse.png", tinyPNGBytes(t)); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DeepIdentificationJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.DeepJobStatusCompleted,
			"completed_at": time.Now().Add(-200 * 24 * time.Hour),
			"expires_at":   time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	svc.janitor.runRetentionSweep()
	svc.janitor.runRetentionSweep()

	var job models.DeepIdentificationJob
	if err := db.First(&job, jobID).Error; err != nil {
		t.Fatalf("job row must survive repeated retention sweeps: %v", err)
	}
	if job.Status != models.DeepJobStatusCompleted {
		t.Fatalf("expected status to remain completed across repeated sweeps, got %s", job.Status)
	}
}

// --- US2 / FR-007..011: re-apply idempotency -------------------------------

func TestFeature354_ApplyToWishlist_SecondClickReturnsExistingCoinNoDuplicate(t *testing.T) {
	proposalSvc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"name": "Diocletian follis",
	})

	first, err := proposalSvc.Apply(jobID, userID, "wishlist", nil)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if first.CoinID == nil {
		t.Fatalf("expected a coin id from the first apply")
	}

	second, err := proposalSvc.Apply(jobID, userID, "wishlist", nil)
	if err != nil {
		t.Fatalf("FR-007/FR-008 violation: re-apply on the same completed job must not error, got %v", err)
	}
	if second.CoinID == nil || *second.CoinID != *first.CoinID {
		t.Fatalf("FR-008 violation: expected the second apply to return the same coin id %v, got %v", first.CoinID, second.CoinID)
	}

	var count int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = 1", userID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("FR-008 violation: expected exactly one wishlist coin after two identical applies, got %d", count)
	}
}

func TestFeature354_ApplyToWishlist_RecreatesAfterLinkedCoinDeleted(t *testing.T) {
	proposalSvc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceIntake, nil, map[string]any{
		"name": "Constantine follis",
	})

	first, err := proposalSvc.Apply(jobID, userID, "wishlist", nil)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := db.Delete(&models.Coin{}, *first.CoinID).Error; err != nil {
		t.Fatalf("delete linked coin: %v", err)
	}

	second, err := proposalSvc.Apply(jobID, userID, "wishlist", nil)
	if err != nil {
		t.Fatalf("FR-009 violation: re-apply after the linked coin was deleted must create a fresh coin, got error %v", err)
	}
	if second.CoinID == nil {
		t.Fatalf("expected a new coin id after the linked coin was deleted")
	}
	if *second.CoinID == *first.CoinID {
		t.Fatalf("FR-009 violation: expected a NEW coin id distinct from the deleted %d, got the same id back", *first.CoinID)
	}

	var count int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = 1", userID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one surviving wishlist coin after delete+reapply, got %d", count)
	}
}

func TestFeature354_ApplyToSavedCoin_TargetMismatchStillRejected(t *testing.T) {
	// FR-012: source/target coupling must remain unchanged even after the
	// idempotency loosening - a saved_coin job can never apply to "wishlist".
	proposalSvc, _, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Existing coin", Category: models.CategoryRoman}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"name": "Existing coin (revised)",
	})

	if _, err := proposalSvc.Apply(jobID, userID, "wishlist", nil); !errors.Is(err, ErrDeepProposalTargetMismatch) {
		t.Fatalf("expected ErrDeepProposalTargetMismatch for saved_coin job applying to wishlist target, got %v", err)
	}
}

func TestFeature354_ApplyToCoin_ReapplyRefreshesAppliedAtWithoutDuplicating(t *testing.T) {
	// FR-011: target="coin" re-apply against an existing owned saved-coin
	// target must be a re-patch, never a duplicate coin, and AppliedAt is
	// refreshed on every successful re-apply.
	proposalSvc, repo, db := newDeepProposalTestDeps(t)
	userID := seedDeepProposalUser(t, db)
	coin := models.Coin{UserID: userID, Name: "Saved coin", Category: models.CategoryRoman}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	jobID := seedDeepProposalJob(t, db, userID, models.DeepJobSourceSavedCoin, &coin.ID, map[string]any{
		"name": "Saved coin (AI-revised)",
	})

	first, err := proposalSvc.Apply(jobID, userID, "coin", nil)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}

	time.Sleep(2 * time.Millisecond)
	second, err := proposalSvc.Apply(jobID, userID, "coin", nil)
	if err != nil {
		t.Fatalf("FR-011 violation: re-apply to the same coin target must not error, got %v", err)
	}
	if second.CoinID == nil || *second.CoinID != coin.ID {
		t.Fatalf("expected the re-apply to keep patching the same coin %d, got %v", coin.ID, second.CoinID)
	}
	if !second.AppliedAt.After(first.AppliedAt) {
		t.Fatalf("FR-011 violation: expected AppliedAt to be refreshed on re-apply, first=%v second=%v", first.AppliedAt, second.AppliedAt)
	}

	job, err := repo.GetJob(jobID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if job.AppliedCoinID == nil || *job.AppliedCoinID != coin.ID {
		t.Fatalf("expected job.AppliedCoinID to remain %d after re-apply", coin.ID)
	}

	var count int64
	if err := db.Model(&models.Coin{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("FR-011 violation: re-apply to coin target must never create a duplicate coin, got %d rows", count)
	}
}

// --- US5 / FR-021..024: wishlist-eligible Coin of the Day pool widening ---

func TestFeature354_PickNextCoinID_IncludesWishlistWhenOptedIn(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	repo := repository.NewFeaturedCoinRepository(db)
	user := models.User{Username: "f354-cotd-in", Email: "f354-cotd-in@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	wishlistCoin := models.Coin{UserID: user.ID, Name: "Wishlist coin", Category: models.CategoryRoman, IsWishlist: true}
	if err := db.Create(&wishlistCoin).Error; err != nil {
		t.Fatal(err)
	}

	id, err := repo.PickNextCoinID(user.ID, true)
	if err != nil {
		t.Fatalf("PickNextCoinID(includeWishlist=true): %v", err)
	}
	if id != wishlistCoin.ID {
		t.Fatalf("FR-022 violation: expected the wishlist coin %d to be eligible when opted in, got %d", wishlistCoin.ID, id)
	}
}

func TestFeature354_PickNextCoinID_ExcludesWishlistWhenOptedOut(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	repo := repository.NewFeaturedCoinRepository(db)
	user := models.User{Username: "f354-cotd-out", Email: "f354-cotd-out@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	wishlistCoin := models.Coin{UserID: user.ID, Name: "Wishlist coin", Category: models.CategoryRoman, IsWishlist: true}
	if err := db.Create(&wishlistCoin).Error; err != nil {
		t.Fatal(err)
	}

	id, err := repo.PickNextCoinID(user.ID, false)
	if err != nil {
		t.Fatalf("PickNextCoinID(includeWishlist=false): %v", err)
	}
	if id != 0 {
		t.Fatalf("FR-023 violation: expected byte-identical owned-only behavior (no eligible coins) when opted out, got coin id %d", id)
	}
}

func TestFeature354_PickNextCoinID_SoldWishlistCoinExcludedEvenWhenOptedIn(t *testing.T) {
	db := setupCoinOfDaySchedulerDB(t, true)
	repo := repository.NewFeaturedCoinRepository(db)
	user := models.User{Username: "f354-cotd-sold", Email: "f354-cotd-sold@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	soldWishlistCoin := models.Coin{UserID: user.ID, Name: "Sold wishlist coin", Category: models.CategoryRoman, IsWishlist: true, IsSold: true}
	if err := db.Create(&soldWishlistCoin).Error; err != nil {
		t.Fatal(err)
	}

	id, err := repo.PickNextCoinID(user.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != 0 {
		t.Fatalf("FR-022 violation: sold coins must remain excluded from the combined pool, got coin id %d", id)
	}
}

func TestFeature354_PickNextCoinID_FairCycleAcrossCombinedPool(t *testing.T) {
	// SC-004/FR-024: never-shown-first fairness must apply over the
	// combined owned+wishlist pool - not treat wishlist as a separate
	// lower-priority tier.
	db := setupCoinOfDaySchedulerDB(t, true)
	repo := repository.NewFeaturedCoinRepository(db)
	user := models.User{Username: "f354-cotd-fair", Email: "f354-cotd-fair@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	owned := models.Coin{UserID: user.ID, Name: "Owned coin", Category: models.CategoryRoman}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatal(err)
	}
	wishlist := models.Coin{UserID: user.ID, Name: "Wishlist coin", Category: models.CategoryRoman, IsWishlist: true}
	if err := db.Create(&wishlist).Error; err != nil {
		t.Fatal(err)
	}
	// Feature the owned coin already today; the wishlist coin has never
	// been shown, so it must win the next pick.
	if err := db.Create(&models.FeaturedCoin{UserID: user.ID, CoinID: owned.ID, FeaturedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}

	id, err := repo.PickNextCoinID(user.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != wishlist.ID {
		t.Fatalf("FR-024 violation: expected the never-shown wishlist coin %d to win the fair cycle, got %d", wishlist.ID, id)
	}
}

func TestFeature354_NaturalEligibilityAfterPromotionToOwned(t *testing.T) {
	// Q2/spec: once a wishlist-source pick is moved to the collection
	// (IsWishlist cleared via the existing coin-update path), it must
	// naturally re-enter the owned pool without any special-case code - the
	// existing never-shown-first fairness sort already prefers it.
	db := setupCoinOfDaySchedulerDB(t, true)
	repo := repository.NewFeaturedCoinRepository(db)
	user := models.User{Username: "f354-promote", Email: "f354-promote@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	coin := models.Coin{UserID: user.ID, Name: "Promoted coin", Category: models.CategoryRoman, IsWishlist: true}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}

	// While still a wishlist coin, it's eligible under includeWishlist=true.
	id, err := repo.PickNextCoinID(user.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != coin.ID {
		t.Fatalf("expected wishlist coin eligible before promotion, got %d", id)
	}

	// Simulate "Move to Collection": clear IsWishlist via the existing
	// coin-update path semantics (mirrors
	// CoinService.UpdateCoinWithFields({isWishlist: false})).
	if err := db.Model(&models.Coin{}).Where("id = ?", coin.ID).Update("is_wishlist", false).Error; err != nil {
		t.Fatal(err)
	}

	// Now it must be discoverable even with includeWishlist=false (owned pool).
	idAfter, err := repo.PickNextCoinID(user.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if idAfter != coin.ID {
		t.Fatalf("expected promoted coin to be naturally eligible in the owned-only pool, got %d", idAfter)
	}
}

// --- US5 / FR-027: deterministic wishlist fallback summary -----------------

func TestFeature354_BuildWishlistFallbackSummary_PrefixesBuildCoinSummary(t *testing.T) {
	coin := &models.Coin{
		Name:         "Diocletian follis",
		Denomination: "Follis",
		Ruler:        "Diocletian",
		Era:          models.EraAncient,
		Mint:         "Antioch",
	}
	base := buildCoinSummary(coin)
	got := buildWishlistFallbackSummary(coin)

	const wantPreamble = "From your wishlist"
	if !strings.HasPrefix(got, wantPreamble) {
		t.Fatalf("FR-027 violation: expected fallback summary prefixed with %q, got %q", wantPreamble, got)
	}
	if base == "" || !strings.Contains(got, base) {
		t.Fatalf("FR-027 violation: expected the fallback to be derived from buildCoinSummary (%q), got %q", base, got)
	}
}

func TestFeature354_BuildWishlistFallbackSummary_NeverEmpty(t *testing.T) {
	// SC-005: every wishlist coin_of_day pick must carry a non-empty
	// summary regardless of Python agent health - the deterministic
	// fallback must hold even for a bare-metadata coin.
	coin := &models.Coin{Name: "Unnamed coin"}
	got := buildWishlistFallbackSummary(coin)
	if got == "" {
		t.Fatalf("SC-005 violation: fallback summary must never be empty")
	}
}
