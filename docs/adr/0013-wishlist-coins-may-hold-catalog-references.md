# ADR 0013: Wishlist Coins May Hold Catalog References

Date: 2026-08-17
Status: Accepted

Amends: Feature 351 spec (`specs/351-vision-first-deep-identification/spec.md`
lines 843-852) and `specs/351-vision-first-deep-identification/tasks.md:277`,
per Constitution SS22 (Amendment Process).

## Context

### What the current rule is

`CoinService` unconditionally discards the `References` relation on any coin
created with `IsWishlist == true`, in two places:

| Location | Code |
|---|---|
| `src/api/services/coin_service.go:154-156` | `if coin.IsWishlist { coin.References = nil }` inside `prepareCoinForCreate` |
| `src/api/services/coin_service.go:175-177` | `if coin.IsWishlist { pendingReferences = nil }` inside `createPreparedCoinInTx`, under the comment at lines 173-174: "Wishlist coins must never carry catalog references -- enforce the invariant here regardless of what the caller set upstream (belt-and-suspenders)." |

Three tests encode it:

- `src/api/services/coin_service_test.go:438` `TestCreateCoin_DropsReferencesForWishlist`
- `src/api/services/coin_service_test.go:490` `TestCreateCoin_WishlistInvariant_AgentStylePayload`
- `src/api/handlers/coin_handler_test.go:2075` `TestCoinHandler_Create_WishlistWithReferencesStoresZeroReferences`
- plus an inline assertion at `src/api/services/wishlist_search_alert_service_test.go:615`
  ("Wishlist coins discard references -- no `coin_references` rows should exist")

### Why it exists -- the recorded rationale, and what is missing

**There is no recorded domain rationale for this rule.** This ADR states that
plainly rather than inventing one. What the record actually contains is:

1. **Origin (2026-07-21, branch `fix/agent-wishlist-reference-ids`).** Per
   `.squad/agents/cassius/history.md:50-62`, the rule was introduced as the
   fourth of four **defensive layers** against a crash: `ConvertCandidateInput.Coin`
   is a raw `models.Coin`, so agent-supplied references arrived carrying non-zero
   primary keys, and GORM's batch `db.Create(&slice)` emits those IDs in the
   INSERT, producing `UNIQUE constraint failed: coin_references.id`. The user-visible
   symptom was "Failed to add coin to wishlist: duplicate references are not
   allowed."

   The three layers that actually fix the root cause are:
   - `CoinReferenceService.NormalizeAndValidate` zeroes `.ID` and `.CoinID`
     (commit `feb2306`);
   - `createPreparedCoinInTx` detaches `pendingReferences` before `txRepo.Create(coin)`
     so GORM cannot auto-cascade;
   - `CoinRepository.Create` uses `Omit("References")` (`coin_repository.go:412`).

   Layer 4 -- dropping references for wishlist coins (commit `4bb4636`) -- was
   described in that same entry as covering "the specific agent/ConvertCandidate
   path". It is **redundant with respect to the crash**: layers 1-3 already
   prevent it, and there are regression tests for each.

2. **Elevation to "invariant" (2026-07-21).** `.squad/agents/brutus/history.md:46`
   records it as "Primary invariant (Brian's spec): Wishlist coins must have ZERO
   persisted catalog references after creation." So Brian did state it as a rule
   at the time -- but the *reason* recorded is the bug, not a numismatic or
   data-modelling argument.

3. **Ratification (2026-08-16, Feature 351).** `specs/351-.../spec.md:843-852`
   calls it "the pre-existing and intended invariant" while establishing that
   `Coin.ReferenceText` (a scalar, `models/coin.go:76`) is distinct from the
   `References` relation (`models/coin.go:92`), so catalogue *text* survives on a
   wishlist coin. 351 inherited the rule; it did not justify it.

**Conclusion: the rule is a bug workaround that was retroactively described as a
domain invariant.** No document in this repository states why a coin a collector
wants to buy should be forbidden from recording which catalogue type it is.

### The rule is already not true in production

Two shipped write paths create `CoinReference` rows for wishlist coins today,
both bypassing `CoinService` entirely:

1. **`QuickCaptureRepository.PromoteDraftTransaction`**
   (`src/api/repository/quick_capture_repository.go:291-300`) calls
   `tx.Create(&models.CoinReference{...})` directly after inserting the coin.
   `quick_capture_service.go:530` sets `coin.IsWishlist` from the normalised
   promotion target, so promoting a draft with a selected Numista reference to
   the **wishlist** persists a reference. This is Feature 341 FR-014 behaviour and
   is covered by `TestQuickCaptureServiceSelectedReferencePromotionCollectionWishlistAndNoSelection`
   (`quick_capture_service_test.go:182`).

2. **`ReferenceMigrationService.MigrateLegacyReferences`**
   (`src/api/services/reference_migration_service.go:47-56`) selects coins by
   `user_id` and `TRIM(rarity_rating) <> ''` with **no `is_wishlist` filter**, then
   `s.db.Create(ref)`. Any wishlist coin with legacy `rarity_rating` text has
   already been given a structured reference by this migration.

