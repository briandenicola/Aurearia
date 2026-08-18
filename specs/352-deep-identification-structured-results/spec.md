# Feature 352 — Deep Identification: Structured Results

- **Feature number**: 352 (confirmed unused; highest existing is 351)
- **Status**: Draft — specification only, no implementation authored.
  Revised 2026-08-17 for Brian's Decisions A (wishlist references), B (additive
  draft table), and C (unified notes format). Those three are **settled**.
- **Author**: Maximus (Lead / Architect)
- **Requested by**: Brian (briandenicola)
- **Builds on**: Feature 344 (deep agentic identification), 345 (OCRE provider),
  351 (vision-first deep identification), 214 (structured numismatic catalog
  references), 341 (Numista lookup / draft selected reference)
- **Governing documents**: `.specify/memory/constitution.md` SS0 Hierarchy of
  Authority, Principle I, Principle III, Principle IV, Principle V, SS17 Quality
  Gate, SS21 Definition of Done, SS22 Amendment Process. ADR 0010 (binding).
  ADR 0013 (prerequisite — amends Feature 351's wishlist-reference invariant).

---

## 1. Summary

Feature 351 shipped a vision-first deep identification pipeline that produces a
confirm-gated proposal of **scalar** coin fields. The evidence the pipeline
already gathers is richer than what it can write: an NGC certification number, a
RIC catalogue type from OCRE, an occasional RPC number read off a slab label, and
a written narrative all exist on the report but land either nowhere or in a
free-text column.

Feature 352 makes deep identification write **structured, catalogue-grade
results**:

1. The NGC certification number becomes a structured `NGC` catalog reference.
2. The OCRE RIC type becomes a structured `RIC` catalog reference with a
   parsed Catalog / Volume / Number split, proposed with a confidence score.
3. An RPC number is proposed **opportunistically** when the vision model reads
   one; the RPC provider remains un-automated.
4. The narrative is proposed as an **append-only, dated block** on coin notes
   that can never destroy hand-written owner text.
5. The draft and wishlist apply paths record that a deep analysis produced the
   item, closing a provenance gap that exists today on all three paths.

Two enabling changes are in scope: quick-capture drafts gain a one-to-many
catalog reference relation (additively), and the existing legacy-reference parser
is promoted to a shared, reusable service helper.

One governance change is a prerequisite:
[ADR 0013](../../docs/adr/0013-wishlist-coins-may-hold-catalog-references.md)
amends Feature 351 so that wishlist coins may hold catalog references, giving
wishlist items the same treatment as saved coins (Brian's Decision A,
2026-08-17).

---

## 2. Findings that contradict the stated premises

These were verified against the code before writing this spec. They change what
the feature has to do and are recorded here rather than silently absorbed.

### F-1. The coin path did **not** journal either — and a concurrent agent has just fixed half of it

The task premise states that the coin target already records a journal entry with
source `"deep_identification"`. That was **not true** when the brief was written.

- `CoinService.updateCoin` (`coin_service.go:223-322`) uses the `source` argument
  **only** to gate the `CurrentValue`-change journal branch. Deep identification
  never proposes `CurrentValue`, so that branch is unreachable. The
  `"deep_identification"` string was decorative.
- The `Apply()` doc comment describing journal behaviour was aspirational, not
  implemented.

**During this session**, Cassius (backend) independently found and fixed the same
defect — see `.squad/decisions/inbox/cassius-deep-journal-wishlist.md`. As of now
`deep_identification_proposal.go` contains `deepProposalJournalEntryText` and
`CreateJournalEntry` calls in **both** `applyToCoin` (line 403) and
`applyToWishlist` (line 474), and the doc comment has been corrected.

**Consequence**: the `coin` and `wishlist` halves of Ask 5 are **already done**
and this spec must not re-specify them. What remains is the **draft** path, which
Cassius correctly reported as not implementable within his scope: a draft has no
`CoinID`, `models.CoinJournal.CoinID` is non-nullable, and the promotion path he
would have had to extend is the same path Feature 352 Phase 7 migrates. Feature
352 is the right home for it. FR-030..FR-034 are restated accordingly.

### F-2. The intake/draft path **already** populates `notes` with the narrative

The task premise states nothing populates `notes`.
`buildDeepIntakeProposalFields` (`deep_identification_pipeline_runner.go:833-886`)
already composes `narrative` + a `"Deep Analysis findings:"` list into a `notes`
proposal field for the **intake** branch. It is only the **saved-coin** branch
(`targetCoinID != nil`) that never emits `notes`, because that branch copies
straight from `report.proposed_fields` and synthesis never proposes `notes`.

**Consequence**: Ask 4 is *two* changes — (a) add a notes block to the saved-coin
branch, and (b) change the *existing, shipped* intake notes block to the dated
heading format. (b) touches 351 behaviour and 351 tests. FR-020..FR-026 cover it.
Brian decided (2026-08-17, Decision C) that both paths use the **same** format —
see Section 11, C-2. (b) is therefore **in scope**, not optional.

### F-3. Wishlist coins were forbidden from carrying catalog references — but two shipped paths already violated that invariant *(RESOLVED by ADR 0013)*

`CoinService.prepareCoinForCreate` (`coin_service.go:154-156`) and
`createPreparedCoinInTx` (`coin_service.go:173-177`) both nil `coin.References`
when `IsWishlist` is true, described in-code as "belt-and-suspenders". Feature
351's spec (`specs/351-.../spec.md:843-852`) explicitly ratified this as "the
pre-existing and intended invariant".

**Two shipped write paths bypass it:**

1. `QuickCaptureRepository.PromoteDraftTransaction`
   (`quick_capture_repository.go:291-300`) creates a `models.CoinReference` with a
   raw `tx.Create` **after** the coin is inserted, and
   `quick_capture_service.go:530` sets `coin.IsWishlist` from the promotion target.
   A draft promoted to **wishlist** already holds a `CoinReference` today.
2. *(found while writing ADR 0013)* `ReferenceMigrationService.MigrateLegacyReferences`
   (`reference_migration_service.go:47-56`) selects coins by `user_id` and
   `TRIM(rarity_rating) <> ''` with **no `is_wishlist` filter**, then calls
   `s.db.Create(ref)` directly. Any wishlist coin with legacy `rarity_rating` text
   was already given a structured reference by that migration.

Tracing the rule to its origin (`.squad/agents/cassius/history.md:50-62`) shows
it was introduced as the **fourth of four defensive layers** against a GORM
batch-insert `UNIQUE constraint failed: coin_references.id` crash — layers 1-3
already fix the root cause. No document in the repository records a
domain rationale for it. Feature 351 inherited the rule and described it as
intended without justifying it.

**Consequence — RESOLVED**: Brian decided (2026-08-17) that wishlist items MAY
hold catalog references. Recorded as
[ADR 0013](../../docs/adr/0013-wishlist-coins-may-hold-catalog-references.md),
which amends 351 per constitution SS22. Phase 6 is **ungated**. See FR-048 /
FR-049 for the enforcement that replaces the removed guards, and Section 11 C-1
for the decision record.

### F-4. `UpdateCoinWithFields` replaces references destructively

If structured references were routed through `updates.References`,
`CoinService.updateCoin` (`coin_service.go:270-282`) calls
`CoinReferenceRepository.ReplaceForCoin`, which **deletes every existing
reference for the coin** before inserting. Applying a deep-ID proposal that way
would silently destroy the owner's hand-entered catalog references.

**Consequence**: the collection-valued write MUST NOT go through
`UpdateCoinWithFields`. FR-013 mandates an additive append path.

### F-5. RPC and RIC are both `VolumeRequired: true`

`database.go:394-395` seeds `RIC` and `RPC` with `VolumeRequired: true`; `NGC`
(line 409) and `SEAR` (line 396) with `false`. `CoinReferenceService.NormalizeAndValidateOne`
rejects a volume-required catalog with an empty volume.

**Consequence**: an opportunistically-read `RPC 1234` off a slab label has no
volume and cannot be persisted as-is. It must be proposed unaccepted with a
"volume required" flag rather than written. FR-018 covers this. The
`Volume: "0"` sentinel used by `ReferenceMigrationService` is a migration-only
artifact and MUST NOT leak into this feature (FR-019).

### F-6. `normalizeCatalogAlias` does not recognise NGC

`reference_migration_service.go:264-278` returns `""` for `NGC`. The shared
parser therefore cannot produce an NGC reference. This is correct and must stay
correct — adding `NGC` to the alias map would change what legacy `rarity_rating`
text migrates, a behaviour change to a shipped migration. FR-006 constructs the
NGC reference directly instead of parsing it.

---

## 3. User scenarios

### US-1 — Slabbed coin, NGC cert reaches the references panel

Brian runs deep analysis on a slabbed coin whose NGC cert number was already read
during Quick Identify. The proposal shows a **Catalog References** group with one
entry: `NGC 6379244-002`, linked to the NGC lookup URL, marked as sourced from
the slab label. He accepts it. The coin's Catalog References panel now shows the
NGC entry alongside the RIC entry he had entered by hand months ago — neither was
lost.

### US-2 — OCRE returns a RIC type; the split is proposed, not assumed

OCRE returns `RIC II Hadrian 39b`. The proposal shows a structured entry with
Catalog `RIC`, Volume `II`, Number `Hadrian 39b`, and a confidence of 0.9, plus
the raw text it was parsed from and the OCRE citation. Brian sees the split is
right and accepts. Had the split been wrong he could have corrected any of the
three parts in the editor before accepting.

### US-3 — Ambiguous RIC type, volume missing

OCRE returns `RIC 39b` with no volume. The entry is proposed **unaccepted** with
confidence 0.3 and a visible "volume required for RIC" warning. Brian cannot
accept it until he types a volume. Nothing is written if he skips it.

### US-4 — Opportunistic RPC from a dealer listing

The vision model reads `RPC III 1520` off a dealer photo caption. An RPC entry is
proposed with the parsed volume. No RPC network call is made. If the label had
read only `RPC 1520`, the entry would be proposed unaccepted per US-3.

### US-5 — Narrative appended, hand-written notes preserved

Brian's coin has three paragraphs of his own notes. He applies the deep analysis
narrative. His three paragraphs are untouched; a new block appears beneath them:

```
## Deep Analysis — 2026-08-17 (job 412)

The obverse legend and portrait style are consistent with ...
```

He re-runs deep analysis a month later and applies again. A second block appears
under a new date and job id. Nothing is overwritten.

### US-6 — Wishlist item carries its references

Brian runs deep analysis from intake, sends the result to his wishlist, and the
resulting wishlist coin carries the NGC and RIC references — the same treatment a
collection coin gets.

### US-7 — Provenance is visible

Opening any coin created or updated by a deep analysis apply, Brian sees a
journal entry recording that deep identification wrote to it and which fields it
touched. For a draft, the draft's lifecycle timeline shows the same, and the
record survives promotion into the coin's journal.

---

## 4. Functional requirements

### 4.1 The structured catalog reference write surface

- **FR-001**: The proposal document MUST gain exactly **one** new field key,
  `catalogReferences`, whose `proposed` value is a **JSON array** of structured
  catalog reference objects. `deepProposalFieldEntry.Proposed` is typed `any` and
  already permits this; no schema-version bump of the proposal document is
  required beyond documenting the new key.

- **FR-002**: `catalogReferences` MUST NOT be added to
  `deepProposalCoinFieldAllowlist` or `deepProposalDraftFieldAllowlist`. Those two
  maps MUST remain exactly what they are today: maps from a JSON key to a
  **scalar** struct field written through `CoinService.UpdateCoinWithFields`. A
  new, separately named allowlist MUST be introduced for collection-valued
  fields, with its own resolver, its own validation, and its own write path.

- **FR-003**: The collection-valued allowlist MUST be closed and MUST contain
  exactly one entry in this feature (`catalogReferences`). A field key present in
  neither the scalar allowlists nor the collection allowlist MUST be rejected at
  both `PATCH .../proposal` time and apply time, with the existing
  `ErrDeepProposalFieldNotAllowed` error.

- **FR-004**: Each element of the `catalogReferences` array MUST carry exactly
  these properties and no others:
  `catalog` (string, required), `volume` (string, may be empty),
  `number` (string, required), `uri` (string, may be empty),
  `sourceProvider` (string, one of the providers that contributed, or `image`),
  `confidence` (number 0.0-1.0), `rawText` (string, the text the split was parsed
  from, may be empty), `needsVolume` (boolean).
  Unknown properties MUST be rejected, not ignored.

- **FR-005**: The `catalogReferences` array MUST be capped at **10** elements. A
  longer array MUST be rejected at apply time.

### 4.2 NGC certification number (Ask 1)

- **FR-006**: When `quick_evidence.ngc.cert_number` is non-empty, the pipeline
  MUST emit a `catalogReferences` element with `catalog: "NGC"`,
  `number` = the cert number verbatim, `uri` = `quick_evidence.ngc.lookup_url`
  when present (empty otherwise), `volume` = empty, `sourceProvider: "ngc"`,
  `confidence: 1.0`, `needsVolume: false`.

- **FR-007**: The NGC element MUST be constructed directly and MUST NOT be routed
  through the shared catalog-reference text parser.
  `ReferenceMigrationService.normalizeCatalogAlias` MUST NOT be modified to
  recognise `NGC` (F-6).

- **FR-008**: The NGC element's confidence is 1.0 because the cert number is
  transcribed, not inferred. Per Feature 351 RD-3 (threshold 0.70) it therefore
  defaults to **accepted**.

- **FR-009**: The existing behaviours that consume the cert number — the NGC
  provider link-out (`providers/ngc.py`) and the hypothesis observations string
  (`hypothesis.py:89-98`) — MUST be left unchanged. This feature adds a
  consumer; it removes none.

### 4.3 RIC as a structured reference (Ask 2)

- **FR-010**: When a `coin_type` claim is available, its value MUST be run
  through the shared catalog-reference parser (FR-015) and, on a successful
  parse against a registry-valid catalog, MUST be emitted as a
  `catalogReferences` element with `sourceProvider: "ocre"` (or the contributing
  provider), `rawText` = the original `coin_type` string, and a confidence
  assigned per FR-017.

- **FR-011 (back-compat decision — binding)**: The existing
  `"coin_type" -> "ReferenceText"` entry in `deepProposalCoinFieldAllowlist`
  **MUST be retained, unchanged**. The structured reference is **additive and
  superseding, never replacing**:

  1. `coin_type` continues to be proposed as a scalar mapping to
     `Coin.ReferenceText`, exactly as today.
  2. When the same `coin_type` value **also** parses into a registry-valid
     structured element, that element is added to `catalogReferences`, and the
     scalar `coin_type` entry's default `accepted` is set to **false** — it
     remains visible and the owner may still opt into it, but the structured
     entry is the default home for the value.
  3. When the parse **fails** (unrecognised catalog, missing number, or a
     volume-required catalog with no volume), the scalar `coin_type` entry keeps
     its normal confidence-driven default (351 RD-3) so the catalogue label is
     never lost.

  **Rationale**: writing both by default would put the same fact in two places
  and make the Catalog References panel and the free-text field disagree the
  moment the owner edits one. Dropping the scalar entirely would regress Feature
  345 and lose data whenever the parse fails. Superseding-by-default preserves
  both properties and leaves the choice with the owner.

- **FR-012 (back-compat for existing data — binding)**: Coins that already carry
  a free-text `Coin.ReferenceText` MUST NOT be migrated, rewritten, or cleared by
  this feature. `Coin.ReferenceText` remains a supported, populated column.
  Backfilling legacy free-text references into `models.CoinReference` is the
  existing `ReferenceMigrationService`'s job and is an explicit non-goal here
  (NG-3).

- **FR-013**: Applying `catalogReferences` to a saved coin MUST be **additive**.
  It MUST NOT use `CoinService.UpdateCoinWithFields` and MUST NOT reach
  `CoinReferenceRepository.ReplaceForCoin` (F-4). A new append-semantics service
  method MUST be used that:
  1. loads the coin's existing references,
  2. drops any proposed element whose `(Catalog, Volume, Number)` triple —
     compared case-insensitively, matching `dedupeKey` in
     `coin_reference_service.go` — already exists on the coin,
  3. validates each surviving element through
     `CoinReferenceService.NormalizeAndValidateOne`,
  4. inserts only the survivors,
  5. deletes nothing.

- **FR-014**: Applying `catalogReferences` twice for the same coin MUST be
  idempotent by construction (FR-013 step 2). The unique index
  `idx_coin_ref_unique` on `(CoinID, Catalog, Volume, Number)` is the backstop;
  the service MUST NOT rely on catching a constraint violation.

### 4.4 Shared volume parsing (Ask 2, Brian's decision)

- **FR-015**: The catalog-reference text parser currently private to
  `ReferenceMigrationService` (`parseLegacyReference`, `normalizeCatalogAlias`,
  `isRomanNumeral`, `isPlausibleVolumeToken`, `formatReference`) MUST be
  extracted into a **single shared service-layer helper** that both
  `ReferenceMigrationService` and this feature call. A second, divergent parser
  MUST NOT be written.

- **FR-016**: The extraction MUST be behaviour-preserving for
  `ReferenceMigrationService`. Every existing test in
  `reference_migration_service_test.go` MUST pass unchanged, with no edits to
  assertions.

- **FR-017**: The parser MUST return a **confidence** for the Catalog / Volume /
  Number split, assigned deterministically (no LLM):

  | Situation | Confidence |
  |---|---|
  | Catalog registry-valid, catalog is not volume-required, number present | 0.90 |
  | Catalog volume-required, volume token is a Roman numeral, number present | 0.90 |
  | Catalog volume-required, volume token is a plausible non-Roman token | 0.50 |
  | Catalog volume-required, no volume token found | 0.30, `needsVolume: true` |
  | Catalog not in registry, or no number found | no element emitted |

  The 0.70 acceptance threshold from Feature 351 RD-3 applies unchanged, so 0.90
  and 1.00 elements default to accepted and 0.50 / 0.30 elements default to
  unaccepted.

- **FR-018**: An element with `needsVolume: true` MUST be rejected at apply time
  unless the owner has supplied a non-empty `volume` via the proposal editor.
  The rejection MUST use the existing `ErrReferenceVolumeRequired` vocabulary.

- **FR-019**: The `Volume: "0"` placeholder sentinel used by
  `ReferenceMigrationService` for un-parseable legacy volumes MUST NOT be
  produced, proposed, or persisted by this feature under any circumstance.

### 4.5 RPC, opportunistically (Ask 3)

- **FR-020**: An `RPC` element MAY be emitted **only** when an RPC-shaped string
  is present in already-captured evidence — specifically `hypothesis.coin_type`,
  `quick_evidence.label_text`, or a `coin_type` claim value — and matches a
  leading `RPC` catalog token per the shared parser.

- **FR-021**: `src/agent/app/teams/deep_identification/providers/rpc.py` MUST NOT
  be modified. It remains a typed `unavailable` stub. No RPC network call, no
  scraping, no new provider automation.

- **FR-022**: Because `RPC` is `VolumeRequired: true` (F-5), an RPC element read
  off a bare label without a volume will be proposed unaccepted per FR-017 and
  FR-018. This is accepted behaviour, not a defect.

### 4.6 Narrative into notes (Ask 4)

- **FR-023**: The narrative MUST be proposed as an **editable proposal field the
  owner confirms** — never auto-applied. It reuses the existing `notes` key and
  the existing `"notes" -> "Notes"` allowlist entry.

- **FR-024 (append semantics — binding)**: The `notes` field MUST gain an
  **append** write mode on the `coin` target. Applying `notes` to a saved coin
  MUST produce:

  ```
  <existing Coin.Notes verbatim, including trailing content>
  <blank line>
  ## Deep Analysis - YYYY-MM-DD (job <jobID>)
  <blank line>
  <accepted notes value>
  ```

  When `Coin.Notes` is empty, the leading existing-notes segment and the
  separating blank line are omitted. The heading MUST use a plain ASCII hyphen
  and MUST contain the literal job id so it is machine-locatable. The date is the
  apply timestamp in UTC, `YYYY-MM-DD`.

- **FR-025 (idempotency rule — binding)**: Before appending, the service MUST
  scan the existing notes for a heading line matching
  `^## Deep Analysis - \d{4}-\d{2}-\d{2} \(job <thisJobID>\)$`.
  - If found, the existing block (from that heading up to the next `## ` heading
    or end of text) MUST be **replaced in place**, not duplicated.
  - If not found, a new block is appended at the end.

  Re-running deep analysis produces a **new job id** and therefore a **new
  block**; the previous analysis's block is preserved as a dated history. A
  second apply of the *same* job is additionally blocked upstream by the existing
  `AppliedAt` guard (`ErrDeepProposalAlreadyApplied`), so FR-025 is defence in
  depth against manual notes editing between applies.

- **FR-026**: The append MUST NOT truncate or reorder the owner's existing notes.
  If the composed result would exceed the `Coin.Notes` binding limit, the
  **appended block** MUST be truncated (with the heading preserved) — never the
  owner's pre-existing text. If the existing notes alone already meet or exceed
  the limit, the apply of `notes` MUST fail with a typed error rather than
  silently discard anything.

- **FR-027**: The saved-coin branch of `buildDeepProposalDocumentJSON` MUST emit
  a `notes` field composed from the report narrative and the findings list, using
  the same builder the intake branch uses today
  (`buildDeepIntakeProposalFields`), so there is exactly one notes-composition
  rule in the codebase.

- **FR-028 (binding — Brian's Decision C, 2026-08-17)**: The intake/draft
  branch's existing notes block (F-2) MUST be brought under the same dated
  heading format. There MUST be exactly **one** notes format in the codebase,
  produced by exactly one composer, for the `coin`, `draft`, and `wishlist`
  targets alike. This is a deliberate, accepted change to shipped Feature 351
  behaviour and to the 351 tests that assert the old shape — Brian accepted that
  it alters output he tested successfully on 2026-08-16. Divergent formats are
  **not** an acceptable fallback if the unification proves awkward; the awkward
  case is to be solved, not routed around.

- **FR-029**: On the `draft` and `wishlist` targets the row is newly created, so
  append degenerates to set. The heading MUST still be emitted (FR-028) so the
  provenance of the text is visible and so a later coin-target append composes
  cleanly. On the `draft` target the composed value remains a **proposed,
  owner-editable field** exactly as on `coin` — Decision C changes the format,
  not the confirm gate.

### 4.7 Provenance on every apply path (Ask 5, widened per F-1)

- **FR-030** *(SATISFIED — landed by Cassius during this session, see F-1)*: The
  `coin` apply target records a `models.CoinJournal` entry via
  `CoinRepository.CreateJournalEntry` naming the applied field names.
  Feature 352 MUST NOT re-implement this; it MUST extend
  `deepProposalJournalEntryText`'s field list to include `catalogReferences` when
  that field is applied, and MUST NOT let reference *values* into the entry text.

- **FR-031** *(SATISFIED — see FR-030)*: The `wishlist` apply target records the
  equivalent entry against the newly created coin.

- **FR-032** *(the remaining gap)*: The `draft` apply target MUST record a
  `models.DraftLifecycleEvent` with a **new** event type constant
  (`deep_analysis_applied`) added to `DraftLifecycleEventType` and to
  `IsValidDraftLifecycleEventType`. `models.CoinJournal.CoinID` is non-nullable
  and a draft has no coin until promotion, so `CoinJournal` is not usable at
  draft-apply time. `DraftLifecycleEvent` is the existing, correct surface and
  requires no new dependency in `DeepIdentificationProposalService` beyond the
  quick-capture write path it already holds.

- **FR-033**: When a draft carrying a `deep_analysis_applied` lifecycle event is
  later promoted, the promotion MUST carry that provenance into the promoted
  coin's `CoinJournal`. The condition MUST be the **presence of the lifecycle
  event**, not `draft.Source == "deep_identification"` — a draft may be
  deep-analysis-enriched without having been deep-analysis-created. This requires
  a journal write path in `QuickCaptureService`/`QuickCaptureRepository`, which is
  a new constructor dependency and a `main.go` DI change; that cost is accepted
  here because Phase 7 is already reworking this exact code path.

- **FR-034** *(SATISFIED)*: The contradictory `Apply()` doc comment has been
  corrected. Feature 352 MUST keep it accurate as the apply path grows.

### 4.8 Draft one-to-many catalog references (Brian's decision)

- **FR-035 (migration shape — binding)**: The draft one-to-many migration MUST be
  **purely additive**. A **new** table and model MUST be introduced
  (working name `QuickCaptureDraftCatalogReference`) with `DraftID` indexed
  **non-uniquely**, `UserID` indexed, `Catalog`, `Volume`, `Number`, a
  **nullable** `URI`, plus `SourceProvider` and `Confidence`.

- **FR-036**: `models.QuickCaptureDraftReference` and the
  `QuickCaptureDraft.SelectedNumistaReference` relation MUST be left **structurally
  untouched**: the `DraftID uniqueIndex` MUST NOT be dropped, the `URI not null`
  constraint MUST NOT be relaxed, and the relation MUST NOT be renamed or
  repurposed. SQLite cannot drop an index-backed constraint or relax `NOT NULL`
  without a destructive table rebuild, which the constitution's audit guidance
  and SS17 forbid on a shipped table holding user data.

- **FR-037**: On migration, every existing `QuickCaptureDraftReference` row MUST
  be backfilled as a `QuickCaptureDraftCatalogReference` row with the same
  `Catalog` / `Number` / `URI`, `Volume` empty, `SourceProvider: "numista"`,
  `Confidence: 1.0`. The backfill MUST be idempotent (guarded by an existence
  check on the `(DraftID, Catalog, Number)` triple) and MUST be safe to re-run on
  every process start.

- **FR-038**: Existing drafts MUST keep their current Numista selection working
  with **zero behaviour change**. Every consumer listed in Section 5 MUST
  continue to compile and pass its existing tests without assertion edits.

- **FR-039**: Draft promotion MUST copy **both** the legacy
  `SelectedNumistaReference` and every `QuickCaptureDraftCatalogReference` into
  `models.CoinReference`, deduplicated by `(Catalog, Volume, Number)`
  case-insensitively, so the backfilled duplicate of the Numista selection cannot
  produce two rows.

- **FR-040**: The rollback story MUST be: stop reading the new table. The legacy
  table, column, constraint, relation, and JSON key are untouched, so reverting
  the code reverts the feature completely. The new table may be dropped
  independently and its loss costs only deep-identification-sourced references.

- **FR-041**: The new relation MUST be exposed on the draft JSON payload under a
  new key (`catalogReferences`) **in addition to** the existing
  `selectedNumistaReference` key. The existing key MUST NOT change shape or
  nullability — `openapi_nullability_test.go:48-49` asserts it.

### 4.9 Cross-cutting constraints

- **FR-042**: `src/api/services/ocre_scoring.go` MUST NOT be modified (ADR 0010,
  binding).

- **FR-043**: Feature 351 FR-040's application-log ban applies unchanged and
  absolutely: no hypothesis values, query terms, candidate details, catalog
  numbers, cert numbers, narrative text, or image data may appear in application
  logs. Cert numbers and catalog references are owner data and are subject to the
  same ban. The owner-scoped SSE progress stream MAY carry this detail; image
  data is banned everywhere.

- **FR-044**: `image` remains valid **only** as an `EvidenceRef.provider`
  (351 FR-025). It is a legal value of `catalogReferences[].sourceProvider`
  because that property records evidence origin, not a provider run; it MUST NOT
  appear in coverage or attribution lists.

- **FR-045**: Every catalog code written or proposed MUST validate against
  `models.CatalogRegistry`. No catalog code may be invented. `NGC`, `RIC`, `RPC`,
  and `SEAR` are already seeded (`database.go:394-410`).

- **FR-046**: Go layering (Principle I) is unchanged: handlers -> services ->
  repository -> models; `models/` stays stdlib-only. The new parser helper is a
  **service**-layer artifact. The append-semantics reference write is a
  **service** method over an existing repository.

- **FR-047**: The Python agent MUST remain stateless (351 FR-032). All catalog
  registry validation and all parsing happen Go-side. The agent MAY carry the raw
  `coin_type` / cert values it already carries; it MUST NOT gain a registry, a
  parser, or a database handle.

### 4.10 Wishlist references (Brian's Decision A, recorded as ADR 0013)

- **FR-048 (guard removal — binding)**: The two `coin.IsWishlist ->
  References = nil` guards MUST be deleted:
  `coin_service.go:154-156` (`prepareCoinForCreate`) and
  `coin_service.go:175-177` (`createPreparedCoinInTx`, with its comment on
  lines 173-174). The `pendingReferences := coin.References; coin.References = nil`
  detach on `coin_service.go:171-172` MUST **stay** — it is GORM
  auto-cascade defence, not wishlist logic, and removing it reintroduces the
  `UNIQUE constraint failed: coin_references.id` crash of 2026-07-21.

- **FR-049 (untrusted-source guard — binding, replaces the removed enforcement)**:
  `WishlistSearchAlertService.ConvertCandidate` MUST clear
  `input.Coin.References` at its own boundary before calling `CoinService`.
  `ConvertCandidateInput.Coin` is a whole `models.Coin` deserialised from the
  request body (`wishlist_search_alert_service.go:140-143`), carrying catalog
  claims that originate from AI search-agent output over scraped dealer pages,
  with **no confirm gate and no provenance record**. Removing the wishlist guard
  without FR-049 would silently begin persisting them. The drop moves from a
  type-wide invariant to a single untrusted-source guard, which is where it
  belongs. FR-049 MUST land in the same change as FR-048 — not after it.

- **FR-050**: Wishlist reference writes MUST route through
  `CoinReferenceService.NormalizeAndValidate` (ID zeroing, `CatalogRegistry`
  validation, `VolumeRequired` enforcement, duplicate rejection) exactly as
  collection-coin writes do. No caller may reach `CoinReferenceRepository` with
  wishlist references directly, and the existing
  `refRepo != nil && refSvc != nil` routing guard MUST NOT gain a bypass.

- **FR-051**: The append path (FR-013) MUST deduplicate on
  `(Catalog, Volume, Number)` case-insensitively **on every target**, not only at
  draft promotion (FR-039). With FR-048 in place, a wishlist coin can now be
  enriched, purchased via `PurchaseCoin`, and enriched again; without a general
  dedupe that produces two rows for the same catalogue type.

- **FR-052**: The tests that encode the old invariant MUST be **rewritten to
  assert the new rule**, deliberately and individually — not deleted, and not
  patched incidentally:
  `coin_service_test.go:438` (`TestCreateCoin_DropsReferencesForWishlist`),
  `coin_service_test.go:490` (`TestCreateCoin_WishlistInvariant_AgentStylePayload`),
  `coin_handler_test.go:2075` (`TestCoinHandler_Create_WishlistWithReferencesStoresZeroReferences`),
  and the inline assertion at `wishlist_search_alert_service_test.go:615`. The
  last of these keeps asserting zero references — but for the FR-049 trust reason,
  with its comment updated to say so.

---

## 5. Consumer list — the `SelectedNumistaReference` breaking-change surface

This is the full enumeration requested. It is the reason FR-035/FR-036 mandate an
additive new table rather than an in-place one-to-many conversion. **34 files, 149
match sites.** Under the additive design, **none of these break**; under an
in-place conversion, every one is a candidate breakage.

### Go — models (2)
| File | Nature |
|---|---|
| `src/api/models/quick_capture_draft.go:49` | `SelectedNumistaReference *QuickCaptureDraftReference` relation, `json:"selectedNumistaReference"`, `extensions:"x-nullable"` |
| `src/api/models/quick_capture_draft.go:55-64` | `QuickCaptureDraftReference` struct: `DraftID uniqueIndex`, `URI not null` |
| `src/api/models/numista.go:164,303-330` | `SelectedNumistaReference` value type, `NewSelectedNumistaReference`, `ParseSelectedNumistaReference`, `Validate` — **distinct type, same name**; a rename would collide |

### Go — repository (2)
| File | Nature |
|---|---|
| `src/api/repository/quick_capture_repository.go:22,42,90,124,216,221,250,280,291-296` | `DraftReferenceMutation`, create/replace/clear, four `Preload("SelectedNumistaReference", ...)` sites, promotion copy to `CoinReference` |
| `src/api/database/database.go:86` | `AutoMigrate` registration |

### Go — services (1)
| File | Nature |
|---|---|
| `src/api/services/quick_capture_service.go:63,85,152,278,426-450` | Create/Update input structs and `normalizeSelectedNumistaReference` |

### Go — handlers (1)
| File | Nature |
|---|---|
| `src/api/handlers/quick_capture.go:80,276,451,476` | Request binding and `ParseSelectedNumistaReference` validation |

### Go — generated API docs (3)
`src/api/docs/swagger.yaml:3324-3358`, `src/api/docs/swagger.json:22398-22451`,
`src/api/docs/docs.go:22405-22458` — regenerated, must not drift.

### Go — tests (12)
`src/api/openapi_nullability_test.go:48-49` (asserts the `x-nullable` ref);
`src/api/database/migration_test.go:94-144` (asserts the `DraftID` index exists
and that old drafts preload to `nil`); `src/api/models/quick_capture_draft_test.go:24-79`
(asserts exact JSON `"selectedNumistaReference":{"catalog":...,"number":...,"uri":...}`);
`src/api/repository/quick_capture_repository_test.go` (15 sites);
`src/api/services/quick_capture_service_test.go` (19 sites);
`src/api/services/deep_identification_proposal_test.go:29`;
`src/api/handlers/quick_capture_handler_test.go` (14 sites, incl. exact JSON
substring assertions at 129/162/258/268);
`src/api/handlers/coin_handler_test.go:44`;
`src/api/handlers/deep_identification_test.go:61`;
`src/api/handlers/images_media_test.go:29`;
`src/api/repository/coin_repository_test.go:27`;
`src/api/integration/numista_workflows_test.go:81,258`,
`src/api/integration/numista_security_test.go:128`,
`src/api/integration/numista_compatibility_test.go:52`.

### TypeScript / Vue (8)
| File | Nature |
|---|---|
| `src/web/src/types/index.ts:450,1416` | `selectedNumistaReference?: SelectedNumistaReference \| null` on the draft type; the interface itself |
| `src/web/src/utils/numistaLookup.ts:7,48-59` | `selectedNumistaReferenceFromCandidate`, `numistaCandidateFromReference` |
| `src/web/src/pages/QuickCaptureDraftPage.vue:21-25,204,280,315` | Renders the link, round-trips the selection |
| `src/web/src/pages/CoinLookupPage.vue:295,542` | Builds the selection on submit |
| `src/web/src/components/quick-capture/QuickCaptureDraftCard.vue:26-29` | Renders `Numista #<number>` |
| `src/web/src/components/quick-capture/PromotionReadinessPanel.vue:143-144` | Readiness text |
| `src/web/src/test/numista-fixtures.ts:7,88-90` | Shared test fixture factory |
| `src/web/src/pages/__tests__/QuickCaptureDraftPage.test.ts`, `QuickCaptureDraftsPage.test.ts:48`, `NumistaStatusWorkflows.test.ts:17,75`, `components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts:17` | Fixture-driven assertions |

---

## 6. Acceptance criteria

Each is phrased to be directly testable.

### Structured references
- **AC-001**: Given a job whose `quick_evidence.ngc.cert_number` is `6379244-002`,
  the generated proposal contains a `catalogReferences` element with
  `catalog="NGC"`, `number="6379244-002"`, `volume=""`, `confidence=1.0`,
  `needsVolume=false`.
- **AC-002**: Given that proposal accepted and applied to a saved coin that
  already has two hand-entered `CoinReference` rows, the coin has **three**
  references afterward and both original rows are byte-identical to before.
- **AC-003**: Applying the same accepted `catalogReferences` payload twice (via
  two distinct jobs producing the same element) leaves exactly one matching
  `CoinReference` row and returns success, not a constraint error.
- **AC-004**: Given an OCRE `coin_type` claim of `RIC II Hadrian 39b`, the
  proposal contains an element `catalog="RIC"`, `volume="II"`,
  `number="Hadrian 39b"`, `confidence=0.90`, `rawText="RIC II Hadrian 39b"`.
- **AC-005**: Given a `coin_type` of `RIC 39b` (volume-required, no volume), the
  element has `confidence=0.30`, `needsVolume=true`, `accepted` defaulting to
  false, and `volume` is **not** `"0"`.
- **AC-006**: Attempting to apply an element with `needsVolume=true` and an empty
  owner-supplied volume returns `ErrReferenceVolumeRequired` and writes nothing.
- **AC-007**: Given `RPC III 1520` present in `hypothesis.coin_type`, an `RPC`
  element is proposed. No HTTP request is made to any `rpc.ashmus.ox.ac.uk` host
  in the test run.
- **AC-008**: A `catalogReferences` element with `catalog="NOTACATALOG"` is
  rejected at apply time and nothing is written.
- **AC-009**: A `catalogReferences` array of 11 elements is rejected at apply
  time.
- **AC-010**: A proposal field key not present in either scalar allowlist nor the
  collection allowlist is rejected with `ErrDeepProposalFieldNotAllowed` at both
  `PATCH .../proposal` and apply.

### coin_type back-compat
- **AC-011**: When the `coin_type` value parses successfully, the proposal
  contains **both** a `coin_type` scalar entry (default `accepted=false`) and the
  structured element (default `accepted=true`).
- **AC-012**: When the `coin_type` value fails to parse, the `coin_type` scalar
  entry keeps its confidence-driven default per 351 RD-3 and **no** structured
  element is emitted.
- **AC-013**: An existing coin whose `ReferenceText` is `"Sear 1234 (old note)"`
  has that value unchanged after any 352 apply that does not accept `coin_type`.

### Notes append
- **AC-014**: Given `Coin.Notes` = `"Bought at Frankfurt.\n\nToned."` and an
  applied narrative, the resulting notes **start with** exactly
  `"Bought at Frankfurt.\n\nToned."` and contain exactly one line matching
  `^## Deep Analysis - \d{4}-\d{2}-\d{2} \(job \d+\)$`.
- **AC-015**: Applying job 412 then job 413 to the same coin yields notes
  containing **two** `## Deep Analysis` headings, in apply order, with the owner's
  original text still first.
- **AC-016**: Given notes that already contain a `(job 412)` block, re-composing
  for job 412 replaces that block in place and the heading count stays at one.
- **AC-017**: When existing notes plus the composed block would exceed the notes
  limit, the owner's existing text is present in full in the result and only the
  appended block is shortened.
- **AC-018**: When existing notes alone are at the limit, applying `notes`
  returns a typed error and `Coin.Notes` is unchanged.

### Provenance
- **AC-019**: After a `coin` apply that includes `catalogReferences`, the
  `CoinJournal` entry names `catalogReferences` among the applied fields and
  contains **no** catalog number, cert number, or reference value.
- **AC-020**: After a `wishlist` apply, the created coin has the equivalent
  `CoinJournal` row (regression guard on Cassius's landed change).
- **AC-021**: After a `draft` apply, a `DraftLifecycleEvent` with type
  `deep_analysis_applied` exists for that draft.
- **AC-022**: Promoting that draft produces a `CoinJournal` row on the promoted
  coin carrying the deep-analysis provenance.

### Draft one-to-many
- **AC-023**: `db.Migrator().HasIndex(&models.QuickCaptureDraftReference{}, "DraftID")`
  still reports the index after migration, and
  `migration_test.go` passes with **no assertion edits**.
- **AC-024**: A draft with an existing Numista selection, loaded after migration,
  still serialises `"selectedNumistaReference":{"catalog":"Numista","number":"12345","uri":"..."}`
  exactly as `quick_capture_draft_test.go:79` asserts.
- **AC-025**: That same draft additionally exposes a `catalogReferences` array
  containing one backfilled Numista element.
- **AC-026**: Running the migration twice produces exactly one backfilled row per
  legacy reference.
- **AC-027**: A draft carrying a backfilled Numista element **and** its legacy
  `SelectedNumistaReference` promotes to a coin with exactly **one** Numista
  `CoinReference` row.
- **AC-028**: A draft carrying NGC + RIC + Numista catalog references promotes to
  a coin with three `CoinReference` rows.
- **AC-029**: `openapi_nullability_test.go` passes unchanged.

### Guardrails
- **AC-030**: `git diff` for the feature branch shows **zero** changed lines in
  `src/api/services/ocre_scoring.go` and
  `src/agent/app/teams/deep_identification/providers/rpc.py`.
- **AC-031**: The existing application-log assertion suite from 351 (FR-040)
  passes with the new fields present; no cert number, catalog number, or
  narrative text appears in any captured application log line.
- **AC-032**: `go test ./...` in `src/api` passes, including
  `architecture_test.go`.

### Wishlist references (ADR 0013)
- **AC-033**: `CreateCoin` with `IsWishlist: true` and two valid references
  persists **two** `coin_references` rows, normalised, with zeroed incoming IDs.
- **AC-034**: `ConvertCandidate` with an `input.Coin.References` slice persists
  **zero** references (FR-049) and still returns 201 — including when those
  references carry non-zero IDs from source data (the original 2026-07-21 crash
  payload).
- **AC-035**: A wishlist coin carrying a RIC reference, then `PurchaseCoin`d,
  retains exactly that one reference — no loss, no duplication.
- **AC-036**: Applying a deep-ID proposal with a `RIC I 12` reference to a coin
  that already has `ric i 12` produces exactly one row (FR-051, case-insensitive
  dedupe).
- **AC-037**: A deep-ID notes apply on the **intake/draft** path produces a block
  matching `^## Deep Analysis - \d{4}-\d{2}-\d{2} \(job \d+\)$`, byte-identical in
  shape to the coin-path block (FR-028, Decision C).

---

## 7. Non-goals

- **NG-1**: Automating the RPC provider. `providers/rpc.py` stays a typed
  `unavailable` stub (FR-021).
- **NG-2**: Automating NGC lookup. NGC terms prohibit automated access; the cert
  number is reused, never fetched.
- **NG-3**: Backfilling existing coins' free-text `Coin.ReferenceText` into
  `models.CoinReference`. That is `ReferenceMigrationService`'s job.
- **NG-4**: Removing, deprecating, or emptying `Coin.ReferenceText`.
- **NG-5**: Dropping the `QuickCaptureDraftReference` table, its unique index, or
  the `SelectedNumistaReference` relation. That is a future cleanup, gated on the
  new relation being proven in production.
- **NG-6**: Adding new catalog codes to `CatalogRegistry`.
- **NG-7**: Changing the 0.70 acceptance threshold or the corroboration bonus
  established by 351 RD-2/RD-3.
- **NG-8**: Any change to `ocre_scoring.go` (ADR 0010).
- **NG-9**: Making the Python agent registry-aware or database-aware.
- **NG-10**: A new provider, a new LLM call, or an increase in LLM calls per job
  (351 FR-031).
- **NG-11**: Editing `specs/351-*/tasks.md`.

---

## 8. Data model changes

All additive.

### New: `models.QuickCaptureDraftCatalogReference`
`ID`, `DraftID` (not null, **non-unique** index), `UserID` (not null, index),
`Catalog` (varchar 32, not null), `Volume` (varchar 64, **nullable/empty
allowed**), `Number` (varchar 128, not null), `URI` (varchar 2000,
**nullable** — NGC and RIC references frequently have none),
`SourceProvider` (varchar 32), `Confidence` (float), `CreatedAt`, `UpdatedAt`.
Registered in `AutoMigrate`. No unique index in v1 — dedupe is enforced in the
service layer so a future index can be added without a rebuild.

### Changed: `models.QuickCaptureDraft`
Gains `CatalogReferences []QuickCaptureDraftCatalogReference` with
`json:"catalogReferences"`. `SelectedNumistaReference` unchanged.

### Changed: `models.DraftLifecycleEventType`
Gains `DraftLifecycleEventDeepAnalysisApplied = "deep_analysis_applied"`, added
to `IsValidDraftLifecycleEventType`.

### Unchanged
`models.CoinReference`, `models.CatalogRegistry`, `models.Coin`,
`models.QuickCaptureDraftReference`, `models.CoinJournal`.

---

## 9. Security and privacy (Principle V)

- Every new read and write is owner-scoped through the existing scopes
  (`OwnedBy`, `OwnedByID`) and the existing job ownership check in
  `loadTerminalJobWithProposal`.
- `QuickCaptureDraftCatalogReference.UserID` is set from the draft's owner, never
  from request input, mirroring `normalizeSelectedNumistaReference`.
- Catalog codes are validated against the registry (FR-045); nothing
  user-supplied is interpolated into SQL.
- NGC cert numbers, catalog numbers, and narrative text are owner data and fall
  under the FR-043 application-log ban.
- No new external network access is introduced (NG-1, NG-2).

---

## 10. Definition-of-Done additions specific to 352

Beyond the standard SS21 checklist:

1. `reference_migration_service_test.go` passes with **zero** assertion edits
   after the parser extraction (FR-016).
2. `migration_test.go`, `quick_capture_draft_test.go`, and
   `openapi_nullability_test.go` pass with **zero** assertion edits (FR-038).
3. Swagger artifacts regenerated and committed; no drift.
4. The corrected doc comment (FR-034) is part of the same change.
5. ADR 0013 is merged (status `Accepted`) **before** any wishlist-reference code
   is merged, and FR-048 and FR-049 land in the **same commit** — the guard
   removal must never ship ahead of the untrusted-source guard that replaces it.
6. The four tests named in FR-052 are rewritten with an explicit reference to
   ADR 0013 in their doc comments, so the next reader learns the rule changed
   rather than assuming the assertion was always this way.

---

## 11. Conflicts — recorded findings and their status

These were raised for Brian's decision rather than settled unilaterally. Brian
decided C-1, C-2, and C-4 on 2026-08-17. **They are settled and are not to be
re-litigated.** They are retained in full because the reasoning is the record of
why the code looks the way it does.

| # | Subject | Status |
|---|---|---|
| C-1 | Wishlist references vs. landed spec 351 | **RESOLVED** — Decision A, ADR 0013 |
| C-2 | Unified notes format changes shipped 351 output | **RESOLVED** — Decision C |
| C-3 | Ask 5 was a bug report; half already fixed | Informational; no decision needed |
| C-4 | Literal one-to-many migration vs. additive table | **RESOLVED** — Decision B |
| C-5 | RPC almost always proposed unaccepted | Open, accepted as-is; UX-only |

**Still live as engineering constraints** (these are *not* conflicts awaiting a
decision, they are permanent traps): **F-4** (`UpdateCoinWithFields` replace
semantics) and **F-5**/**F-6** (`VolumeRequired`, NGC alias absence). F-4 in
particular must remain a loud warning for the life of this feature — see plan
Phase 2 and risk R2.

### C-1 — Wishlist references contradicted a decision ratified in Feature 351 — **RESOLVED (Decision A)**

**Decision (Brian, 2026-08-17): option (a). Wishlist items MAY hold catalog
references.** Recorded as
[ADR 0013](../../docs/adr/0013-wishlist-coins-may-hold-catalog-references.md),
which amends `specs/351-.../spec.md:843-852` and
`specs/351-.../tasks.md:277` per constitution SS22. Phase 6 is **ungated**;
US-6 stays in scope. Requirements: FR-048..FR-052.

The original analysis, preserved:

Asks 1-3 require the `wishlist` apply target to write structured
`CoinReference` rows. `CoinService` deliberately nils `coin.References` for
wishlist coins in two places, and Feature 351's spec ratified that as "the
pre-existing and intended invariant" (`specs/351-.../spec.md:843-852`).
Feature 351 is a landed spec, so per constitution SS22 and SS18.2 this could not
be silently reversed by a downstream feature.

Complicating the picture: the invariant was **already inconsistent** — two shipped
paths bypass it (F-3), and tracing it to origin showed it was a GORM crash
workaround retroactively described as a domain rule, with no recorded domain
rationale anywhere in the repository. The options were (a) relax it, (b) keep it
and scope 352's references to `coin` and `draft`, (c) tighten it and regress
Feature 341 FR-014. My recommendation was (a); Brian chose (a).

**Cost of Decision A that was not visible when the question was put to Brian —
see ADR 0013 Consequences item 5.** Removing the guards also un-blocks
`WishlistSearchAlertService.ConvertCandidate`, which takes a whole `models.Coin`
off the request body and would begin persisting unconfirmed AI-search-agent
catalog claims. FR-049 exists to hold that line and is a hard prerequisite, not a
nicety.

### C-2 — Unified notes format changes shipped Feature 351 behaviour — **RESOLVED (Decision C)**

**Decision (Brian, 2026-08-17): unify. One format across both paths.** The
dated-heading, job-id-keyed append format applies to the intake/draft path as
well as the saved-coin path, even though the intake path already composes a
narrative into `notes` today (`buildDeepIntakeProposalFields`). Brian explicitly
accepted that this changes output he tested successfully on 2026-08-16. See
FR-028, AC-037, plan Phase 5.

The original concern, preserved: bringing the intake notes block under the dated
heading alters output that 351 shipped and that has tests asserting its shape. It
was proposed because two divergent notes formats is worse; it is a real change to
landed behaviour and Brian should confirm rather than discover it. He confirmed.

**Residual regression risk (accepted, not eliminated)**: the intake notes block
is a *shipped, user-visible surface* Brian has already run against real coins.
Changing it is not a no-op refactor. Existing drafts created before this change
keep their old-format notes; nothing migrates them, so a collector will see both
formats side by side in their draft list until old drafts age out. That is
acceptable — the old text stays readable and hand-written notes are never touched
— but it should not surprise anyone. Tracked as risk R9.

### C-3 — F-1: Ask 5 was a bug report, and half of it is already fixed

The premise was that the coin path already journals. It did not — the doc comment
was aspirational. Cassius found the same defect independently during this session
and landed the `coin` and `wishlist` journal writes
(`.squad/decisions/inbox/cassius-deep-journal-wishlist.md`). Feature 352 therefore
inherits only the **draft** half, plus the obligation to keep the entry text free
of reference values (FR-043).

Two coordination notes for Brian:
- Cassius explicitly stopped short of the draft path because it would have
  required touching `quick_capture_draft.go` and `PromoteDraft` — the exact
  surface this spec's Phase 7 migrates. That was the right call; the two changes
  must land in one place, and this is it.
- FR-033 adds a journal write path to `QuickCaptureService`, which is a new
  constructor dependency and a `main.go` DI change. Cassius listed that as a
  reason not to do it in isolation. It is proportional here only because Phase 7
  is already in that code.

### C-4 — Brian's "migrate to one-to-many" vs. FR-035's additive table — **RESOLVED (Decision B)**

**Decision (Brian, 2026-08-17): accept the additive table.** He explicitly
acknowledged this is not the literal in-place migration he first described, and
accepted it on two grounds: SQLite cannot drop the `DraftID` unique index or
relax `URI NOT NULL` without a destructive table rebuild, and the existing
single-reference surface spans the 34 consumer files enumerated in Section 5.
FR-035..FR-041 stand as written. NG-5 (a deprecated table left behind) is an
accepted cost.

The original framing, preserved: Brian's stated decision was to migrate drafts to
one-to-many. FR-035/FR-036 deliver one-to-many **semantics** via a new additive
table rather than by relaxing the existing `uniqueIndex` and `URI not null`. I
believed this was what he wanted (drafts holding NGC + RIC + RPC) at a fraction of
the risk, but it is not literally what he said. He agreed.

### C-5 — RPC will almost always be proposed unaccepted

`RPC` is `VolumeRequired: true` (F-5). A number read off a slab label is
overwhelmingly likely to be volume-less, so the RPC element will usually arrive
at confidence 0.30, unaccepted, requiring manual volume entry. Ask 3 will
therefore feel less useful in practice than it sounds. The alternative — flipping
`RPC.VolumeRequired` to `false` — is a registry change affecting the manual
reference editor and every existing RPC reference, and I am not proposing it
unilaterally.
