# Squad Decisions

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

