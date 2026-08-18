# Squad Decisions

---

### Decision: Feature 353 Production Startup — Hotfix 2 (Correction of Incomplete Hotfix 1 / 1df5a99)

**Date:** 2026-08-18
**Author:** Cassius (Backend Developer)
**Requested by:** Brian DeNicola (second urgent production startup failure after 1df5a99)
**Feature:** specs/353-wishlist-availability-run-observability/
**Status:** IMPLEMENTED — PR #630 merged to main at d625b08; Docker images published

## Incident

After hotfix 1 (`1df5a99`: parent-before-child AutoMigrate ordering + `constraint:-` on
`AvailabilityRun.User`), production still failed to start. New evidence:
`GORM/glebarez SQLite migrator reached DROP TABLE availability_runs during the temp-table
rebuild and the DROP failed repeatedly with "constraint failed: FOREIGN KEY constraint
failed (787)"`.

## Why Hotfix 1 Was Incomplete

Hotfix 1's own regression test (`TestFeature353Migration_ProductionOrderPreservesLegacyDataAndAddsCycleSupport`)
used a hand-picked legacy fixture (`legacyAvailabilityRun`/`legacyAvailabilityResult` in
`database_test.go`) that declares **no** GORM relation/foreignKey tags at all. Empirically
verified: AutoMigrate-ing that fixture produces `availability_runs`/`availability_results`
tables with **zero** physical FK constraints (`PRAGMA foreign_key_list` returns no rows for
either table). Production's real schema is different — `models.AvailabilityRun` has shipped
since the very first availability-check commit with:

- `User User gorm:"foreignKey:UserID"` (belongs-to, no `constraint:-` until 1df5a99)
- `Results []AvailabilityResult gorm:"foreignKey:RunID"` (has-many, **never** suppressed by
  1df5a99 — only the `User` field was touched)

Both generate real, physical SQLite `CONSTRAINT ... FOREIGN KEY` clauses by default. So
hotfix 1's test could never have caught this — it validated a schema shape production never
actually had. Brutus's follow-up fixture (`trueLegacyAvailabilityRun`/`trueLegacyAvailabilityResult`
in `feature353_migration_order_regression_test.go`, using real GORM associations, no
`constraint:-`) reproduces the incident deterministically.

## Root Cause (Proven, Not Assumed)

Built a standalone reproduction harness against the real `github.com/glebarez/sqlite@v1.11.0`
migrator (not the abstract GORM interface) and confirmed via `PRAGMA foreign_keys=ON`
instrumented runs:

1. `models.AvailabilityRun.CycleID` (new, nullable FK to `AvailabilityCycle`) has **no**
   reverse struct field, but GORM still builds an implicit belongs-to relation from
   `AvailabilityCycle.Children []AvailabilityRun gorm:"foreignKey:CycleID"` (has-many). During
   `AutoMigrate(&models.AvailabilityRun{})`, since `!HasConstraint(...)` is true for this new
   relation, GORM calls `Migrator().CreateConstraint(...)`.
2. `glebarez/sqlite`'s `CreateConstraint` (unlike its `AlterColumn`/`DropColumn`) is **not**
   wrapped in `RunWithoutForeignKey` — it calls `recreateTable` directly, which does:
   `CREATE availability_runs__temp (with fk_availability_cycles_children)` → `INSERT ... SELECT`
   → **`DROP TABLE availability_runs`** → `RENAME ... TO availability_runs`, all with
   `PRAGMA foreign_keys=ON` still in effect (set by `database.Connect()` before `AutoMigrate`
   runs).
3. `availability_results.run_id` still has its live, physical FK to `availability_runs.id`
   (from the un-suppressed `Results` association, item above). SQLite refuses to `DROP TABLE`
   a table that a live physical FK still references while enforcement is on → `SQLITE_CONSTRAINT_FOREIGNKEY (787)`.

The pre-existing `User` FK is a **red herring** for this second failure: GORM's `AutoMigrate`
loop only ever *creates* a constraint that is declared-but-missing (`!HasConstraint`); it
never actively calls `DropConstraint` for a relation whose tag changed to `constraint:-`.
Adding `constraint:-` to `User` in hotfix 1 was therefore inert with respect to any existing
physical `fk_availability_runs_user` constraint — it neither caused nor prevented a rebuild by
itself. The actual trigger is exclusively the **new** `CycleID`/`Children` relationship's
`CreateConstraint` call landing on a table with a live incoming FK from a sibling table.

## Fix (Smallest Safe Change, No Global Flag)

Considered and rejected `gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}`: verified
in GORM's migrator source that this flag also gates the FK-embedding loop inside `CreateTable`,
so it would silently strip physical FKs from ~75 registered models on every **fresh** install,
not just the two availability tables — a broad, asymmetric blast radius disproportionate to
Principle IV.

Instead, applied the same, already-established, per-relation `constraint:-` suppression pattern
used for `User` in hotfix 1, extended to the two relations that actually reach
`CreateConstraint`/`recreateTable` on `availability_runs`/`availability_cycles`:

- `models/availability_check.go`: added an explicit `Cycle *AvailabilityCycle
  gorm:"foreignKey:CycleID;references:ID;constraint:-" json:"-"` field to `AvailabilityRun`
  (mirrors the existing `User` field), and added `constraint:-` to the pre-existing
  `Results []AvailabilityResult gorm:"foreignKey:RunID;constraint:-"` field (the association
  hotfix 1 left untouched and which is what physically blocks the rebuild).