So there is no clean state to protect. The database already contains wishlist
coins with catalog references.

### What forced the decision

Feature 352 (`specs/352-deep-identification-structured-results/`) requires the
`wishlist` apply target of `DeepIdentificationProposalService.Apply` to write
structured NGC / RIC / RPC references, exactly as the `coin` target does. Spec
352 Conflict C-1 flagged this as blocked pending an amendment.

## Decision

**Wishlist coins MAY hold catalog references.** The `IsWishlist` flag records
*acquisition intent*; it is not a statement about how much is known about the
coin type. A collector tracking a specific RIC type they intend to buy has
exactly as much need for a structured reference as an owner cataloguing one they
already own -- arguably more, since the reference is the search key.

The two guards in `CoinService` are removed. `IsWishlist` stops participating in
reference handling at all: a wishlist coin's references travel the same
`NormalizeAndValidate` -> `CreateBatch` path as a collection coin's.

### What replaces the removed enforcement

Nothing new is required for correctness -- the crash the guards were added for is
prevented by three independent, tested layers that remain in place. But because
removing the guards re-opens an *unvalidated, non-confirm-gated* input path (see
Consequences, item 5), the following are **required** as part of the change:

- **V-1.** Every wishlist-coin reference write MUST go through
  `CoinReferenceService.NormalizeAndValidate` (ID zeroing, catalog-registry
  validation, `VolumeRequired` enforcement, duplicate rejection). No caller may
  reach `CoinReferenceRepository` with wishlist references directly. The existing
  `refRepo != nil && refSvc != nil` guard in `createPreparedCoinInTx` already
  enforces the routing; do not add a bypass.
- **V-2.** `WishlistSearchAlertService.ConvertCandidate` MUST NOT persist
  references taken verbatim from `ConvertCandidateInput.Coin.References`. That
  input is an unvalidated `models.Coin` from the request body, ultimately sourced
  from LLM web-search output. Until candidate references are confirm-gated in the
  UI the same way deep-identification proposals are, `ConvertCandidate` MUST clear
  `input.Coin.References` at its own boundary -- moving the drop from a
  type-wide invariant to a **single untrusted-source guard**, which is where it
  should have been in the first place.
- **V-3.** The three tests listed above are rewritten to assert the new rule
  (wishlist coins persist their normalised references), and a new test asserts
  V-2 (the alert-candidate path still persists zero references, for a stated
  trust reason rather than a wishlist reason).

## Consequences

### 1. Enforcement points removed

| File:line | Action |
|---|---|
| `src/api/services/coin_service.go:154-156` | Delete the `if coin.IsWishlist { coin.References = nil }` block in `prepareCoinForCreate`. |
| `src/api/services/coin_service.go:173-177` | Delete the comment and the `if coin.IsWishlist { pendingReferences = nil }` block in `createPreparedCoinInTx`. The `pendingReferences := coin.References; coin.References = nil` detach on lines 171-172 MUST STAY -- it is GORM cascade defence, not wishlist logic, and deleting it reintroduces the original crash. |

### 2. Downstream effects of removing the strip in `createPreparedCoinInTx` -- checked, one by one

- **Value snapshots: no effect.** `createPreparedCoinInTx` ends with
  `txRepo.RecordValueSnapshot(coin.UserID)`. Snapshots aggregate coin value
  columns; they never read `coin_references`. The reference write and the snapshot
  are in the same transaction and remain so.
- **Reference normalisation: no effect, and it is now reached more often.**
  `NormalizeAndValidate` is called on `pendingReferences` at
  `coin_service.go:183`. Today wishlist coins never reach it because
  `pendingReferences` was nilled first. After this change they do, which means
  wishlist references get catalog-registry validation, `VolumeRequired`
  enforcement, and ID zeroing -- strictly more validation than the current
  bypassed-into-nothing path. This is the desired direction.
- **`ReferenceMigrationService`: no effect.** It has never filtered on
  `is_wishlist` (`reference_migration_service.go:52-53`) and does not use
  `CoinService`. Its behaviour is unchanged by this ADR. What changes is that its
  output is no longer anomalous. Its duplicate guard
  (`reference_migration_service.go:86-88`, keyed on
  `coin_id + catalog + volume + number`) will now also protect against
  re-creating a reference a deep-ID apply already wrote to a wishlist coin.
- **`CoinReferenceRepository`: no effect.** Every method scopes by
  `coin_id IN (owned coins)` (`coin_reference_repository.go:34-35, 45-46, 78-79,
  87-88`). No method filters on `is_wishlist`. Owner scoping is preserved.
- **`CoinRepository.Duplicate`: behaviour improves.** It already copies
  `source.References` (`coin_repository.go:472-486`) and already copies
  `IsWishlist` (line 456). Today, duplicating a wishlist coin copies zero
  references because there were none. It will now copy them, which is the
  consistent outcome.
