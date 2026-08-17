# Feature 352 — Implementation Plan

**Spec**: `specs/352-deep-identification-structured-results/spec.md`
**Author**: Maximus (Lead / Architect)
**Status**: Plan only. No implementation authored.

---

## 0. Plan-level constraints

Applies to every phase below.

| Constraint | Source |
|---|---|
| `src/api/services/ocre_scoring.go` must not change | ADR 0010 (binding) |
| `src/agent/.../providers/rpc.py` must not change | Spec FR-021 |
| No hypothesis values, query terms, candidate details, catalog/cert numbers, narrative text, or image data in application logs | 351 FR-040 (binding) |
| `image` is only ever an `EvidenceRef.provider` | 351 FR-025 |
| handlers -> services -> repository -> models; `models/` stdlib-only | Constitution Principle I |
| Catalog codes validate against `models.CatalogRegistry`; none invented | Spec FR-045 |
| Simplest complete proportional change | Constitution Principle IV |

**Sequencing rule (revised 2026-08-17 after Brian's Decisions A/B/C)**: Phase 6
is **no longer gated**.
[ADR 0013](../../docs/adr/0013-wishlist-coins-may-hold-catalog-references.md)
records the amendment to Feature 351 required by constitution SS22, and must reach
status `Accepted` before Phase 6 code merges — but that is a documentation step
now under way, not an open question.

**Re-checked phase ordering.** Removing the wishlist invariant does change what
must land first. Phase 6 splits in two, and its first half moves to the front:

- **Phase 6a (governance + guard swap)** — ADR 0013, delete the two guards
  (FR-048), add the `ConvertCandidate` untrusted-source guard (FR-049), rewrite
  the four tests (FR-052). This half depends on **nothing**. It touches
  `coin_service.go` and `wishlist_search_alert_service.go` only, and it is the
  change with the widest blast radius in the feature, so it should land **first
  and alone**, on its own commit, where a regression is unambiguous. FR-048 and
  FR-049 MUST be in the same commit.
- **Phase 6b (apply path)** — `applyToWishlist` calls `AppendForCoin`. Still
  depends on Phases 2 and 3, unchanged.

**One requirement moves phases**: FR-051 (case-insensitive
`(Catalog, Volume, Number)` dedupe on every append, not just at promotion) belongs
in **Phase 2**, not Phase 7. With wishlist references allowed, the
enrich -> `PurchaseCoin` -> enrich sequence can duplicate a reference on the coin
path, so the dedupe is no longer a promotion-only concern.

**Branch/commit convention**: Conventional Commits, `Co-authored-by: Copilot`
trailer, spec section IDs quoted in commit messages (Constitution SS17,
Principle VII).

---

## Phase 1 — Extract the shared catalog-reference parser

**Goal**: one parser, callable from two places, with a confidence score.
**Depends on**: nothing. Start here.

### Files touched

| File | Layer | Change |
|---|---|---|
| `src/api/services/catalog_reference_parser.go` | service | **New.** Exported `ParseCatalogReferenceText(text string, registry map[string]*models.CatalogRegistry) (ParsedCatalogReference, bool)` plus `ParsedCatalogReference{Catalog, Volume, Number, Confidence, NeedsVolume, RawText}`. Contains the moved `normalizeCatalogAlias`, `isRomanNumeral`, `isPlausibleVolumeToken` logic verbatim. |
| `src/api/services/reference_migration_service.go` | service | Delegate `parseLegacyReference` to the new helper; keep the `Volume: "0"` sentinel + journal-message behaviour **inside the migration service**, not in the shared helper. |
| `src/api/services/catalog_reference_parser_test.go` | test | **New.** Confidence-table coverage per FR-017, plus explicit "never emits volume 0" assertion (FR-019). |
| `src/api/services/reference_migration_service_test.go` | test | **Must not change.** It is the regression oracle. |

### Design note

The `Volume: "0"` sentinel and the "manual review needed" journal string are
**migration-specific policy**, not parsing. They stay in
`ReferenceMigrationService`. The shared helper returns
`NeedsVolume: true, Volume: ""` and lets each caller decide: the migration
service substitutes `"0"` and journals; feature 352 proposes at 0.30 unaccepted.
This is what keeps FR-016 (behaviour-preserving) and FR-019 (no sentinel)
simultaneously satisfiable.

### Risk

**Medium.** A behaviour-preserving extraction of live migration code. The
specific hazard is the four-way branching in `parseLegacyReference` around
volume-required catalogs, each of which returns a *different* origin-text string
for the journal message (`text` vs `first` depending on whether a `;` was
present). Getting that wrong silently changes migration journal output for a
subset of users. **Mitigation**: run `reference_migration_service_test.go` before
and after with zero edits; if any assertion needs editing, the extraction is
wrong.

**Do not** add `NGC` to `normalizeCatalogAlias` (FR-007, F-6) — it would newly
migrate legacy `rarity_rating` text starting with `NGC`.

---

## Phase 2 — Additive append path for coin references

**Goal**: a service method that adds references to a coin without deleting any.
**Depends on**: nothing. Can run parallel to Phase 1.

### Files touched

| File | Layer | Change |
|---|---|---|
| `src/api/services/coin_reference_service.go` | service | **New method** `AppendForCoin(coinID, userID uint, refs []models.CoinReference) ([]models.CoinReference, error)`. Loads existing via `repo.ListByCoin`, filters proposed refs whose `dedupeKey` already exists, validates survivors via `NormalizeAndValidateOne`, inserts via `repo.CreateBatch`. Reuses the existing `dedupeKey` helper. **FR-051**: the key MUST be `(Catalog, Volume, Number)` compared **case-insensitively**, on every target — confirm `dedupeKey` already folds case; if it does not, that is a Phase 2 fix, not a Phase 7 one. |
| `src/api/repository/coin_reference_repository.go` | repository | Possibly a `CreateBatchInTx` convenience; otherwise unchanged. `ReplaceForCoin` untouched. |
| `src/api/services/coin_reference_service_test.go` | test | New: append-preserves-existing, append-is-idempotent, unknown-catalog-rejected, volume-required-rejected. |

### Design note

`AppendForCoin` is deliberately a **sibling** of `ReplaceForCoin`, not a mode
flag on it. `ReplaceForCoin` is the manual-editor semantic (the owner said "these
are my references"); `AppendForCoin` is the agent semantic ("here is one more
thing I found"). Conflating them behind a boolean is exactly how F-4 would
recur.

### Risk

**Low-medium.** The risk is a future caller reaching for `ReplaceForCoin` out of
habit on an agent path. **Mitigation**: doc comment on `ReplaceForCoin` stating
it is owner-editor-only and pointing at `AppendForCoin`; consider an
architecture-test-style assertion that `deep_identification_*.go` does not
reference `ReplaceForCoin`.

---

## Phase 3 — The collection-valued write surface

**Goal**: extend the allowlist concept, deliberately, to a collection-valued
field. This is the architectural heart of the feature.
**Depends on**: Phases 1 and 2.

### Files touched

| File | Layer | Change |
|---|---|---|
| `src/api/services/deep_identification_proposal.go` | service | **New** `deepProposalCollectionFieldAllowlist` — a separate, closed map with exactly one entry, `catalogReferences`. New `deepProposalCatalogReference` struct mirroring FR-004 with `DisallowUnknownFields` decoding. New `resolveDeepProposalCatalogReferences(entry)` decoding `Proposed`/`OwnerValue` into `[]models.CoinReference` + per-element metadata. `Apply` gains a second pass: scalar fields via the existing path, collection fields via `CoinReferenceService.AppendForCoin`. `UpdateProposal` gains per-element validation of edited `catalogReferences`. |
| `src/api/services/deep_identification_proposal.go` | service | `selectDeepAppliedFieldNames` / allowlist lookups must consult **both** maps so an unknown key is still rejected (FR-003). |
| `src/api/handlers/deep_identification.go` | handler | Doc/Swagger annotations only. No new endpoint. |
| `specs/351-.../contracts/deep-identification.openapi.yaml` equivalent for 352 | contract | Document the `catalogReferences` field entry shape. |
| `src/api/services/deep_identification_proposal_test.go` | test | AC-001..AC-003, AC-008..AC-010. |
| `src/api/services/deep_identification_contract_drift_test.go` | test | Extend drift coverage to the new key. |

### Design note — why one key holding an array, not one key per reference

Per-reference keys (`catalogReference.NGC`, `catalogReference.RIC`, ...) would
require a **dynamic** allowlist, which destroys the property that makes the
allowlist a guard: that it is a closed, statically readable set. Instead:

- One static key, `catalogReferences`, holding an array.
- Field-level `accepted` gates the whole set.
- Per-element opt-out is expressed as an **owner edit**: the editor writes a
  filtered array into `ownerValue` and sets `ownerEdited = true`. This reuses the
  existing 351 owner-edit machinery exactly as designed — `Proposed` is already
  typed `any`.

The write surface therefore widens by exactly one statically declarable key, and
the guard property is preserved.

### Risk

**High.** This is the first collection-valued write in the feature and the first
time `Proposed: any` carries a non-scalar. Specific hazards:

1. **Type coercion.** `setCoinFieldFromProposalValue` currently coerces
   `float64`/`string`. A JSON array decodes to `[]any` of `map[string]any`. The
   decoder must be strict (`DisallowUnknownFields`), not `fmt.Sprintf`-tolerant —
   FR-004 requires rejecting unknown properties, and the existing
   `deepProposalValueToString` would happily stringify a whole array into a
   scalar field if the two paths ever cross.
2. **Path crossing.** A bug that lets `catalogReferences` reach
   `setCoinFieldFromProposalValue` would write a stringified array into
   `Coin.ReferenceText`. **Mitigation**: the two allowlists must be consulted in
   a single `switch` with an explicit `default:` rejection, and a test must assert
   that `catalogReferences` is absent from both scalar allowlists.
3. **Transactionality.** Scalar fields go through `UpdateCoinWithFields` (its own
   transaction); references go through `AppendForCoin` (another). A failure
   between them leaves a partial apply while `ApplyJob` may already have marked
   the job applied. **Mitigation**: perform both writes **before** calling
   `repo.ApplyJob`, and on reference-write failure return an error without
   marking applied. Accept that the scalar write may have landed — document it,
   and note that a re-apply is blocked by `AppliedAt` only if `ApplyJob` ran.
   This ordering must be explicit in review.

---

## Phase 4 — Pipeline: emit NGC, RIC, and opportunistic RPC elements

**Goal**: populate `catalogReferences` from evidence that already exists.
**Depends on**: Phases 1 and 3.

### Files touched

| File | Layer | Change |
|---|---|---|
| `src/api/services/deep_identification_pipeline_runner.go` | service | `buildDeepProposalDocumentJSON`: new `buildDeepCatalogReferenceField(...)` producing the `catalogReferences` entry from (a) `quick_evidence.ngc.cert_number` -> NGC element, constructed directly (FR-006/FR-007); (b) each `coin_type` claim -> parsed element via Phase 1 (FR-010); (c) RPC-shaped text in `hypothesis.coin_type` / `label_text` / `coin_type` claims -> parsed RPC element (FR-020). Applied to **both** the intake and saved-coin branches. |
| `src/api/services/deep_identification_pipeline_runner.go` | service | Set the scalar `coin_type` entry's default `accepted` to `false` when a structured element for the same value was emitted (FR-011.2). |
| `src/api/services/deep_identification_service.go` | service | Whatever plumbing is needed to make the catalog registry available to the runner (constructor injection of `CatalogRegistryRepository`; no globals). |
| `src/api/services/deep_identification_pipeline_runner_test.go` | test | AC-004..AC-007, AC-011, AC-012. |
| `src/agent/...` | agent | **No change expected.** All parsing and registry validation is Go-side (FR-047). Confirm `quick_evidence.ngc` and `hypothesis.coin_type` already reach Go on the report; if `coin_type` claims are not currently forwarded with enough fidelity, that is a `frame_translator` change, not an agent change. |

### Design note

The registry must be loaded **once per job**, not once per claim. Load it in the
runner and pass the map down, mirroring `ReferenceMigrationService.MigrateLegacyReferences`.

### Risk

**Medium.** Hazards:

1. **Log leakage.** This code handles cert numbers and catalog numbers directly.
   FR-043 is absolute. Any `log.Printf` added here — including a well-meaning
   "parsed RIC II Hadrian 39b" debug line — violates it. **Mitigation**: the 351
   log-assertion suite must be extended to cover the new code path (AC-031).
2. **`coin_type` multiplicity.** OCRE emits *multiple* `coin_type` claims to
   preserve ambiguity (351 FR-013). Emitting one structured element per claim
   would flood the array. **Decision to make in implementation**: emit an element
   only for the **top-ranked** `coin_type` claim; surviving lower-ranked claims
   stay visible as evidence on the scalar `coin_type` entry. This is a real
   design decision, not an oversight — record it.
3. **RPC false positives.** Matching `RPC` in free label text could pick up
   unrelated tokens. **Mitigation**: require the token to be leading and
   word-boundaried, and reuse the shared parser rather than a bespoke regex.

---

## Phase 5 — Narrative into notes, append-only, one format everywhere

**Goal**: the narrative reaches coin notes without ever destroying owner text,
in a **single format shared by the coin, draft, and wishlist paths**
(Brian's Decision C, spec C-2).
**Depends on**: Phase 3 (shares the apply-path restructuring).

### Files touched

| File | Layer | Change |
|---|---|---|
| `src/api/services/deep_identification_proposal.go` | service | `deepProposalCoinFieldAllowlist` entries become a small struct carrying `GoField` + `Mode` (`set` \| `append`). `notes` -> `{Notes, append}`. New `composeDeepAnalysisNotes(existing, block, jobID, appliedAt)` implementing FR-024 heading format and FR-025 in-place replacement, plus FR-026 truncation policy. `applyToCoin` reads `existing.Notes` and composes before assigning. `applyToWishlist` / `applyToDraft` compose against empty (FR-029). |
| `src/api/services/deep_identification_pipeline_runner.go` | service | Saved-coin branch emits `notes` using the same builder the intake branch uses (FR-027). Intake branch's block is **rewritten** to the dated heading (FR-028, Decision C). After this phase there must be exactly **one** notes composer in the codebase — verify by grep, not by memory. |
| `src/api/services/deep_identification_proposal_test.go` | test | AC-014..AC-018. |
| `src/api/services/deep_identification_pipeline_runner_test.go` | test | **Existing 351 assertions on the intake notes block MUST be updated** (FR-028). AC-037 asserts the intake block is shape-identical to the coin block. |

### Design note — the idempotency rule, precisely

```
heading := fmt.Sprintf("## Deep Analysis - %s (job %d)", appliedAt.UTC().Format("2006-01-02"), jobID)
```

1. Scan existing notes line-by-line for a line matching
   `^## Deep Analysis - \d{4}-\d{2}-\d{2} \(job <jobID>\)$` — the **job id**, not
   the date, is the identity. A block written yesterday for job 412 is still
   job 412's block.
2. If found: the block runs from that heading to the next line beginning `## ` or
   to end of text. Replace it, heading included (so the date refreshes).
3. If not found: append `"\n\n" + heading + "\n\n" + block`, omitting the leading
   `"\n\n"` when existing notes are empty.
4. Re-running deep analysis creates a **new job id**, so step 3 fires and the
   previous analysis's block survives as dated history. That is the intended
   behaviour, not accidental accumulation.

### Risk

**Medium-high.** This writes to a column that holds irreplaceable, hand-written
user data. Hazards:

1. **The mode flag changes a shipped map's type.** Every read of
   `deepProposalCoinFieldAllowlist` must be updated. There are several
   (`applyToCoin`, `applyToWishlist`, `buildDeepProposalDocumentJSON`,
   `buildDeepIntakeProposalFields`). Missing one is a compile error, which is the
   good case; a `map[string]string` retained alongside for compatibility would be
   the bad case. Do not keep two maps.
2. **Truncation direction.** FR-026 requires truncating the *appended* block, not
   the owner's text. The natural implementation (`truncateDeepProposalText` on
   the composed string) does exactly the wrong thing. This must have a dedicated
   test (AC-017).
3. **Heading collision with owner text.** An owner who happens to have typed
   `## Deep Analysis - ...` themselves could have text replaced. The job-id
   requirement in the regex makes this vanishingly unlikely; note it and move on.
4. **Decision C makes this phase a change to shipped, user-tested output.** The
   intake notes block is not new code — it landed in 351 and Brian ran it against
   real coins on 2026-08-16. Rewriting its format is a deliberate regression
   surface, and the mitigation is *not* "be careful": it is that AC-037 asserts
   the intake and coin blocks are shape-identical, and that the 351 assertions are
   updated in a **separate, clearly-labelled commit** from the rest of Phase 5 so
   the diff that changed shipped output is easy to find later. Drafts created
   before this change keep their old-format notes; nothing migrates them, so both
   formats will coexist in the draft list until old drafts age out. That is
   accepted (spec C-2), not overlooked.

---

## Phase 6 — Wishlist structured references *(ungated; ADR 0013 accepted)*

**Goal**: the `wishlist` apply target writes structured references like `coin`.
**Split**: 6a depends on **nothing** and lands first, alone.
6b depends on Phases 2 and 3.

### Phase 6a — governance and guard swap *(lands first, own commit)*

| File | Layer | Change |
|---|---|---|
| `docs/adr/0013-wishlist-coins-may-hold-catalog-references.md` | docs | **Prerequisite, written.** Records the relaxation, the absence of any recorded domain rationale for the original rule, the two shipped bypasses (spec F-3), and amends 351 per Constitution SS22. Must reach status `Accepted` before 6a merges. |
| `specs/351-vision-first-deep-identification/spec.md:843-852` | spec | Inline pointer to ADR 0013. **Amend the invariant claim only** — the `ReferenceText`-is-a-scalar argument around it is correct and load-bearing. Do not rewrite the landed spec. |
| `specs/351-vision-first-deep-identification/tasks.md:277` | spec | Same inline pointer, same restraint. |
| `src/api/services/coin_service.go` | service | Delete the two guards at `:154-156` and `:173-177` (FR-048). **Keep `:171-172`** (`pendingReferences :=` detach) — it is cascade defence, not wishlist logic. |
| `src/api/services/wishlist_search_alert_service.go` | service | `ConvertCandidate` clears `input.Coin.References` at its own boundary (FR-049). **Same commit as the guard deletion, no exceptions.** |
| `src/api/services/coin_service_test.go` | test | Rewrite `TestCreateCoin_DropsReferencesForWishlist` (:438) and `TestCreateCoin_WishlistInvariant_AgentStylePayload` (:490) to assert the new rule; cite ADR 0013 in the doc comments (FR-052). AC-033, AC-035. |
| `src/api/handlers/coin_handler_test.go` | test | Rewrite `TestCoinHandler_Create_WishlistWithReferencesStoresZeroReferences` (:2075) (FR-052). |
| `src/api/services/wishlist_search_alert_service_test.go` | test | The `:615` assertion keeps expecting zero references, but its comment changes from "wishlist coins discard references" to the FR-049 trust reason. AC-034. |

### Phase 6b — apply path

| File | Layer | Change |
|---|---|---|
| `src/api/services/deep_identification_proposal.go` | service | `applyToWishlist` calls `AppendForCoin` after `CreateCoin`. |
| `src/api/services/deep_identification_proposal_test.go` | test | Wishlist apply persists structured references; US-6. |

### Risk

**6a is the highest-blast-radius change in the feature.** The code diff is four
lines; the behaviour change reaches every wishlist create path in the
application. Specific hazards:

1. **FR-049 is load-bearing, not hygiene.** `ConvertCandidateInput.Coin` is a raw
   `models.Coin` off the request body (`wishlist_search_alert_service.go:140-143`)
   carrying LLM-sourced catalog claims, wired to the reference-enabled
   `CoinService` (`main.go:304,306`). Removing the guards without FR-049 begins
   silently persisting unconfirmed AI output as structured catalogue data. If a
   reviewer sees a commit deleting the guards without the `ConvertCandidate`
   change, that is a `BLOCK`.
2. **Do not delete `coin_service.go:171-172` along with the guard.** The detach
   and the wishlist nil sit three lines apart and read as one block. Deleting both
   reintroduces the 2026-07-21 `UNIQUE constraint failed: coin_references.id`
   crash. `coin_reference_regression_test.go` is the oracle and must stay green.
3. **`CountReferencesUsing` semantics shift.**
   `catalog_registry_repository.go:71-75` will now count wishlist references, so a
   catalog used only by wishlist coins becomes undeletable. Correct, but a visible
   admin-surface change worth a line in the PR description.
4. **Frontend assumption of emptiness.** Any component that hid a references panel
   on wishlist coins because the array was always empty will now render content.
   Phase 9 must check, not assume.

---

## Phase 7 — Draft one-to-many catalog references

**Goal**: drafts hold NGC + RIC + RPC, without touching the shipped table.
**Depends on**: Phase 1 (for parsing) and Phase 3 (for the proposal shape).
Independent of Phases 5 and 6.

### 7a — Model and migration

| File | Layer | Change |
|---|---|---|
| `src/api/models/quick_capture_draft.go` | models | **New** `QuickCaptureDraftCatalogReference` (spec SS8). New `CatalogReferences []QuickCaptureDraftCatalogReference` on `QuickCaptureDraft`. `QuickCaptureDraftReference` and `SelectedNumistaReference` **untouched** (FR-036). New `DraftLifecycleEventDeepAnalysisApplied` constant + `IsValidDraftLifecycleEventType` case. Stdlib-only, as required. |
| `src/api/database/database.go` | database | Register the new model in `AutoMigrate`. Add an idempotent backfill (FR-037) alongside the existing `seedCatalogRegistry` call site. |
| `src/api/database/migration_test.go` | test | **Existing assertions must not change** (AC-023). New assertions for the new table and the idempotent backfill (AC-026). |

### 7b — Repository and service

| File | Layer | Change |
|---|---|---|
| `src/api/repository/quick_capture_repository.go` | repository | Preload `CatalogReferences` everywhere `SelectedNumistaReference` is preloaded (lines 90, 124, 250, 280). Promotion (lines 291-300) copies **both** sources into `models.CoinReference`, deduped by `dedupeKey` (FR-039). |
| `src/api/services/quick_capture_service.go` | service | Accept catalog references on create/update input; set `UserID` from the draft owner, never from request input. |
| `src/api/handlers/quick_capture.go` | handler | Bind and validate the new payload key; existing `selectedNumistaReference` binding unchanged. |
| `src/api/services/deep_identification_proposal.go` | service | `applyToDraft` writes `catalogReferences` into the new relation. |

### 7c — Contract and frontend

| File | Layer | Change |
|---|---|---|
| `src/api/docs/*` | generated | Regenerate. `models.QuickCaptureDraftReference` definition and the `x-nullable` `selectedNumistaReference` ref must be byte-identical (AC-029). |
| `src/web/src/types/index.ts` | frontend | Add `catalogReferences?: DraftCatalogReference[]` to the draft type. `selectedNumistaReference` unchanged. |
| `src/web/src/pages/QuickCaptureDraftPage.vue`, `components/quick-capture/QuickCaptureDraftCard.vue`, `PromotionReadinessPanel.vue` | frontend | Render the new list. Existing Numista rendering untouched. Use `.chip-sm` for reference pills, `--radius-full`, `--accent-gold` for values, per the design system. Optional chaining / `??` on all array index access (Docker TS strictness). |
| `src/web/src/test/numista-fixtures.ts` | frontend test | Extend the factory additively; do not change `makeSelectedNumistaReference`'s shape. |

### Risk

**High — the riskiest phase, which is why FR-035 makes it additive.**

1. **Blast radius.** 34 files, 149 sites (spec Section 5). The additive design
   reduces this from "every one is a candidate breakage" to "every one must keep
   compiling and passing unchanged". The regression oracle is:
   `migration_test.go`, `quick_capture_draft_test.go`,
   `openapi_nullability_test.go`, `numista_compatibility_test.go` — **all four
   must pass with zero assertion edits.**
2. **Name collision.** `models.SelectedNumistaReference` (a value type in
   `numista.go:164`) and `QuickCaptureDraft.SelectedNumistaReference` (a relation
   field) share a name. Any rename attempt hits this. Another reason not to
   rename.
3. **SQLite.** The additive design performs **no** `ALTER TABLE` on
   `quick_capture_draft_references` — no index drop, no `NOT NULL` relaxation, no
   rebuild. GORM `AutoMigrate` only adds a new table. **Rollback**: revert the
   code; the new table is inert and may be dropped separately (FR-040).
4. **Double-copy on promotion.** The backfill (FR-037) means a legacy Numista
   selection exists in *both* tables. Promotion must dedupe or it writes two
   Numista rows and trips `idx_coin_ref_unique`. AC-027 is the specific guard.
5. **Nullable URI.** The new table allows an empty `URI`; the legacy one does
   not. Promotion copies into `models.CoinReference.URI`, which is already
   nullable — no problem there. But the frontend must not assume a link exists.

---

## Phase 8 — Provenance: the remaining draft gap

**Goal**: record deep-analysis provenance on the draft path. The `coin` and
`wishlist` paths were fixed during this session by Cassius (spec F-1) — do not
re-implement them.
**Depends on**: Phase 7 (lifecycle-event constant and the promotion rework).

| File | Layer | Change |
|---|---|---|
| `src/api/services/deep_identification_proposal.go` | service | `applyToDraft` writes a `DraftLifecycleEvent` of the new `deep_analysis_applied` type (FR-032). Extend `deepProposalJournalEntryText`'s input so `catalogReferences` appears in the coin/wishlist entries when applied — **field names only**, never values (FR-030, FR-043). |
| `src/api/services/quick_capture_service.go` + `src/api/main.go` | service / DI | New journal write dependency for FR-033. Constructor injection, no globals. |
| `src/api/repository/quick_capture_repository.go` | repository | On promotion, if a `deep_analysis_applied` lifecycle event exists for the draft, write a `CoinJournal` entry on the promoted coin. Condition on the **event's presence**, not on `draft.Source` (FR-033). |
| `src/api/services/deep_identification_proposal_test.go` | test | AC-019..AC-022. Cassius's existing journal assertions in `TestDeepIdentificationProposal_ApplyRoutesThroughCoinServiceOnly` and `TestDeepIdentificationProposal_WishlistApplyCreatesWishlistCoin` must keep passing. |

### Risk

**Medium.** Two hazards:

1. **Log/journal leakage.** Journal entry text must list **field names**, never
   values (FR-043). "Deep Analysis applied: denomination, mint, catalogReferences
   updated" is fine; "set mint to Antioch" is not. Cassius's
   `deepProposalJournalEntryText` already has the right shape — preserve it.
2. **DI widening.** FR-033 adds a dependency to `QuickCaptureService`, which
   Cassius correctly declined to do in isolation. It is proportional only because
   Phase 7 is already reworking `PromoteDraft`. If Phase 7 is descoped, Phase 8's
   FR-033 half must be descoped with it — FR-032 alone still stands.

---

## Phase 9 — Frontend proposal editor

**Goal**: the owner can see, edit, and per-element opt out of catalog references,
and can see the notes block before applying.
**Depends on**: Phases 3, 4, 5.

| File | Layer | Change |
|---|---|---|
| `src/web/src/components/deep-identification/DeepProposalEditor.vue` | frontend | New rendering branch for the `catalogReferences` array field: one row per element showing catalog / volume / number / source / confidence, an include checkbox writing a filtered `ownerValue`, editable volume for `needsVolume` elements, and a blocking validation state that prevents accepting while a required volume is empty. |
| `src/web/src/utils/deepProposalAcceptance.ts` | frontend | Extend default-acceptance logic to the array field; the 0.70 threshold constant is unchanged (351 RD-3, NG-7). |
| `src/web/src/types/index.ts` | frontend | Types for the new field entry shape. |
| `src/web/src/components/deep-identification/__tests__/DeepProposalEditor.test.ts` | frontend test | Per-element opt-out, volume-required blocking, notes-block preview. |

### Design system compliance

Reference rows use `.chip-sm` for the catalog pill, `--accent-gold` for the
number, `--text-muted` at `0.7rem` / `letter-spacing: 0.08em` for the
"SOURCE" / "CONFIDENCE" labels, `var(--radius-sm)` for the row card,
`1.5rem` between the references group and adjacent sections, `0.35rem` between
chips. No emojis. No hardcoded radii or colours.

### Risk

**Medium.** Hazards:

1. **Docker TS strictness.** Array index access on `catalogReferences` needs `?.`
   and `?? ''` / `?? 0`. Local `vue-tsc --noEmit` will pass where
   `vue-tsc --build` fails.
2. **Owner-edit round-trip.** Writing a filtered array into `ownerValue` and
   setting `ownerEdited = true` must round-trip through `PATCH .../proposal`
   without the backend stringifying it. This is the frontend half of Phase 3's
   type-coercion risk and should be tested end-to-end, not just in isolation.

---

## Phase 10 — Documentation, contracts, and gate

| File | Layer | Change |
|---|---|---|
| `docs/ARCHITECTURE.md` | docs | Note the collection-valued proposal write surface — it is a new architectural pattern. |
| `.github/copilot-instructions.md` | docs | Add the `catalogReferences` write surface and the notes-append rule to Notable Endpoints & Features. |
| `specs/352-.../contracts/` | contract | Proposal document schema delta. |
| `src/api/docs/*` | generated | Regenerate; assert no drift. |
| `.squad/decisions/inbox/` | governance | The C-1 ADR outcome. |

### Quality gate (Constitution SS17)

```
cd src/api && go build ./... && go vet ./... && go test ./...
cd src/web && npm run build
cd src/agent && ruff check app/ tests/ && pytest tests/ -v
```

Plus the 352-specific DoD additions in spec Section 10: four regression-oracle
test files passing with **zero assertion edits**, and `git diff` showing zero
changed lines in `ocre_scoring.go` and `providers/rpc.py`.

---

## Phase dependency graph

```
Phase 6a (ADR 0013 + guard swap) ── independent, LANDS FIRST, own commit

Phase 1 (parser) ─┬─> Phase 4 (pipeline emit) ─┬─> Phase 9 (editor)
                  │                            │
Phase 2 (append   ─┴─> Phase 3 (write surface) ─┤
 + FR-051 dedupe)                              ├─> Phase 5 (notes, unified) ──> Phase 9
                                               │
                                               ├─> Phase 6b (wishlist apply)
                                               │
                                               └─> Phase 7 (draft 1:N) ──> Phase 8 (provenance)

Phase 10 (docs/gate) last.
```

Phases 1 and 2 are parallelisable. **Phase 6a is now the first thing to land**:
it depends on nothing, it is the widest behaviour change in the feature, and
isolating it makes any regression unambiguous. Phase 6b is a two-line apply-path
change once Phases 2 and 3 exist. Phase 8's FR-032 (draft lifecycle event) is
independent; its FR-033 (promotion-time journal) is bound to Phase 7 and descopes
with it. The `coin` and `wishlist` journal writes are **already landed** and are
not a phase.

### Concurrency warning

Cassius has an in-flight change in `src/api/services/deep_identification_proposal.go`
(journal entries, see `.squad/decisions/inbox/cassius-deep-journal-wishlist.md`).
Phases 3, 5, and 8 all edit that same file. Rebase on his change before starting,
and do not revert `deepProposalJournalEntryText` or the corrected `Apply()` doc
comment.

---

## Risk register, ranked

| # | Risk | Phase | Severity | Mitigation |
|---|---|---|---|---|
| R1 | **Guard removal begins persisting unconfirmed AI search-agent catalog claims** via `ConvertCandidate`'s raw `models.Coin` body binding — a trust decision Brian did not make when he decided A | 6a | **Critical** | FR-049 in the **same commit** as FR-048. AC-034. A guard-removal commit without it is a reviewer `BLOCK`. See ADR 0013 Consequences item 5. |
| R2 | Structured reference write reaches `ReplaceForCoin` and deletes owner references | 2, 3 | **Critical** | `AppendForCoin` as a separate method; doc comment on `ReplaceForCoin`; architecture-style test. **Permanent constraint (spec F-4), not a decision item — this does not go away when the feature ships.** |
| R3 | Notes append truncates or overwrites hand-written owner text | 5 | **Critical** | FR-024/025/026; dedicated tests AC-014..AC-018; truncate the block, never the owner text. |
| R1b | Deleting `coin_service.go:171-172` (cascade detach) along with the adjacent wishlist guard reintroduces the 2026-07-21 UNIQUE-constraint crash | 6a | **High** | Explicit note in Phase 6a; `coin_reference_regression_test.go` green is the oracle. |
| R4 | Draft migration breaks one of 34 consumers | 7 | High | Purely additive; four regression-oracle test files must pass with zero assertion edits. |
| R5 | Array `Proposed` value stringified into a scalar column | 3 | High | Strict decoding; single `switch` over both allowlists with explicit `default:`; test asserting key absence from scalar maps. |
| R6 | Cert/catalog/narrative values leak into application logs | 4, 8 | High | FR-043 absolute; extend 351's log-assertion suite (AC-031). |
| R7 | Parser extraction silently changes legacy migration journal output | 1 | Medium | `reference_migration_service_test.go` unchanged is the oracle. |
| R8 | Partial apply between scalar write and reference write | 3 | Medium | Both writes before `ApplyJob`; documented ordering; explicit review item. |
| R9 | **Decision C rewrites 351's shipped, user-tested intake notes format**; pre-existing drafts keep the old format, so both coexist | 5 | Medium | Accepted by Brian. AC-037 asserts shape parity; 351 assertion updates go in a separate labelled commit. No data migration — old text stays readable. |
| R10 | RPC elements almost always unaccepted (C-5) | 4 | Low (UX) | Flagged for Brian; do not unilaterally flip `RPC.VolumeRequired`. |
| R11 | Multiple `coin_type` claims flood the references array | 4 | Low | Top-ranked claim only; record the decision. |
| R12 | Docker `vue-tsc --build` rejects array index access that local passes | 9 | Low | `?.` / `??` at every index access and nullable prop boundary. |
| R13 | Duplicate references from enrich -> `PurchaseCoin` -> enrich, now possible because wishlist coins carry references | 2 | Low | FR-051: case-insensitive `(Catalog, Volume, Number)` dedupe in `AppendForCoin`, not only at promotion. AC-036. |
| R14 | `CountReferencesUsing` now counts wishlist references, making a wishlist-only catalog undeletable; frontend components that hid the references panel on wishlist coins now render content | 6a, 9 | Low | Both are correct outcomes; call them out in the PR description and check the frontend rather than assuming. |

---

## What is explicitly NOT in this plan

- Any change to `ocre_scoring.go` (ADR 0010).
- Any change to `providers/rpc.py`.
- Any Python agent change beyond confirming existing fields reach Go.
- Dropping `QuickCaptureDraftReference`, its unique index, or the
  `SelectedNumistaReference` relation.
- Backfilling `Coin.ReferenceText` into `models.CoinReference`.
- Any edit to `specs/351-*/tasks.md` **beyond** the single inline ADR-0013
  pointer at line 277 required by Phase 6a. The landed spec is not rewritten.
- Any new catalog code in `CatalogRegistry`.
- Any new LLM call or provider.