- `models/availability_cycle.go`: added `constraint:-` to `Children []AvailabilityRun
  gorm:"foreignKey:CycleID;constraint:-"` (the has-many side of the new relation — needed in
  addition to the `Cycle` field above; empirically confirmed a single-sided suppression was
  insufficient, GORM parses each side's relation definition independently).

No changes to `database.go`'s AutoMigrate ordering (already correct from hotfix 1) or to the
sqlite DSN/config helper (not required).

## Proof

Standalone reproduction (`tmp_fk_probe`, deleted after use, not committed):
- **Before fix** (hotfix-1-only models, real GORM-authored legacy fixture with genuine
  `user_id->users.id` and `run_id->availability_runs.id` physical FKs): deterministically
  reproduced `constraint failed: FOREIGN KEY constraint failed (787)` at
  `DROP TABLE availability_runs`, verbatim to the reported production error.
- **After fix**: same fixture, same production `AutoMigrate` call list, migration succeeds;
  `availability_runs`/`availability_results` row counts unchanged; no `..._temp` table left
  behind.

Test suite (Brutus's enhanced fixture, already present uncommitted in
`database/feature353_migration_order_regression_test.go`, exercising the real, live, ~75-model
production `AutoMigrate(...)` call read directly from `database.go`'s source):
- `TestFeature353Migration_RealProductionAutoMigrateListStillFailsWithFK787` — now **passes**
  end-to-end (previously intentionally `t.Fatal`'d with a BLOCK message documenting the
  incomplete fix; not weakened, the underlying bug was fixed).
- `TestFeature353Migration_RepeatedUpgradeIsDeterministic` (6 iterations) — passes.
- `TestFeature353Migration_FreshInstallSucceedsWithRealProductionList` (fresh DB + idempotent
  restart) — passes.
- `TestPreFeature353FixtureShapeMatchesRealProductionHistory` — passes (documents old fixture's
  gap vs. the corrected one).
- All pre-existing Feature 353 tests from hotfix 1 — still pass.
- Full suite: `go build ./...`, `go vet ./...`, `go test ./...` — all clean, all packages `ok`.

## Recovery

Every failed production restart's `AutoMigrate` runs inside a single `gorm.DB.Transaction`
(`recreateTable`'s `CREATE`/`INSERT`/`DROP`/`RENAME` sequence); the failed `DROP TABLE` rolled
the whole transaction back atomically each time. No partial `availability_runs__temp` tables,
no row loss, no manual cleanup required at any point across both incidents — confirmed by the
repeated-run/idempotency test asserting no leftover `..._temp` table after a failed or
successful pass.

## Schema Asymmetry Accepted (Documented, Not a Bug)

- **Upgraded (pre-353) databases:** `availability_runs.user_id -> users.id` physical FK
  remains live post-migration (rebuilding it away is out of scope and not needed for
  correctness — `constraint:-` only suppresses *future* DDL generation, it cannot retroactively
  drop a constraint nobody asks GORM to remove). `availability_results.run_id ->
  availability_runs.id` also remains live — intentionally left in place since removing it was
  never required to fix the incident and doing so would be an unproportional, unrequested
  destructive schema change.
- **Fresh installs going forward:** neither the `User`, `Results`, nor the new `Cycle`/`Children`
  relation gets a physical FK at all (all four now carry `constraint:-`); referential integrity
  for all of them is enforced at the service/repository layer, consistent with the project's
  existing "nullable lookup FKs added post-launch use `constraint:-`" convention.
- Both shapes are safe and already covered by the enhanced fixture/tests above. Ownership
  scoping, `Preload("Results")`/`Preload("User")` behavior, and legacy `UserID = 0` readability
  are all unaffected — `constraint:-` only suppresses DDL, not GORM's Preload/association
  query behavior, verified by the full test suite (including `availability_repository_test.go`
  and `valuation_repository_test.go`, both `Preload("Results")`/`Preload("User")` consumers).

## Files Changed

- `src/api/models/availability_check.go` — added `Cycle` field with `constraint:-`; added
  `constraint:-` to `Results`
- `src/api/models/availability_cycle.go` — added `constraint:-` to `Children`
- No changes to `src/api/database/database.go` (ordering already correct) or any test file
  (Brutus's enhanced fixture was already present uncommitted; validated against it, not edited)

## Outcome

Root cause of the second failure identified and proven (new `CycleID`/`Children` relation's
unprotected `CreateConstraint`/`recreateTable`, not the inert `User` `constraint:-` change).
Fix applied with the smallest possible blast radius (four relation tags, no global config
change, no AutoMigrate ordering change). Full Quality Gate (build/vet/test) green. Merged to
main; Docker images published. Awaiting external deployment confirmation.

## Approvals

- **Brutus (QC):** Fixture gap analysis confirmed; root cause proven via deterministic reproduction; full test suite passing; FK 787 specifically validated resolved. Approved.
- **Maximus (Architect):** Architecture pattern consistent (mirrors User field); constraint tag convention already established (post-launch nullable FK rule); Principle IV satisfied (4 tags, 2 files, no global config); schema asymmetry documented and safe; backward-compatibility verified (GORM Preload behavior unchanged). **Strict Lockout (independent revision)** — earlier recommendation to explore global flag withdrawn; focused constraint-tag approach is lower-risk and sufficient. Approved for merge.

---

### Decision: Feature 353 production startup blocker — AutoMigrate parent/child ordering + latent `constraint:-` FK gap

**Date:** 2026-08-18
**Feature:** 353-wishlist-availability-run-observability (production hotfix)
**Agent:** Cassius (Backend Developer)
**Status:** IMPLEMENTED — PR #629 opened, hotfix validated by Brutus & Maximus

## Root Cause (two distinct, compounding bugs — ordering alone was insufficient)

1. **AutoMigrate registration order.** `database.go`'s single `DB.AutoMigrate(...)` call
   registered `&models.AvailabilityRun{}` (the child, gaining a new nullable `CycleID *uint`
   column) far earlier in the slice than `&models.AvailabilityCycle{}` (the new parent table),
   which was the very last argument. On any pre-existing on-disk `availability_runs` table
   (every real production DB), adding `cycle_id` is a SQLite full table-rebuild (GORM/glebarez
   `recreateTable`: `CREATE availability_runs__temp` with the new FK → `INSERT INTO
   __temp SELECT ... FROM availability_runs` → `DROP` → `RENAME`). GORM's own `ReorderModels`
   dependency sort does **not** catch this: the FK is only discoverable from
   `AvailabilityCycle.Children []AvailabilityRun gorm:"foreignKey:CycleID"` (a `HasMany` on the
   *parent* struct), and `AvailabilityRun` itself declares no reverse `Cycle` field, so GORM
   can't infer "Run depends on Cycle" from Run's own schema. With `PRAGMA foreign_keys=ON`
   (set by `Connect()` before `AutoMigrate`), the temp-table copy validates the new FK against
   `availability_cycles`, which doesn't exist yet at that point in the call → exact production
   error `SQL logic error: no such table: main.availability_cycles (1)`, `Connect()`'s
   `log.Fatalf` kills the process.
   - **Fix:** moved `&models.AvailabilityCycle{}` immediately before `&models.AvailabilityRun{}`
     in the `AutoMigrate` slice (parent-before-child), matching the same convention already
     followed for every other parent/child pair in that list (e.g. `DeepIdentificationJob`
     before its child event/run/artifact models).

2. **Pre-existing (pre-Feature-353) `AvailabilityRun.User` belongs-to FK is incompatible with
   the legacy `UserID = 0` admin-run sentinel.** `User User gorm:"foreignKey:UserID"` has
   existed since the original wishlist-availability feature (commit `8d52d27`) and generates a
   `FOREIGN KEY (user_id) REFERENCES users(id)` constraint in the table's DDL. Legacy
   admin-triggered `AvailabilityRun` rows intentionally use `UserID = 0` (a non-owner sentinel,
   never a real user id — see the type doc comment on `AvailabilityRun`). That FK was never
   actually validated against existing data until *now*, because SQLite only re-validates a
   table's full constraint set during a rebuild — and Feature 353 is the first change ever to
   force one on `availability_runs`. Confirmed empirically: reordering alone (fix #1) still
   fails with `FOREIGN KEY constraint failed` on any DB containing legacy `UserID = 0` rows;
   the identical reorder succeeds immediately when no such legacy rows are present.
   - **Fix:** added `;constraint:-` to `AvailabilityRun.User`'s gorm tag
     (`src/api/models/availability_check.go`), which tells GORM to skip generating any
     DDL-level FK constraint for this specific belongs-to association (Preload/joins via
     `foreignKey:UserID` are unaffected — only physical constraint creation is skipped).
     This is the same established pattern already used for `Coin.StorageLocationID` per
     Cassius's history notes ("nullable lookup FKs added post-launch should use `constraint:-`
     ... to avoid destructive table rebuilds; enforce ownership/referential correctness in
     services/repositories instead"), extended here to an FK that predates the feature that
     exposed it. Referential correctness for `UserID` is already enforced at the
     service/handler layer (ownership checks on every read/write path) and was never actually
     relying on the DB-level constraint in practice, since it silently went unvalidated for the
     entire lifetime of legacy admin rows.

## Recovery Behavior For Already-Broken Databases

SQLite's `recreateTable` runs the `CREATE __temp` / `INSERT ... SELECT` / `DROP` / `RENAME`
sequence inside a single `db.Transaction(...)` (glebarez `migrator.go:400-410`). Every failed
production restart therefore rolled back atomically: the original `availability_runs` table
and all legacy rows/results are left completely untouched, and no `availability_runs__temp`
table is ever left behind. No manual recovery/cleanup is required for any environment that hit
this crash — the very next successful startup with this fix applied completes the migration
normally against the still-intact original table.

## Validation

- New repro test (disk-based, not `:memory:` — the failure does not reproduce on `:memory:`
  because a same-process `:memory:` DB is fully populated by the time GORM inspects it in one
  continuous session; disabled/removed after confirming both root causes and the fix) proved:
  (a) buggy order + legacy `UserID=0` row → `no such table: main.availability_cycles`
      (exact production error)
  (b) reordered (cycle-before-run) + legacy `UserID=0` row → `FOREIGN KEY constraint failed`
      (second, previously-latent bug)
  (c) reordered + `constraint:-` on `AvailabilityRun.User` + legacy `UserID=0` row → succeeds
- `gofmt -l` clean on both changed files
- `go build ./...` and `go vet ./...` clean
- `go test ./database/... -v` — all pass, including
  `TestAvailabilityCycleMigration_AdditiveAndPreservesLegacyRows` (idempotency + byte-equivalent
  legacy row preservation)
- `go test ./...` (full suite) — all packages pass (root, capture, database, handlers,
  integration, middleware, models, repository, services, testutil)

## Files Changed

- `src/api/database/database.go` — moved `&models.AvailabilityCycle{}` to immediately precede
  `&models.AvailabilityRun{}`/`&models.AvailabilityResult{}` in the `AutoMigrate` call
  (parent-before-child).
- `src/api/models/availability_check.go` — added `constraint:-` to `AvailabilityRun.User`'s
  gorm tag to stop GORM from generating a DDL FK constraint incompatible with the
  legacy `UserID = 0` admin-run sentinel.

## Approvals

- **Brutus (QC):** Reproduced exact production error; validated both root causes; confirmed fix succeeds with legacy rows; idempotency tests pass. Approved.
- **Maximus (Architect):** Reviewed parent-before-child pattern precedent, `constraint:-` semantics, recovery atomicity, and ownership enforcement layer. Architecture and §17 gates preserved. Approved.

## Release Status

PR #629 opened; normal merge process in progress (auto-merge unavailable; awaiting beta sync).

---

## Baseline Note: Pre-existing flaky test in `src/api/services` (unrelated to Feature 344)

**Date:** 2026-08-15
**Feature:** 344-deep-agentic-coin-identification (implementation session)
**Status:** BASELINED — not modified, not caused by Feature 344 changes

## Observation

While running the full Go test suite (`go test ./...`) after landing Feature
344's Phase 2 foundational changes (T004–T021), a single test in
`github.com/briandenicola/ancient-coins-api/services` intermittently fails
under `go test ./...` (full-package run) but passes reliably in isolation
(`go test ./services/... -run <name>`). The specific failing test name
varies between runs, which points to shared/global test state or ordering
sensitivity within the `services` package rather than a defect in any
individual test.

This flake was verified as pre-existing and unrelated to Feature 344 by
using `git stash` to remove all Feature 344 changes and re-running the same
test against the clean branch tip, where the same class of flake reproduced
identically.

## Decision

No fix is attempted as part of Feature 344. Future work (outside Feature
344's scope) should investigate shared mutable state across `services`
package tests as the likely root cause.

---

### Decision: Feature 344 Phase 6 — Provider-Tool Boundary Implementation (T051–T054)

**Date:** 2026-08-15
**Feature:** 344-deep-agentic-coin-identification
**Phase:** 6 (Foundational — Go provider tool boundary, T051–T054)
**Status:** IMPLEMENTED

## Key Decisions

1. **Job-scoped token family uses new `MintForJob(userID, jobID)` method**
   - Go has no method overloading; existing `Mint(userID)` has external callers
   - New method uses distinct 4-field token shape (`userID:jobID:expiry:sig`)
   - Two token families are mutually rejecting; existing internal-token routes unmodified
   - New middleware `InternalJobTokenRequired` used only for three new provider-tool routes

2. **Nomisma call budget is Go constant, not settings key**
   - `const deepNomismaCallBudget = 3` in `handlers/internal_tools.go`
   - Matches contract §2 sample and Phase 1/2 settings schema (no entry existed)
   - Future phases can promote to admin-tunable setting if needed

3. **Provider budgets (`DeepProviderBudgetTracker`) are in-memory, per-job**
   - `sync.Mutex`-guarded map keyed by `"<jobID>:<provider>"`
   - Ephemeral per-run enforcement; no persistence needed (jobs are time-boxed to ≤900s)
   - Avoids new migration/table; remains additive and small
   - `Reset(jobID)` provided for future wiring from job-completion path if needed

4. **Status-vocabulary mapping for `numista_search`/`numista_detail`**
   - Provider-layer `NumistaErrorKind` vocabulary is richer than contract §7 requirement
   - Mapping: `invalid_request`/`malformed_response`/`cancelled` → `unavailable`
   - Mapping: `unauthorized` → `unconfigured` (both mean "cannot use Numista right now")
   - Mirrors spirit of non-fabrication: never invent a false no_match

## Alignment
- Principle IV: Smallest complete changes, no unrelated refactoring
- Principle I: Clear layering (handler → service → repository pattern maintained)
- §17 Quality Gate: All gates passed; tests prove token families distinct and tamper-rejecting

---

### Decision: Feature 344 Phase 7 — Python LangGraph Pipeline & Go↔Python Streaming (T055–T078)

**Date:** 2026-08-15
**Feature:** 344-deep-agentic-coin-identification
**Phase:** 7 (Python agent implementation + Go pipeline runner, T055–T078)
**Status:** IMPLEMENTED — **See Remediation below for proposal contract correction**

## Key Decisions

1. **Job-scoped internal token TTL extended via `MintForJobWithTTL`**
   - Deep pipeline can run up to 900s; 30s default TTL too short
   - New method `MintForJobWithTTL(userID, jobID uint, ttl time.Duration)` reuses HMAC logic
   - `MintForJob` now delegates to this method with 30s default (unchanged for existing callers)
   - Pipeline runner mints with `total_timeout_s + 30s` margin

2. **Per-run bounds derived in code, not settings keys**
   - `deepPipelineBounds()` derives from constants matching Python agent defaults
   - Max concurrency 2, provider timeout 45s, recursion limit 12
   - `total_timeout_s = clamp(HardTimeout - 20s, [30, 900])`
   - Keeps settings schema unchanged for this phase; future phases can promote to tunable

3. **`quick_evidence` intentionally omitted (null) from Go→Python request**
   - Field is optional in Python contract; Go leaves it `nil` (omitted in JSON)
   - Wiring Feature 341 Quick Lookup into job creation is not Phase 7 scope
   - Revisit when later phase explicitly integrates Quick Identify results

4. **NGC provider node is fixed link-out only, no OCR**
   - Reuses `quick_evidence.ngc` (already extracted upstream by Feature 341)
   - Returns `not_automated` + fixed link URL; never performs extraction/OCR itself
   - No live NGC API call (Terms of Use prohibited)

5. **OCRE/RPC catalog entries always non-automatable (unconditional)**
   - Settings flags `OCREEnabled`/`RPCEnabled` exist but not read by provider catalog
   - Reserved for deferred T155/T156; this phase hardcodes `automatable: false`
   - When T155/T156 implemented, both catalog-building and settings-plumbing must be revisited together

6. **Terminal SSE frames (`synthesis`/`error`) not persisted as individual events**
   - `SettleTerminal` already appends exactly one terminal event transactionally
   - Pipeline runner's `onFrame` callback does not call `AppendEvent` for terminal frames
   - Only router_selected/provider_started/provider_result/evaluation/synthesis_started/progress persisted

7. **Proposal JSON originally flat shape — See Remediation for superseding decision**
   - Initial Phase 7 implementation used minimal flat `{"fields": {field: value}}` shape
   - This shape was insufficient for review/confirm/apply workflows
   - **This decision was superseded by Remediation decision #1 below**

## Pre-existing production bug found (not fixed — out of scope)
- `collection_tools.py:75` uses literal placeholder `"Authorization": "******"` instead of f-string
- All existing collection tools send non-functional headers; this was even encoded as expected in tests
- Left unfixed per explicit instruction not to modify unrelated code
- Flagged for coordinator/QC to fix in separate change with test update

## Alignment
- Principle IV: Proportional changes; no fabricated completion
- Principle II: Python stateless/DB-free; Go is authoritative
- §17: Security constraints honored; secrets/logging discipline maintained
- §21: Owner-scoping and authorization layers preserved

---

### Decision: Feature 344 Remediation — Proposal Contract & Security Allowlisting (Post-QC Block)

**Date:** 2026-08-15
**Feature:** 344-deep-agentic-coin-identification
**Agent:** Cassius (independent revision after QC blocks)
**Status:** IMPLEMENTED — **Supersedes Phase 7 decision #7; correction history preserved**

## Correction History

**Prior state (Phase 7, superseded):** Proposal was shipped as minimal flat `{"fields": {field: value}}` map, deferring the rich shape to "a later phase."

**QC Finding:** This flat shape was insufficient; deep identification requires proposal to round-trip through PATCH edit/accept and POST apply. The existing `deep_identification_proposal.go` parser already consumed rich `deepProposalDocument` (schemaVersion, per-field proposed/confidence/evidence[]/ownerEdited/ownerValue/accepted), so flat proposals failed and dropped citations + confidence.

**Accepted truth (Remediation, this record):** Proposal is the full rich `deepProposalDocument`. This was the actual application contract all along; the initial deferral was incomplete. Commits `896955e` and `080e598` corrected the implementation.

## Key Decisions

1. **Proposal built as rich `deepProposalDocument` in Go**
   - Directly reuses existing application-owned DTOs from `deep_identification_proposal.go`
   - No duplicate/incompatible structs; JSON tags match TS `DeepProposal` types and OpenAPI schemas
   - `DeepProposalEditor.vue` receives exactly the shape it consumes
   - `deepProposalDocument` carries: schemaVersion, per-field `proposed`/`confidence`/`evidence[]`/`ownerEdited`/`ownerValue`/`accepted`

2. **Evidence/citations resolved Go-side from `provider_result` frames**
   - Python synthesis carries lightweight `{provider, claim_index}` pointers, not resolved citations
   - Full citations/excerpts/confidence live in per-provider `provider_result` frames streamed during pipeline
   - Runner accumulates `provider_result` claims and resolves each evidence_ref into full claim on document build
   - **No Python contract change:** rich shape was always Go/OpenAPI/TS concern; bug was Go emitting wrong shape
   - Preserves citations + per-field confidence without changing Python synthesis contract

3. **Field allowlist + citation-host allowlist re-validated at translation (document build)**
   - Field allowlist: `deepProposalCoinFieldAllowlist` filters to coin-updatable fields only before persistence
   - Citation allowlist: `deepCitationHostAllowed` re-validates every resolved citation URI host against allowlist at translation
   - Claims with non-allowlisted citation hosts are dropped from proposal
   - Owner-edited/owner-value/accepted initialized pristine (false/nil/nil) — no auto-accept, no auto-write
   - In-memory validation at translation time provides defense-in-depth with apply-time field allowlist

4. **Deep Analysis capability probe: `GET /deep-identification/capability`**
   - Exposes only boolean `{"enabled": bool}` derived from `SettingDeepIdentificationEnabled`
   - Protected (JWT required); never exposes underlying settings
   - Frontend composable `useDeepIdentificationCapability` fails closed (hidden on error)
   - Backend independently 403s job creation when disabled
   - Quick Identify workflow untouched

5. **Terminal error frame emits `code` not `error_kind`**
   - Contract §3 defines error frame as `{code, message}` with typed codes: llm_unavailable | timeout | invalid_model_output | internal
   - Python implementation was emitting `error_kind` (different field); this was implementation/contract mismatch, not spec change
   - Narrow error classifier: ValidationError/OutputParserException → invalid_model_output; connection/Unavailable → llm_unavailable; else → internal
   - Go `runJob` maps any run error to stored failure code `agent_unavailable` regardless; `code` drives message string

6. **`bounds.recursion_limit` bound into LangGraph config (contract §6)**
   - Production driver `run_deep_identification_stream` is hand-written async generator (emits per-provider SSE frames)
   - Does not invoke compiled graph, so recursion limit currently inert for serving path
   - Binding via `compiled.with_config({"recursion_limit": N})` ensures any graph-based caller is capped at contract bound
   - Documented honestly as inert for current streaming driver rather than fabricating graph invocation SSE envelope cannot use

## Integration Test Coverage

`deep_identification_proposal_integration_test.go` drives realistic runner synthesis through:
- Persistence → Parse → PATCH accept/owner-edit → Apply
- Asserts: (a) no coin write before Apply, (b) citation + confidence survive round-trip
- Fails under old flat shape by construction (cannot express proposed/confidence/citation)

## Alignment
- Principle IV: Simplest complete change; resolved evidence Go-side using existing DTOs
- Principle V: Input validation, allowlisting, owner-scoping all enforced at translation + apply
- §17 Quality Gate: All gates passed; integration tests cover realistic workflows
- §21 Definition of Done: T123/T124 proposal acceptance confirmed complete

---

## Decision: Real vision-hypothesis structured output + degrade-ladder deviation

**Author:** Cassius (Backend Dev)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 3+4 (T019-T033)  
**Status:** IMPLEMENTED

Implemented structured vision extraction with `get_structured_model(config, schema)` factory. Degrade ladder: structured → retry once → prose regex → quick-evidence fallback → typed-empty. Deviated from tasks.md literal to prefer quick-evidence hypothesis over typed-empty (strictly better, zero cost, matches prior shipping behavior). Provider-specific methods: Anthropic `function_calling`, Ollama `json_schema`. Wiring to router/query-builder/evaluator deferred to later phases per task dependency ordering.

**Key decision:** `include_raw=True` surfaces schema-validation failures as `parsing_error` in return value, not exception — enables prose extraction from same response with zero additional LLM calls. Retry-once logic lives in `build_hypothesis_from_vision` only on failure branch; happy path makes exactly one call.

**Test coverage:** +20 new tests, 299 passing (was 279). Verified via `pytest tests/ -q` and `ruff check app/ tests/`.

---

## Decision: Fix deep-identification worker SQLITE_BUSY claim contention

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 2 infrastructure  
**Status:** IMPLEMENTED

Root cause: deferred SQLite transactions in `ClaimNextQueuedJob` race an upgrade lock from read to write, failing with `SQLITE_BUSY` under concurrent workers. Fix: added `_txlock=immediate&_pragma=busy_timeout(5000)` to DSN in `database/database.go`, enforcing write-lock acquisition at `BEGIN` rather than on first write.

**Hard constraint verified:** Reproduction test (`TestDeepIdentificationRepository_ConcurrentClaimNoLockContention`) confirmed 29/300 failures before fix, 0/300 after, even with just `busy_timeout` alone. Blast radius assessed safe across all schedulers (valuation, health, auction, coin-of-day, etc.); all benefit from reduced transient lock contention.

**Test coverage:** New `TestDeepIdentificationRepository_ConcurrentClaimNoLockContention` on real on-disk WAL SQLite file; 5+ consecutive runs passing. Log line downgraded from ERROR to WARN (transient, self-healing condition).

---

## Decision: Phase 10 wishlist save destination

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 10 (T072-T075, T119)  
**Status:** IMPLEMENTED

Extended `DeepIdentificationProposalService.Apply` to support `target="wishlist"` alongside `"draft"` and `"coin"`. Builds `models.Coin{IsWishlist: true}` via existing 14-entry allowlist (no schema migration). `isWishlist` remains intent-only (set from destination, never proposed). Name derivation: reads `proposal.Fields["workingTitle"]` or falls back to `"Unidentified Coin (Deep Analysis)"` when hypothesis empty.

**Validation:** No new `validateCoinMinimumForPromotion`-equivalent; `CoinService.prepareCoinForCreate` already validates Era/Category. Confirmed `isWishlist` rejection via `TestDeepIdentificationProposal_WishlistApplyRejectsIsWishlistAsProposedField`.

---

## Decision: Phase 5 provider query terms + candidate ranking

**Author:** Cassius (Backend Dev)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 5 (T034-T043, T121-T123)  
**Status:** IMPLEMENTED (partial wiring)

Built deterministic query composition (`query_terms.py`): precedence `numista_query` → `label_text` → hypothesis (ruler+denomination → ruler → denomination+material → obverseInscription) → notes[:200]. Built candidate ranker (`candidate_ranking.py`) over provider results using hypothesis reverse-type/legend tokens. Deleted placeholder `_DEFAULT_QUERY = "unidentified ancient coin"`; now return `no_match`/`insufficient_query_evidence`/zero-call when no terms available.

**Implementation gap (not mine to fix):** Hypothesis parameter added to `numista.run()`, `nomisma.run()`, `ocre.run()` with default `None`, but `graph.py`'s provider-fanout call site does not yet pass `state.get("hypothesis")` through. One-line wire-up needed; hypothesis-derived tiers (3) and ranking currently unreachable in live pipeline. Quick-evidence tiers (1-2, 4) and zero-placeholder guarantee (FR-011) already live.

**Test coverage:** +36 new tests, 335 passing. Verified `pytest tests/ -q` and `ruff check app/ tests/`.

---

## Decision: Phase 6 deterministic router + wiring addenda

**Author:** Maximus (Lead/Architect)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 6 (T044-T050)  
**Status:** IMPLEMENTED (with two addenda)

Replaced LLM-driven router with pure function of `(catalog, provider_override, bounds, quick_evidence, hypothesis)`. Removed one LLM call per job. Added `skipped[]` array to `router_selected` SSE frame. RD-7 inclusion-by-default: selects all automatable, in-bounds providers unless evidence indicates non-Roman coin → skip OCRE. Determinism proven by `test_route_is_deterministic_across_identical_runs`.

**Addendum 1 — Evaluator wiring:** Fixed `evaluator_node` to pass `hypothesis=state.get("hypothesis")` to `evaluate()` call. Prior: hypothesis available in unit tests only, inert in pipeline.

**Addendum 2 — Provider wiring:** Added `hypothesis` parameter to `_run_one_provider()` and threaded through from `provider_fanout_node`. Updated 13 test fakes with `hypothesis=None` signature.

**Test coverage:** +27 new/updated router and SSE tests from Phase 6, +10 from both addenda. Final count 337 passing (was 299). One pre-existing timing flake in `test_deep_identification_sse.py` regressed (timing-sensitive, not caused by this batch).

---

## Decision: Phase 7 image as first-class claim source

**Author:** Brutus (QA/Integration)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — Phase 7 (T051-T055)  
**Status:** IMPLEMENTED (partial wiring)

Extended evaluator to flatten `CoinHypothesis` fields into claim-disagreement pipeline as `(source="image", claim_index=None, value)` entries. Image/provider claims never resolved by precedence (both kept). Field with one image + one provider claim → unresolved; one image only (or one provider only) → no disagreement, not counted in `resolved_count`.

**Type safety:** `EvidenceRef.provider` accepts `"image"` as string; `ProviderEvidence.provider` rejects it (literal union). Verified by `test_image_never_becomes_a_provider_name`.

**Constraint verified:** `detect_disagreements()` takes no `model` parameter (pure function). Model used only by `_summarize()` for `unresolved_questions`; poisoned/error tests prove return values identical regardless.

**Implementation gap (not mine to fix):** `graph.py::evaluator_node` still calls `evaluate(model, state.get("evidence", []))` without `hypothesis=` kwarg. Parameter defaults to `None`; hypothesis invisible in pipeline.

---

## Decision: Deep Analysis activity timeline UI

**Author:** Aurelia (Frontend Developer)  
**Date:** 2026-08-17  
**Feature:** `specs/351-vision-first-deep-identification` — FR-040 frontend half  
**Status:** SHIPPED

Implemented `DeepAnalysisActivityTimeline.vue` deriving step-by-step progress from existing `DeepStreamEvent[]` stream (no backend contract changes). Recognizes lifecycle events (`job_accepted`, `router_selected`, `evaluation`, `synthesis_started`, `terminal`) with curated labels. `progress` events use `knownPhaseLabels` map with titleCase fallback for unknown phases — new backend phases render immediately without frontend deploy.

**Accessibility:** Icon + text label per step (never color alone); `role="log"` with `aria-live="polite"`. `<details>` auto-collapses on terminal status but respects manual toggle afterward. Elapsed-time deltas (`+12s`, `+1m 4s`) from consecutive event timestamps.

**Backend contract request (out of scope, for backend agent):** Backend should emit `progress` phases: `vision_completed`, `provider_query_dispatched`, `synthesis_started` (with hypothesis/query/outcome detail per FR-040's binding limits: owner-scoped stream only, no images/logs, bounded length).

**Test coverage:** +9 new tests in `DeepAnalysisActivityTimeline.test.ts`, updated `DeepAnalysisProgressTimeline.test.ts` for new DOM shape. Full suite 131 files / 830 tests passing (was 821).

---

## Active Decisions

### Decision: Feature 344 — Preserve provider claims across the analysis stream (B1 re-remediation)

**Date:** 2026-08-15
**Agent:** Aurelia (remediation; original implementer and Cassius locked out under Strict Lockout §18.2)
**Status:** Proposed — READY FOR RE-REVIEW
**Supersedes wording of:** commit `896955e` B1 paragraph (see Correction below)

## Context

The prior B1 remediation (`896955e`) made the Go pipeline runner build the rich
`deepProposalDocument` by resolving `proposed_fields.evidence_refs` against the
`claims` carried on internal `provider_result` frames. But production Python
(`run_deep_identification_stream`) emitted only `{type, provider, status}` for
`provider_result` — it never serialized the application-owned `ProviderEvidence`
claims (contract §3/§4). Result: in production `providerClaims` was always empty,
every proposal's evidence/citation arrays were dropped, violating the internal
contract and the MVP citation requirement (SC-006). The existing Go integration
tests hand-injected a `providerClaims` map into `buildDeepProposalDocumentJSON`,
so they passed while production was broken (false assurance).

## Decision

1. **Python emits the full ProviderEvidence** on each `provider_result` internal
   frame via Pydantic serialization (`result.model_dump(mode="json")`), never an
   ad-hoc dict — field names/types/nullability match the Go mirror exactly. All
   fields are length-bounded by the model; no raw provider payload, user notes,
   or image data enter the frame. `_emit`'s sanitizer still redacts token-shaped
   strings without stripping valid claims/citations.
   (`src/agent/app/teams/deep_identification/graph.py`)

2. **Persistence/privacy split (§5 of the task).** The internal Go↔Python
   `provider_result` frame carries full claims (contract §4); the runner consumes
   them **in-memory** to build the confirm-gated proposal, but persists only the
   bounded public payload `{provider, status, confidence, claimCount, errorKind?,
   linkOut?}` (contracts/sse-events.md §2) into the user-visible, replayable event
   log. Full claims/citations therefore never leak into the owner-facing event
   stream (FR-036) and the log is not bloated with per-claim evidence.
   (`src/api/services/deep_identification_pipeline_runner.go`)

3. **Go re-validation preserved.** `buildDeepProposalDocumentJSON` still enforces
   the coin-field allowlist and re-checks each claim's citation host against the
   per-provider allowlist before it enters the proposal; the full streamed claim
   list is retained in emitted order so synthesis `claim_index` values stay
   index-aligned (no reindexing). `claimCount` in the public payload counts only
   host-allowlisted claims, so a hostile frame cannot inflate it or surface an
   arbitrary URL.

4. **Real cross-boundary regression replaces false assurance.**
   - Python: `test_provider_result_frame_carries_complete_claims` asserts the real
     stream emits full, typed claims (field/value/confidence/citation/excerpt),
     index-aligned with the terminal synthesis' evidence_refs (no live network).
   - Go: `TestRunnerAccumulatesStreamedClaimsIntoProposal` feeds the exact
     Python-shaped SSE frames through the actual `Run`/stream parser and asserts
     the persisted `ProposalJSON` carries confidence + citation evidence, and that
     the persisted `provider_result` event carries only the bounded public payload.
     Plus off-allowlist-host rejection and malformed-frame-skip tests.
   - Integration: `seedDeepJobWithRunnerProposal` now drives the **actual runner**
     over a fake Python agent (no hand-built `providerClaims`) so the saved-coin
     and new-intake Get/PATCH/Apply regressions consume genuine runner output.

5. **`DeepIdentificationProviderRun.ClaimsJSON` (task §7).** The entity is
   provisioned/migrated but intentionally **not written** in the MVP; per-provider
   outcomes live in the append-only event log and the proposal. Documented as a
   deliberate deferral in `data-model.md §4` (not silent drift) — raw claims are
   deliberately not duplicated into a second user-visible store.

## Correction (task §11)

Commit `896955e`'s B1 wording — "Evidence/citations resolved Go-side from
provider_result frames; field + citation-host allowlists enforced at translation
and apply" — was **aspirational, not operative**, because production Python never
emitted claims, so nothing was resolved Go-side at runtime. The accurate record:
citation-host allowlisting is enforced at **proposal-build (translation) time**
inside `buildDeepProposalDocumentJSON`; **apply** enforces only the coin/draft
**field** allowlist (`applyToCoin`/`applyToDraft`). This fix makes the "resolved
from provider_result frames" behavior actually true end-to-end.

## Validation

- Go: gofmt (CRLF-only), `go build ./...`, `go vet ./...`, full `go test ./...`,
  OpenAPI drift, architecture test — all pass.
- Python: `ruff check`, full `pytest` — pass.
- Vue: `vue-tsc --build`, `vite build`, full Vitest, no-emoji/design-token
  regression — pass (no Vue changes in this fix).
- Manual fixture-backed end-to-end: real Python-shaped claim frame → Go runner →
  stored rich proposal evidence/citation → PATCH/Apply, no auto-write, no live
  provider calls.


### Decision: Feature 342 — Measured Numista Text-Query Tuning (T001–T025)

**Date:** 2026-08-11  
**Agent:** Sabinus (implementation), Brutus (QA review), Cassius (backend contract), Maximus (follow-up)  
**Status:** APPROVED for Beta (T001–T025 valid; T026 pending Maximus review)

## Context

Feature 342 delivers canonical Go-based Numista text-query construction with measured improvements over divergent TypeScript assembly. Brutus identified three blocking issues in initial review: enrichment attribution loss, `generationVersion` validation gap, and incomplete test coverage for `reverseType` and alias cases. Sabinus addressed all three with bounded focused fixes: enrichment faithfully preserves source/attempt/query through successful/partial/failed detail fetch; `generationVersion` is now required for generated and user-edited requests; exact SMN/SMNT aliasing and reversed-type builder logic are exercised in expanded deterministic replay. Independent revision clears focused QA.

## Decision

Sabinus delivers:

- Pure injected Go `NumistaQueryBuilder` (`numista-query-v2`) in `src/api/services/numista_query.go` with exact `SMN`/`SMNT` → `Nicomedia` alias allowlist (unapproved mintmarks omitted)
- `POST /api/numista/query-proposal` authenticated, bounded, strict-decoded handler with no provider or telemetry calls
- Generated-only relaxed fallback (one distinct query after empty server-verified primary only; manual/edited/error/NGC paths remain one-query)
- Frontend query attribution preserved through enrichment: `NumistaLookupPanel` submits trimmed `effectiveQuery`, backend returns verified source/attempt/query metadata, Vue panel displays explicit relaxed-query disclosure
- `generationVersion` validation enforced for generated and user-edited request sources
- 12-case sanitized deterministic replay in `src/api/services/testdata/numista/query_v2_comparison.json`
- Live-evidence sample (six cases, sanitized) in `specs/342-numista-text-query-tuning/live-evidence.md` measures improvement from 0/6 to 3/6 top-three inclusion (+50 percentage points)
- 24/24 deterministic scorer benchmark maintained
- All T001–T025 acceptance and implementation tests pass; T026 (Maximus final review) remains pending

## Validation

- Focused Feature 342 Go and Vitest tests ✓
- 12-case deterministic replay and enrichment attribution matrix ✓
- 24/24 known-coin scorer benchmark ✓
- `go build ./...`, `go vet ./...`, `go test ./...` ✓
- `npm run test` (113 files, 702 tests) ✓
- `vue-tsc --build` ✓
- `npm run build` ✓
- OpenAPI regeneration (byte-stable, no route drift) ✓
- Gitleaks history/worktree scan ✓
- `git diff --check` ✓
- No live Numista access, credentials, or E2E browser tests

## Alignment

- **Principle III (Explicit typing)**: All request/response contracts typed; `generationVersion` required for generated/edited; source/attempt enums safe
- **Principle IV (Simple proportional change)**: Three focused fixes addressing root causes; no unrelated refactoring
- **Principle V (Security/Privacy)**: No credentials/images/raw prose in tests or payloads; sanitized evidence only
- **Principle VII (Proven tests)**: 12-case deterministic replay; test-first per protocol; no live provider access
- **Principle X (All gates pass)**: Full Go/frontend/OpenAPI suite clean; no pre-existing regressions
- **§17 Quality Gate**: All lint/build/test gates pass; code review and approval documented; T001–T025 valid
- **§21 Definition of Done**: Acceptance criteria met; tests pass; measurement documented; T026 pending

## Pending

- T026: Maximus reviews fixture/live comparison, alias allowlist, request ceiling, NGC/no-eager regressions, and explicit absence of image search before release approval.

---

### User Directive (2026-08-11)

**By:** Brian DeNicola  
**What:** Do not pursue Numista image search because of its cost. Improve and empirically test text-query construction instead.  
**Why:** Live Numista testing showed concise expanded terms can find candidates, while exact mintmarks and catalog references may eliminate all results.

---

### User Directive (2026-08-12)

**By:** Brian DeNicola  
**What:** Finish the current Numista query-tuning work, but avoid overengineering future changes. Default to the smallest measured query transformation and focused tests; add APIs or generalized infrastructure only when evidence requires them.  
**Why:** User wants proportional implementation scope after Feature 342 expanded beyond the apparent size of the original query adjustment.

---

## Active Decisions

### Decision: Feature 341 Phase 7 — Progressive Numista Enrichment (T064–T072)

**Date:** 2026-08-11  
**Agent:** Domitian (implementation), Brutus (QA review)  
**Status:** APPROVED

## Context
Feature 341 Phase 7 delivers progressive detail enrichment for Numista lookup: after broad candidates render, the backend fetches details for a bounded deterministic subset (default 5), reranks them, and streams the enriched response. Frontend progressively updates candidate cards without changing explicit selection.

Brutus identified two P1 blockers during initial QA:

1. **Deterministic ranking contract mismatch**: `rankNumistaEnrichmentTargets` was comparing `ProviderPosition` before numeric ID, causing different candidate subsets to receive detail requests than the binding application-owned deterministic order (data-model.md §117-119).

2. **Effective-query contract not preserved**: Enrichment request returned raw `request.Query` instead of trimmed query; frontend submitted trimmed `effectiveQuery` to endpoint. This divergence broke first enrichment / retry identity contract (data-model.md §46-50, 142).

## Decision
Domitian delivered an independent revision that:

- Refactors `rankNumistaEnrichmentTargets` in `numista_lookup_service.go` to delegate deterministic reranking to a new shared `numistaCandidateRanksBefore` function in `numista_scoring.go`
- Enforces binding order: score > exact ID > completeness > normalized title > numeric ID > provider position (matching spec exactly)
- Adds `TrimSpace` to `NumistaEnrichmentRequest.Validate()` to trim surrounding whitespace while direct FR-006 lookup preserves exact raw query
- Frontend submits `effectiveQuery.trim()` to enrichment endpoint
- All T064–T067 acceptance tests (provider detail, enrichment service, authenticated handler, progressive component) pass deterministic fakes with injected transports and clocks
- All T068–T072 implementation tests (backend caching, concurrent detail fetches, handler auth guards, frontend progressive state display) pass

## Validation
- `go build ./...` ✓
- `go vet ./...` ✓
- `go test ./... -count=1` ✓ (full suite including enrichment unit/integration)
- `npm run test` ✓ (112 test files, 689 tests, 80% coverage)
- `vue-tsc --build` ✓ (strict type-check)
- `npm run build` ✓ (production build)
- OpenAPI regeneration ✓ (byte-stable, no route drift)
- `git diff --check` ✓
- No live Numista access, browser E2E, or provider credentials in tests

## Alignment
- **Principle III (Explicit typing)**: All request/response contracts typed; binding order documented and enforced in both frontend and backend
- **Principle IV (Simple proportional change)**: Two focused fixes addressing root causes; no unrelated refactoring
- **Principle VII (Proven tests)**: Acceptance and implementation tests written test-first per protocol; deterministic fakes verify behavior without live provider access
- **Principle X (All gates pass)**: Full Go/frontend/OpenAPI suite clean; no pre-existing regressions
- **§17 Quality Gate**: All lint/build/test gates pass; code review and approval documented
- **§21 Definition of Done**: Acceptance criteria met; tests pass; no unrelated changes; commit message cites spec sections and principles

---

### Decision: Sets Type Refinement + Goal Completion Formula

**Date:** 2026-07-26  
**Agent:** Cassius  
**Status:** IMPLEMENTED

## Context
Set semantics were refined: legacy `defined` is removed, `open` is renamed to `standard`, and Goal completion no longer uses target matching.

## Decision
- Normalize legacy set type values in DB migration logic:
  - `defined` -> `goal`
  - `open` -> `standard`
  - legacy `dynamic` -> `tracker` with `creation_mode='dynamic'`
- Add `creation_mode` on `coin_sets` (`manual` default, `dynamic` allowed only for `tracker` sets).
- Update Goal completion to: `collection_items / (collection_items + wishlist_items)` using set memberships + `coins.is_wishlist`.
- Keep tag-to-set migration idempotent and additive so newly tagged coins join existing migrated sets.

## Validation
- `go test ./repository -run "TestSetRepository_"`
- `go test ./services -run "TestSetService_CreateSet"`
- `go test ./database -run "TestMigrateCoinSetTypes_NormalizesLegacyValues"`
- `go test ./handlers -run "TestSetHandler_ReorderCoins"`
- `go test ./testutil`

---

### Decision: Frontend set-type normalization during Standard/Goal migration

**Date:** 2026-07-26  
**Agent:** Aurelia  
**Status:** IMPLEMENTED

## Context
Set APIs are moving from legacy `open`/`defined` to `standard`/`goal`, with Tracker and Dynamic Tracker creation mode added. During rollout, frontend may receive mixed legacy/new set type values.

## Decision
Frontend now writes only `standard`, `goal`, `smart`, and `tracker`, while treating `open` and `defined` as legacy read aliases through a shared `normalizeCoinSetType()` helper in `src/web/src/types/index.ts`.

Membership and workflow gates branch on normalized values:
- Tracker and Smart sets are non-manual membership in Set Detail and Coin Tags surfaces.
- Completion panel loads for normalized `goal` and `tracker`.
- Collection set filter includes normalized `standard` sets.

## Rationale
This keeps UI behavior stable during mixed-contract deployments, prevents accidental legacy writes, and centralizes compatibility logic to avoid drift between components.

---

### Decision: Tracker set creation mode contract alignment

**Date:** 2026-07-26  
**Agent:** Maximus  
**Status:** IMPLEMENTED

## Context
`SetCreationWizard` emitted `trackerCreationMode`, but backend set creation reads `creationMode`. Tracker sets created through the dynamic flow therefore persisted with backend default `manual` mode.

## Decision
Align frontend payload contract to backend by emitting `creationMode` in `CreateCoinSetRequest` and wizard submit payload. Keep prompt behavior unchanged, and add a focused frontend regression test asserting dynamic tracker submission emits `creationMode: "dynamic"` and not `trackerCreationMode`.

## Validation
- `npm.cmd run test -- src/components/sets/__tests__/SetCreationWizard.test.ts`
- `npm.cmd run type-check`

## Alignment
- Principle III: explicit typed frontend/backend contract field match
- Principle IV: smallest complete fix with targeted regression coverage

---

### Decision: Coin Grading as AI Analysis Sub-Action

**Date:** 2026-07-02
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context
Coin grading is an AI-assisted per-coin workflow that returns an estimate report without mutating the saved coin grade.

## Decision
The user-facing grading entry point lives inside `CoinAIAnalysis.vue` beside obverse/reverse analysis instead of chat or a new store. It uses the existing async AI job polling pattern, requires both obverse and reverse images before start, displays `gradingReport` in-place, and includes permanent limitation copy that the estimate is not professional certification and does not update `Coin.grade` automatically.

## Alignment
- Principle III: typed API contract and frontend type-check passed.
- Principle IV: reused existing AI job polling surface without a new store.
- Principle VI: token-based in-panel UI with no emoji and existing button hierarchy.

---

### Decision: Coin Grading ships as a dedicated coin-detail AI job

**Date:** 2026-07-01
**Agent:** Maximus
**Status:** IMPLEMENTED
**Issue:** #374

## Context

Coin grading exists in `src/agent/app/teams/coin_grading.py` and the supervisor router advertises a `grading` intent, but the streaming chat path does not carry images. Routing grading requests through chat currently lands on a passthrough/dead-end unless a caller injects a grading node.

## Decision

Implement Coin Grading as a dedicated authenticated coin-detail action backed by the existing AI job system, not as image-capable chat attachments for this slice.

Recommended surface:
- Add a `Grade Coin` action on the coin detail AI Analysis page, using existing stored obverse/reverse/detail images.
- Queue `AIJobTypeCoinGrading` through Go, pass owner-scoped coin context and image bytes to Python `/api/grade`, then store the report in the job result.
- Do not write the estimated grade into `Coin.Grade` automatically. Offer an explicit `Apply to Grade` follow-up only after the user reviews the confidence/limitations.
- Remove supervisor grading advertising/dead-end behavior unless/until chat supports image attachments.

## Rationale

This keeps the feature aligned with Constitution Principle I/II: Vue calls Go, Go owns auth/image access/job persistence, Python remains stateless and receives only bounded per-request context. Reusing AI jobs matches the existing analysis/value UX, avoids introducing image-capable chat contracts prematurely, and provides clear regression points for no-image, success, and model-failure paths.

## Validation expected

- Agent: request model and `/api/grade` route tests for no-image, success, and graph failure.
- Go: service/job tests for no images, successful grading result persistence, and agent failure status.
- Frontend: component tests for disabled/no-image state, queued/success polling, failure display, and confidence/limitations copy.
- Contract: regenerate OpenAPI and run route drift test for any new Go route.

---

### Decision: Coin Grading Revision

**Date:** 2026-07-02
**Agent:** Maximus
**Status:** IMPLEMENTED

Coin grading remains exposed through the dedicated `/api/grade` agent endpoint and Go `POST /coins/:id/grade` AI job workflow only. Collection chat no longer advertises or routes to a `grading` supervisor capability until that path has a real wired implementation instead of a passthrough dead-end.

---

### Decision: Coin Grading Backend Contract

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED
**Issue:** #374

## Decision

Issue #374 uses a dedicated async AI job endpoint, `POST /api/coins/:id/grade`, backed by Python `POST /api/grade`.

## Contract

- Go endpoint returns `202` with the normal `services.AIJobSubmissionResponse`.
- New job type is `coin_grading`.
- Python grading response is `{ "report": string }`.
- Completed Go job result is JSON `{ "gradingReport": string }`.
- The workflow requires at least one owner-scoped coin image; image-less coins return `400 {"error":"No image available for grading"}`.
- The saved `Coin.Grade` field is not updated automatically.

## Rationale

This preserves the existing AI job polling/notification contract, avoids chat attachment coupling, and keeps grade updates explicitly user-controlled.

---

### Decision: Python Agent Dynamic Set Builder Workflow (Phase 2)

**Date:** 2026-07-26
**Agent:** Cassius
**Status:** IMPLEMENTED
**Issue:** specs/011-dynamic-set-builder-correction-plan.md (Phase 2)

## Context

Spec 011 (Dynamic Set Builder) requires a Python multi-agent group-chat/magentic workflow that turns a free-text prompt into a structured Set Proposal for human review, without ever creating a set itself.

## Decision

- Added `POST /api/set-builder/run` to the existing router in `src/agent/app/routes.py`, covered by existing `InternalServiceAuthMiddleware`.
- Added `SetBuilderRequest` (Pydantic, `extra="forbid"`) with bounded `prompt` (500 chars), `max_turns` (1-8, default 4), `max_slots` (1-300, default 200), `enable_external_lookup` (default `True`), and optional `feedback`.
- Added response DTOs: `SetBuilderResponse` (status, proposal, clarification_question, failure_reason, transcript_summary, turns_used). Status is one of `completed`, `clarification_needed`, `rejected`, `failed`, `limit_reached` — there is no "set created" outcome.
- Added LangGraph `StateGraph` with five nodes (intent_analyst, roster_researcher, collection_matcher, validator, finalize). Each LLM-calling node increments turn counter; conditional edges route to finalize once status is set or turns exhausted.
- Roster research uses `get_search_model`/`get_chat_model` from `app/llm/provider.py` per existing per-request LLM config pattern.
- Top-level `run_set_builder_workflow(request)` is what routes.py calls and what tests monkeypatch.

## Validation

- `python -m pytest tests/test_set_builder.py -v` (12 passed)
- `python -m pytest -q` — full agent suite, 182 passed
- `python -m ruff check app/models/requests.py app/models/responses.py app/routes.py app/teams/set_builder.py tests/test_set_builder.py` — clean

---

### Decision: Phase 3 Backend Submission Slice Complete

**Date:** 2026-07-26
**Agent:** Cassius
**Status:** IMPLEMENTED
**Issue:** specs/011-dynamic-set-builder-correction-plan.md (Phase 3)

## Context

Task was to finish and verify `POST /set-builder/runs` as a real, safe submission endpoint: queue a run only, create no set or proposal.

## Decision

- `POST /set-builder/runs` is wired under the `protected` (JWT-required) route group in `main.go`, calling `SetBuilderHandler.CreateRun` → `SetBuilderService.CreateRun`.
- Only inserts a `SetBuilderRun` row with status `queued` and owner-scoped `UserID` from JWT context. No `SetProposal`, `ProposalSlot`, `CoinSet`, or `CoinSetTarget` is touched.
- Regenerated OpenAPI because the new route/request/response shapes were undocumented and drift detection was failing.

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` (full suite) — all packages pass, including `TestRegisteredAPIRoutesAreDocumentedInOpenAPI`.
- `go test ./handlers -run TestSetBuilder -v`, `go test ./services -run TestSetBuilder -v`, `go test ./repository -run TestSetBuilder -v` — all pass.

## Alignment

- Principle I: handler stays thin; all persistence lives in `SetBuilderService` / `SetBuilderRepository`.
- Principle VII: Swagger annotations present; OpenAPI regenerated.
- Principle XI: owner scoping enforced via JWT auth middleware under `protected` group.

---

### Decision: User custom mint/location CRUD is already fully implemented

**Date:** 2026-07-27
**Agent:** Cassius
**Status:** Informational (test coverage added)

## Context

Task requested backend support for user-scoped custom mint/location CRUD, but discovered the entire feature already exists in the codebase.

## Finding

- Model: `MintLocation.UserID *uint` (nil = global, non-nil = private)
- Repository: `CreatePrivate`, `FindOwnedByID`, `ListVisibleTo`, `ExistsVisibleTo`
- Service: `CreatePrivate`, `UpdatePrivate`, `DeletePrivate`, `List`
- Handler: `CreatePrivate`, `UpdatePrivate`, `DeletePrivate`, `List` with Swagger
- Routes: Protected routes under `main.go:308–312`
- AutoMigrate: `MintLocation` included

## Added This Session

Service-layer test coverage (8 new tests):
- `TestMintLocationService_CreatePrivateSetsUserID`
- `TestMintLocationService_CreatePrivateDuplicateRejectsNameCollidingWithGlobal`
- `TestMintLocationService_CreatePrivateTwoUsersSameNameAllowed`
- `TestMintLocationService_UpdatePrivateOwnerScopingRejectsOtherUser`
- `TestMintLocationService_UpdatePrivateOwnerCanRenameOwnMint`
- `TestMintLocationService_UpdatePrivateCannotMutateGlobalMint`
- `TestMintLocationService_DeletePrivateOwnerScopingRejectsOtherUser`
- `TestMintLocationService_DeletePrivateCannotDeleteGlobalMint`
- `TestMintLocationService_ListReturnsGlobalAndUsersOwn`

## Key Design Notes

- Owner-scoping enforced at repository layer (`FindOwnedByID` uses `WHERE id = ? AND user_id = ?`).
- Private mint name cannot shadow a global entry visible to the user.
- Two users may hold private mints with the same name — separate visibility scopes.
- In-use guard applies to both global and private delete paths.

---

### Decision: Agentic Set Submit UX Uses Set Builder Run, Not Local Set Creation

**Date:** 2026-07-26
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

Phase 3 of Dynamic Set Builder correction plan required unblocking the Agentic option in `SetCreationWizard.vue`. Coordinator had already added `createSetBuilderRun` to API client.

## Decision

- `SetCreationWizard.vue` enables normal submit button for `setType === 'agentic'` and emits standard `submit` payload (including `agenticPrompt`) without attempting local set creation.
- `SetsPage.vue`'s `createSet` branches on `value.setType === 'agentic'`: calls `createSetBuilderRun({ prompt: value.agenticPrompt || value.name })`, closes modal, resets form, shows confirmation dialog.
- Both agentic and standard/goal/smart/CSV error paths use `useDialog().showAlert(...)` instead of `window.alert`.

## Validation

- `npm run test -- src/components/sets/__tests__/SetCreationWizard.test.ts src/pages/__tests__/SetsPage.test.ts src/pages/__tests__/SetDetailPage.test.ts` — 9/9 passed
- `npm run type-check` — clean

## Alignment

- Principle III: strict typing preserved
- Principle IV: minimal change scoped to wizard button/copy and one branch in SetsPage
- Constitution "no raw `window.alert`" convention: replaced with `useDialog`

---

### Decision: User Custom Mint Locations UI — Global/User Separation

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

Cassius added user-scoped `/mint-locations` protected routes. `getMintLocations()` returns mixed list of global (userId=null) and user-owned (userId=number) locations. UI needed to present only user-owned entries as editable in Settings.

## Decision

`SettingsDataSection.vue` calls `getMintLocations()` and filters result to `m.userId != null` via computed property (`userMintLocations`). Global/admin locations excluded from list, badge count, and form context. Create/update/delete use user-scoped wrappers, never admin paths.

## Validation

- 10 Vitest tests including explicit test: "renders only user-scoped locations, not global"
- `npm.cmd run type-check` passes
- Commit `84e01f1` on beta branch

---

### Decision: CategoryEraConfirmModal z-index fix

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

During Add Coin agentic mode, `CategoryEraConfirmModal` opened invisibly because `CoinSearchChat` sidebar (`z-[1400]`) covered it — modal used `z-[300]`.

## Root Cause

Modal uses `<Teleport to="body">`, placing overlay as sibling of `CoinSearchChat` root div. Since 300 < 1400, sidebar always painted on top.

## Decision

Raise `CategoryEraConfirmModal`'s overlay from `z-[300]` to `z-[1600]`.

**Z-index scale after fix:**
| Element | z-index |
|---|---|
| CoinSearchChat sidebar | 1400 |
| Note-draft modal | 1500 |
| CategoryEraConfirmModal | 1600 |

## Test Pattern

For teleported Vue components: stub `Teleport: true` in `global.stubs` so content renders inline — `wrapper.find()` and `wrapper.trigger()` work normally.

## Files Changed

- `src/web/src/components/chat/CategoryEraConfirmModal.vue` — z-[300] → z-[1600]
- `src/web/src/components/__tests__/CategoryEraConfirmModal.test.ts` — new (4 tests including z-index regression guard)

## Alignment

- Principle IV: smallest complete fix, targeted regression coverage
- §17 Quality Gate: type-check passed, targeted tests pass

---

### Decision: Mint Map List + Grid Layout

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

Mint map page showed only Leaflet map with coins revealed via floating drawer on marker click. No summary overview of which mints were represented without mousing over each marker.

## Decision

Added `MintListPanel.vue` alongside `MintMapLeaflet.vue` in two-column CSS grid layout:

- **Desktop:** 260px scrollable list on left, map fills remaining width
- **Mobile:** Single-column stack; map first (primary), list below capped at 280px
- List items show: `displayName`, `region` (uppercase label), count badge, coin name preview (first 2, with `+N more` suffix)
- Selecting mint from list emits `select-mint` to page, updates `selectedMintId`, highlights marker, opens drawer
- Detail view: existing `MintCoinDrawer` (fixed-position, `z-[1100]`) continues as selected-mint surface

## Validation

- 9 new `MintListPanel.test.ts` unit tests
- 2 new `MintMapPage.test.ts` integration tests
- All 23 targeted tests pass; `vue-tsc --build` clean

## Alignment

- Principle IV: reused MintCoinDrawer, existing groupCoinsByMint, existing selectedMintId state
- Principle VI: design tokens throughout; dark theme; mobile-responsive
- §17 Quality Gate: targeted tests pass, type-check clean

---

### Decision: Mint Map Layout — Equal-Height Panel and Map Cards

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

After `MintListPanel.vue` was added, stray vertical scrollbar appeared in gap between list panel and map. List panel taller than map card, causing mismatched heights on desktop.

Root cause: `.map-layout` used `align-items: start`, so each grid column sized independently. List panel (~692px) exceeded map (~640px). Scrollbar on `.mint-list` leaked into column gap.

## Decision

1. Set `height: min(70vh, 640px)` on `.map-layout`, remove `align-items: start` (default stretch). Both columns fill same row height.
2. Changed `.mint-map-leaflet` from `height: min(70vh, 640px)` to `height: 100%`. Added `@media (max-width: 768px)` override to `height: min(50vh, 380px)` (mobile grid is `height: auto`).
3. Removed `max-height: min(70vh, 640px)` from `.mint-list`. Added `min-height: 0` (critical flex-child shrink property). Grid row height bounds panel; `flex: 1; min-height: 0; overflow-y: auto` produces correctly scrollable list with no overflow artifact.
4. Mobile grid overrides to `grid-template-columns: 1fr; height: auto`. List panel `max-height: 280px` retained on mobile.

## Validation

- All 22 targeted tests pass
- Added layout structure test: verifies `.map-layout` contains both `MintListPanel` and `MintMapLeaflet`
- `vue-tsc --noEmit` clean

## Alignment

- Principle IV: minimal surgical change; 3 CSS properties across 3 files
- §17 Quality Gate: targeted tests pass, type-check clean

---

### Decision: TrayCoin Adapters Must Forward All MuseumTrayWell Caption Fields

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

`MuseumTrayWell.displayCaption` reads three fields from `TrayCoin`: `purchaseDate`, `placeholder`, `wishlistPlaceholder`. Any adapter that converts a `Coin` into `TrayCoin` must explicitly map all three or captions fall back to `'TBD'`.

`ImperialFigureWellGrid.toTrayCoin()` was missing `purchaseDate`, causing every owned emperor-tracker well to display `TBD` even when coin had valid purchase date.

## Decision

- Any `Coin → TrayCoin` adapter must include `purchaseDate: coin.purchaseDate`
- Unowned figure/goal placeholders with no purchase date correctly show `'TBD'`
- Collection > Tray continues to pass `showCaptions: false` — unaffected

## Validation

- `npm.cmd run test -- src/components/__tests__/MuseumTrayWell.test.ts src/components/__tests__/MuseumTray.test.ts src/components/emperor-tracker/__tests__/ImperialFigureWellGrid.test.ts src/pages/__tests__/TrayViewPage.test.ts` — all 37 tests pass
- `npm.cmd run type-check` — clean

## Alignment

- Principle IV: smallest complete fix
- Principle IX: UI correctness — real dates shown where real dates exist

---

### Decision: Unknown Mint moved into the side panel as a first-class list entry

**Date:** 2026-07-27
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

`UnattributedMintBucket` lived below map row as collapsible card. Consumed vertical space below map/list pair; required separate expand interaction; second-class compared to matched-mint list.

## Decision

All unattributable coins (both `aggregation.unknown` and `aggregation.unmatched`) collapsed into single virtual `MintGroup` using sentinel `UNKNOWN_MINT_ID = 0`. This group appended to `panelGroups` and passed to `MintListPanel`.

- `MintMapLeaflet` continues to receive only `aggregation.matched` (no phantom marker)
- `selectedGroup` checks `selectedMintId === UNKNOWN_MINT_ID`, returns virtual group, opens `MintCoinDrawer`
- `UnattributedMintBucket` no longer imported or rendered
- Map-layout height increased from `min(70vh, 640px)` to `min(80vh, 800px)` since below-map section removed
- Mobile stacked layout preserved

## Validation

- `npm.cmd run test -- src/pages/__tests__/MintMapPage.test.ts src/components/map/__tests__/MintListPanel.test.ts src/utils/__tests__/mintMap.test.ts` — 29/29 pass
- `npm.cmd run type-check` — clean

## Alignment

- Principle IV: simplest complete change; reuses existing panel + drawer
- Principle VI: consistent tap-to-select interaction
- §17: targeted test coverage for all acceptance criteria

---

### Decision: Phase 2 Python Set-Builder QA Coverage

**Date:** 2026-07-26
**Agent:** Brutus
**Status:** IMPLEMENTED
**Issue:** specs/011-dynamic-set-builder-correction-plan.md (Phase 2)

## Context

Cassius landed the Phase 2 Python set-builder workflow while Brutus was drafting QA coverage in the same session. Test coverage was added against the landed contract.

## Confirmed Contract

1. Route: `POST /api/set-builder/run`, guarded by existing `X-Internal-Service-Token` middleware
2. Route body: `await run_set_builder_workflow(request)` — route does not build/invoke graph itself
3. Response contract:
   - `SetBuilderResponse.status` is one of `completed`, `clarification_needed`, `rejected`, `failed`, `limit_reached`
   - Only `status == "completed"` populates `proposal`; ambiguous/unbounded prompts return clarification/failed/limit with `proposal = None`
   - `SetBuilderSlot` carries `criteria`, `group`, `sort_order`, `verification_status`

## Test Additions

`src/agent/tests/test_set_builder_api.py` (24 tests, all passing):
- Request validation: empty/missing/oversized prompt, oversized feedback, bounds checks
- Response schema: slot defaults, verification status, proposal presence/absence by status
- Route: requires internal token (401), rejects invalid body (422), returns structured response, handles workflow failure as structured failure (not 500)

## Alignment

- Constitution §17/§21: targeted regression coverage without editing implementation files
- Principle IV: test-only change validated against actual landed contract

---

### Decision: Agentic Set Builder Correction — QA Checklist & Phase 3 Submit Coverage

**Date:** 2026-07-26
**Agent:** Brutus
**Status:** IN PROGRESS (tracking, not implementation-blocking)

## Context

`specs/011-dynamic-set-builder-correction-plan.md` defines Phases 0-6. Cassius/Aurelia have active uncommitted work. Brutus added only a new, independent test file.

## What Exists Today

- Models: `SetBuilderRun`, `SetProposal`, `ProposalSlot`
- Repository: full lifecycle (`CreateRun`, `StartRun`, `CompleteRun`, `FailRun`, `CreateProposalWithSlots`, `ListProposals`, `GetProposalForUser`, `MarkApproved`, `MarkRejected`, `MarkExpired`)
- Service: `CreateRun`, `FindPendingProposalByPrompt`, `CreateProposalFromWorkflow`, `ListProposals`, `GetProposal`, `RejectProposal`
- Handler: only `POST /set-builder/runs` (`CreateRun`) implemented and wired
- Not yet implemented: GET list/get routes, PUT edit, POST approve (transactional set creation), POST reject, POST regenerate handlers

## Added This Session

`src/api/handlers/set_builder_handler_test.go` — proves currently live Phase 0/Phase 3 contract at HTTP layer:
- `TestSetBuilderHandlerCreateRunRequiresAuth` — 401 without bearer token
- `TestSetBuilderHandlerCreateRunRejectsBlankPrompt` — 400, zero persisted runs
- `TestSetBuilderHandlerCreateRunPersistsQueuedRunWithoutCreatingSet` — 202, run persisted with status=queued, trimmed prompt, correct userId, **zero** `CoinSet`/`CoinSetTarget`/`SetProposal` rows
- `TestSetBuilderHandlerCreateRunIsScopedPerUser` — two users get independent runs

All new tests pass: `go test ./handlers -run TestSetBuilderHandler -v` (4/4 PASS).

## Full Phase Test Checklist (summary)

### Phase 0 — Stop misleading behavior
- [x] Submitting prompt does not create set immediately
- [ ] Python agent unavailable surfaces actionable error
- [ ] Legacy deterministic Agentic rows remain readable

### Phase 1 — Backend data model
- [x] Proposals persist without creating sets
- [x] `ProposalSlot` verification defaults

### Phase 2 — Python agent workflow
- [ ] Tests for set-builder endpoint
- [ ] Intent analyst rejects non-numismatic
- [ ] Roster researcher structured-only output
- [ ] Orchestrator enforces max turns/budget
- [ ] Backend/provider logs show real Python calls

### Phase 3 — Backend agent proxy and proposal service
- [x] Prompt submission creates run, no set
- [x] Proposal review fetch is user-scoped
- [ ] Dedup identical pending prompts end-to-end
- [ ] GET proposals handlers
- [ ] PUT edit handler
- [ ] POST approve (transactional)
- [ ] POST reject handler
- [ ] POST regenerate handler
- [ ] Expired proposal cannot be approved

### Phase 4 — Human review UI
- [ ] Wizard submits builder run instead of creating set
- [ ] Notification deep-links to proposal review
- [ ] Review screen renders roster
- [ ] Approve/Reject/Regenerate actions

### Phase 5 — Roman-Emperor-style matching
- [ ] Matching auto-fills owned coins into slots
- [ ] Matching recalculates on coin lifecycle
- [ ] Deterministic tie-break
- [ ] Tray renders every slot

### Phase 6 — Integration tests
Items 1 and 3 covered; items 2, 4-10 and all frontend/integration open.

## Alignment

- Principle IX / §17: regression coverage without modifying implementation files
- Constitution §18.2 Strict Lockout: no BLOCK; Phase 3 behavior matches documented acceptance criterion

---

### Decision: Auction Sync Auto-Creates In-App Calendar Events

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED

## Context

`/auctions/sync` upserts NumisBids and CNG watchlist lots, but newly tracked active lots were not linked to in-app calendar entries despite `AuctionLot.EventID` and `AuctionEvent` already existing.

## Decision

Add repository-level `UpsertWithCalendarEvent` for sync paths only. It creates an `AuctionEvent` and links `AuctionLot.EventID` in the same transaction only when the source-aware upsert inserts a new lot with status `watching` or `bidding`. Existing lots update without new events, and `passed`/`won`/`lost` lots do not auto-create events.

## Validation

- `go test -v .\repository -run "TestAuctionLotRepository_Upsert"`
- `go test -v .\handlers -run "TestAuctionLotHandlerUpdateStatus"`
- `go test ./...`

## Alignment

- Principle I: GORM and multi-step create/link live in repository transaction.
- Principle IV: Small, source-aware extension of existing upsert/sync workflow.
- §17: Targeted regression coverage plus full Go API tests pass.

---

### Decision: Cassius Scraper Transport Helper

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED

## Context

Issue #373 starts with auditing shared scraper behavior across NumisBids and CNG. Both providers need authenticated HTTP session mechanics, but their login payloads, auth verification rules, URL safety, pagination, parsing, and provider-specific sentinel errors must remain provider-owned.

## Decision

Added a package-private shared helper in `src/api/services/scraper_transport.go` for cookie-jar client creation, request/header construction, form POST construction, request execution, status checks, response body read/close behavior, and request error wrapping. The first segment intentionally does not refactor `NumisBidsService` or `CNGAuctionService` to use it yet.

## Validation

- `go test -v ./services -run "Test(NewScraper|DoScraper|ReadScraper|CNGAuctionService|CanonicalCNG|ParseWatchlist|WatchlistDiagnostics|FetchWatchlist|Login|VerifyAuthentication)"`

## Alignment

- Principle I: helper stays in service layer and remains HTTP-provider agnostic.
- Principle IV: simple, focused extraction without broad provider refactor.
- §21: new helper methods have focused regression coverage.

---

### Decision: User-Initiated Camera Start

**Date:** 2026-06-30
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

iOS/PWA users should not see a camera permission prompt just by opening Add Coin agentic mode or Identify Coin. The app still needs to preserve the guided live camera experience once the user intentionally starts capture.

## Decision

`src/web/src/pages/AddCoinPage.vue` and `src/web/src/pages/CoinLookupPage.vue` no longer start camera streams from page mount, agentic mode entry, or Identify Coin retake. Both pages show a clear "Start Camera" placeholder action that calls `startCamera()` only from a user tap. Existing upload-library actions remain available, shutter buttons stay disabled until `cameraReady`, and Add Coin continues stopping active streams when leaving agentic mode.

## Validation

- `npm.cmd run test -- src/pages/__tests__/CoinLookupPage.test.ts src/__tests__/ui-patterns.test.ts`
- `npm.cmd run type-check`

## Alignment

- Principle III: Vue strict type-check passed.
- Principle IV: Simple complete change across both affected camera entry points.
- Principle VI: Preserves existing dark, token-based camera UI and upload fallback.

---

### Decision: Shared Typed HTTP Client for Numista Lookup

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Legacy lookup had two independent Numista integrations: direct passthrough at `GET /api/numista/search` and disconnected photo analysis at `POST /api/coins/lookup`. Both duplicated HTTP/cookie-jar setup, error handling, and status mapping.

## Decision

Implement single injected `NumistaClient` interface with typed request/response DTOs, private provider-specific structs, and safe error taxonomy. All Numista HTTP goes through this client.

**HTTP Client Design:**
- `NumistaClient` interface: `SearchBroad(ctx, query) → []Candidate | error`, `EnrichDetail(ctx, id) → *Detail | error`
- `HTTPNumistaClient` implements interface using `net/http` with four/three-second deadlines, context cancellation, and one bounded transient retry
- Provider structs (request/response) are private; application DTOs are public
- Safe error mapping: 400 validation errors, 401 auth, 403 forbidden, 429 quota, 5xx unavailable, timeout, cancellation

**Injection Pattern:**
- Single client instance created in `main.go` during startup
- Injected into `NumistaLookupService`, `NumistaCache`, handlers
- Live provider replaced by interfaces in tests; `httptest` harness for contract validation

## Validation

- Phase 2 tasks T006–T007: httptest and fake-RoundTripper tests for provider URL/header mapping, retries, deadline handling, all error codes
- `go test ./services -run TestNumistaClient`
- `go build ./...` && `go vet ./...`

## Alignment

- Principle I: handler-thin, service-owned HTTP, interface-based dependency injection
- Principle III: explicit public application DTOs, private provider payloads, Swagger-documented contracts
- Principle V: no secrets leaked to logs, bounded retries, context-safe cancellation
- Principle IX: live provider replaced by interfaces; httptest ensures no drift

---

### Decision: Bounded TTL Caches with In-Flight Request Coalescing

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Numista has a shared quota allowance. Repeated identical queries should reuse cached results. In-flight duplicate requests should coalesce to a single provider call.

## Decision

Implement injectable-clock bounded TTL caches in `NumistaCache` with independent search/detail namespaces:

**Search Cache:**
- Key: SHA-256(normalized query string)
- Value: `[]Candidate` or empty-outcome status
- TTL: configurable per setting (default 24 hours for success, 1 hour for empty)
- Eviction: bounded in-memory map with LRU-style deletion on capacity exceed (default 500 searches)

**Detail Cache:**
- Key: `numista_id` (numeric identifier)
- Value: enriched detail object
- TTL: configurable per setting (default 7 days for success)
- Eviction: bounded, separate namespace (default 5,000 details)

**In-Flight Request Coalescing:**
- Same-key in-flight requests wait on a channel for first completion
- Only first caller hits provider; others receive cached result or error
- Context cancellation propagates safely without partial writes

**TTL Check:**
- `CheckFresh(key, now) → bool` returns true if entry exists and not expired
- `Get(key, now)` returns value if fresh, else `nil`
- Expired entries automatically deleted on `Get` miss

## Validation

- Phase 2 tasks T008–T009: fake-clock tests for TTL, eviction, coalescing, cancellation
- `go test ./services -run TestNumistaCache`

## Alignment

- Principle I: service-owned cache, interface-based injection of injectable clock
- Principle IV: simple deterministic eviction without background worker
- Principle V: no key material in cache; bounded memory to prevent DoS

---

### Decision: Deterministic Versioned Scoring with Weighted Dimensions

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Numista's default ordering is provider-driven and may not rank evidence relevant to the collector's coin. Application must score candidates deterministically so repeated queries produce stable results.

## Decision

Implement `NumistaScorer` (pure service) using `numista-v1` versioned normalization and weighted scoring:

**Scoring Dimensions (versioned numista-v1):**
- **Exact ID Match** (weight: 1.0) — if collector provides NGC/exact Numista ID
- **Inscription Match** (weight: 0.8) — substring match, case-insensitive, non-ASCII normalized (NFKC)
- **Ruler/Issuer Match** (weight: 0.7) — exact after normalization
- **Denomination Match** (weight: 0.6) — normalized abbreviations (AR, AV, etc.)
- **Mint/Location Match** (weight: 0.6) — normalized place name
- **Date Range Match** (weight: 0.5) — candidate date overlaps coin date range (BCE/CE-aware)
- **Material Match** (weight: 0.4) — normalized material (Gold, Silver, Bronze, etc.)
- **Missing Data** (weight: neutral) — absent evidence does not penalize; only present evidence scores

**Relevance Reasons:**
- For each matched dimension, generate a human-readable reason: "Inscription matches 'IMP CAES'", "Date range overlaps 117–138 CE"
- Redact full inscription text in public reasons; show only truncated preview
- Deterministic tie-breaking: by Numista ID ascending if scores are equal

**Validation & Determinism:**
- All tie-breaks based on immutable Numista ID, not provider order
- Results cached, so identical queries always return same rank order
- `numista-v1` version pinned in code; future scoring changes go to `numista-v2` with migration

## Validation

- Phase 2 tasks T010–T011: table-driven scorer tests for every dimension, edge cases (BCE ranges, punctuation, duplicates), stable tie-breaking
- `go test ./services -run TestNumistaScorer`

## Alignment

- Principle I: pure service, no mutable state
- Principle III: explicit scoring version (numista-v1) in code; contract documents reasons
- Principle IV: weights are simple sums, no neural network or hidden logic
- Principle IX: deterministic scoring is fully testable without Numista access

---

### Decision: Transactional Selected-Reference Persistence for Quick Capture Drafts

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Photo lookup and Quick Capture must let collectors select one Numista result without creating a coin. The selected reference must survive draft edits and be copied to the coin during promotion (collection or wishlist).

## Decision

Add `quick_capture_draft_references` table with one-to-zero-or-one relationship to `quick_capture_drafts`:

**Schema:**
```sql
CREATE TABLE quick_capture_draft_references (
  id INTEGER PRIMARY KEY,
  draft_id INTEGER NOT NULL,
  catalog VARCHAR(50) NOT NULL,  -- always 'numista' for now
  catalog_number INTEGER NOT NULL,  -- Numista ID
  canonical_url TEXT,  -- HTTPS numista.com URL
  created_at TIMESTAMP,
  FOREIGN KEY (draft_id) REFERENCES quick_capture_drafts(id) ON DELETE CASCADE,
  UNIQUE(draft_id),
  INDEX(draft_id)
);
```

**Lifecycle:**
1. Photo lookup returns proposed evidence + editable query (no Numista call)
2. Collector edits query and triggers lookup (shared `NumistaLookupService`)
3. Results display with explicit radio selection (not auto-selected)
4. `POST /api/quick-capture-drafts/{id}/reference` creates or updates row, persisting only selected candidate ID
5. Draft edit operations (photo add/remove, field edit) preserve reference unless explicitly cleared
6. `POST /api/quick-capture-drafts/{id}/promote` transaction:
   - Copies selected reference to new `CoinReference` with `linkType='lookup_selected'`, owner-scoped
   - If no reference selected, promotes without one (null `CoinReference`)
   - Idempotent: repeated promotion attempts use same reference, never duplicate

**Constraints:**
- Owner-scoped: can only create/read own references
- One reference per draft: upsert semantics
- Reference is optional: draft can be promoted with or without one
- Additive schema: no existing table modifications

## Validation

- Phase 2 task T036: schema migration tests for additive table, rollback compatibility
- Phase 4 tasks T028–T030: repository tests for create/preserve/replace/remove, promotion transaction
- `go test ./repository -run TestQuickCapture`
- `go test ./database -run TestMigration`

## Alignment

- Principle I: repository owns transaction for draft + reference + coin + lifecycle event in one atomic write
- Principle III: explicit `linkType='lookup_selected'` in CoinReference; typed DTO contract
- Principle IV: single table, minimal schema, reuses existing promotion transaction
- Principle V: owner-scoped at repository layer; no provider keys stored

---

### Decision: Six Explicit Domain Statuses for Lookup Outcomes

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Direct lookup and photo analysis currently return raw provider errors. Collectors cannot distinguish "no results" from "service unavailable" from "quota exhausted". Frontend and backend need explicit, actionable statuses.

## Decision

Define six domain statuses for all lookup outcomes, mapping provider errors and HTTP codes to application-owned enums:

**Domain Statuses:**
1. **success** — provider returned results; candidate list may be empty after filtering
2. **empty** — provider returned HTTP 200 with no candidates; recommend query revision
3. **unconfigured** — Numista API key not set or invalid; admin users see configuration link, others see supportive message
4. **quota_limited** — provider returned HTTP 429 or documented quota exhaustion; include `Retry-After` header if present
5. **timeout** — provider request exceeded deadline (3–4 seconds); query remains editable, retry offered
6. **unavailable** — provider returned HTTP 500–599 or is otherwise unhealthy; transient failure, safe retry guidance

**Provider → Domain Mapping:**
- HTTP 200 → success or empty (based on result count)
- HTTP 400 validation error → success with empty results (invalid query)
- HTTP 401 auth failure → unconfigured
- HTTP 403 forbidden → unconfigured (e.g., plan limit)
- HTTP 429 or documented quota → quota_limited
- `context.DeadlineExceeded` → timeout
- `context.Canceled` → timeout (user cancellation treated same as deadline)
- HTTP 500–599 → unavailable
- Other error → unavailable (safe fallback)

**Role-Aware Guidance:**
- Non-admin users: no raw error text, no API key hints
- Admin users: configuration link for unconfigured state, error summary for unavailable
- All users: query/selection preserved across status changes; retry available for quota_limited and timeout

## Validation

- Phase 5 tasks T046–T048: service tests for all six statuses, HTTP layer tests for guidance boundaries, Vue component tests for state rendering
- `go test ./services -run TestNumistaLookupStatus`
- `npm run test -- ...NumistaLookupPanel.status.test.ts`

## Alignment

- Principle III: explicit enum, no stringly-typed status; Swagger contract documents each state
- Principle V: admin-only error details; no secret/config hints to non-admin users
- Principle VI: non-color guidance (text + icon + action), aria-live announcements
- Principle IX: status state machine is fully testable without Numista access

---

### Decision: Redacted Bounded Telemetry for Numista Lookup Health

**Date:** 2026-08-11
**Agent:** Maximus
**Status:** DECISION (spec/research documented; implementation pending)
**Feature:** specs/341-improve-numista-lookup

## Context

Instance administrators need visibility into Numista health and quota usage without exposing collector search terms, images, or API keys.

## Decision

Implement thread-safe bounded `NumistaTelemetry` ring buffer with 500-operation limit, redaction, and aggregate metrics:

**Telemetry Record (one per lookup):**
- Timestamp, path (lookup or enrich), HTTP status code or domain status
- Cache result: fresh, hit, miss, expired
- Enrichment: count of details requested, count of details succeeded
- Quota: Retry-After value if present
- Correlation digest: first 16 hex chars of SHA-256(normalized query), never full query
- Duration (milliseconds)
- Zero secrets, user text, image data, raw provider error messages

**Aggregate Metrics:**
- Last N operations (N ≤ 500)
- Count by status: success, empty, unconfigured, quota_limited, timeout, unavailable, cached
- Count by cache state: fresh, hit, miss, expired
- p50, p95 latency percentiles (for fresh + cached, separate)
- Quota-limited count and latest Retry-After
- Provider error count (no detail)

**Redaction Rules:**
- Query terms: truncated to first 50 chars, no sensitive fields (addresses, names)
- Inscription text: never stored; digest only
- Image data: never stored
- API key: never stored
- User context: correlation digest only

**Access Control:**
- Telemetry endpoint (`GET /api/admin/numista/telemetry`) requires admin JWT
- Non-admin: 403 Forbidden
- Public health check (`GET /ai-status`) returns simple availability, never telemetry

## Validation

- Phase 2 tasks T012–T013: concurrency tests for ring buffer, atomic updates, redaction guards against secret/user-text fields
- `go test ./services -run TestNumistaTelemetry`

## Alignment

- Principle V: no keys/full-text/images in logs; role-based access to telemetry
- Principle IX: redaction rules enforced by type system (correlation digest, not raw query)
- Constitution §17: telemetry scoped to operational health, not privacy-invasive

---

## Feature 341 User Story 6 / Phase 5A — Canonical Catalog References + Actions Refactor

### Decision: Gate saved-coin Numista disclosure collapse on refreshed persistence

**Date:** 2026-08-11
**Agent:** Aurelia
**Feature:** 341 User Story 6

Saved-coin Numista lookup is composed inside Catalog References as a compact peer disclosure. The disclosure records the confirmed candidate identifier, requests the existing coin refresh, and collapses only when refreshed references contain the matching Numista number; a compatibility fallback recognizes a newly returned Numista reference when older emitters provide no candidate payload.

Actions links to the existing `/coin/:id#catalog-references` anchor rather than owning another lookup panel or route. This follows FR-030–FR-032, NFR-009, Constitution Principle IV, and §17.

---

### Decision: Feature 341 Phase 5 Frontend Status Contract

**Date:** 2026-08-11  
**Agent:** Aurelia  
**Scope:** T048, T051, T052, T053

`NumistaLookupPanel` uses one typed presentation helper for `idle`, `loading`, `success`, `empty`, `unconfigured`, `quota-limited`, `timeout`, and `unavailable`.

- Every state has a visible text label, title, guidance, and retry eligibility.
- Only administrators see Numista API-key instructions and `/admin?tab=system`; ordinary users receive safe administrator-contact guidance.
- Retry-After appears only when `retryAfterSeconds` is supplied.
- Freshness appears only when cache metadata accompanies `success` or `empty`; no remaining-quota estimate is shown.
- The terminal status region receives focus after lookup completion.
- Once the collector edits a query, later parent evidence changes cannot overwrite it. Errors, retries, and status changes retain the edited query and explicit selection.

## Alignment

- Constitution Principle III: typed state/presentation contract.
- Principle IV: one shared helper and panel behavior across direct, photo, and draft paths.
- Principle V: role-safe configuration guidance with no raw errors or quota estimates.
- Principle VI and NFR-008: aria-live, focus management, non-color text, retained mobile-safe controls.

---

### Decision: Feature 341 Phase 5 Numista Outcome and Cancellation Boundaries

**Date:** 2026-08-11  
**Agent:** Cassius  
**Feature:** 341 Improved Numista Lookup, Phase 5

Only typed Numista/provider failures map to expected HTTP 200 domain outcomes. Unknown internal errors propagate to the handler and receive the generic safe HTTP 500 response. Caller cancellation and caller deadlines propagate as context errors and do not pollute the six-status health taxonomy. Retry-After is propagated only when it is a positive observed value. Configuration remains checked before cache access. The service emits the established admin guidance code `numista_configuration_required`; the authenticated handler replaces it with `numista_contact_administrator` for non-admin callers. The deprecated GET adapter continues returning its legacy generic HTTP 503 failure for all non-success/non-empty states.

## Alignment

- Constitution Principles I, III, IV, V, IX and §17 Quality Gate and §21 Definition of Done
- Feature 341 FR-016, FR-017, FR-024, FR-028 and NFR-005–NFR-007

---

### Decision: Feature 341 Phase 5 Strict Lockout Cleared

**Date:** 2026-08-11
**Reviewer:** Brutus
**Status:** APPROVED
**Feature:** specs/341-improve-numista-lookup

Marcus's independent revision clears the rejected Phase 5 backend mapping. Provider 401/403 is `unconfigured`, provider 400 is HTTP 200 `empty`, non-admin guidance remains non-privileged, and legacy GET compatibility behavior remains covered. Configuration precedence, Retry-After, cancellation/deadline handling, safe 500 responses, frontend retention/accessibility, the reconciled lower-authority data model, and T046–T053 all passed focused and full gates.

Alignment: Constitution §0, Principles III/V/VI/IX, §17, §18.2, and §21.

---

### Decision: Feature 341 User Story 6 QA Acceptance Tests and Final Approval

**Date:** 2026-08-11  
**Reviewer:** Brutus  
**Scope:** T087–T096  
**Verdict:** APPROVE  
**Feature:** 341 Improved Numista Lookup

T087–T096 satisfy FR-030–FR-038, NFR-009, and SC-011–SC-014 without reopening landed Feature 214/336 artifacts or adding routes, endpoints, schema, provider calls, cache, telemetry, or enrichment behavior.

- Saved-coin lookup is canonical under Catalog References, preserves manual reference management, waits for persistence plus matching refresh before collapse, returns focus, stays open on failure, and renders the persisted reference.
- Actions contains one compact contextual anchor and no full panel. Identify Coin preserves non-NGC lookup, NGC-first behavior, zero eager NGC-path Numista requests, editable override, labels, selection retention, and narrow-layout containment.
- Draft cards render the exact retained identifier from the owner-scoped preloaded list relation, omit it otherwise, wrap safely, and add no API request.
- Gates passed: focused Go and 34 focused frontend tests; full Go build/vet/tests; architecture/OpenAPI/migration gates; full frontend 109 files/671 tests; design tokens 8/8; type-check; production build; ESLint 0 errors/168 warnings; diff check; targeted secret scan.
- Residual risks are limited to structural jsdom mobile assertions rather than a physical 375 px browser run and unavailable optional external scanners; neither blocks this proportional frontend placement amendment.

Alignment: Constitution Principles III/IV/V/VI/VII/X, §17 Quality Gate, §21 Definition of Done.

---

## Feature 341 MVP — Reviewed & APPROVED

### Decision: Feature 341 Backend MVP — Numista Direct Lookup Architecture

**Date:** 2026-08-11  
**Agent:** Cassius  
**Scope:** T001–T027 backend/shared foundations  
**Status:** IMPLEMENTED & APPROVED

## Context
Feature 341 implements direct Numista lookup workflow as an authenticated API feature with service-to-service caching and deterministic relevance scoring.

## Decision
The direct Numista workflow uses one process-wide injected composition:
```
NumistaHandler -> NumistaLookupService -> NumistaClient
```

The HTTP client owns provider URL/header mapping, private provider DTOs, response limits, timeouts, cancellation, retry policy, and safe errors. The lookup service owns configuration precedence, cache orchestration, candidate sanitization, deterministic scoring, statuses, compatibility mapping, and redacted telemetry.

Search cache keys are SHA-256 digests of normalized query plus result limit. The server reconstructs every canonical catalog URL from the positive numeric Numista ID and never trusts provider/client URLs.

Compatibility: `GET /api/numista/search` remains authenticated and returns the legacy `{count,types}` shape while delegating to the shared service. New direct clients use authenticated `POST /api/numista/lookup`.

## Note
Generated Swagger/OpenAPI artifacts were not refreshed as OpenAPI regeneration is explicitly T077, outside T001–T027 boundary. Handler annotations and Swagger aliases are present.

## Validation
- Go build/vet/full tests PASS
- Architecture test PASS
- Targeted Numista service tests PASS

## Alignment
- Principle I: Clear Layered Architecture (handler → service → client)
- Principle IV: simplest complete service composition
- Constitution §17: all gates passed before merge

---

### Decision: Feature 341 Frontend MVP — Direct Date Evidence & Panel Composition

**Date:** 2026-08-11  
**Agent:** Aurelia  
**Scope:** T002, T018–T019, T024–T027  
**Status:** IMPLEMENTED & APPROVED

## Context
Frontend Numista panel maps coin attributes to OpenAPI contract evidence fields. Existing frontend `Coin` model lacks dedicated date-range field, requiring contract alignment for date evidence.

## Decision
Direct lookup maps `Coin.era` to the approved Numista contract's `evidence.dateText` field, with all other evidence mapped from coin name, ruler, denomination, mint, material, and inscriptions.

When lookup returns HTTP 200 domain statuses, expected outcomes are handled by the reusable panel. If POST rejects unexpectedly, the panel presents the safe `unavailable` state while retaining exact editable query and any explicit selection; raw transport details are not exposed.

No OpenAPI deviations introduced. All frontend tests pass (640/640).

## Validation
- `npm run type-check` PASS
- `npm run build` PASS
- Full `npm run test -- --run` PASS
- ESLint 0 errors

## Alignment
- Principle III: explicit typed API contract
- Principle IV: reused panel composition, no new routes
- Principle VI: PWA-compatible, no emoji in UI text

---

### Decision: Feature 341 MVP Cache Coalescing & Cancellation Safety

**Date:** 2026-08-11  
**Agent:** Tacitus  
**Status:** IMPLEMENTED & APPROVED

## Context
Cache coalescing must preserve caller cancellation/deadline errors without poisoning healthy waiters and must prevent a cancelled first caller from blocking replacement attempts.

## Decision
Same-key cache misses use an explicit per-key call state containing an internally owned provider context, waiter count, completion signal, and result. Cache lookup, in-flight discovery/creation, and successful publication occur under the cache mutex; provider I/O remains outside it.

A caller cancellation detaches only that caller. The provider context is cancelled and the state removed only when the final waiter leaves. A later caller may then create one replacement call; the removed call cannot publish over that replacement because publication verifies pointer identity.

This preserves caller cancellation/deadline errors, prevents a cancelled first caller from poisoning healthy waiters, bounds takeover calls, and avoids retaining provider contexts after completion.

## Validation
- Targeted Numista service tests pass including:
  - 100-iteration cold-cache fan-in
  - Cancelled-first-caller, cancelled waiter, all-cancelled scenarios
  - Bounded replacement failure, cache publication coverage
  - Deterministic stress tests and synchronization audit passed
- Note: `-race` unavailable because CGO_ENABLED=0; residual synchronization risk assessed through deterministic tests and inspection

## Alignment
- Principle IV: simplest complete state machine without race-condition surface
- Principle IX: fully testable without external dependencies
- Constitution §17, §21.6–7

---

### Decision: Feature 341 MVP QA Approval — Final Sixth Revision

**Date:** 2026-08-11  
**Reviewer:** Brutus  
**Status:** APPROVED

## Context
Five prior iterations were blocked under Constitution §18.2 Strict Lockout for contract semantics, cache safety, acceptance-level test coverage, and data model completeness. Tacitus's independent sixth revision addresses all findings.

## Verdict
APPROVE — All prior Strict Lockout findings cleared. The per-key coalescing state machine is accepted: leader cancellation does not poison healthy waiters, all-canceled work is canceled without publication, cache/in-flight operations are atomic, and superseded calls cannot overwrite replacement work.

## Verified Gates
- ✅ Exact 1 MiB/1 MiB+1 provider body boundaries proven
- ✅ Mutation-sensitive missing date/ruler/denomination/inscription neutrality proven
- ✅ Full Go build/vet/test suite passed
- ✅ Frontend 640/640 tests passed
- ✅ Route drift test passed
- ✅ Architecture test passed
- ✅ Strict types/build passed
- ✅ Focused stress tests passed
- ✅ Lint passed
- ✅ Diff hygiene passed
- ✅ T001–T027 checksum complete

## Residual Risk
Limited to unavailable `go test -race` execution because `CGO_ENABLED=0`. Deterministic stress and synchronization review were sufficient for approval.

## Alignment
- Principle I, IV, VII, X: Architecture, simplicity, convention, audit
- Constitution §17 (Quality Gate): all gates passed
- Constitution §21 (Definition of Done): all 15-item checklist verified

---

### Decision: Feature 341 Phase 4 Backend — Selected Reference Persistence

**Date:** 2026-08-11  
**Agent:** Cassius  
**Scope:** T028–T032, T036–T042  
**Status:** IMPLEMENTED

## Context

The active spec/plan/data model govern selected-reference persistence: an additive child row stores owner, canonical `Catalog`/`Number`/`URI`, and promotion copies the existing `CoinReference` shape transactionally for both collection and wishlist targets.

An older `.squad/decisions.md` sketch mentions `catalog_number`, `linkType='lookup_selected'`, and a separate reference route. Per Constitution §0, implementation follows the higher-ranked Feature 341 artifacts and additive Quick Capture create/update DTOs. Those historical fields/routes do not exist in the authoritative active contract or current `CoinReference` schema.

## Decision

Canonical selection validation is performed before draft writes and then delegated to the existing `CoinReferenceService` catalog rules. Repository transactions own create/replace/remove and promotion copy, preserving rollback, ownership, CAS concurrency, and repeated-promotion idempotency.

## Alignment
- Principle I: Clear layered architecture (handler → service → repository)
- Principle III: Exact typed multipart DTO contract
- Principle IV: Simple complete proportional change to Quick Capture flow
- Constitution §0: Higher-ranked Feature 341 spec takes precedence

## Validation
- Transaction tests: create/preserve/replace/remove, owner isolation, rollback, CAS concurrency, repeated promotion
- Repository and service contract tests pass
- Handler compatibility tests pass
- Migration tests pass (additive, no row rewrites)

---

### Decision: Quick Capture Numista Selection — Explicit Tri-State Mutation

**Date:** 2026-08-11  
**Agent:** Aurelia  
**Feature:** 341 Improved Numista Lookup, Phase 4  
**Status:** IMPLEMENTED

## Context

Quick Capture updates must distinguish an unrelated edit from replacing or removing the optional selected Numista reference. The Identify Coin integration must keep the Numista lookup panel visible for manual entry, even when photo evidence produces an empty proposed query (matching spec's required disabled-search guidance).

## Decision

The frontend tracks selection mutation as three explicit states:
- **unchanged** — Unrelated draft updates omit all selection multipart fields
- **replace** — Selection changes send approved `selectedNumistaId`/`selectedNumistaUrl` pair
- **clear** — Removal sends only `clearSelectedNumista=true`

The shared lookup panel accepts an initial selection so a retained reference remains visible outside later result sets. Promotion is disabled while a reference change is pending (promotes saved draft relation, not unsaved frontend state).

Identify Coin leaves the Numista panel visible with a disabled search button when photo evidence produces an empty `proposedNumistaQuery`, allowing manual entry at the disabled-search stage (matches active spec).

## Alignment
- Principle III: Exact typed multipart DTO contract
- Principle IV: Explicit local state without hidden inference
- Principle VI: Stable, keyboard-operable selection on mobile and desktop
- Constitution §17/§21: Focused serialization and workflow regression coverage

## Validation
- Type-check: `npx vue-tsc --noEmit` (strict)
- Tests: Draft resume, Identify Coin integration, multipart serialization
- Frontend build: Vue Vite production build pass
- Lint: ESLint 0 errors

---

### Decision: Feature 341 Phase 4 QA Review — Final Approval

**Date:** 2026-08-11  
**Reviewer:** Brutus  
**Status:** APPROVED

## Context

Three Strict Lockout blocks (Constitution §18.2) were issued:
1. **Generated Swagger nullability mismatch** — Runtime nullable pointers not reflected in OpenAPI contract
2. **Identify Coin panel visibility** — Numista panel hidden when empty proposed query (contradicted spec guidance)
3. **Handler/model contract drift** — Changed contracts not propagated to generated Swagger snapshots

Hadrian's independent revision addressed block 1 (schema alignment). Aurelia revised block 2 (panel visibility logic). All snapshots were regenerated for block 3.

## Verdict

**APPROVE** — All three Strict Lockout blocks cleared. T028–T045 complete and verified.

## Verified Gates
- ✅ Nullable runtime pointers and generated Swagger schemas aligned (`x-nullable` / `allOf` / `$ref` accurate)
- ✅ All four OpenAPI artifacts regenerate byte-identically (docs/openapi.json, src/api/docs/docs.go, swagger.json, swagger.yaml)
- ✅ Identify Coin panel visible with disabled search for empty proposed query (manual entry allowed)
- ✅ Prior manual-entry/no-eager-call/NGC-first/contract/task findings cleared
- ✅ T028–T045 checksum complete
- ✅ Focused contracts/Go/frontend tests pass
- ✅ Full Go build/vet/test suite pass
- ✅ Frontend 654/654 tests pass
- ✅ Frontend lint 0 errors
- ✅ Type-check (strict) pass
- ✅ Production build pass
- ✅ Diff/secret checks pass
- ⚠️ Gitleaks/Trivy unavailable (residual)

## Residual Risk

Limited to unavailable Gitleaks and Trivy scanning. All code quality and functional gates passed.

## Alignment
- Principle I, III, IV, VII, X: Architecture, typing, simplicity, convention, audit
- Constitution §17 (Quality Gate): all gates passed
- Constitution §21 (Definition of Done): T028–T045 checksum complete

---
---

### Decision: Feature 341 Phase 6 QA Review and Approved Implementation

**Date:** 2026-08-11  
**Reviewers:** Brutus (QA/approval), Claudius (implementation fix)  
**Status:** APPROVED

## Context

Phase 6 cycles refined Numista cache telemetry ownership, health API, and admin UI:

**Cassius-Aurelia-Augustus iterations** identified three critical misalignments:
1. Real provider retry attempts not captured in production (only test-constructed events)
2. TypeScript health contract type mismatch with sparse backend maps
3. Admin UI missing successful-empty state; incomplete retry/coalesced/failure semantics

**Tiberius-Germanicus revision** corrected false cache-hit classification. The subsequent **Vespasian-Nerva revisions** remained blocked on orthogonal issues:
- Cancelled loader events still emitting provider aggregates
- Shallow detail cache copy allowing nested reason mutation
- Orphaned/late provider results emitting despite cancellation

**Claudius independent revision** (2026-08-11 15:04) implemented the definitive fix:
- Loader closures prepare redacted telemetry callbacks only after cache confirms pointer-identity ownership
- Superseded or orphaned late results discard callbacks and emit zero provider aggregates
- Successful/failed cold fan-in produces exactly one provider event; replacement ownership emits once
- Fresh cache hits, coalesced waiters, unconfigured bypasses retain existing semantics
- Barrier-controlled ownership/cancellation/replacement tests pass at -count=100

## Verdict

**APPROVE** — All Phase 6 blockers cleared. T054–T063 satisfy FR-021–FR-026, NFR-005–NFR-007, SC-006, SC-008, SC-009.

## Requirements Coverage

- **FR-021** (Real provider retry telemetry): Search and detail loaders capture and emit retry attempt counts from provider layer
- **FR-022** (Separate fresh/coalesced metrics): Distinct event types for fresh cache hits, coalesced waiter reuse, and provider loads
- **FR-023** (Cache state visibility): Admin health API exposes cache hit rate, refresh rate, bypass rate, failure rate, and provider latency percentiles
- **FR-024** (Admin health UI): Real-time cache metrics with sparse/empty/partial/populated rendering; redaction for non-admin users
- **FR-025** (Ownership semantics): Only cache loaders own provider status/latency/failure aggregates; cancellation emits only cancellation events
- **FR-026** (Coalesce transparency): Concurrent callers correctly attributed as coalesced (one provider call, N-1 waiters reuse result, zero provider latency for reuse)
- **NFR-005** (Percentile fidelity): Rolling R-7 p50/p95 calculated per reconciled definition; boundary tests validate small/large windows
- **NFR-006** (Bounded retention): Health ring stores ≤100 events; oldest FIFO expiry; zero-event ring handled explicitly
- **NFR-007** (Telemetry redaction): Non-admin GET /admin/health/numista returns sparse maps, zero counts, and 
ull latency/retry details

## Verified Gates
- ✅ Ownership/cancellation/replacement/late-success regressions at -count=100
- ✅ Go build, vet, architecture rules, OpenAPI contract checks, full test suite
- ✅ Frontend Vitest (654 tests), strict type-check, production build, lint (0 errors)
- ✅ OpenAPI regeneration byte-stable (docs/openapi.json, Swagger artifacts)
- ✅ Numista domain deep-copy mutation isolation
- ✅ Admin layout/auth/redaction/responsive/empty/partial/populated state coverage
- ✅ Diff and privacy compliance checks
- ⚠️ Go race detector (unavailable: CGO disabled), Gitleaks, Trivy (residual)

## Implementation Summary: 9 Cycles

| Cycle | Agent | Focus | Outcome |
|-------|-------|-------|---------|
| 1 | Cassius | Cache structure, basic telemetry | BLOCK: production retries missing |
| 2 | Aurelia | Health UI, TS contract | BLOCK: type mismatch, missing states |
| 3 | Augustus | Retry telemetry, coalesce accounting | BLOCK: detail retries incomplete, double-count waiters |
| 4 | Tiberius | Retry ownership, cache-hit classification | BLOCK: false freshness classification |
| 5 | Germanicus | Hit correction, fresh/provider separation | BLOCK: fresh hits in provider p50/p95, incomplete activity detection |
| 6 | Vespasian | Cancellation/replacement ownership | BLOCK: cancelled events emit provider aggregates |
| 7 | Nerva | Late orphaned-result handling | BLOCK: non-cooperative cancellation emits double provider events |
| 8 | Claudius | Pointer-identity ownership callback | PASS: telemetry conditional on cache ownership confirmation |
| 9 | Brutus | Final QA review | APPROVE: all requirements satisfied, gates passed |

## Alignment

- Constitution Principles III (Consistent API), IV (Simple Complete Changes), V (Security/Privacy), IX (Test-Driven), X (Audit)
- Constitution §17 (Quality Gate): all 15 gate checks passed
- Constitution §21 (Definition of Done): T054–T063 checksum and cycle-log complete
- Feature 341 Phase 6 Specification (.specify/specs/341-improve-numista-lookup/)

## Residual Risk

Limited to unavailable race detector and binary scanners. All code, architecture, contract, and functional gates passed.

---
## Feature 341 Release Review — adr-0008-accepted.ToUpper().Replace('-', ' ')

# ADR 0008 Acceptance

**Date:** 2026-08-11
**Author:** Cincinnatus
**Status:** ACCEPTED
**Feature:** 341 Improved Numista Lookup

## Decision

Brian explicitly selected “Approve ADR 0008 (Recommended).” ADR 0008 is
therefore accepted as the one-time §22 waiver for the exact four immutable
Feature 341 public-history deviations in its exception matrix.

Public `beta` history remains unchanged. The waiver is non-precedential and
does not relax future commit hygiene: subsequent AI-assisted commits and the
future PR/release evidence must use an allowed conventional prefix, include a
parseable required Copilot co-author trailer, and disclose the accepted ADR
and all four exceptions.

T086's evidence acceptance is satisfied by the accepted ADR, transparent
matrix, completed quality/workflow checks, and prospective enforcement.
Maximus's final-review block is not cleared by this decision and still
requires his explicit re-review and clearance.

## Authority

Constitution §0, Principle VII, §17, §21 items 14 and 17, and §22.


---

## Feature 341 Release Review — directive-20260811T163624-0500.ToUpper().Replace('-', ' ')

### 2026-08-11T16:36:24-05:00: User directive
**By:** Brian DeNicola (via Copilot)
**What:** Group the Numista API key and lookup-limit settings directly above Numista Health, and place the combined settings and health content inside one visually bounded Numista section.
**Why:** User approved this Admin System information-architecture refinement based on the current UI.


---

## Feature 341 Release Review — directive-20260811T164500-0500.ToUpper().Replace('-', ' ')

### 2026-08-11T16:45:00-05:00: User directive
**By:** Brian DeNicola (via Copilot)
**What:** Accept the documented immutable-history exception for Feature 341 and proceed without rewriting public beta history.
**Why:** Public history contains nonstandard historical subjects and one malformed co-author trailer; preserving published history takes precedence over retroactive correction.


---

## Feature 341 Release Review — feature-341-adr0008-rereview-block.ToUpper().Replace('-', ' ')

# Feature 341 ADR 0008 Final Re-Review

**Date:** 2026-08-11
**Reviewer:** Maximus
**Verdict:** BLOCK

## Cleared

- Brian's explicit approval and accepted ADR 0008 provide a narrowly bounded,
  non-precedential waiver for only `31cb603`, `a8f59b3`, `8e77500`, and
  `460dbfc`. The matrix, subjects, and parsed trailer states are accurate.
- Public `beta` history is preserved. The ADR requires future conventional
  subjects, parseable required Copilot trailers, and full PR disclosure.
- SC-002 remains credible at 24/24 fixtures, six candidates, three
  permutations, no `ExactNumistaID`, required field reasons, and fourth-place
  discrimination.
- Admin settings and Health are grouped in one labelled Numista boundary;
  focused tests cover structure, token-backed responsive layout, save/bounds,
  loading, error, retry, empty/sparse/populated states, redaction, and
  accessibility.
- T073-T085 evidence remains credible. Focused integration and Admin tests,
  frontend type-check, architecture/OpenAPI route checks, Gitleaks history and
  worktree scans, and Trivy High/Critical scan of the published beta image pass.
  GitHub run 31536414201 checked out and published SHA `011c635`.
- The worktree contains only the reviewed Phase 8 integration tests, Admin UI
  refinement/tests, Gitleaks hardening, Feature 341 evidence/tasks, ADR/index,
  and append-only squad history. No generated artifacts or secrets are
  package candidates.

## Blocker

Constitution §22 states that deviations from any Principle must be explicitly
justified in the PR and tracked in the plan's Complexity Tracking table.
Accepted ADR 0008 waives four Principle VII/§17/§21 historical deviations, but
`specs/341-improve-numista-lookup/plan.md` still says:

> No constitutional violations require justification.

That higher-authority process requirement conflicts with T086's completed
state and the quickstart's claim that the DoD is fully reconciled. Update only
the plan's Complexity Tracking entry to identify ADR 0008's exact immutable
exception matrix and non-precedential scope, then request Maximus re-review.

## Gate note

Focused `go test -race ./integration` was attempted but could not run because
the local Go environment has CGO disabled. Normal focused integration tests
pass. Physical-browser and live-provider E2E remain explicitly unperformed.

Strict Lockout remains active. Scribe must not commit, push `beta`, or open the
`beta`-to-`main` PR until Maximus explicitly clears this blocker.


---

## Feature 341 Release Review — feature-341-final-clearance-approved.ToUpper().Replace('-', ' ')

# Feature 341 Final Release Clearance

**Date:** 2026-08-11
**Reviewer:** Maximus
**Verdict:** APPROVE

## Clearance

Scipio's plan reconciliation clears the sole remaining reviewer block.
`specs/341-improve-numista-lookup/plan.md` no longer claims that no waiver or
Complexity Tracking entry is required. It now records ADR 0008's exact
exceptions:

- nonconventional subjects on `31cb6033875bcb6da0db82e9fc59a1278a56b0f6`,
  `8e77500f05dde63ed7335fa12ba14614fe6e2ba2`, and
  `a8f59b3bf7e2479e1083ee21f0737369c89c3a91`;
- the unparseable required Copilot co-author trailer on
  `460dbfcd0ba4bd36d39d150945d9c39546551be3`;
- no trailer waiver for the reconciliation merge;
- preservation of published history and audited references;
- rejection of amend/rebase/reset/rewrite/force-push;
- expiry after this Feature 341 `beta`-to-`main` review, non-precedent, and
  mandatory prospective prefix/trailer enforcement.

The design and post-design gates remain accurately scoped to product design.
T086 and the §21 Definition of Done remain legitimate: the accepted waiver
covers only the disclosed immutable history, while the next commit and PR must
comply prospectively. The older “pending final clearance” evidence is resolved
by this explicit review record.

## Package and gates

The worktree package remains limited to the previously reviewed Phase 8
integration tests, Admin System refinement/tests, Gitleaks hardening, Feature
341 evidence/tasks, ADR/index, append-only squad records, and the approved plan
reconciliation. No unrelated candidate or generated artifact appeared.
`beta` and `HEAD` both remain
`011c6350fd067d64597c8ecb601c649bf097f78f`; public history was not rewritten.

Final focused checks:

- `git diff --check` — PASS
- Feature 341/ADR local Markdown links — PASS
- contradiction search for false no-waiver/no-deviation claims — PASS
- ADR/plan/quickstart/tasks exception-matrix consistency — PASS
- four immutable SHA subjects and parsed trailer states — PASS
- worktree package/status comparison to prior approved evidence — PASS

The previously approved Go, Vue, Python, architecture, OpenAPI, Gitleaks, and
published-beta Trivy evidence is unaffected, so full suites were not rerun.
The recorded CGO race-detector and physical-browser/live-provider limitations
remain unchanged and non-blocking.

## Authorization

Strict Lockout is cleared. Scribe is authorized to:

1. create one prospectively compliant commit with an allowed Conventional
   Commit prefix and a parseable required Copilot co-author trailer;
2. push `beta` without rewriting history; and
3. open the `beta`-to-`main` pull request with the title and body below.

## PR title

`feat: improve Numista lookup workflows and operations`

## PR body

### Summary

- deliver typed, authenticated Numista lookup and bounded enrichment across
  saved-coin and Quick Capture workflows;
- add deterministic scoring, cache/telemetry health, admin configuration, and
  transactional selected-reference promotion;
- preserve legacy compatibility while documenting rollout, operations,
  security, and release evidence.

### Validation

- Go build, vet, full tests, architecture, route drift, and deterministic
  provider-independent integration workflows: pass
- Vue focused/full tests, type checks, Docker-equivalent `vue-tsc --build`,
  and production build: pass
- Python Ruff and 189 tests: pass
- OpenAPI regeneration byte stability and documentation links: pass
- Gitleaks history/worktree scans: pass
- published `beta`/immutable image Trivy scan: 0 High/Critical

No physical-browser or live-provider E2E was performed. The focused race test
was unavailable because local CGO is disabled; normal focused integration
tests pass.

### Immutable release-history waiver

[ADR 0008](https://github.com/briandenicola/Aurearia/blob/beta/docs/adr/0008-feature-341-immutable-public-history-waiver.md) is
accepted under Constitution §22. It waives only:

| SHA | Exception |
| --- | --- |
| `31cb6033875bcb6da0db82e9fc59a1278a56b0f6` | disallowed `scribe:` subject |
| `8e77500f05dde63ed7335fa12ba14614fe6e2ba2` | disallowed `merge:` subject; no trailer waiver asserted |
| `a8f59b3bf7e2479e1083ee21f0737369c89c3a91` | disallowed `merge:` subject |
| `460dbfcd0ba4bd36d39d150945d9c39546551be3` | required Copilot trailer is a list item and not parseable |

No amend, rebase, reset, rewrite, or force-push occurred. The waiver expires
with this Feature 341 release, is not precedent, and does not relax later
commit or PR enforcement.

### Constitution and Definition of Done

This PR complies with Constitution §0, Principles I–IX, §17, §21, and §22,
subject only to ADR 0008's exact immutable exceptions.

- [x] Builds, tests, architecture checks, type checks, and linters pass
- [x] Workflow/config contracts and exact regression paths are covered
- [x] Swagger/OpenAPI and documentation are synchronized
- [x] Material decisions are recorded in ADRs 0007 and 0008
- [x] Active Feature 341 tasks, including T086, are complete
- [x] Secrets and container vulnerability gates pass
- [x] Change is simple, complete, and proportional
- [x] New release-evidence commit uses an allowed prefix and parseable required
      Copilot co-author trailer
- [x] Historical exceptions are fully disclosed above


---

## Feature 341 Release Review — feature-341-final-clearance-block.ToUpper().Replace('-', ' ')

# Feature 341 Final Combined Clearance

**Date:** 2026-08-11  
**Reviewer:** Maximus  
**Verdict:** BLOCK

## Cleared findings

- SC-002/T076 is genuine: 24 distinct Roman, Greek/Hellenistic, Byzantine,
  and medieval fixtures; six plausible candidates each; three deterministic
  permutations; no `ExactNumistaID`; field-reason and fourth-place score
  discrimination; 24/24 top-three. Focused and full integration tests pass.
- Aurelia's Admin System refinement places the API key and six limits directly
  above Health inside one labelled, token-backed, mobile-first Numista
  boundary. Unrelated settings remain outside. Save, bounds, loading, error,
  retry, empty, sparse, populated, redaction, and accessibility tests pass.
- T073–T085 evidence is credible. OpenAPI regeneration is byte-stable; Go,
  Vue, and Python gates pass; Gitleaks history/worktree scans report no leaks;
  Trivy reports 0 High/Critical for the published beta image. GitHub run
  31536414201 proves the beta and immutable SHA tags were built from
  `011c6350fd067d64597c8ecb601c649bf097f78f`.
- The immutable exception matrix is accurate: nonconventional subjects
  `31cb603`, `8e77500`, and `a8f59b3`; malformed/unparseable required
  co-author at `460dbfc`; `Copilot-Session` is optional. Public beta history
  remains unchanged.

## Remaining blocker

Brian's directive in
`.squad/decisions/inbox/copilot-directive-20260811T164500-0500.md` explicitly
accepts the immutable-history exception. It is not, however, the waiver ADR
required by the Constitution header and §22. Under §0, an inbox decision cannot
override the Constitution. Transparent PR disclosure is necessary but cannot
alone make Principle VII, §17, and §21 item 17 compliant.

T086 therefore remains unchecked. The requirements checklist remains
consistent; T087–T096 are complete.

## Lockout and required action

The prior T086 rejection is not cleared. Octavian remains locked out from
revising the rejected closure. Governance owner Brian must authorize an
ADR-backed waiver through §22; an independent governance author must record
it, and Maximus must explicitly clear the block. Scribe must not commit/push
or open the beta-to-main PR before clearance.

## Gates run

- Focused benchmark and all `src/api/integration` tests: PASS
- Focused Admin Numista Vitest, full Vitest, type-check, `vue-tsc --build`,
  production build: PASS
- Go build, vet, full tests, architecture, route drift: PASS
- Ruff and 189 Python tests: PASS
- OpenAPI byte stability and `git diff --check`: PASS
- Gitleaks git/dir: PASS
- Trivy published `bjd145/ancient-coins:beta`: 0 High/Critical

Residual evidence limitation remains explicit: no physical-browser or
live-provider E2E was performed.


---

## Feature 341 Release Review — feature-341-phase8-docs.ToUpper().Replace('-', ' ')

# Feature 341 Phase 8 Documentation Reconciliation

**Date:** 2026-08-11  
**Agent:** Maximus  
**Status:** DOCUMENTED

## Decision

Phase 8 documentation follows merged runtime and higher-authority Feature 341
artifacts. Lower-authority sketches describing inferred persistence,
`quota_limited`, a separate draft-reference route,
`/admin/numista/telemetry`, query truncation, non-loader latency, a one-hour
empty TTL, or a 100-event ring are not runtime contracts.

The shipped contract uses typed lookup/enrichment POST routes, deprecated GET
compatibility, six statuses including `quota-limited`, raw broad effective
query and trimmed enrichment query, 500-event publication-owned telemetry at
`GET /api/admin/numista/health`, 24/168-hour TTL defaults, additive Quick
Capture selection fields, and exact transactional promotion copy.

## Alignment

Feature 341 FR-001–FR-038; Constitution §0, Principles I, III, IV, V, VIII,
IX, §17, and §21.

## Release evidence correction

The public merge description on
`a8f59b3bf7e2479e1083ee21f0737369c89c3a91` incorrectly labels T087–T096
as Quick Capture promotion. The active task list governs: those tasks are
Phase 5A / User Story 6 canonical catalog-reference placement and contextual
lookup UX. Transactional Quick Capture promotion is User Story 2 work,
principally T030–T031 and T038–T040. This correction is append-only; public
history is not amended.

Principle VII exceptions are also explicit:
`31cb6033875bcb6da0db82e9fc59a1278a56b0f6` uses `scribe:` and
`a8f59b3bf7e2479e1083ee21f0737369c89c3a91` uses `merge:`, neither of which
is an allowed conventional prefix. Both have the required Copilot trailer.
The Phase 8 follow-up commit and future PR must be compliant and must disclose
these immutable historical exceptions rather than claiming strict historical
subject compliance.


---

## Feature 341 Release Review — feature-341-phase8-final-block.ToUpper().Replace('-', ' ')

# Feature 341 Phase 8 Final Re-Review

**Date:** 2026-08-11  
**Reviewer:** Maximus  
**Verdict:** BLOCK

## Blocking findings

1. **T076 does not prove SC-002.** In
   `src/api/integration/numista_performance_test.go`,
   `TestNumistaDeterministicScoringFixturesAndDefaultDetailCeiling` supplies
   the expected candidate as `ExactNumistaID`, while broad candidates are
   generic and enriched candidates have identical comparison evidence. The
   reported 20/20 top-three result is therefore tautological, not the
   specification's curated known-coin scoring benchmark. Replace this with a
   sanitized, diverse known-coin fixture set whose correct candidates are
   ranked from realistic title/issuer/denomination/mint/date/material/
   inscription evidence, and prove the SC-002 threshold without supplying the
   answer as exact-ID evidence except where exact-ID is genuinely the tested
   scenario.

2. **T086's immutable-history disclosure is incomplete.** The quickstart and
   existing decision inbox state there are only two nonconventional subjects,
   but `8e77500` also uses the disallowed `merge:` prefix. Additionally,
   `460dbfc` is AI-assisted Feature 341 history but its
   `- Co-authored-by: ...` bullet is not a Git trailer
   (`git interpret-trailers --parse` returns none). Public history must remain
   immutable, but the append-only correction and future PR must disclose all
   subject/trailer exceptions accurately. `Copilot-Session` is optional under
   the constitution: it is present on the main Feature 341 product sequence
   and `a8f59b3`, absent on `31cb603`, `97bba2e`, `6726b09`, `011c635`, and
   not parseable on `460dbfc`/`8e77500`; no claim should make it mandatory.

## Passing evidence

- Focused integration tests and deterministic repetitions pass.
- Architecture and route-drift tests pass.
- OpenAPI generation is byte-stable across two runs.
- Tightened Gitleaks configuration passes full history and worktree scans;
  exclusions are path/placeholder scoped, and the three commit exclusions
  contain verified documentation placeholders only.
- The immutable `sha-011c6350fd067d64597c8ecb601c649bf097f78f`
  production image and mutable `beta` image scan with 0 High/Critical
  vulnerabilities. GitHub run `31536414201` checked out `011c635` and
  published the immutable SHA tag, so published-image evidence is acceptable
  despite local Docker unavailability.
- Quickstart correctly discloses that no physical-browser/live-provider
  walkthrough occurred.

Octavian remains locked out under Constitution §18.2 until Maximus explicitly
clears these findings.

---

### Decision: Feature 343 Phase 1 — Nomisma OpenRefine Wire Contract Verification and Fix

**Date:** 2026-08-14  
**Agent:** GitHub Copilot CLI (QC follow-up)  
**Branch:** `343-nomisma-mint-authority-linking`  
**Status:** RESOLVED — T026 complete, verified live

## Context

Live QC for Feature 343 Phase 1 discovered that `HTTPNomismaClient.Search` always returned `no_match` even for valid mint names ("Roma", "Rome"). Root cause analysis identified two compounding contract bugs against the real `nomisma.org/apis/reconcile` OpenRefine-compatible reconciliation API:

1. **Request double-wrapping.** Client marshalled query-id map under an outer `{"queries":{...}}` key, violating the OpenRefine wire spec (single query-id map expected).
2. **Response unwrapping mismatch.** Parser expected top-level `{"results":{...}}` but real API returns query-id map at top level.

Test fixtures (`httptest.Server`) masked both bugs by hand-crafting both incorrect shapes.

## Fix Verified Live

Confirmed against `https://nomisma.org/apis/reconcile` for "Roma"/"Rome":

- **Request:** Query-id map **directly** in `queries` param (no outer wrapper): `{"q1":{"query":"Roma","limit":5}}`
- **Response:** Query-id map at top level: `{"q1":{"result":[...]}}`  (no `results` wrapper)
- **ID field:** Short local id (e.g., `"rome"`), expanded to Nomisma concept URI `http://nomisma.org/id/<id>` before returning
- **Match field:** JSON string (`"true"`/`"false"`), not boolean; requires custom `UnmarshalJSON` for compatibility

## Changes Applied

- `src/api/services/nomisma_client.go`: Direct query-id map marshalling; response parser updated to `map[string]struct{Result []nomismaResultItem}`; short `id` expanded to concept URI; `match` field decoded via `nomismaMatch` custom type
- `src/api/services/nomisma_client_test.go`: All fixtures corrected to live wire shape; two regression tests added (`TestHTTPNomismaClient_Search_RequestShapeMatchesLiveContract`, `TestHTTPNomismaClient_Search_ResponseShapeMatchesLiveContract`)
- `specs/343-nomisma-mint-authority-linking/research.md` §1 and `docs/adr/0009-nomisma-authority-linking.md`: Corrected wire description and updated HTTP → HTTPS

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` all pass, including new regression tests
- Live end-to-end (admin user → global mint location → search "Rome" on live nomisma.org → link `http://nomisma.org/id/rome` → verify persistence → unlink → verify idempotent clear): pass
- Frontend component tests and existing Nomisma test suite pass
- Architecture and OpenAPI generation green

---

### Decision: OCRE/RPC Identified as Required Phase 2 Identify Coin Data Sources (Deferred, Distinct from F343)

**Date:** 2026-08-14  
**Agent:** Specifier / Feature 343 Phase 1 closure  
**Status:** DEFERRED to Phase 2 specification  
**Branch:** `343-nomisma-mint-authority-linking`

## Context

Feature 343 Phase 1 delivers Nomisma authority linking for global mint locations. During Phase 1 planning, the broader Identify Coin feature (future) was analyzed as a consumer of authority data. Nomisma is suitable for mint-location global authority but insufficient for coin identification, which also requires:

- **OCRE** (Online Catalog of Roman Monetary Empires): RPC-indexed data for Roman coin reverse identification
- **RPC** (Roman Provincial Coinage): Reference types and typology required for date/mint/reverse binding in Identify Coin

OCRE/RPC are separate API contracts, licensing terms, and UI workflows from Nomisma mint authority linking and must not be conflated with F343.

## Decision

1. **F343 Phase 1 closes as planned** without OCRE/RPC integration (not in scope)
2. **OCRE/RPC are mandatory Phase 2 sources** for the future Identify Coin feature, requiring:
   - Dedicated feature specification and research (new F344 or similar)
   - API contract discovery, licensing review, and ADR
   - Separate schema/handler/service implementation
   - Independent QC and release cycle
3. **No regression in F343**: Nomisma mint authority linking remains production-ready; Identify Coin UI does not yet depend on OCRE/RPC (future feature backlog)

---



# Decision: CoinActionsPanel.vue god-component extraction (F6 backlog, audit finding)

**Author:** Aurelia (Frontend Dev)
**Date:** 2026-08-17
**Requested by:** Brian (@briandenicola)

## Context

Audit finding F6 flagged `src/web/src/components/coin/CoinActionsPanel.vue` at 372 script
lines / 9 responsibilities. By the time this task started it had grown to **550 total lines**
(script + template) because Feature 344 bolted the Deep Analysis launch onto it directly
instead of extracting a composable. Per the task brief, the audit description was treated as
stale and the file was re-inventoried from scratch rather than refactored from the old list.

## Real inventory found (not the audit's)

1. **Image upload** — file picker, pasted-URL fetch, in-app camera capture, all writing one
   shared `uploadStatus`/`uploadError` pair.
2. **AI value-estimate background job** — start, resume-on-mount (active-jobs endpoint +
   `sessionStorage` fallback), poll-until-terminal, apply-to-coin, dismiss. By far the largest
   single block (~190 lines) and the most tangled state (six refs + a timer handle all read
   and written by the same six functions).
3. **Deep Analysis modal launch** — already routed entirely through `useDeepAnalysisLauncher`
   (extracted earlier this session for the at-capacity work). Nothing left to duplicate here.
4. **Static catalog-references link** — one `RouterLink`, no state.

## What was extracted

- `src/web/src/composables/useCoinImageUpload.ts` — owns responsibility #1 in full (upload
  type/status/error refs, image-URL field, three upload handlers, one `onUploaded` callback).
- `src/web/src/composables/useCoinValueEstimate.ts` — owns responsibility #2 in full, including
  the `onMounted`/`onUnmounted` resume/cleanup hooks (Vue allows lifecycle hooks inside any
  composable invoked during a component's `setup()`; verified against the existing
  `useDeepIdentificationCapability` composable, which does the same thing and is already
  covered by a mount-based test).
- Both composables accept `coinId`/`imageCount` as `MaybeRefOrGetter<number>` (resolved with
  `toValue()`) so they stay reactive to prop changes exactly like the original inline
  functions, which read `props.coinId` fresh on every call.

Result: `CoinActionsPanel.vue` dropped from 550 to 247 total lines. The remaining script is
now almost entirely wiring: two composable calls, the Deep Analysis launcher (unchanged), and
one small `safeComparableUrl` helper for template-only URL sanitization.

## What was deliberately NOT extracted, and why

**The "AI Value Estimate" template block (~40 lines: confidence badge, comparables list,
apply/dismiss buttons) was left inline in `CoinActionsPanel.vue` rather than pulled into a
child component.** Once the *stateful* logic moved into `useCoinValueEstimate`, the remaining
template block is short, reads directly off values the composable already returns, and is used
in exactly one place. Splitting it into a sibling `.vue` file would have meant threading six-plus
reactive values (`estimating`, `estimateStatusMessage`, `estimateError`, `valueEstimate`, plus
two handlers) through props/emits for a component with no reuse target and no independent
lifecycle — a smaller line count bought by scattering one tightly-coupled render block across
two files, for zero cohesion gain. This mirrors the call Maximus made the same day on the Go
side, declining to split a seam that shared a mutex: partial, well-reasoned extraction beats a
forced full split.

## Behaviour change check

- Same actions, same messages, same styling — no template text or CSS was touched, only
  variable/function origins moved.
- The existing `CoinActionsPanel.test.ts` (4 tests) was **not edited** and passed unmodified —
  the strongest available signal that behaviour did not change.
- Added `useCoinImageUpload.test.ts` (8 tests) and `useCoinValueEstimate.test.ts` (6 tests) for
  the newly extracted composables' logic.

## Gates (from `src/web/`)

- `npm.cmd run type-check` — exit 0
- `npx.cmd vitest run` — **133 files / 862 tests passed** (baseline was 131 files / 848 tests;
  the +14 is exactly the two new composable test files)
- `npm.cmd run build` — succeeded (Docker-equivalent `vue-tsc --build` strictness)

No existing test was modified.


### Decision: Feature 351 Phase 12b — Hypothesis panel, image-vs-cited marking, confidence-driven acceptance (T089–T092, T120)

**Date:** 2026-08-17
**Agent:** Aurelia (Frontend Dev)
**Feature:** `specs/351-vision-first-deep-identification` — Phase 12 (T089-T092, T120)
**Branch:** `beta`

## What shipped

- **`DeepReportPanel.vue`**: a collapsible, default-collapsed "What the images
  alone said" section (RD-6) rendering `report.image_hypothesis`'s typed
  fields with per-field confidence plus `observations`. States plainly when
  `legible: false` ("not legible enough… different from the analysis not
  running at all"), distinct from the separate "legible but nothing found"
  case. Hidden entirely (no empty shell) when `image_hypothesis` is absent —
  verified against a report fixture with no such key.
- **`DeepProposalEditor.vue`**: image-derived fields (`evidence.length === 0`,
  i.e. an empty array present — not simply absent) get a `.chip-sm` "Image
  only" marker, visibly distinct from provider-cited fields.
- **Confidence-driven acceptance default (RD-3, reversing the earlier
  "image-only opt-in" default)**: a field renders accepted once its
  confidence is ≥ `DEEP_PROPOSAL_ACCEPTANCE_THRESHOLD` (0.70), regardless of
  source. Single named constant + `effectiveDeepProposalAcceptance()` in new
  `src/web/src/utils/deepProposalAcceptance.ts`, consumed by
  `DeepProposalEditor.vue` (render + `confirmDisabled`) and
  `DeepAnalysisPage.vue` (see below). No bare `0.70` literal anywhere (T120).
- **Disagreement claim marking**: `DeepClaim.citation` is now optional (an
  image-sourced claim has no citation per contract §3); a citation-less claim
  renders a `.chip-sm` "From images" marker instead of a broken/missing link.
- Extracted `formatFieldName()` to `src/web/src/utils/format.ts`, replacing
  the duplicated inline label-formatting regex.

## A finding other agents should know

**RD-3's default cannot be purely cosmetic.** Traced the actual write path:
`src/api/services/deep_identification_pipeline_runner.go` always persists
`Accepted: nil` when a proposal is built, and
`src/api/services/deep_identification_proposal.go::Apply()` only ever writes
fields whose **persisted** `Accepted == true` — there is no field-filter
bypass. So a confidence-qualifying field that the owner never explicitly
touches would silently vanish from Apply if the FE only rendered it as
"accepted" without ever telling the server. Fixed entirely within
`src/web/**`: `DeepAnalysisPage.vue::onApplyProposal` now walks the proposal
immediately before calling Apply and sequentially `await`s a PATCH
(`accepted: true`) for every field still at `accepted: null` that qualifies
on confidence, before the Apply call fires. This is not a backend change —
it uses the existing `PATCH .../proposal` endpoint — but any backend agent
touching `Apply()`/`selectDeepAppliedFieldNames` should know the FE now
depends on this pre-apply PATCH sequence to make RD-3 actually take effect,
not just look right.

**No Python/Go-side acceptance constant exists to keep in sync.** Checked
per T120's instruction — Go always sets `Accepted: nil` regardless of
confidence at proposal-build time, so there is no synthesis-side default
threshold anywhere in `src/api` or `src/agent`. Nothing routed to a backend
agent on this point.

## Wire-shape note for whoever next touches `DeepReport`

`DeepSynthesis.image_hypothesis` reaches the browser **verbatim** as
snake_case — Go's report handler returns `json.RawMessage(job.ReportJSON)`
unmodified (`src/api/handlers/deep_identification.go:97`), and the pipeline
runner stores the Python `synthesis` frame's `report` payload raw
(`deep_identification_pipeline_runner.go:218`). `types/index.ts`'s
`DeepReport.image_hypothesis` is deliberately named to match that wire shape
rather than following this interface's otherwise-camelCase convention —
documented inline with a comment so a future edit doesn't "fix" it into
`imageHypothesis` and silently break the panel.

## Verification

- `npm.cmd run type-check` (vue-tsc --build): exit 0.
- `npx.cmd vitest run`: **131 files / 842 tests passed** (baseline 131/830 —
  the +12 delta is exactly the new assertions in `DeepProposalEditor.test.ts`
  and `DeepReportPanel.test.ts`).
- Tasks T089, T090, T091, T092, T120 ticked in
  `specs/351-vision-first-deep-identification/tasks.md` via targeted string
  edits.

## Scope respected

Only `src/web/**` touched. No Go or Python files were edited.

## Follow-up (2026-08-18): pre-Apply finalize PATCH was reworked after code review

Code review caught that the sequential-PATCH loop described above had a
silent-failure hole: `useDeepIdentification.ts::updateProposalField` catches
its own error, sets `error.value`, and **returns `null` without throwing**.
The original `onApplyProposal` loop `await`ed each call but never checked the
return value, so a single failed PATCH would be swallowed and execution
would fall straight through to `applyProposal`, applying a partial proposal
while reporting success — precisely the "fails quietly, reports nothing"
defect class this entire feature effort exists to eliminate.

Fixed, still entirely within `src/web/**`:

- Confirmed `src/api/services/deep_identification_proposal.go::UpdateProposal`
  (lines 176-208) already validates every field name in an incoming
  multi-key `fields` map up front and commits all edits in one atomic
  document save — so batching is both safe and strictly better than N
  sequential PATCHes (also fixes atomicity: no more half-written proposal on
  a mid-loop failure).
- Added `updateProposalFields(jobId, edits)` to `useDeepIdentification.ts` —
  same optimistic-update/error-swallow-and-return-`null` contract as the
  single-field version, but sends every qualifying field in one PATCH.
- Rewrote `onApplyProposal` to build one `edits` map (still gated on
  `entry.accepted === null` to preserve the explicit-user-rejection guard),
  send it in a single `updateProposalFields` call, and **return early with a
  visible `applyError` message, without calling `applyProposal`**, if that
  call returns `null`.
- Added a regression test in `DeepAnalysisPage.test.ts` that mocks
  `patchDeepIdentificationProposal` to reject and asserts
  `applyDeepIdentificationProposal` is never called, no navigation occurs,
  and the failure message renders.

Verified: `npm.cmd run type-check` exit 0; `npx.cmd vitest run` **131 files /
843 tests passed** (up from 131/842 — the +1 is the new guard test).



## 2026-08-17 — Deep Analysis: `job_at_capacity` (409) UX (Aurelia, requested by Brian, paired with Cassius backend fix)

**Context:** Deep Analysis's `MaxActivePerUser` (default 1) had a wrong-data bug: submitting for coin B while coin A's analysis was still running returned HTTP 200 with a fully populated report/proposal *for coin A*, flagged `reused=true`, and the UI rendered it as a normal success — the user was silently shown a finished report for the wrong coin. Cassius changed the backend contract for this specific case (a genuinely different in-flight job at capacity) to `409 Conflict`, `code: "job_at_capacity"`, with a user-legible server message. Idempotent duplicate submissions (same images resubmitted) are unchanged — still 200/`reused:true`.

**What changed (frontend only, `src/web/`):**
- `src/web/src/api/client.ts` — added `getApiErrorCode(error)`, a sibling to the existing `getApiErrorMessage(error)`, extracting the backend's `code` field from `response.data` for branching UI behavior (message extraction logic itself is untouched).
- `src/web/src/composables/useDeepIdentification.ts` — `start()` now also sets a new `errorCode` ref via `getApiErrorCode`; on `job_at_capacity` the fallback text (used only if the server sends no message) is `"An analysis is already running. Wait for it to finish or cancel it."` The existing `error`/`starting` reset-in-`finally` behavior is unchanged — this was **not** a case of a missing error path, just a missing code-aware branch.
- `src/web/src/composables/useDeepAnalysisLauncher.ts` — the existing `listDeepIdentificationJobs({ activeOnly: true })` probe (previously used only to disable the entry button) now also remembers the active job's id (`activeJobId`). `submitDeepAnalysis` sets a new `capacityConflictJobId` only when the rejection's `errorCode === 'job_at_capacity'`, after refreshing the probe so the id is current.
- `src/web/src/components/deep-identification/DeepAnalysisStartPanel.vue` — new optional `conflictJobId` prop; when set alongside `submitError`, renders a `.btn.btn-secondary.btn-sm` `RouterLink` to `/deep-analysis/:id` reading "View running analysis", next to the error text.
- `CoinActionsPanel.vue` and `CoinLookupPage.vue` (the two call sites sharing the panel) each pass `:conflict-job-id="launcher.capacityConflictJobId.value"`.

**What the user now sees on a 409:** the modal stays open (never silently closes or navigates), the submit spinner clears (no stuck loading state — `starting` already reset in a `finally`, unchanged), the server's own message renders (or the local fallback above if the server sends none), and a "View running analysis" link appears pointing at the job that is actually running. That job's own page (`/deep-analysis/:id`) already has the Cancel control (`DeepAnalysisProgressTimeline`) — so the user has a fully worked path to either wait or cancel without a second cancel control being added to the modal itself.

**Deliberately NOT done:** a duplicate/inline cancel button inside the start-panel modal. The existing job page already owns cancel; wiring a second cancel affordance into the modal would mean either duplicating `useDeepIdentification.cancel()` wiring in two more places or threading the conflicting job's full state into the panel just to support one button — not a clean small addition per Principle IV, so left as a navigation link to the existing affordance instead.

## Alignment
- Principle IV: reused `getApiErrorMessage`'s extraction pattern for the new `getApiErrorCode` helper; reused the already-existing `listDeepIdentificationJobs({activeOnly:true})` probe instead of adding a new endpoint call; did not add a second cancel affordance where a navigation link to the existing one suffices.
- Principle V/IX: no hardcoded colors/spacing (`.btn .btn-secondary .btn-sm` global classes only), no emoji, dark theme default preserved, 44px+ touch target inherited from `.btn-sm` sizing already in use elsewhere in this panel.
- §17 Quality Gate: `npm.cmd run type-check` exit 0; full `npx.cmd vitest run` 131 files / 848 tests passing (baseline 131/843, +5 new: 2 in `useDeepIdentification.test.ts`, 2 in `DeepAnalysisStartPanel.test.ts`, 1 in `CoinLookupPage.test.ts`).


## 2026-08-17 — Feature 351 Phase 9 T068/T069: the Maximinus regression gate

**Status: T068 and T069 PASS.** `src/agent/tests/test_deep_identification_maximinus.py`
(new, Python-only, per my charter's scope boundary — did not touch any Go
file; T070/T071 are being handled concurrently by another agent) drives the
named Maximinus fixture end to end through the real production entry point
`run_deep_identification_stream` — never `synthesize()` or a bare node
function directly.

**T068** replays: two legible face images, empty notes, **empty quick
evidence** (the cruel part — the vision hypothesis is the only source of
truth in this run), all three automatable providers (numista, nomisma,
ocre) stubbed to `no_match` at the real `ProviderToolsClient` HTTP
boundary, NGC `not_automated`. Confirmed, per spec.md US2's before/after
table:
1. Narrative is not `FALLBACK_NARRATIVE_NO_EVIDENCE` and names both ruler
   ("Maximinus I (Thrax)") and denomination ("Denarius").
2. `proposed_fields` has ≥4 entries.
3. Every entry carries `evidence_refs: [{"provider": "image"}]`.
4. No provider call carried a placeholder query — numista/nomisma were
   called with the real hypothesis-derived text ("Maximinus I (Thrax)
   Denarius"); OCRE made zero calls at all (its bound signals come only
   from `quick_evidence.coin_fields`, absent here — a structural
   zero-call, not a placeholder-avoidance code path).
5. `image_hypothesis` is present and populated in the emitted synthesis
   payload.

**T069** widens the same no-placeholder invariant across a 5-case corpus
in the same module spanning every `build_query_terms` precedence tier
(full hypothesis, ruler-only hypothesis, `quick_evidence.numista_query`,
notes-only, fully-empty/zero-signal) — a regex blocklist for
generic-placeholder shapes PLUS a positive check that each captured query
actually contains that fixture's own evidence substring, so a future
reintroduction using a *different* generic constant (not only the deleted
literal `"unidentified ancient coin"`) is still caught.

**Proved the gate can fail** (per the explicit instruction not to ship a
gate that cannot go red): temporarily reverted the FR-020 fallback-gating
fix in `synthesis.py` — T068 went red reproducing Brian's exact symptom.
Separately, reintroduced a literal placeholder default in
`query_terms.py` — the T069 `fully_empty_zero_signal` case went red.
Both production files were restored immediately after
(`git status --short` confirmed zero diff on both).

**Verification:** `ruff check app/ tests/` clean. `pytest tests/ -q` from
`src/agent` → **352 passed** (baseline 346 + 6 new), zero failures, no
flaky SSE test hit.

Tasks.md T068/T069 ticked via targeted string edits (re-read immediately
before editing; did not touch surrounding lines/other agents' in-flight
edits).

No production code was modified as part of this task — I found no defect
in the current `app/**` implementation; the gate passes cleanly against
today's code.


## 2026-08-17 — Feature 351 Phase 15 T105 (Brutus) — same-coin concurrent-submit semantics

**Task:** Close the concurrency gap at `src/api/services/deep_identification_service_test.go:1110`
(`TestDeepIdentificationService_CreateJobFromIntake_DuplicateSubmitIsIdempotent`), which only ever
calls `CreateJobFromIntake` **twice sequentially** with byte-identical images. That proves nothing
about two goroutines racing into `CreateJobFromIntake` for the same coin with **different** image
bytes — the exact shape of bug this feature has repeatedly shipped (the prior SQLITE_BUSY outage
came from a read→write lock upgrade inside a deferred transaction).

New file (as instructed, to avoid colliding with another agent editing the existing test file):
`src/api/services/deep_identification_concurrency_test.go`.

## Semantics encoded (read from the current post-884f709 code, not invented)

`CreateJobFromIntake` computes `InputFingerprint` from the actual uploaded bytes
(`ComputeInputFingerprint`, keyed on `sha256(obverse)`/`sha256(reverse)` among other fields) before
touching the database, then holds `svc.intakeMu.Lock()` (a full write lock, not `RLock`) across
`StartJob` + artifact persistence — per the function's own comment, "so neither the wake signal nor
the polling fallback can claim incomplete evidence." This means production job creation is fully
serialized through one global mutex; two concurrent submissions never race the DB directly, they
race for entry into that critical section, and the second one's dedupe check runs strictly after
the first one's is durably committed.

Given that ordering, `StartJob`'s own doc comment states the contract, which I encoded as three
assertions under real goroutines + `sync.WaitGroup` (no artificial serialization added by the test):

1. **Distinct fingerprints, both under `MaxActivePerUser`** → both submissions are admitted as two
   separate job rows, each `reused=false`, with different `InputFingerprint` values.
2. **Distinct fingerprints, but the user is already at `MaxActivePerUser`** → exactly one
   submission creates the job; the other is bounced back to that *same* job with `reused=true`,
   even though its own fingerprint doesn't match anything on that job. This is intentional
   FR-007 behavior ("returns ... the user's existing active job if they are already at their
   concurrency limit") — it means "you're at capacity, here's your job in flight," not a content
   dedupe. Documenting this explicitly matters because it is easy to misread `reused=true` as "we
   recognized this exact image," which it does not mean in this branch.
3. **Byte-identical concurrent submissions** (the real-race version of the existing sequential
   test) still dedupe to exactly one job row, with exactly one of the two goroutines reporting
   `reused=true`.

All three subtests passed cleanly, including 10 repeated `-count=10` runs of the whole test with no
flakiness observed.

## Falsification (both required, both succeeded)

1. **Broke the fingerprint comparison** in `repository.DeepIdentificationRepository.CreateJob` by
   dropping `AND input_fingerprint = ?` from the dedupe `WHERE` clause (so any active job for the
   user is treated as a match regardless of content). Subtest 1 went RED immediately
   ("submission 0 unexpectedly reused an existing job; distinct image bytes must produce distinct
   fingerprints under the per-user cap"). Restored the exact original line; `git status --short`
   showed zero diff; test went GREEN again.
2. **Broke the `MaxActivePerUser` gate** in `services.DeepIdentificationService.StartJob` by
   short-circuiting `if activeCount >= int64(settings.MaxActivePerUser)` to `if false && ...`
   (never enforced). Subtest 2 went RED immediately ("expected exactly one of the two racing
   submissions to be marked reused under MaxActivePerUser=1, got 0 reused"). Restored the exact
   original line; `git status --short` showed zero diff; test went GREEN again.

Both falsifications prove the new assertions are load-bearing, not decorative.

## Gates run (verbatim results)

- `go build ./...` — clean, no output.
- `go vet ./...` — clean, no output.
- `go test -count=1 ./...` — all 10 packages `ok` (root, capture, database, handlers, integration,
  middleware, models, repository, services, testutil).
- `go test -run TestArchitecture ./...` — `ok` across all packages.
- `go test -run TestNoDirectDatabaseImports .` — `ok`.

## What was NOT verified

**`-race` was not run.** This environment has no CGO/C compiler, so the Go race detector is
unavailable here, same limitation noted in prior 341/343 reviews. The new tests are written to be
race-meaningful when CI does run `-race`: real goroutines, a real `sync.WaitGroup`, a shared
`start` channel used only to align goroutine launch (not to serialize the calls under test), and no
artificial mutex/sleep inserted by the test itself — any race in `StartJob`/`CreateJobFromIntake`
(e.g. a future regression that narrows `intakeMu` to `RLock`, or moves the fingerprint check outside
the lock) would be free to manifest and should trip `-race` in CI. This is a limitation of the local
environment, not a claim that the race dimension has been verified.

## No defect found requiring routing

Unlike some prior QC cycles, I did not find the current same-coin-different-bytes behavior to be
incoherent or wrong — `intakeMu`'s full serialization plus the fingerprint-keyed dedupe plus the
documented `MaxActivePerUser` capacity-reuse fallback together form a defensible, already-intentional
design. Nothing here needs routing to Cassius or Maximus; T105 is closed by tests only, no product
code was changed (both falsification edits were reverted exactly).


# Cassius — T107 contract-drift test (DeepIdentifyRequest/DeepSynthesis)

## What I built

New, standalone file `src/api/services/deep_identification_contract_drift_test.go`
(no forbidden files touched: `deep_identification_service.go`,
`deep_identification_service_test.go`, `deep_identification_pipeline_runner.go`,
and `src/api/integration/` were left untouched). Three tests:

- `TestDeepIdentifyRequestContractFieldsMatchPython`
- `TestDeepSynthesisProposedFieldContractMatchesPython`
- `TestDeepSynthesisKnownTopLevelFieldsMatchPython`

## Comparison mechanism

**Python side**: a checked-in JSON Schema fixture,
`src/api/services/testdata/deep_identify_contract_schema.json`, produced by
calling `DeepIdentifyRequest.model_json_schema()` and
`DeepSynthesis.model_json_schema()` on the *actual* Pydantic models — never a
hand-transcribed field list. Regeneration command is documented in the test
file's header comment.

**Go side**: read via `reflect.TypeOf` + `json` struct tags on the real,
shipped mirror structs — `DeepIdentifyProxyRequest`, `DeepIdentifyImageProxy`,
`DeepProviderCatalogEntryProxy`, `DeepIdentifyBoundsProxy`,
`DeepQuickEvidenceProxy`, `DeepQuickEvidenceNGCProxy`, `LLMConfig`, and
`deepSynthesisProposedField` (+ its nested anonymous `evidence_refs` element
type, resolved via `reflect.Type.Elem()` rather than duplicated). No test
hardcodes a duplicate Go field list.

**Nullability convention**: a Go field counts as nullable iff its type is a
pointer (the convention this codebase already uses, e.g. `NGC
*DeepQuickEvidenceNGCProxy`); a Python field counts as nullable iff its JSON
Schema property is `{"type": "null"}` or has an `anyOf` member of that type
(i.e. `Optional[X]`/`X | None`). Field-name presence is checked
symmetrically in both directions — a Go field with no Python counterpart
fails just as loudly as the reverse.

## What is NOT fully mechanical, and why

Go has **no single named struct that types the entire `DeepSynthesis`
contract**. By design, `deep_identification_pipeline_runner.go`'s
`buildDeepProposalDocumentJSON` and `deep_identification_frame_translator.go`'s
`handleSynthesis` treat most of the terminal synthesis report as an opaque
`json.RawMessage` pass-through to the persisted event/report and the
frontend, typing out only `narrative`/`proposed_fields` (+ nested
`evidence_refs`) and `partial_success` into narrow, private structs used to
build the owner-facing proposal.

Where Go *does* have a reflectable struct (`deepSynthesisProposedField` and
its nested `evidence_refs` entry), the test compares it mechanically against
Pydantic's `ProposedFieldValue`/`EvidenceRef` models
(`TestDeepSynthesisProposedFieldContractMatchesPython`).

Where it does not (`disagreements`, `unresolved_questions`, `coverage`,
`attributions`, `image_hypothesis`, and `partial_success`'s own anonymous
struct plus the `narrative`/`proposed_fields` wrapper key names themselves),
the test falls back to `TestDeepSynthesisKnownTopLevelFieldsMatchPython`: a
pinned, hand-maintained snapshot of the exact top-level property set read
from the schema fixture. This is the **one deliberate non-mechanical
exception** in this file, and it exists only because there is no Go type to
reflect over for those fields — it still turns red on any
rename/addition/removal, but a human must consciously update the pinned list
(and check whether Go's opaque pass-through and the frontend's TypeScript
mirror still agree), rather than a future Go struct change updating it for
free.

This test also **cannot detect drift introduced between fixture
regenerations** — if a Pydantic model changes and nobody regenerates
`testdata/deep_identify_contract_schema.json`, this test will still pass
against the stale fixture. That is the deliberate trade-off of not adding a
live Python dependency (or a running Python process) to the Go test path per
the task's hard constraints (fast, hermetic, always-on in CI, no live
service). T106 (Maximus, live SSE round trip in the same batch) is the
complementary test that would catch drift that has not yet been
regenerated into this fixture.

`schema_version`, `call_budget`, `hint_kind` are present on both sides and
are not flagged as drift (per the task's explicit instruction) — they simply
compare equal since both sides define them identically today.

## Coverage of the five T093 drift points

T093 fixed five documentation drift points in
`specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`:

1. **§1** `Mint(userID)`/`InternalTokenRequired` → `MintForJob(userID, jobID)`/
   `InternalJobTokenRequired` — a token-minting *function signature*, not a
   field of `DeepIdentifyRequest`/`DeepSynthesis`. **NOT caught** — out of
   scope for a field-name/nullability comparison of these two models.
2. **§2** `llm_config` → `llm` (a `DeepIdentifyRequest` field rename).
   **CAUGHT** — mechanically, by
   `TestDeepIdentifyRequestContractFieldsMatchPython`. This is the exact
   field I used for the falsification test below.
3. **§2** deletion of the never-real `quick_evidence.numista_evidence` line
   (`QuickEvidence` is `extra="forbid"` and would reject it). **CAUGHT** as a
   side effect — the request-side comparison flags any Go field with no
   matching Pydantic property, which is exactly the shape this drift would
   take if it reappeared.
4. **§3** `evaluation` frame payload → `{disagreement_count,
   resolved_count}` plus the missing `synthesis_started` row — an SSE
   *frame* contract (`contracts/sse-events.md`), not a field of
   `DeepIdentifyRequest`/`DeepSynthesis`. **NOT caught** — out of scope.
5. **§5** add `attributions` to the `DeepSynthesis` example. **Partially
   caught** — via the documented non-mechanical fallback
   (`TestDeepSynthesisKnownTopLevelFieldsMatchPython`), since Go has no
   struct typing this field.
6. **§7** add the `ocre_search` row — a tool/provider-catalog documentation
   table entry, not a field of either model. **NOT caught** — out of scope.

**Summary**: 2 of 5 points are directly, mechanically caught (#2, #5), #3 is
caught as a side effect, and #1/#4/#6 live on wire surfaces (token minting,
SSE frame shape, tool catalog docs) that a `DeepIdentifyRequest`/
`DeepSynthesis` field-drift test is not positioned to cover. A complete
guard against all six historical drift points would need at least two more
purpose-built tests: one for the internal-token minting signature, and one
for the SSE `evaluation`/`synthesis_started` frame shapes (T106's live round
trip likely covers the latter in practice, but not as a hermetic unit test).

## Falsification evidence

Temporarily renamed `DeepIdentifyProxyRequest.LLM`'s json tag from `llm` to
`llm_config` in `agent_proxy_deep_identify.go` — deliberately recreating the
exact T093 drift point #2.

- **RED**: `TestDeepIdentifyRequestContractFieldsMatchPython` failed with:
  `DeepIdentifyRequest: Python field "llm" has no matching Go field (json
  tag) - Go mirror struct has drifted behind the Pydantic model` and
  `DeepIdentifyRequest: Go field "llm_config" (json tag) has no matching
  Python property - Go mirror struct has drifted ahead of the Pydantic
  model (or is sending a field Pydantic's extra="forbid" would reject)`.
- Restored the field exactly; `git diff --stat -- src/api/services/agent_proxy_deep_identify.go`
  showed no diff.
- **GREEN**: re-ran the same test — passed.

## Validation (verbatim, from `src/api/`)

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -count=1 ./...` — all 10 packages `ok` (root, capture, config
  (no tests), database, docs (no tests), handlers, integration, middleware,
  models, repository, services, testutil).
- `go test -run TestArchitecture ./...` — pass.
- `go test -run TestNoDirectDatabaseImports .` — pass.

No Python files were changed; the checked-in schema fixture was generated
once from the existing, unmodified `src/agent` Pydantic models, so no Python
gates were re-run beyond the one-off generation.

## Follow-up: staleness gap closed on the Python side

The gap I flagged above — "this test cannot detect drift introduced
between fixture regenerations" — turned out to be a live risk in practice,
not just a theoretical one: T106 (the live Go<->Python SSE round trip) is
CI-excluded/tagged, so it does **not** run on every default `pytest`
invocation and cannot be relied on to catch a stale fixture on the default
path. Without a Python-side guard, the fixture could silently drift behind
the real Pydantic models forever while the Go test kept passing against the
stale snapshot — exactly the "looks green while the contract has drifted"
failure mode T107 exists to prevent.

Closed it with a new, hermetic pytest,
`src/agent/tests/test_contract_schema_fixture_is_current.py`, which
recomputes `DeepIdentifyRequest.model_json_schema()`/
`DeepSynthesis.model_json_schema()` on every `pytest` run and asserts the
result is byte-for-byte identical (`json.dumps(..., sort_keys=True)`,
excluding only the `_generated_by` provenance key) to the checked-in
fixture. It resolves the fixture path relative to the test file itself
(never cwd) and skips with a clear message rather than failing if the
expected repo layout can't be found. On failure it prints the exact
regeneration command from this test's own header, so whoever hits it knows
exactly what to run.

**Falsification**: renamed `DeepSynthesis.attributions` -> `attribution_list`
in `src/agent/app/models/responses.py` (the same field named in T093 drift
point #5) — confirmed RED with the regeneration command in the failure
message, restored exactly (`git diff --stat` clean), confirmed GREEN.

**Gate** (from `src/agent/`): `ruff check app/ tests/` clean;
`python -m pytest tests/ -q` -> 351 passed (baseline 350 + 1 new test);
`test_deep_identification_maximinus.py` all 6 passed in isolation.

With this in place, the chain is closed on both sides without needing a
live service: the Go test catches Go drifting from the fixture, and this
new Python test catches the fixture drifting from Python — no live
Python/Go process required for either.


### Decision: Feature 351 Phase 14 (T100, T101 Python portion, T102) — Python dead-code removal

**Date:** 2026-08-17
**Agent:** Cassius (Backend Developer)
**Branch:** `beta`
**Status:** T100 complete, T101 partial (Python only — Go/web deferred), T102 complete

## Context

Phase 14 was deliberately sequenced last so the cleanup moves settled code, not
code still in flight (Maximus concurrently refactoring
`deep_identification_service.go`/`deep_identification_pipeline_runner.go`;
Aurelia concurrently in `src/web/**`). This batch covers only the Python-side
items: T100 (`build_graph`), the Python half of T101 (`numista_detail()`), and
T102 (unread request/state fields).

## T100 — `build_graph()` handled honestly, not just deleted

`src/agent/app/teams/deep_identification/graph.py::build_graph` was test-only:
production always ran the hand-written async generator
`run_deep_identification_stream`, never the compiled `StateGraph`. The old
`tests/test_deep_identification_graph_topology.py::test_build_graph_has_expected_node_topology`
therefore asserted node/edge shape on a graph nobody executes — passing while
proving nothing about production, the same class of defect (something fully
tested that production never ran) that caused the original outage.

**Order followed exactly as instructed:**
1. Rewrote the topology test **first** to drive
   `run_deep_identification_stream` itself (LLM calls faked via
   `monkeypatch.setattr(graph_module, "get_chat_model", ...)` and
   `hypothesis_module.get_structured_model`, the numista provider node
   swapped for a fake — same pattern already used by
   `test_deep_identification_sse.py`) and assert its emitted
   progress/stage frames occur in exactly
   `prepare_evidence -> router -> provider_fanout -> evaluator -> synthesizer`
   order (via the `image_evidence_ready`/`vision_completed` ->
   `router_selected` -> `provider_fanout_started`/`provider_started`/
   `provider_result` -> `evaluation_started`/`evaluation` ->
   `synthesis_started`/`synthesis` stage markers already present on every
   real production run).
2. Ran the new test in isolation and confirmed it passed against the
   still-present `build_graph`.
3. Only then deleted `build_graph` (and the now-unused `StateGraph`/`END`
   imports from `langgraph.graph`) from `graph.py`, and the two tests that
   only exercised `build_graph`'s own `recursion_limit`-binding mechanism
   (`test_build_graph_binds_recursion_limit_into_invocation_config`,
   `test_build_graph_without_recursion_limit_is_unbound`) — that binding has
   no production equivalent (the streaming driver never calls
   `.ainvoke`/`.astream` on a compiled graph), so there was nothing left to
   preserve for those two.

Net test count for that file: 3 build_graph-based tests -> 1 new
production-based test (the 4 attribution tests in the same file are
untouched). This accounts for the full -2 in the full-suite count below.

## T101 (Python portion) — `numista_detail()` removed

**Grep proof of zero call sites** (ran before removal):

```
grep -rn "numista_detail" src/agent
```

Result — only the method's own definition and docstring:
```
src/agent/app/tools/provider_tools.py:67:    async def numista_detail(self, numista_id: int) -> dict:
src/agent/app/tools/provider_tools.py:68:        """POST /api/internal/tools/numista_detail — {status, candidate, identifier}."""
src/agent/app/tools/provider_tools.py:69:        return await self._post("numista_detail", {"id": numista_id})
```

No provider node, test, or router calls `ProviderToolsClient.numista_detail`
anywhere in `src/agent`. Removed the method entirely.

**Not touched (Go/web, explicitly out of scope this batch):** `RPCEnabled`,
`listDeepIdentificationJobs`, `stream.reset()`, `DeepStreamTruncatedPayload`.
Noted for the record: the Go `/api/internal/tools/numista_detail` HTTP
endpoint (`internal_tools.go::NumistaDetailRequest`/`NumistaDetail`, wired in
`main.go:835`) still exists server-side and is unaffected by this
change — it isn't one of T101's named Go items, so it's out of scope, but
flagging it here in case a future pass wants to trace whether it now has any
real caller either.

## T102 — kept wire fields, removed genuinely-internal ones

Verified each field's actual status by grepping both the Python codebase and
the contract document before deciding, per the brief's instruction not to
assume the task text was still accurate.

**Kept and documented as forward-compatibility placeholders** (all three
appear in `specs/344-deep-agentic-coin-identification/contracts/
agent-internal-contract.md` §2 as fields Go actively sends on every request):

- `DeepIdentifyImage.hint_kind` (`src/agent/app/models/requests.py`) —
  contract §2 example: `{"role": "hint", ..., "hint_kind": "label"}`. Zero
  Python reads (`grep -rn "\.hint_kind" src/agent` → no matches outside the
  field declaration).
- `DeepProviderCatalogEntry.call_budget` (`src/agent/app/models/requests.py`)
  — contract §2 example: `{"provider": "numista", ..., "call_budget": 4}`.
  Zero Python reads (`grep -rn "entry.call_budget|catalog_entry.call_budget"
  src/agent` → no matches; only test-fixture constructor calls that *set* it).
- `DeepIdentifyRequest.schema_version` (`src/agent/app/models/requests.py`) —
  contract §2 example: `{"schema_version": 1, ...}`, also referenced as a
  versioned-contract field in `specs/344.../plan.md` (Principle III: Strict
  Types). Zero Python reads.

Each now carries an inline comment stating plainly that it is accepted from
Go but not yet branched on by Python logic, so a future reader doesn't
mistake "unread" for "unused/removable" again. No Go mirror change was made
or needed — the fields are simply documented in place, matching the task's
own stated preference ("keep wire fields and document them as
forward-compatibility placeholders; remove only the genuinely internal
ones").

**Removed as genuinely internal, zero-dependency dead declarations**
(`src/agent/app/teams/deep_identification/state.py`):

- `DeepIdentificationState.errors: Annotated[list[str], operator.add]` —
  `grep -rn "errors" src/agent/app/teams/deep_identification` found only the
  field's own declaration; nothing ever wrote `state["errors"]` or read it.
  No wire-contract mirror exists for a bare `state.errors` — it was never a
  request/response field, only an internal `TypedDict` key nobody used.
- `DeepIdentificationState.tools_base_url: str` and
  `DeepIdentificationState.internal_token: str` — these are **state**
  fields, distinct from `DeepIdentifyRequest.tools_base_url`/
  `.internal_token` (which *are* real, actively-used wire fields — Go sends
  them, and `run_deep_identification_stream` builds its `ProviderToolsClient`
  straight from `request.tools_base_url`/`request.internal_token`, confirmed
  at `graph.py` lines 409-410). The **state dict construction** in
  `run_deep_identification_stream` never populates `state["tools_base_url"]`
  or `state["internal_token"]` at all — the `TypedDict` declared keys that no
  code ever set or read. Removing them changes zero runtime behavior; the
  request-level fields of the same name are untouched.

No Go-side change was required or made for T102 — reported here per the
brief rather than acted on, though in this case nothing needed sequencing to
Maximus since the wire fields were kept, not removed.

## Verification

- `ruff check app/ tests/` → clean, no warnings/errors.
- `pytest tests/ -q` → **350 passed** (baseline 352). The drop of 2 is
  entirely explained by T100: 3 `build_graph`-only tests replaced by 1
  production-generator test (net -2); no other test was removed or skipped.
- `pytest tests/test_deep_identification_maximinus.py -v` → all 6 tests
  passed (the Phase 9 regression gate is green).
- The known-flaky `test_deep_identification_sse.py` timing test did not
  fail in this run's full-suite pass; not re-run in isolation since it did
  not fail.
- No Go files were built/tested this batch (no Go source was touched).

## Files changed

- `src/agent/app/teams/deep_identification/graph.py` — deleted `build_graph`
  and the `StateGraph`/`END` imports it alone used; updated module/function
  docstrings that referenced it.
- `src/agent/tests/test_deep_identification_graph_topology.py` — replaced the
  3 `build_graph`-based tests with 1 production-generator-based test;
  attribution tests in the same file untouched.
- `src/agent/app/tools/provider_tools.py` — removed `numista_detail()`.
- `src/agent/app/models/requests.py` — documented `hint_kind`, `call_budget`,
  `schema_version` in place as forward-compatibility placeholders (no
  removal).
- `src/agent/app/teams/deep_identification/state.py` — removed `errors`,
  `tools_base_url`, `internal_token` `TypedDict` fields; added a module
  docstring note explaining why.
- `specs/351-vision-first-deep-identification/tasks.md` — T100 and T102
  ticked complete; T101 left unticked with a note that only the Python
  portion (`numista_detail()`) is done and the four Go/web items remain for
  Maximus/Aurelia to sequence.


### Decision: SQLite Concurrency DSN — Single Source of Truth (Drift Guard)

**Date:** 2026-08-17
**Agent:** Cassius (Backend Dev)
**Branch:** `beta`
**Status:** RESOLVED — uncommitted, ready for review/commit by requester

## Context

The SQLITE_BUSY outage fix — `_txlock=immediate&_pragma=busy_timeout(5000)`
appended to the SQLite DSN — was duplicated verbatim in two places that
cannot legally import each other:

- `src/api/database/database.go` (`database.Connect`, production)
- `src/api/repository/deep_identification_repository_test.go`
  (`newDeepIdentificationFileTestDB`, the concurrency regression test)

`repository/` must not import `database/` (Principle I/II, enforced by
`architecture_test.go`'s `TestNoDirectDatabaseImports`). The two copies were
kept in sync only by a code comment asking future editors to remember. That
means the regression test that exists specifically to catch this class of
bug could not actually detect it: if someone removed `_txlock=immediate`
from production, the test's own hardcoded copy of the string would keep
passing, because it was never derived from production's value in the first
place.

## Decision

Added `models.SQLiteConcurrencyDSNParams` (`src/api/models/sqlite_config.go`)
as the single definition of the DSN parameter string. Both consumers now
read this constant instead of hand-copying the literal:

- `database/database.go`: `dsn := dbPath + "?" + models.SQLiteConcurrencyDSNParams`
- `repository/deep_identification_repository_test.go`:
  `newDeepIdentificationFileTestDB(t, models.SQLiteConcurrencyDSNParams)`

## Why `models/` is the only import-legal placement

Read `architecture_test.go` first, specifically `TestPackageImportMatrix`:

- `repository/`'s allowed **internal** import for production (non-test)
  files is `models` — and only `models`. Any new dedicated package (e.g. a
  hypothetical `dbconfig`) would violate that matrix for `repository/`'s
  own non-test code, even though today no non-test `repository/` file needs
  this constant.
- `models/` is documented (and enforced) as stdlib-only, and is importable
  by every layer (`handlers`, `services`, `repository`, `database`, plus
  `main`). A plain exported `const` string requires no imports itself, so
  adding it to `models/` doesn't violate that stdlib-only constraint.
- `TestNoDirectDatabaseImports` explicitly forbids importing
  `github.com/briandenicola/ancient-coins-api/database` from `repository/`
  — including test files, since that check does **not** skip `_test.go`
  (unlike `TestPackageImportMatrix`, which explicitly exempts test files
  from the internal-import allowlist). So the test file cannot resolve this
  by importing `database/` even opportunistically.

Given those two constraints together, `models/` was the only fully
import-legal home for a value shared between `database/` production code
and a `repository/` test. No other placement was viable without either
violating the import matrix or the direct-import ban, so this wasn't a
close call between multiple valid options.

## Other hand-built SQLite DSNs — left alone, and why

Grepped the whole `src/api` tree for `sqlite.Open(`. Every other occurrence
(~140 call sites across `handlers/`, `services/`, `repository/`, `database/`
test files, `integration/`, `testutil/`) uses either:

- Plain `:memory:`, or
- `file:<name>?mode=memory&cache=shared` (unique per-test in-memory shared
  cache, keyed by `time.Now().UnixNano()` or an atomic counter)

None of these use `_txlock=immediate` or `_pragma=busy_timeout(...)` — they
don't need to, because in-memory SQLite has no real file-lock contention to
guard against (the concurrency defect this DSN fixes only reproduces against
a real on-disk WAL file, per the comment on
`newDeepIdentificationFileTestDB`). These were deliberately left untouched:
folding them into `models.SQLiteConcurrencyDSNParams` would be a no-op at
best and would misleadingly suggest they exercise the same locking
behavior, when they structurally cannot. Only
`deep_identification_repository_test.go`'s file-backed test DB shares
production's real concurrency semantics, so it's the only place that needed
this guard.

## Falsification (proof the guard works)

1. Temporarily changed `models.SQLiteConcurrencyDSNParams` to drop
   `_txlock=immediate` (kept only `_pragma=busy_timeout(5000)`).
2. Ran `TestDeepIdentificationRepository_ConcurrentClaimNoLockContention`
   (`repository/`) — **went RED**: 19 of 150 concurrent claims failed with
   `database is locked (5) (SQLITE_BUSY)` (test asserts zero errors).
3. Restored the constant to `_txlock=immediate&_pragma=busy_timeout(5000)`.
4. Reran `go test -count=1 ./...` — **GREEN**, 10/10 packages (`ok` for
   root, `capture`, `database`, `handlers`, `integration`, `middleware`,
   `models`, `repository`, `services`, `testutil`; `config`/`docs` have no
   test files).

This proves the shared constant is a real drift guard, not cosmetic
de-duplication: a silent regression in the DSN parameters now fails a test
that both production and the test consume from the same source.

## Gates run (verbatim summary)

- `go build ./...` — clean, no output
- `go vet ./...` — clean, no output
- `go test -count=1 ./...` — 10/10 packages `ok`
- `go test -run TestArchitecture .` — `PASS` (all 5 subtests, including
  `package_import_matrix` and `no_direct_database_imports`)
- `go test -run TestNoDirectDatabaseImports .` — `PASS`

## Files changed

- `src/api/models/sqlite_config.go` (new) — `SQLiteConcurrencyDSNParams` const
- `src/api/database/database.go` — builds DSN from the shared constant
- `src/api/repository/deep_identification_repository_test.go` — test call
  site and doc comment updated to reference the shared constant instead of
  a hand-copied literal

No runtime behavior changed: the effective DSN string is byte-identical to
before (`_txlock=immediate&_pragma=busy_timeout(5000)`), for both production
and the test.

## Alignment

Constitution Principle I (Clear Layered Architecture), Principle II
(Dependency Injection / import boundaries), Principle IV (Simple Complete
Changes — smallest change that closes the actual drift risk, no new
package, no behavior change).


### Decision: Phase 9 regression gate — image-only proposal fields never dropped (T070/T071)

**Author:** Cassius (Backend Developer)
**Date:** 2026-08-17
**Feature:** `specs/351-vision-first-deep-identification` — Phase 9 (T070, T071)
**Status:** IMPLEMENTED

## Context

Phase 9 turns Brian's Maximinus I bug (a correctly image-identified coin producing an empty/junk Deep Analysis result) into a permanent automated gate. Brutus owns the Python end-to-end fixture (T068/T069). I own the two Go companions.

## T070 — verified, no defect present

The brief asked me to verify honestly whether `buildDeepProposalDocumentJSON` (`deep_identification_pipeline_runner.go`) drops a `proposed_fields` entry whose only `evidence_refs` is `{"provider":"image"}`, since `image` is correctly skipped as a *ref* (FR-025: image is never a provider on any provider-facing surface) but that ref-level skip must not cascade into dropping the *field*.

I ran a debug probe against the real function output before writing any assertion. Result: **the field is retained.** The per-field loop always executes `fields[name] = entry` after the inner evidence-ref loop finishes, regardless of whether any evidence survived the `image`/citation-allowlist filtering. An image-only field ends up in the persisted document with `Proposed` set to the AI value and `Evidence` as an empty/nil slice — never dropped.

Added `TestDeepIdentificationProposal_ImageOnlyFieldRetainedWithEmptyEvidence` to `src/api/services/deep_identification_proposal_integration_test.go`, reusing the existing `realisticRunnerStreamFrames`/`seedDeepJobWithRunnerProposal` harness (its `notes` field is already the image-only case). Asserts the field is present, the proposed value is preserved, evidence length is 0, and the document as a whole is non-empty.

## T071 — backward-compatibility fixture

Added `TestDeepIdentificationBackwardCompatibility_PreAndPostImageHypothesisFixtures` to `src/api/services/deep_identification_pipeline_runner_test.go` with two subtests, both driven through the real `Get -> UpdateProposal (PATCH accept) -> Apply` service round trip against a saved coin:

1. A pre-351 report/proposal shape (no `image` provider anywhere, every `evidence_ref` names a real automatable-provider claim) — loads, renders, and applies with zero errors.
2. A post-351 proposal with an image-only field (`evidence_refs: [{"provider":"image"}]`) — same.

This protects Brian's already-persisted deep-identification jobs from before this feature shipped: neither shape assumption is new-row-only.

## Note for whoever owns the frontend `Claim`/proposal TS types

`deepProposalFieldEntry.Evidence` is tagged `json:"evidence,omitempty"` (`deep_identification_proposal.go`). A field with zero evidence therefore **omits** the `evidence` key from the wire JSON rather than serializing `"evidence": []`. This is immaterial on the Go side (`len(entry.Evidence) == 0` either way, field is present) and I did not change it — out of scope for T070/T071 and `src/web/**` is off-limits for this task — but if `DeepProposalEditor.vue` or its TS types assume `evidence` is always a present array, this is the exact wire shape it needs to tolerate for image-only fields.

## Verification

- `go build ./...` — exit 0
- `go vet ./...` — exit 0
- `go test -count=1 ./...` — all 10 packages `ok`, 0 FAIL (api, capture, database, handlers, integration, middleware, models, repository, services, testutil)
- Targeted: `go test ./services -run "TestDeepIdentification|TestBuildDeepProposal" -v` — all pass, including the two new tests

## Alignment

- Principle IV: no fix applied because none was needed — verified the premise before writing tests, exactly as instructed
- FR-021, FR-025, FR-033: image-only fields survive to the owner-facing proposal; `image` never appears as a provider name; backward compatibility for existing persisted jobs
- §17/§21: full local build/vet/test gate green before reporting


### Decision: Deep Analysis StartJob at-capacity, non-matching-fingerprint path stops returning a foreign job

**Date:** 2026-08-17
**Agent:** Cassius (Backend Developer)
**Requested by:** Brian (@briandenicola), explicitly approved live
**Status:** SHIPPED — approved breaking change to a shipped endpoint

## Context

`DeepIdentificationService.StartJob` (`src/api/services/deep_identification_service.go`)
enforces `MaxActivePerUser` (default `1`). When a user was already at
capacity, the genuine-duplicate path (matching `InputFingerprint`) correctly
returned the existing job with `reused: true` — that is the documented
FR-007 idempotency contract. But the *non-matching* fingerprint branch,
reached whenever a user submitted a second, genuinely different coin while
their first job was still in flight, fell through to `ListJobs` and handed
back the user's most recent active job anyway, still marked `reused: true`.

The handler (`CreateJob`) then returned HTTP 200 with that unrelated job's
`report`/`proposal` populated, presented as the answer to the new
submission. With `MaxActivePerUser` defaulting to 1, this triggered on
nearly every second concurrent submission. This was not authorized by
FR-007 (which governs genuine duplicate consumption, not capacity), and the
Swagger/OpenAPI description only ever documented "duplicate submissions
return the existing job" — a matching-fingerprint scenario. The capacity
branch's non-matching case was undocumented contract drift as well as a
wrong-data bug.

## Decision

At capacity with a **non-matching** fingerprint, `StartJob` now returns a
new sentinel `ErrDeepJobAtCapacity` instead of substituting an unrelated
job. The genuine-duplicate (matching-fingerprint) path is unchanged and
still returns `reused: true`.

- **New sentinel:** `services.ErrDeepJobAtCapacity`
- **HTTP status:** `409 Conflict`
- **Error code:** `job_at_capacity`
- **Message:** `"An analysis is already running. Wait for it to finish or cancel it."` (Principle V: generic, no internal detail or IDs)

This is a deliberate, approved breaking change to the shipped
`POST /deep-identification/jobs` endpoint's undocumented capacity-collision
behavior. It is pinned so the frontend (Aurelia, in parallel) can build
against it without further negotiation.

## Contract naming note (as requested — deviation check)

`contracts/deep-identification.openapi.yaml`'s original planning text
already anticipated this exact scenario but called for **`429`**, not
`409`, with the description "Per-user active job limit reached (a different
in-flight job exists)". No handler code implemented this `429` — the
shipped code took the wrong-job-returned path instead, so there was no
existing `429`-mapped error code to collide with `job_at_capacity`. Per
Brian's live-pinned contract, the shipped behavior is `409`/`job_at_capacity`,
not the originally-planned `429`. I did not silently pick a different name
or status; the OpenAPI contract has been updated in place to document `409`
as the shipped behavior and explicitly notes that it supersedes the `429`
text from the original planning contract, with the date and reason
recorded inline. Flagging this explicitly per instruction, since it is a
deviation from the pre-existing (never-implemented) planning document.

## RetryJob impact (checked, as requested)

`DeepIdentificationHandler.Retry` → `DeepIdentificationService.RetryJob` →
`CreateJobFromIntake` → `StartJob`. Retry shares the exact same `StartJob`
call path as create, so a retry submitted while the user is already at
capacity with a non-matching fingerprint now also receives
`ErrDeepJobAtCapacity`, mapped through the same `respondDeepJobError`
central switch to `409 job_at_capacity`. No separate handling was needed;
behavior is coherent between create and retry with no code duplication.

## Changes Applied

- `src/api/services/deep_identification_service.go`: new `ErrDeepJobAtCapacity`
  sentinel; `StartJob`'s non-matching-fingerprint-at-capacity branch now
  returns it instead of scanning `ListJobs` for a substitute job.
- `src/api/handlers/deep_identification.go`: `respondDeepJobError` maps the
  sentinel to `409`/`job_at_capacity`/generic message; Swagger annotation and
  description on `CreateJob` updated to document the `409` outcome and stop
  implying reuse is the only non-2xx-adjacent outcome at capacity.
- `contracts/deep-identification.openapi.yaml`
  (`specs/344-deep-agentic-coin-identification/contracts/`): documented the
  `409 job_at_capacity` outcome, with an explicit note that it supersedes
  the never-implemented `429` text from the original planning contract.
- `src/api/docs/{docs.go,swagger.json,swagger.yaml}` and `docs/openapi.json`
  regenerated via `swag init` to keep the shipped Swagger surface in sync
  (CI enforces this; done manually here since the `task openapi` Taskfile
  target's version-bump step has a pre-existing, unrelated PowerShell
  templating bug on this environment — main.go's `@version` line was not
  touched).
- `src/api/services/deep_identification_concurrency_test.go`: third
  sub-test rewritten from asserting `reused=true` with a foreign job to
  asserting the losing racer gets `ErrDeepJobAtCapacity` with no job
  returned; doc comment above the test updated to state this is an
  approved, deliberate contract change, not a weakened assertion.
- `src/api/services/deep_identification_service_test.go`: the pre-existing
  `TestDeepIdentificationService_StartJob_QueueDepthAndPerUserLimit` also
  pinned the old behavior at the unit level (outside the file named in the
  original task) and was broken by this fix; updated for the same reason —
  tightly coupled to the change, not incidental.
- `src/api/handlers/deep_identification_test.go`: added
  `TestDeepIdentificationHandler_CreateJob_AtCapacityWithDifferentFingerprintReturns409`
  asserting the 409 status, `job_at_capacity` code, no job envelope in the
  response body, and exactly one job row created.

## Verification

- `go build ./...`, `go vet ./...`: clean.
- `go test -count=1 ./...`: all 10 packages `ok`.
- `go test -run TestArchitecture ./...` and
  `go test -run TestNoDirectDatabaseImports .`: pass.
- Falsification: reverted only the `StartJob` capacity branch back to the
  old `ListJobs`-substitution behavior (sentinel definition and handler
  mapping left in place to isolate the behavioral regression) — the new
  concurrency sub-test and the new handler test both went RED
  (`--- FAIL: .../distinct_fingerprints_at_the_per-user_cap:_one_is_admitted,_the_other_is_refused_with_ErrDeepJobAtCapacity`
  and `--- FAIL: TestDeepIdentificationHandler_CreateJob_AtCapacityWithDifferentFingerprintReturns409`).
  Restored the fix; both went GREEN again, along with the full suite.


### Decision: Deep Analysis Activity Timeline — Python emission half (FR-040)

**Date:** 2026-08-17
**Agent:** Cassius (Backend Developer)
**Branch:** `beta`
**Status:** DONE — implemented, tested, gates green. Frontend half already shipped by Aurelia (see `.squad/decisions/inbox/aurelia-activity-timeline.md`).

## Context

Brian: "Can we add additional logging and visuals for the user while the
system does its workflow?" FR-040 (amendment to FR-030, authorized by Brian
2026-08-17) narrows the `progress`-payload prohibition: the owner-scoped SSE
stream may now carry hypothesis values, application-authored query terms,
candidate counts, and per-provider outcomes — the application-log
prohibition is unchanged and absolute. This batch implements the four
emission points Aurelia's frontend timeline needs, in
`src/agent/app/teams/deep_identification/graph.py` and
`src/agent/app/teams/deep_identification/hypothesis.py`.

## What shipped (`src/agent/**` only — no Go, no web)

1. **`phase: "vision_completed"` progress frame.** New `_vision_completed_message()`
   in `graph.py` states structural facts (populated-field count, a confidence
   bucket derived from the hypothesis's own bounded per-field scores) and
   honestly reports degradation: which rung of the hypothesis ladder actually
   produced the result (`structured` / `prose` / `deterministic_fallback` /
   `no_images`). Emitted immediately after `image_evidence_ready`, before the
   router runs.

   To know the rung without changing `build_hypothesis_from_vision`'s public
   signature (17 existing tests/call-sites depend on it returning a bare
   `CoinHypothesis`), added `build_hypothesis_from_vision_traced()` in
   `hypothesis.py` returning `(CoinHypothesis, source)`; the original function
   is now a two-line wrapper that discards the tag. `graph.py`'s
   `prepare_evidence_node` calls the traced variant and stashes a new
   `hypothesis_source` key on `DeepIdentificationState` (never a claim/
   citation source, never persisted to the coin record — consumed only by
   this one progress message).

2. **Query terms on `provider_started`.** New `_provider_started_detail()` in
   `graph.py`, called from `provider_fanout_node`'s `run_and_report` right
   before the existing `on_provider_event({"type": "provider_started", ...})`
   call. Numista/nomisma get the exact deterministic query text the provider
   node is about to use (reusing the shared `query_terms.build_query_terms`,
   nomisma's 200-rune bound included) as `query_terms`, or
   `skip_reason: "insufficient_query_evidence"` when no precedence tier
   yields usable terms. OCRE (structured-field decode, not free text) and
   NGC/RPC (no automated call at all — terms of use / no public API) get a
   fixed, non-invented `detail` string instead of a fabricated query.
   Rides `provider_started` per Aurelia's stated preference — zero Go
   changes, since `deep_identification_pipeline_runner.go`'s `onFrame` has no
   reducing case for that event type (verified by reading it, not assumed).

3. **Per-provider settle (`provider_result`) — no change needed, claim
   verified.** Confirmed `on_result`/`on_provider_event("provider_result")`
   already fire synchronously inside each per-provider task as soon as that
   task's own `_run_one_provider` await resolves — `asyncio.gather` only
   waits for every task to finish, it does not delay any task's own internal
   side effects, so results were already streaming live, not batched.
   Added `test_provider_result_frames_are_emitted_live_not_batched` (an
   artificially slow numista raced against near-instant ngc/ocre/rpc) to
   prove this through the real stream rather than trust the brief's claim.

4. **`synthesis_started` detail.** New `_synthesis_started_message()` reports
   the contributing-provider count and whether image evidence also feeds
   synthesis (counts/structural facts only, never claim content). Added as a
   `message` field directly on the existing `synthesis_started` frame —
   which, like `provider_started`, is raw-passthrough on the Go side
   (`deepPipelineEventType` maps it straight through with no `onFrame`
   reducing case), so this needed zero Go changes either.

## Correction to the task brief (flagged per instructions, not silently routed around)

The brief asserted both `provider_started` **and** `provider_result` are
raw-passthrough in `deep_identification_pipeline_runner.go`. Verified
directly: `provider_started` genuinely is. `provider_result` is **not** — its
`case` in `onFrame`'s switch reassigns `persistPayload` via
`deepProviderResultPublicPayloadJSON`, reducing it to a fixed six-field
bounded shape (`provider, status, confidence, claimCount, errorKind,
linkOut`). This did not block item 3 (that shape already carries everything
needed, including `errorKind` for `insufficient_query_evidence`), but the
premise itself was wrong and is recorded here explicitly.

## Binding limits honored

- No hypothesis value, query term, or legend text was added to any
  `logger.*`/Python `logging` call — the application-log prohibition is
  untouched. Verified with a new `caplog`-based test driving the real stream.
- No image data (data URIs, bytes, base64) appears in any new field.
- Every new message is a fixed template with bounded, typed inputs (int
  counts, a small fixed vocabulary of source/status tags) or an explicitly
  truncated/bounded string (`query_terms[:300]`, `nomisma` additionally
  bounded to 200 runes matching its Go client) — no unbounded upstream string
  is ever interpolated directly.
- `"image"` never appears as a `provider_started`/`provider_result` provider
  value in any new code path.
- Detail flows only through the existing owner-scoped stream/event
  vocabulary (`progress`, `provider_started`, `synthesis_started`) — no new
  shared/admin/aggregate surface was touched or introduced.

## Tests

Nine new tests added to `src/agent/tests/test_deep_identification_sse.py`,
all driving the real `run_deep_identification_stream` entry point (never an
emission helper directly), following the file's existing
`test_evaluator_node_receives_hypothesis_via_the_real_graph_path` pattern so
a helper that exists but isn't actually wired into the pipeline cannot pass
silently:

- `test_vision_completed_progress_reports_field_count_and_degradation`
- `test_vision_completed_reports_empty_hypothesis_honestly`
- `test_provider_started_carries_query_terms`
- `test_provider_started_surfaces_insufficient_query_evidence_skip_reason`
- `test_provider_started_static_detail_for_non_query_providers`
- `test_provider_result_frames_are_emitted_live_not_batched`
- `test_synthesis_started_reports_contributing_counts`
- `test_synthesis_started_omits_image_evidence_when_hypothesis_empty`
- `test_fr040_hypothesis_and_query_detail_never_reach_application_logs`
  (the FR-040 log-boundary regression test, using `caplog`)

## Verification (actual observed numbers)

- `ruff check app/ tests/` (from `src/agent/`): **clean**.
- `pytest tests/ -q` (from `src/agent/`): **346 passed** — up from the stated
  baseline of 337 (the delta is exactly the 9 new tests above; re-ran
  `test_deep_identification_sse.py` in isolation twice more to confirm no
  flakiness, including the timing-sensitive live-settle test).
- No Go files were touched this batch, so `go build`/`go vet`/`go test` were
  not run (nothing to regress).
- `src/web/**` was not touched.

## tasks.md

Ticked T083 (`[FR-030] Audit every new log/event emission ... for user
content`) with an inline note recording the FR-040 amendment and pointing at
the new log-boundary test — targeted string edit only, no wholesale rewrite.


## Decision: Phase 13 record reconciliation — Feature 351 implementation is complete; the written record now matches it

**Author:** Maximus (Lead/Architect)
**Date:** 2026-08-17
**Feature:** `specs/351-vision-first-deep-identification` — Phase 13 (T093-T099)
**Status:** IMPLEMENTED (documentation-only; no behavior change)

## Purpose

This closes the gap between shipped code and the written record for Feature
351/ADR 0012. Every substantive decision below was already made and, in most
cases, already recorded piecemeal by Cassius/Brutus/Aurelia during
implementation; this note is the single consolidated closure record Maximus
authored/authorized as spec-351's original author, tying the threads together
for anyone reading `decisions.md` end-to-end rather than session-by-session.

## Numbers finally chosen (asked for explicitly)

- **B5 total-timeout clamp**: `total_timeout_s = clamp(HardTimeout - 20s, [30, 900])`
  (`services/settings_service.go`) — already landed, re-verified against code
  today.
- **`DeepIdentificationHardTimeoutSeconds`**: default **420s**, admin-tunable
  range **[1, 900]** (`readInt(..., 420, 1, 900)`, `settings_service.go:312`).
- **`DeepIdentificationQuickLookupTimeoutSeconds`**: default **90s**,
  admin-tunable range **[5, 300]**, additionally bounded above in practice by
  the agent proxy's own 5-minute request ceiling (`settings_service.go:313`;
  `docs/features/ai-analysis.md`).

## OQ/RD defaults actually taken

All seven open questions in `spec.md` are resolved (RD-1 through RD-7, all by
Brian, 2026-08-16/17). Summary for the record:

- **RD-1 (wishlist mechanism)**: confirmed intake results may create a
  `models.Coin` with `IsWishlist = true` directly via `CoinService` — no
  `QuickCaptureDraft` schema change, no migration.
- **RD-2 (corroboration upgrade)**: flat `min(1.0, max(image_conf, provider_conf) + 0.10)`,
  applied once per field, never stacked across providers, never LLM-adjusted.
- **RD-3 (acceptance default)**: confidence-driven, not source-driven — accepted
  by default at confidence ≥ 0.70 regardless of image-only vs. provider-corroborated
  source. Reverses the originally-stated source-driven opt-in default.
- **RD-4 (reverse legend/type)**: excluded from query terms entirely; used only
  as a post-hoc ranking/disambiguation signal over candidates a provider already
  returned (new FR-039 for Numista/Nomisma; OCRE's existing ADR-0010-governed
  scoring math is unchanged — only its token *source* widens to include the
  hypothesis).
- **RD-5 (rollout)**: straight cutover, no transitional A/B flag; existing
  `SettingDeepIdentificationEnabled` remains the sole kill switch.
- **RD-6 (hypothesis visibility)**: build the collapsible, default-collapsed
  "what the images alone said" panel (reverses the originally-stated
  do-not-build default) — this is what makes "unreadable" visibly distinct from
  "dropped," which is the exact failure this feature exists to close.
- **RD-7 (OCRE routing)**: inclusion by default; OCRE is skipped only on a
  *positive* non-Roman-Imperial signal, never on the mere absence of a Roman
  one, and every skip carries a stated reason in `skipped[]`.

## Decisions folded in from implementation (not originally in tasks.md)

1. **FR-040 amendment (Brian-authorized).** The application-log prohibition
   remains absolute. The owner-scoped SSE progress stream may now carry
   hypothesis values, application-authored query terms, candidate counts, and
   per-provider outcomes. Image data remains banned everywhere, with no
   exception.
2. **Degrade-ladder deviation from tasks.md T020/T027/T032** (already recorded
   by the implementing session, restated here as the architectural record):
   tasks.md specified vision failure → typed-EMPTY hypothesis directly. Shipped
   behavior is **structured call → one retry → prose extraction → deterministic
   quick-evidence hypothesis → typed-empty**. This is an intentional, authorized
   deviation, not an oversight: typed-empty on merely-malformed JSON would
   silently recreate the exact bug (vision ran, produced something, and the
   pipeline discarded it) that this entire feature exists to fix.
3. **Router LLM call deleted.** `route()` is now a pure function of
   `(catalog, provider_override, bounds, quick_evidence, hypothesis)` —
   `ROUTER_PROMPT` and the router's LLM invocation are gone. The pipeline now
   makes **one fewer** LLM call per job than Feature 344, and identical inputs
   produce byte-identical `selected`/`skipped`/`rationale` (proven by
   `test_route_is_deterministic_across_identical_runs`).
4. **SQLITE_BUSY fix.** DSN is now `?_txlock=immediate&_pragma=busy_timeout(5000)`
   (glebarez syntax — this is **not** mattn's sqlite3 driver syntax, and the two
   are not interchangeable). Root cause was a read→write lock upgrade inside a
   deferred transaction in `ClaimNextQueuedJob`, which `busy_timeout` alone
   cannot rescue; `_txlock=immediate` forces the write lock at `BEGIN`.
5. **Contract fact worth writing down for anyone extending the SSE stream**:
   Go re-encodes `progress` frames to exactly `{phase, message}` (a strict
   whitelist) before persistence, while `provider_started` passes its payload
   through verbatim and `provider_result` is reduced to a bounded six-field
   owner-facing payload. These three frames are **not** treated uniformly —
   check the actual translation code, not just the frame name, before assuming
   a field survives to the browser.

## Open item — recommendation, not a decision

`DeepIdentificationQuickLookupTimeoutSeconds` is not currently exposed in
Admin Settings, while ten sibling deep-identification settings are (including
its sibling `DeepIdentificationHardTimeoutSeconds`). This was never explicitly
decided either way — it's an omission, not a considered exclusion.

**Recommendation: expose it.** It already has a proper `readInt` bound
`[5, 300]` and a sane default (90s), it directly affects operator-observable
behavior (how long the quick-lookup pass runs before the pipeline proceeds
without it), and every other admin-tunable deep-identification bound is
already surfaced. Leaving it hidden means the only way to change it is a
direct database write to `AppSetting`, which is inconsistent with how every
other bound in this feature is operated. This is a small, additive UI change
(one more settings-form field, `src/web`) — not implemented in this batch
because Aurelia is actively working in `src/web/**` (Phase 12b) and I was
instructed not to touch that directory beyond the single T096 build-define
change. Whoever picks this up next should treat it as a small, low-risk
follow-on, not a new spec.

## Verification performed before recording

- Re-grepped shipped code (not just the original audit) for every fact stated
  above: `MintForJob`/`InternalJobTokenRequired`, `llm` request key,
  `QuickEvidence` `StrictRequestModel(extra="forbid")`, `disagreement_count`/
  `resolved_count`/`synthesis_started` emission, `attributions` in
  `DeepSynthesis`, `ocre_search` internal tool route — all confirmed present
  in `src/api` and `src/agent` as of this session, not assumed from the prior
  audit.
- `go build ./...`, `go vet ./...`, and `go test -count=1 ./...` all pass
  clean in `src/api` after the T096 version-canonicalization change.


### Decision: T106 — Real Go↔Python Deep-Identification Seam Test

**Date:** 2026-08-17
**Agent:** Maximus (Lead/Architect)
**Branch:** `beta`
**Status:** DONE — T106 complete, verified live

## Context

Every deep-identification test that existed before this change drove one side of the wire against hand-written fixtures shaped for the *other* side by convention only:

- `src/api/services/deep_identification_pipeline_runner_stream_test.go` drives the real Go `DeepIdentificationPipelineRunner.Run` against an `httptest.NewServer` fake that emits hand-crafted, Go-authored "Python-shaped" SSE frames.
- The Python `tests/` suite drives `run_deep_identification_stream` and its nodes against hand-crafted, Python-authored "Go-shaped" `DeepIdentifyRequest` payloads.

Both sides were internally consistent and both were wrong about each other at least once — that mismatch is exactly the shape of the 080e598 production bug this task exists to prevent from silently recurring.

## What was built

`src/api/integration/deep_identification_seam_test.go` (new file; none of the three restricted files — `deep_identification_service.go`, `deep_identification_service_test.go`, `deep_identification_pipeline_runner.go` — were touched):

- Boots the **real** `uvicorn app.main:app` Python agent process from `src/agent` (using `src/agent/.venv`, overridable via `DEEP_SEAM_PYTHON`).
- Drives the **real**, exported `DeepIdentificationPipelineRunner.Run` (constructed via its real, exported constructor) over the **real** `AgentProxy.StreamDeepIdentification` HTTP/SSE client against that live process — a genuine `DeepIdentifyRequest` → SSE → `DeepSynthesis` round trip, with no fixture standing in for either side.
- Asserts: (1) the terminal `DeepSynthesis` report has every field contract §5 promises (`narrative`, `proposed_fields`, `disagreements`, `unresolved_questions`, `coverage`, `attributions`, `partial_success`); (2) every persisted event type is inside `deepPipelineEventType`'s closed whitelist; (3) `progress` events re-encode to exactly `{phase, message}`; (4) `provider_result` events carry the bounded 6-field public payload (`provider`, `status`, `confidence`, `claimCount`, `errorKind`, `linkOut`) and never a `provider: "image"` row (FR-025); (5) at least one real `router_selected` and `provider_result` frame is observed.

## CI-exclusion mechanism (defense in depth, both required)

1. **Build tag**: `//go:build seam` at the top of the file. `go build ./...`, `go vet ./...`, and `go test ./...` never compile it without `-tags=seam`.
2. **Env var**: even compiled in, the test `t.Skip`s unless `DEEP_SEAM_TEST=1` is set.

Verified: `go build ./...`, `go vet ./...`, and `go test -count=1 ./...` (no tag) all pass — 10/10 packages ok, seam file entirely absent from the build. `go build -tags=seam ./...` and `go vet -tags=seam ./...` also pass with the file compiled in. Running `go test -tags=seam -run TestDeepIdentificationSeam -v ./integration/...` with the tag but without `DEEP_SEAM_TEST=1` produces a clean `SKIP`, not a failure.

## The LLM tradeoff — stated and justified

The test configures the LLM provider as `ollama` pointed at a local TCP port nothing is listening on. **This is not a stub of the seam.** FR-006/FR-040 already require every LLM call site in the pipeline (vision hypothesis node, evaluator's disagreement summary, synthesis narrative) to degrade to a deterministic fallback on *any* LLM failure, never to raise — verified by reading `hypothesis.py::build_hypothesis_from_vision_traced` (never raises; falls through structured → retry-once → prose → `deterministic_fallback`), `evaluator.py::evaluate`/`_summarize` (wrapped in `try/except`, falls back to deterministic disagreement text), and `synthesis.py::synthesize`/`_write_narrative` (wrapped in `try/except`, falls back to `FALLBACK_NARRATIVE_ON_ERROR`). Pointing at an unreachable endpoint is therefore a *real* LLM call, over the real network stack, that fails fast (connection refused) and is handled entirely by the pipeline's own documented resilience path — not by any test-side interception. This keeps the test hermetic (no API key required, no external network egress, no nondeterministic model output) while still genuinely exercising the vision node's real structured-output call path (the retry ladder is observably invoked — this is why the round trip takes ~15s rather than being instant).

A parallel choice was made for the provider fan-out: the pipeline runner's real, unmodified `deepPipelineProviderCatalog(settings)` default catalog is used (numista/nomisma/ngc/ocre/rpc — nothing is special-cased for the test), but `tools_base_url` is left `""`. This is the exact same code path production takes whenever the tools client is unconfigured (`graph.py::_run_one_provider`: `if tools is None and entry.automatable: return unconfigured`) — not a test-only branch — so `numista`/`nomisma` settle immediately with zero upstream calls, while `ngc`/`rpc` (never automated, no public API) and `ocre` (disabled by default) still execute their real, always-network-free provider nodes. No traffic to `numista.org`/`nomisma.org`/etc. occurs. This was confirmed by reading each provider node (`numista.py`, `nomisma.py` early-return before touching `tools` when automatable is false or query is empty; `ngc.py`/`ocre.py` explicitly check `catalog_entry.automatable` first).

## Verification performed (all actually observed, not claimed)

- `cd src/api && go build ./...` — clean, no output.
- `cd src/api && go vet ./...` — clean, no output.
- `cd src/api && go test -count=1 ./...` — `ok` for all 10 testable packages (models/config/docs have no test files or are non-test packages), seam file entirely excluded from compilation.
- `cd src/api && go test -run TestArchitecture ./...` — `ok`.
- `cd src/api && go test -run TestNoDirectDatabaseImports .` — `ok`.
- `cd src/api && go build -tags=seam ./...` and `go vet -tags=seam ./...` — both clean.
- `cd src/api && go test -tags=seam -run TestDeepIdentificationSeam -v ./integration/...` with `DEEP_SEAM_TEST` unset — `SKIP`, `PASS` overall (proves the guard).
- `cd src/api && $env:DEEP_SEAM_TEST="1"; go test -tags=seam -run TestDeepIdentificationSeam -v ./integration/...` — **`PASS` in 15.48s**, genuinely booting `src/agent/.venv`'s `uvicorn app.main:app`, driving the real pipeline runner over real HTTP/SSE, and observing exactly the expected provider settlement (`numista`/`nomisma` → `failed`/`unconfigured`/`call_count=0`; `ngc`/`ocre` → `not_automated`; `rpc` → `unavailable`; all `call_count=0`, confirming zero network egress) plus a genuine terminal `DeepSynthesis` report and a fully whitelist-compliant, correctly-shaped persisted event log. This is a real, observed pass against a real Python process, not a claimed one.

## Interpretation note for future readers

T106 says "the real Go handler." This test targets the exported `DeepIdentificationPipelineRunner.Run`/`AgentProxy.StreamDeepIdentification` layer — the literal Go↔Python wire client — rather than the full `CreateJob` HTTP handler → worker-pool → SSE-subscriber chain in `deep_identification_service.go` (a file explicitly off-limits this batch and under concurrent edit by other agents this same batch for T105/F3/F5 work). `Run` is the actual seam the 080e598 bug class lives in (wire-fixture drift between Go structs and Python Pydantic models); the worker-pool/queue/SSE-broker machinery around it is a separate, already-tested concern (`deep_identification_service_test.go`) that does not touch the Python wire at all. This scoping keeps the new test stable and independent of concurrent same-batch edits to the restricted files while still fully proving the wire contract.


### Decision: Feature 351 Phase 14 — Deep Identification Service Decomposition (T103/T104)

**Date:** 2026-08-17
**Agent:** Maximus (Lead/Architect)
**Branch:** `beta`
**Status:** COMPLETE — T103 (partial by design), T104 (complete)

## Context

Phase 14 was sequenced last per Brian's explicit instruction on F5, so the
refactor moves settled code, not code in flight. That precondition held:
vision hypothesis, deterministic router, query terms, image-as-claim-source,
wishlist destination, progress emission, and the Maximinus regression gate
were all already landed on `beta`.

Governing constraint: **no behavior change**. Every observable behavior —
frame translation, event persistence, provider settle recording, error
paths, timing, log output, concurrency semantics — had to remain equivalent.

## T104 — Run decomposition (complete, no caveats)

`DeepIdentificationPipelineRunner.Run`'s 8-case inline `onFrame` closure was
extracted into a new `deepFrameTranslator` type
(`deep_identification_frame_translator.go`) with one named method per frame
type: `handleProviderStarted`, `handleRouterSelected`, `handleSynthesis`,
`handleProviderResult`, `handleEvaluation`, `handleProgress`, `handleError`.

This seam was fully clean: `StreamDeepIdentification` invokes `onFrame`
synchronously, once per frame, in arrival order (confirmed by reading
`agent_proxy_deep_identify.go`) — there is no concurrent access to the
translator's fields, so no locking was introduced and no new race is
possible. The translator is constructed fresh per `Run` call and discarded
at the end of it.

Preserved exactly, byte-for-byte:
- `progress` frames are still re-encoded to strictly `{phase, message}`.
- `provider_started` is still persisted verbatim (frame.Raw, untouched).
- `provider_result` is still reduced to the bounded six-field public payload
  via `deepProviderResultPublicPayloadJSON`.
- `synthesis`/`error` still never call `AppendEvent` (returned via the
  translator's `lastSynthesis`/`lastErrorCode`/`lastErrorMessage` fields,
  consumed by `Run` exactly as before).
- The dead-but-harmless `seq++` counter (incremented, never read) was kept
  in `Run` itself rather than moved into the translator, since it isn't
  frame-type-specific — preserves the exact same operation count.

## T103 — Service decomposition (partial by design — two seams declined)

### Extracted cleanly

1. **`deepIdentificationArtifactStore`** (`deep_identification_artifacts.go`):
   `ValidateAndSaveArtifact`, `ReuseSavedCoinImage`,
   `savedImageFingerprintHash`, `DeleteHintArtifacts`, `DeleteJobArtifacts`,
   `deleteArtifacts`, `listArtifacts`, `createArtifact`,
   `markArtifactDeleted`, `detectMimeType`. Confirmed fully self-contained
   (repo/imageRepo/imageSvc/uploadDir/metrics only) before moving — no
   shared locks with any other seam.
2. **`deepIdentificationJanitor`** (`deep_identification_janitor.go`):
   `StartJanitor`, `recoverStaleAndSweepHints`, `runRetentionSweep`.
   Depends on the artifact store (constructor-injected) for hint/job
   artifact deletion, but owns no state shared with the worker pool or job
   lifecycle. `SetProviderBudgetTracker`/`SetInternalTokenService` on
   `DeepIdentificationService` now propagate to the janitor too, since both
   the worker pool's `runJob` and the janitor's recovery sweep independently
   need to reset/revoke a settled job's budget/token.

`DeepIdentificationService`'s public methods (`ValidateAndSaveArtifact`,
`ReuseSavedCoinImage`, `DeleteHintArtifacts`, `DeleteJobArtifacts`,
`StartJanitor`, `RecoverStaleAndSweepHintsForTest`) became one-line
delegations, so the handler-facing API surface is unchanged. Internal
call sites that previously called `s.listArtifacts`/`s.savedImageFingerprintHash`
directly (`resolveRoleHash`, `RetryJob`) now call
`s.artifacts.listArtifacts`/`s.artifacts.savedImageFingerprintHash` — legal
same-package unexported-method access, no new public surface.

`imageSvc` and `uploadDir` were removed as fields from
`DeepIdentificationService` entirely (grep-confirmed unused elsewhere in the
package/tests) since their only consumers moved into the artifact store.

### Declined: job lifecycle + worker pool stay on one type

The task listed five seams: job lifecycle, worker pool, janitor/retention,
SSE broker, capability/limits. SSE broker (`DeepIdentificationBroker`) and
capability/limits (`DeepProviderBudgetTracker`) were **already** separate,
independently constructed types before this change — `DeepIdentificationService`
only holds references to them via DI, which is not god-object coupling.

Job lifecycle (`StartJob`, `CreateJobFromIntake`, `RetryJob`, `GetJob`,
`ListJobs`, `ListEventsSince`, `RequestCancel`) and worker pool
(`StartWorkers`, `workerLoop`, `runJob`, `notifyWorkers`) were **not**
split further, and this is a deliberate decision, not an oversight:

1. **They share `intakeMu` as a single load-bearing mutex.**
   `workerLoop` holds `intakeMu.RLock()` around `ClaimNextQueuedJob`;
   `CreateJobFromIntake` holds `intakeMu.Lock()` across `StartJob` +
   artifact persistence, specifically so a worker cannot claim a queued
   job before its obverse/reverse artifacts exist. This is exactly the
   class of bug this feature has already shipped once (a read→write lock
   upgrade inside a deferred transaction caused SQLITE_BUSY) — the prompt
   explicitly calls out this history.
2. **`deep_identification_service_test.go`'s
   `TestDeepIdentificationService_WorkerCannotClaimUntilIntakeArtifactsAreReady`
   asserts directly on `svc.intakeMu`** to deterministically reproduce the
   production race window (`svc.intakeMu.Lock()` before `StartJob`, then
   asserting the worker cannot claim before `Unlock()`). This is a
   deliberately-authored concurrency regression test for the exact
   mechanism in question.
3. They also share the `cancels` map/`cancelMu`: `runJob` registers/
   unregisters a job's `context.CancelFunc`; `RequestCancel` (job lifecycle)
   looks it up and calls it for a running job.

Splitting these into two separate types is *mechanically* possible — the
same `*sync.RWMutex`/map could be constructor-injected into both — but it
would (a) require rewriting the one test that directly exercises this
exact lock's contention semantics, on the single most concurrency-sensitive
primitive in this file, (b) add a layer of indirection around identical
shared mutable state with no reduction in actual coupling, and (c) do so
**without `-race` available on this machine to verify no regression was
introduced**. Per Brian's framing — "a long, correct file beats an
elegantly decomposed race condition" — I am declining to force this split.
`DeepIdentificationService` retains `intakeMu`, `cancelMu`/`cancels`,
`wakeMu`/`wake`, `runnerMu`/`runner`, and the job-lifecycle/worker-pool
methods directly.

## Verification

- `go build ./...`: exit 0
- `go vet ./...`: exit 0
- `go test -count=1 ./...`: all packages `ok` (root, capture, database,
  handlers, integration, middleware, models, repository, services,
  testutil) — no `FAIL`
- `go test -count=1 -run TestArchitecture ./...`: green across all packages
- `go test -count=1 -run TestNoDirectDatabaseImports .`: green
- `gofmt -l` flags all five touched/new files, but `gofmt -d` confirms this
  is the repo-wide CRLF artifact (whole-file diff, no actual formatting
  delta), consistent with the ~400-file baseline noted in the task brief —
  not a real signal.
- **`-race` was NOT run** (no CGO/C compiler on this machine per the task
  brief). The concurrency-relevant portions of this refactor — the
  worker-pool/job-lifecycle seam I declined to split, and the frame
  translator's single-threaded-callback assumption (verified by reading
  `agent_proxy_deep_identify.go`, not by a race-detector run) — were
  **not race-verified locally**. Weigh accordingly before merge.

## Before/after line counts

| File | Before | After |
|---|---|---|
| `deep_identification_service.go` | 1130 | 842 |
| `deep_identification_pipeline_runner.go` | 1046 | 893 |
| `deep_identification_artifacts.go` (new) | — | 265 |
| `deep_identification_janitor.go` (new) | — | 186 |
| `deep_identification_frame_translator.go` (new) | — | 261 |

(Note: the task brief's "980 lines"/"161 lines" figures for the two files
predate other Phase-9-era additions landed since; the pre-refactor counts
above are what this change actually started from.)

## Test changes

Three direct unexported-method call sites in
`deep_identification_service_test.go` were updated to reach through the new
janitor field (`svc.recoverStaleAndSweepHints()` →
`svc.janitor.recoverStaleAndSweepHints()`; same for `runRetentionSweep`).
No test assertions, fixtures, or expected values were changed — only the
receiver path to the same, unmoved logic.

## Alignment

Feature 351 Phase 14 (F5); Constitution Principle I (layered
architecture/DI), Principle IV (simple, complete, proportional change),
Principle IX (architecture enforced by automated tests), §17 Quality Gate.


### Decision: `task openapi` Windows version-bump bug — reproduced, root-caused, fixed

**Date:** 2026-08-17
**Agent:** Maximus (Lead/Architect)
**Requested by:** Brian (@briandenicola) — backlog cleanup
**Branch:** `beta` (uncommitted; Taskfile.yml change only)

## Context

During the Phase 16 quality gate, Cassius reported that `task openapi` had a
pre-existing PowerShell bug in its version-bump step on Windows and worked
around it by regenerating docs directly. That workaround is not acceptable
as a standing state: `tasks.md` T115 and the Definition of Done both instruct
contributors to run `task openapi` to regenerate API docs.

## Reproduction

Ran `task openapi` from repo root on this Windows environment. It failed
immediately on the Windows-only version-bump `cmd:` line with:

```
You must provide a value expression following the '+' operator.
+ ('' + )
```

## Root cause (two independent, compounding bugs — both verified in isolation)

1. **Task's own shell eats PowerShell's `$` before PowerShell ever sees it.**
   Task parses every `cmd:` line with its embedded POSIX-like shell
   (mvdan.cc/sh) before invoking `powershell.exe`. Inside a double-quoted
   argument, that shell expands bare `$name` itself (undefined in Task's
   environment → empty string), silently stripping `$v`/`$c`/`$1` to nothing
   and producing the `('' + )` PowerShell parser error actually observed.
   **Fix:** escape every PowerShell variable in the Windows `cmd:` line as
   `\$variable` so Task's shell passes it through literally.

2. **A second, silent, more dangerous bug independent of the shell-parsing
   issue.** The replacement expression `('$1' + $v)` concatenates the bare
   `$1` regex backreference directly against the version string (e.g.
   `"$1" + "4.0.0"` → `"$14.0.0"`). .NET regex substitution reads `$14` as
   "capture group 14" (which doesn't exist, since the version's leading
   digit fuses onto the `$1` placeholder), and silently emits the literal
   text `$14.0.0` into `main.go` instead of the version — **no error, just
   corrupted output.** Reproduced this in isolation with a minimal
   `-replace` test before touching the Taskfile. This would have broken at
   `VERSION=4.0` too — any version string starting with a digit fuses with
   `$1`, so it isn't specific to the 4.0 → 4.0.0 format change, though the
   version bump prompted discovering it now.
   **Fix:** use the braced group-reference syntax `('${1}' + $v)`, which
   cannot fuse with trailing digits regardless of `VERSION`'s shape.

CI (`.github/workflows/ci.yml`) never invokes this version-bump step at all
— it runs `swag` directly and diffs the committed docs against the working
tree — so this defect had zero CI blast radius; it was purely a local
Windows-dev-workflow break. The Linux/macOS `perl` step was already immune,
since perl distinguishes `$1` (capture group) from `$v` (named variable) —
no digit-fusion is possible there.

## Fix applied

`Taskfile.yml`, `openapi` task, Windows-only `cmd:` step: escaped all `$`
variables for Task's shell and switched `$1` → `${1}` in the regex
replacement, with inline comments explaining both bugs for the next
contributor who touches this line. No change to the Linux/macOS `perl` step
(already correct) and no change to CI.

## Verification

- Ran `task openapi` end-to-end after the fix: completed cleanly (swag
  install, `swag init`, copy to `docs/openapi.json`, existence check).
- `git diff --stat` on all regenerated artifacts (`src/api/docs/*`,
  `docs/openapi.json`, `src/api/main.go`) is **empty** — the regenerated
  output is byte-identical to what was hand-verified and committed at
  `521c924`. This was checked deliberately per the guardrail in this task;
  a wholesale rewrite would have been treated as a regression requiring
  investigation, not silently accepted.
- Confirmed the `409` `job_at_capacity` response documentation (added by the
  at-capacity fix) is present and unchanged in `docs/openapi.json`.
- Gates, run from `src/api/`:
  - `go build ./...` — pass
  - `go vet ./...` — pass
  - `go test -count=1 ./...` — 10/10 packages `ok`
  - `go test -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .` — pass

## Scope discipline

Touched only `Taskfile.yml`. Did not touch `src/web/`, `src/api/database/`,
`src/api/repository/`, `tasks.md`, or `main.go`'s runtime behavior. Left
other agents' concurrent uncommitted work in the tree untouched
(`src/api/database/database.go`, `src/api/models/sqlite_config.go`,
`src/web/**` changes observed via `git status` belong to Cassius/Aurelia's
in-progress batches and were not inspected or modified).

## Status

Left **uncommitted** on `beta` per instructions. Ready for Brian or the next
agent to commit; `Taskfile.yml` is the only change.

---

### Decision: Deep-analysis journal entries on "coin" and "wishlist" apply targets; "draft" is out of reach

**Date:** 2026-08-17
**Feature:** Deep Analysis / Deep Identification proposal apply (US4)
**Status:** IMPLEMENTED (coin, wishlist) / NOT IMPLEMENTED (draft — genuinely not possible at apply time)
**Agent:** Cassius (Backend Developer)

## What the brief assumed vs. what was actually true

The brief's premise, quoting Apply()'s doc comment, was that the "coin" apply
target already wrote a CoinJournal entry with source "deep_identification",
and asked me to add the same record for "wishlist" and "draft".

That premise was not accurate. I checked every CreateJournalEntry call
site in services/ before writing any code and confirmed deep_identification_proposal.go
was not among them — no target (coin, wishlist, or draft) wrote a journal entry
before this change. The "deep_identification" string passed to CoinService.UpdateCoinWithFields
as source is only consulted by one branch inside updateCoin — the
CurrentValue-changed check (source != "estimate") — and CurrentValue
is not in deepProposalCoinFieldAllowlist, so that branch can never fire for
a deep-analysis apply. The doc comment describing "journal source
deep_identification" for the coin path was aspirational, not implemented.

## What I changed

Added deepProposalJournalEntryText(fieldNames []string) string (terse,
house-style: "Deep Analysis applied: <field1>, <field2> updated", naming
only field keys, never proposed values) and wired a
s.coinRepo.CreateJournalEntry(&models.CoinJournal{}) call — the existing
repository write path, no new DB access — into:

- applyToCoin — right after CoinService.UpdateCoinWithFields succeeds,
  attached to the existing coin's ID.
- applyToWishlist — right after CoinService.CreateCoin succeeds, attached
  to the newly created coin's ID.

## Why "draft" is not implemented

models.QuickCaptureDraft has no CoinID until it is promoted
(PromotedCoinID *uint, nil until promotion), and models.CoinJournal.CoinID
is gorm:"not null". There is no coin row to attach a journal entry to at
the moment applyToDraft runs. I did not implement the promotion-time option
either, to avoid widening the change past "small, isolated" as instructed.

## Follow-up (same day): journal write must be best-effort, not fatal

Brian's review caught a real defect in the first pass: Apply() is not
transactional across CreateCoin/UpdateCoinWithFields -> journal write ->
ApplyJob. My first implementation returned the journal write's error, which would
leave a wishlist coin created but the job never marked applied — a client retry would
then call applyToWishlist again, creating a second wishlist coin.

Fix: recordDeepProposalJournalEntry now swallows CreateJournalEntry's error
and logs it instead of returning it — matching the existing best-effort precedent
in reference_migration_service.go. Added an optional *Logger field +
WithLogger() method to DeepIdentificationProposalService, wired in main.go.
The log line names only field keys, never proposed values (FR-040 discipline).

New test: TestDeepIdentificationProposal_ApplySucceedsWhenJournalWriteFails
forces a genuine CreateJournalEntry failure by dropping the real coin_journals
table with the actual CoinRepository, then asserts: Apply returns no error,
the coin update lands, the job is marked applied, a second Apply call is rejected
as already-applied, and the swallowed error was logged.

## Verification

- go build ./..., go vet ./... clean
- go test -count=1 ./... → 10/10 packages ok
- TestArchitecture, TestNoDirectDatabaseImports PASS

## Files changed

- src/api/services/deep_identification_proposal.go
- src/api/services/deep_identification_proposal_test.go
- src/api/main.go

**Outcome:** committed as 755593f.

---

### Decision: Wishlist coins may hold catalog references (ADR 0013) + Feature 352 Decisions A/B/C

**Author:** Maximus (Lead / Architect)
**Date:** 2026-08-17
**Requested by:** Brian (@briandenicola)
**Artifacts:** docs/adr/0013-wishlist-coins-may-hold-catalog-references.md,
specs/352-deep-identification-structured-results/{spec.md,plan.md}
**Status:** ADR Proposed. Uncommitted, awaiting review. No implementation code written.

## Decisions (settled by Brian — do not re-litigate)

**A. Wishlist items MAY hold catalog references.** This reverses a rule ratified
in landed spec 351, so it is recorded as ADR 0013 per constitution SS22. Feature 352
Phase 6 is ungated.

**B. Draft one-to-many via a new additive table** (QuickCaptureDraftCatalogReference),
not an in-place migration. QuickCaptureDraftReference's DraftID uniqueIndex
and URI NOT NULL stay. Rationale: SQLite cannot relax either without a destructive
rebuild, and the single-reference surface spans 34 consumer files.

**C. One notes format everywhere.** The dated-heading, job-id-keyed append format
applies to the intake/draft path as well as the saved-coin path, replacing the
narrative block buildDeepIntakeProposalFields already writes today. Brian
accepted that this changes output he tested on 2026-08-16.

## What every agent needs to know

1. **The wishlist/no-references rule had no recorded domain rationale.** It was
   the fourth of four defensive layers against a GORM batch-insert crash. Layers
   1-3 already fix the root cause. Do not cite it as a design principle.
2. **Two shipped paths already create wishlist references:**
   QuickCaptureRepository.PromoteDraftTransaction and
   ReferenceMigrationService.MigrateLegacyReferences.
3. **FR-048 and FR-049 must land in the same commit.** Deleting the guards without
   clearing input.Coin.References in WishlistSearchAlertService.ConvertCandidate
   would silently persist unconfirmed AI search-agent claims. A guard-removal commit
   lacking FR-049 is a reviewer BLOCK.
4. **Do NOT delete coin_service.go:171-172** (pendingReferences := coin.References; coin.References = nil).
   It is GORM cascade defence. Removing it reintroduces the 2026-07-21 crash.
5. **UpdateCoinWithFields remains a permanent trap.** Its updateCoin routes
   updates.References into ReplaceForCoin, which deletes every existing reference.
   Structured deep-ID references MUST use the new additive CoinReferenceService.AppendForCoin.
6. **Phase 6a lands first and alone.** Smallest diff, widest blast radius.
7. **Four tests are deliberately rewritten** (FR-052): all FR-048 gating must pass
   before they change.

## Flagged cost Brian has not yet weighed

Decision A un-blocks a path he did not ask about. ConvertCandidateInput.Coin is
an unvalidated models.Coin carrying unconfirmed AI search-agent catalog claims
with no confirm gate. Brian decided confirm-gated deep identification may write
wishlist references. He did not decide that the search agent may. FR-049 holds
that line, but it is worth an explicit "yes, keep it blocked" from him.

## Smaller consequences

- PurchaseCoin now carries references across the purchase instead of dropping them.
- CoinRepository.Duplicate now copies references when duplicating a wishlist coin.
- CatalogRegistryRepository.CountReferencesUsing now counts wishlist references,
  so a catalog used only by wishlist coins becomes undeletable.
- Any frontend component that hid a references panel on wishlist coins will now render content.

---

### Decision: Feature 352 — Deep Identification Structured Results: architecture decisions

**Date:** 2026-08-17
**Author:** Maximus (Lead / Architect)
**Feature:** specs/352-deep-identification-structured-results/
**Status:** Spec + plan authored. No implementation code written. Not committed.

## Scope confirmed

352 was unused (highest existing spec directory was 351). Spec and plan written
to specs/352-deep-identification-structured-results/.

## Architecture Decisions

### D-1. Collection-valued write surface is a NEW, separate allowlist

deepProposalCoinFieldAllowlist / deepProposalDraftFieldAllowlist stay unchanged.
Catalog references get a third, closed map (deepProposalCollectionFieldAllowlist)
with its own resolver and write path. One static key holding an array (catalogReferences),
not one key per reference.

### D-2. coin_type free text is SUPERSEDED, never replaced

- "coin_type -> ReferenceText" stays in the scalar allowlist, unchanged.
- When the value parses into a registry-valid element, that element is emitted and
  the scalar entry's default accepted flips to false.
- When the parse fails, the scalar keeps its normal confidence-driven default
  so the catalogue label is never lost.

Rationale: writing both by default puts one fact in two places that diverge when
edited. Dropping the scalar regresses Feature 345 and loses data on parse failure.

### D-3. Reference write must be ADDITIVE — new AppendForCoin

CoinService.updateCoin routes updates.References to ReplaceForCoin, which
deletes every existing reference before inserting. New CoinReferenceService.AppendForCoin
is a sibling, deliberately not a mode flag. ReplaceForCoin is owner-editor; AppendForCoin
is agent semantic.

### D-4. Notes append: dated heading, job-id keyed idempotency

## Deep Analysis - YYYY-MM-DD (job <jobID>)

Identity is the job id, not the date. Before appending, scan for an existing
block with this job id and replace it in place; otherwise append.

### D-5. Draft one-to-many is delivered ADDITIVELY

New table (QuickCaptureDraftCatalogReference, non-unique DraftID, nullable URI)
plus idempotent backfill, leaving QuickCaptureDraftReference structurally untouched.
SQLite cannot drop index-backed constraints without destructive rebuild. Additive turns
"every one is a candidate breakage" into "every one keeps compiling unchanged".

### D-6. Parser is EXTRACTED, not duplicated; migration policy stays put

parseLegacyReference + helpers move to services/catalog_reference_parser.go.
The Volume: "0" sentinel and "manual review needed" journal string are migration
policy, not parsing — they stay in ReferenceMigrationService. Each caller decides
what to emit. Confidence table: 0.90 clean / 0.90 Roman-numeral / 0.50 inferred /
0.30 + needsVolume sentinel. 351's 0.70 threshold unchanged. NGC is not added to
normalizeCatalogAlias.

## Highest risks (ranked)

1. Wishlist references reversing a landed-spec decision without an ADR (governance).
2. A structured-reference write reaching ReplaceForCoin and deleting owner data.
3. The notes append truncating or overwriting hand-written owner text.
4. The draft migration breaking one of 34 consumers.
5. An array Proposed value being stringified into a scalar column.

---

### Decision: Feature 353 — Wishlist Availability Run Observability (Specification & Clarifications)

**Date:** 2026-08-17  
**Author:** Brian DeNicola (Product Owner) via Copilot directive  
**Feature:** specs/353-wishlist-availability-run-observability/  
**Status:** Clarifications settled; revision ready for implementation

## User Decisions

Three blocking concerns from Feature 353 initial design review were settled by Brian:

1. **Cycle retention (settled):** Parent `AvailabilityCycle` retains last 20 **globally** (terminal, completed status). Child `AvailabilityRun` (per owner) retains last 20 **per owner**. Dual-level retention prevents unbounded growth while maintaining per-user observability.

2. **Per-coin unavailable alerts (settled):** Keep existing `NotifyWishlistUnavailable` per-coin notifications unchanged. New `wishlist_availability_run` per-run notification is an **addition**, not a replacement. Both coexist and fire on run completion — per-coin for each affected coin, per-run for the aggregate.

3. **Legacy data handling (settled):** Additive-only schema migration: new `AvailabilityCycle` table + nullable `AvailabilityRun.CycleID` FK. Zero synthetic backfill, zero reparenting, zero `TriggerType` retagging, zero ADR needed. Legacy `UserID=0` rows remain unmodified in `run_history`, readable without synthesized parents, with "Legacy" UI label (FR-021a).

## Alignment

- Principle IV: Simple, complete, proportional (additive, not destructive)
- Principle V: Security/Auth by Default (clear scoping of notifications)
- §17 Quality Gate: Additive schema, no fabricated state, clear spec settlement

---

### Decision: Feature 353 — Block Resolution via Independent Revision

**Date:** 2026-08-17T17:44:34-05:00  
**Author:** Cassius (Backend Developer)  
**Authorization:** Strict Lockout (Maximus reassigned)  
**Feature:** specs/353-wishlist-availability-run-observability/  
**Status:** Revision complete, approved

## Scope

Cassius independently revised Feature 353 spec/plan/tasks to resolve Brutus's three BLOCK findings:

### Spec.md (FR-014, FR-019, FR-021/FR-021a)

- **FR-014 rewritten:** Explicit non-suppression of per-coin alerts. `NotifyWishlistUnavailable` remains active; new `wishlist_availability_run` is additional, not replacement.
- **FR-019 confirmed:** 20 terminal cycles globally + 20 per-owner child runs. No ambiguity.
- **FR-021 replaced:** Additive-only schema change (new table + nullable FK). No backfill, no reparenting, no synthetic state.
- **FR-021a added:** UI labels `UserID=0` admin rows as "Legacy" for visibility.
- **Trigger vocabulary simplified:** Removed `legacy_cycle` and `legacy` types (not needed; "Legacy" is UI-only label).

### Plan.md (Phases, D4, D6, Tasks T027/T029/T036–T038/T042)

- **Phase 4:** Rewritten from synthetic migration to schema-additive test (verify table/column exist, verify legacy rows unmodified).
- **D4/D6:** Confirmed dual-level retention and coexistent notifications.
- **Tasks rewritten:** T027 (both notifications fire), T029 (per-coin alerts remain), T036–T038 (additive schema test, not data movement), T042 (add "Legacy" label).

### Tasks.md (T001–T051 re-anchored)

- All 51 tasks re-anchored to updated FR/SC/US IDs
- No orphans remain
- Scope discipline: specs-only, no application code

## Outcome

All three BLOCK findings resolved. Strict Lockout clearance approved by Brutus. Ready for implementation delegation.

---

### Decision: Feature 352 Phase 1 — Catalog Reference Parser Extraction

**Date:** 2026-08-17  
**Author:** Cassius (Backend Developer)  
**Feature:** specs/352-deep-identification-structured-results/  
**Phase:** 1 (Foundational)  
**Status:** Implemented, tested, uncommitted

## Scope

- **New:** `src/api/services/catalog_reference_parser.go` with exported `ParseCatalogReferenceText()`, `ParsedCatalogReference{Catalog,Volume,Number,Confidence,NeedsVolume,RawText,Reason}`, and `CatalogParseReason` typed enum.
- **Modified:** `src/api/services/reference_migration_service.go` — `parseLegacyReference` now delegates token/volume/number parsing to the new shared helper, reconstructs migration-specific journal messages.
- **New:** `src/api/services/catalog_reference_parser_test.go` — confidence table (0.90 clean / 0.90 Roman-numeral / 0.50 inferred / 0.30 + NeedsVolume), never-emits-Volume-0 assertion, not-found branch coverage.
- **Unchanged:** `reference_migration_service_test.go` (regression oracle, zero edits).

## Design Decision Worth Recording

`CatalogParseReason` enum (CatalogParseOK, CatalogParseEmpty, CatalogParseUnrecognizedCatalog, CatalogParseNotInRegistry, CatalogParseNoNumber) follows codebase convention (services.LogLevel) and prevents fragile future callers from re-inventing disambiguation tricks.

## Verification

- `reference_migration_service_test.go` unedited, all green before and after extraction.
- Added `TestParseCatalogReferenceText_ReasonOKOnSuccess` and `Reason` assertions on not-found branches.
- Falsified `CatalogParseNoNumber`, got real RED, restored, confirmed GREEN.
- Full gate: `go build ./...`, `go vet ./...`, `go test -count=1 ./...` — 10/10 packages, TestArchitecture and TestNoDirectDatabaseImports both PASS.

---

### Decision: Feature 352 Phase 3 — Collection-Valued Proposal Write Surface

**Date:** 2026-08-17  
**Author:** Cassius (Backend Developer)  
**Feature:** specs/352-deep-identification-structured-results/  
**Phase:** 3 (Foundational)  
**Status:** Implemented, uncommitted (test-file constructor wiring awaiting Brutus)

## Scope

- `src/api/services/deep_identification_proposal.go`:
  - New closed `deepProposalCollectionFieldAllowlist` (exactly `catalogReferences`, FR-002/FR-003)
  - New `deepProposalCatalogReference` DTO (FR-004) with `sourceProvider` vocabulary (closed: all `models.DeepProviderName` + "image")
  - 10-element cap (FR-005)
  - `decodeDeepProposalCatalogReferences` re-marshals proposal `any` value and re-decodes with `DisallowUnknownFields`, then validates each element through `CoinReferenceService.NormalizeAndValidateOne` (FR-045)
  - `applyToCoin` dispatches through explicit switch over two allowlists; scalar names use existing `UpdateCoinWithFields`, `catalogReferences` uses new `CoinReferenceService.AppendForCoin` (additive, FR-013)
  - `applyToDraft`/`applyToWishlist` left unchanged (scalar allowlists only; wishlist reference persistence is Phase 6b)
  - `UpdateProposal` decodes/validates `catalogReferences` owner edit before persisting
- `src/api/main.go`: Widened constructor to take 5th parameter `*CoinReferenceService` (reused existing instance)

## Known Consequences

Two test files have constructor call sites (`deep_identification_proposal_test.go:38`, `handlers/deep_identification_test.go:89`) that need the new 5th argument. Production code `go build ./...` is clean; `go vet ./...` fails only on these two test-file sites (Brutus's responsibility to wire).

## Gap Found (Not Fixed, Out of Phase 3 Scope)

`respondDeepProposalError` default branch maps non-sentinel errors to HTTP 500. A `catalogReferences` validation failure surfaces as plain `fmt.Errorf`, returning 500 instead of 400. Recommend either a new sentinel (`ErrDeepProposalInvalidCatalogReferences`) wrapping validation failures, or explicit decision that 500 is acceptable. Phase 3's authorization scoped handler edits to Swagger/docs-only, so this was left for Phase 3's follow-up or a separate task.

## Verification

- `go build ./...` clean (production).
- `go vet ./...` clean except the two known test-file sites.
- Phase 2 (`AppendForCoin`) verified and committed in isolation first (commit `7a4fc30`), all tests pass with Phase 3 stashed.

---

### Decision: Feature 352 Phase 3 — Client-Error Handling for Catalog References (BLOCK Cleared)

**Date:** 2026-08-17  
**Author:** Maximus (Lead/Architect)  
**Authorization:** Strict Lockout (independent revision)  
**Feature:** specs/352-deep-identification-structured-results/  
**Phase:** 3 (Foundational)  
**Status:** BLOCK cleared, revision ready for re-review

## Block Condition

Brutus identified that `decodeDeepProposalCatalogReferences` returned malformed/invalid content as plain `fmt.Errorf`, so `respondDeepProposalError` fell through to HTTP 500 instead of client 4xx (FR-004/FR-005/FR-045). Cassius had flagged this gap in Phase 3 but left it unfixed (out of scope); Brutus independently confirmed.

## Revision (Typed Fix)

- New sentinel `ErrDeepProposalInvalidCatalogReferences` in `deep_identification_proposal.go`
- `decodeDeepProposalCatalogReferences` wraps every malformed/registry-invalid return with this sentinel via multi-`%w`, preserving error chain
- New guard `isDeepProposalCatalogReferenceValidationError` distinguishes validation errors (4xx) from unwrapped registry-repository errors (5xx)
- New handler case in `respondDeepProposalError`: `errors.Is(err, ErrDeepProposalInvalidCatalogReferences)` → `http.StatusBadRequest` with `code: "invalid_catalog_references"`
- Both PATCH (`UpdateProposal`) and Apply (`applyToCoin`) surfaces fixed by one change

## Verification

- `go vet ./...` clean
- Targeted: `go test ./services/... ./handlers/... -run "DeepProposal|DeepIdentification" -v` — all pass, including Brutus's sentinel assertions
- Full: `go test ./...` — all packages pass

## Outcome

BLOCK condition resolved. Ready for Brutus re-review/clear.

---

### Decision: Feature 352 Phase 4 — Catalog References Pipeline Emission (BLOCK Condition & Remediation)

**Date:** 2026-08-17  
**Author:** Brutus (Reviewer/QA) — block identified  
**Status:** BLOCK issued; remediation by Maximus in progress

## Block Condition

`buildDeepProposalDocumentJSON`'s saved-coin branch has an early guard:

```go
if len(report.ProposedFields) == 0 {
    return ""
}
```

This runs **before** the new `buildDeepCatalogReferenceField(...)` call. When a synthesis genuinely produces zero automatable `proposed_fields` but has a legible NGC cert (reachable: e.g. poor image quality for AI but clear slab), the entire proposal is dropped and the NGC evidence is silently lost. Violates FR-006 (catalog references emitted unconditionally when cert is non-empty) and AC-001 (NGC-driven proposal). Reproduced and documented with characterization test `TestBuildDeepProposalDocumentJSON_KnownDefect_SavedCoinEmptyProposedFieldsDropsNGCCatalogReference` (asserts current `out == ""` behavior with comment to flip assertion once fixed).

Intake branch has no bug (builds `catalogReferences` before early-return check).

## Non-Issues Investigated & Cleared

- **Registry-load degradation** (empty non-nil map + log on DB failure): matches runner's existing degrade-and-log convention, content-free (job ID + driver error only). Not a silent-failure violation.
- **DI wiring** (`deepIdentificationCatalogRegistryRepo`): correctly ordered and independently instantiated per codebase pattern. `deep_identification_service.go` needs zero changes.

## Out-of-Scope Finding (Not Blocking)

`src/api/integration/deep_identification_seam_test.go:172` still calls pre-Phase-4 7-arg constructor signature; will fail to compile under `-tags=seam` until the 8th argument (`catalogRegistry`) is added. Outside authorized test-file list, left untouched.

---

### Decision: Feature 352 Phase 4 — Saved-Coin Early-Return BLOCK Cleared

**Date:** 2026-08-17  
**Author:** Maximus (Lead/Architect)  
**Authorization:** Strict Lockout (independent revision)  
**Feature:** specs/352-deep-identification-structured-results/  
**Phase:** 4 (Foundational)  
**Status:** BLOCK cleared, revision ready for re-review

## Revision (Smallest Change)

Removed the early `if len(report.ProposedFields) == 0 { return "" }` guard from the saved-coin branch of `buildDeepProposalDocumentJSON`. The function's sole terminal empty-check `if len(fields) == 0 { return "" }` (after `catalogReferences`/supersession applied) is now the only place an empty proposal is produced — and only when genuinely no scalar or collection fields exist.

- `fields` map construction, `buildDeepCatalogReferenceField` call, and scalar supersession logic unchanged and now always run
- No changes to intake branch, parser, ranking, registry loading, schema version, or test files

## Verification

- `gofmt -l`/`gofmt -d` clean
- `go vet ./...` clean
- Targeted: `go test ./services/... -run "Phase4|CatalogReference|Proposal" -v` — all pass except the tripwire test (fails as expected, per its own comment "KNOWN DEFECT NO LONGER REPRODUCES... flip assertion")
- Full: `go test ./...` — all other packages pass; `services` shows single expected tripwire failure only

## Outcome

BLOCK condition resolved. Requesting Brutus re-review/clear and tripwire test assertion flip (follow-up test-authorized pass).

---

### Decision: Feature 352 Phase 6a — Wishlist Catalog References & ADR 0013 Acceptance

**Date:** 2026-08-17  
**Author:** Cassius (Backend Developer)  
**Feature:** specs/352-deep-identification-structured-results/  
**Phase:** 6a (Foundational)  
**Status:** Implemented, uncommitted (Brian's review pending)

## Scope

- ADR 0013 (`docs/adr/0013-wishlist-coins-may-hold-catalog-references.md`): Status flipped `Proposed` → `Accepted`
- `src/api/services/coin_service.go`: Deleted two `if coin.IsWishlist { ... = nil }` guards (FR-048)
- `src/api/services/wishlist_search_alert_service.go`: `ConvertCandidate` now clears `input.Coin.References` at its own boundary (FR-049, distinct trust-boundary guard, not duplicate)
- Four Phase 6a tests rewritten (FR-052) to assert new rules, citing ADR 0013/FR-049:
  - `coin_service_test.go`: Drop guard assertions → persist reference assertions
  - `coin_handler_test.go`: Drop guard → persist reference
  - `wishlist_search_alert_service_test.go:~615`: Comment changed from "discard references" to FR-049 trust reason
- New test `TestConvertCandidate_ReferenceSupportEnabled_StillPersistsZeroReferences` wires real `CoinReferenceService` support to catch FR-049 regression (existing test cannot because its mock never calls support)

## What Phase 6a Did Not Touch

- ADR-0013 pointers in specs/351 — Brian handling separately
- `applyToWishlist` (Phase 6b)
- `ocre_scoring.go` (ADR 0010)

## Real Gap Found During Verification

Existing regression test `TestConvertCandidate_CoinWithNonZeroReferenceIDsDoesNotConflict` stayed GREEN even with FR-049 guard deleted because its `CoinService` never wires reference support. Test alone is not reliable oracle; new test actually catches guard regression by wiring support.

## Verification

- `go build ./...`, `go vet ./...` clean
- `go test -count=1 ./...` — 10/10 packages ok
- TestArchitecture and TestNoDirectDatabaseImports both pass
- Falsified FR-049 guard three times, confirmed real RED each time, restored, confirmed GREEN

## Minor Footgun Documented (Not a Bug)

Test-authoring hazard: bare `:memory:` DSN + `SetMaxOpenConns(1)` + `CoinService` with reference support will deadlock on wishlist reference writes because `NormalizeAndValidate*` calls `CatalogRegistryRepository.FindByCatalog` through an unwrapped connection. Previously invisible because wishlist coins never reached that code path. Recorded in `.squad/agents/cassius/history.md` as a footgun, not filed as bug.