- **`PurchaseCoin` (wishlist -> collection): behaviour improves.**
  `coin_service.go:368-380` flips `is_wishlist` to false via an `UpdateField`-style
  map. It does not touch references. Today a wishlist coin arrives in the
  collection with no references and the owner re-types them; after this change the
  references carry across the purchase. This is the single clearest user-facing
  win.
- **`CatalogRegistryRepository.CountReferencesUsing`
  (`catalog_registry_repository.go:71-75`): count increases.** It counts all
  `coin_references` rows for a catalog code and is used to protect a catalog from
  deletion while in use. Wishlist references will now be counted, so a catalog
  used only by wishlist coins becomes undeletable. This is correct, and worth
  knowing.
- **Availability checking: no effect.** The wishlist availability scheduler and
  service key on the scalar `Coin.ReferenceURL` column, not the `References`
  relation (`availability_scheduler_test.go:228`).
- **PDF export / statistics / valuation: no effect.** These exclude wishlist coins
  by `IsWishlist` at the coin level (`export_pdf.go:49,286`) and never branch on
  reference presence.
- **One-time `invoice_number` backfill (`database.go:345-385`): no effect.** It is
  version-gated by an `AppSetting` and joins `coin_references` without a wishlist
  predicate; it is historical and already indifferent.

### 3. Spec 351 text that must be amended

- `specs/351-vision-first-deep-identification/spec.md:843-852` -- the paragraph
  describing the nilling as "the pre-existing and intended invariant". The
  surrounding argument (that `ReferenceText` is a scalar distinct from the
  `References` relation, so `coin_type` survives) remains **correct and load-bearing**
  and MUST NOT be deleted; only the invariant claim is amended.
- `specs/351-vision-first-deep-identification/tasks.md:277` -- same claim, same
  treatment.

Per ADR README's status lifecycle, 351 is landed; this ADR is the amendment
record. The 351 documents get an inline pointer to ADR 0013, not a rewrite.

### 4. What could break

- The four test sites listed in Context will fail and must be deliberately
  rewritten, not incidentally patched.
- Any frontend surface that renders a wishlist coin will now receive a non-empty
  `references` array where it previously always received an empty one. The
  serialisation shape does not change, so this is a data-volume change rather than
  a contract change -- but any component that assumed emptiness (for example, by
  hiding a references panel on wishlist coins) will now show content.
- Import/export round-trips of wishlist coins will now carry references.

### 5. Risk I am NOT comfortable with -- read this

**Removing the guard silently changes an unrelated, untrusted path.**

`WishlistSearchAlertService.ConvertCandidate` takes
`ConvertCandidateInput.Coin` as a **whole `models.Coin` struct straight off the
request body** (`wishlist_search_alert_service.go:140-143`), including its
`References` slice. That path is wired to the reference-enabled `CoinService`
(`main.go:304, 306`). The references it carries originate from AI search-agent
output -- catalog claims an LLM produced from scraped dealer pages.

Today those references are silently dropped by the wishlist guard. The moment the
guard is removed, **they are persisted**, with no owner confirmation step and no
provenance record. That is a materially different trust decision from the one
Brian actually made. He decided that *deep identification*, a confirm-gated
pipeline with per-field confidence, may write wishlist references. He did not
decide that *the search agent* may write them unconfirmed.

This is the cost of Decision A that was not visible when the question was put to
him. It is why V-2 is written as a requirement rather than a suggestion. If V-2
is not implemented in the same change as the guard removal, this ADR should not
be merged.

Two smaller residual concerns, stated for the record and accepted:

- **Duplicate references across the purchase boundary.** A wishlist coin with a
  RIC reference that is purchased and then also enriched by deep identification
  could accumulate two rows for the same type. The `(coin_id, catalog, volume,
  number)` dedupe that Feature 352 FR-039 mandates for draft promotion should be
  applied by the append path generally, not only at promotion.
- **Loss of a coarse "unowned data is thin" signal.** Nothing in the codebase
  reads it, so no consumer breaks -- but any future analytics that wants to
  distinguish researched from unresearched wishlist entries can no longer infer it
  from reference presence and must use an explicit field.

## Related

- Supersedes the wishlist-reference invariant asserted in
  `specs/351-vision-first-deep-identification/spec.md:843-852`.
- Enables `specs/352-deep-identification-structured-results/` Phase 6 and US-6;
  resolves that spec's Conflict C-1.
- [ADR 0012](0012-vision-first-deep-identification.md) -- the pipeline whose
  output this reference write carries.
- [ADR 0011](0011-deep-agentic-coin-identification.md) -- persistence and
  write-boundary decisions for deep identification.
- [ADR 0007](0007-shared-numista-lookup.md) -- the draft selected-reference
  boundary that `PromoteDraftTransaction` implements.
- Constitution SS22 (Amendment Process), Principle V (Security, Auth, and Privacy
  by Default -- the V-2 requirement), Principle IV (simplest complete
  proportional change).
