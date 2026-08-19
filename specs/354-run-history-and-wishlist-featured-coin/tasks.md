---
description: "Task list for feature 354 — Deep-Identification Run History & Wishlist-Eligible Coin of the Day"
---

# Tasks: Deep-Identification Run History & Wishlist-Eligible Coin of the Day

**Input**: Design documents from `specs/354-run-history-and-wishlist-featured-coin/`
**Prerequisites**: `spec.md`, `plan.md`

**Tests**: Included. Every implementation task has a tests-first counterpart per constitution §17 Quality Gate.

## Format: `[ID] [P?] [Story] [Owner] Description`

- **[P]**: Can run in parallel with other [P] tasks (touches different files, no dependencies).
- **[Story]**: `US1`–`US6` from `spec.md`. Cross-cutting tasks use `[X]`.
- **[Owner]**: Suggested owner — `Cassius` (backend Go), `Brutus` (Python agent + QA), `Aurelia` (Vue/PWA), `Maximus` (architecture review).
- File paths are absolute-in-repo.

---

## Phase 1: Setup & Baseline

- [ ] **T001** [X] [Cassius] Confirm baseline green: `cd src/api && go test ./...` and `cd src/web && npm run type-check`. Record output; no source edits.
- [X] **T002** [X] [P] [Brutus] Confirm agent baseline: `cd src/agent && pytest tests/ -q`. Record output.

---

## Phase 2: Foundational — Data Model Changes (BLOCKING)

Additive DDL only. Two independent columns + one nullable relaxation.

### Tests first

- [ ] **T003** [X] [Cassius] Write migration regression test at `src/api/database/feature354_migration_regression_test.go`:
  - Seed a legacy `deep_identification_jobs` row with `expires_at` non-null, an owned `featured_coin` row without `source_type`, and a legacy `users` row.
  - Run AutoMigrate. Assert: `expires_at` is nullable (or a sentinel path is documented); `featured_coins.source_type = 'owned'`; `users.coin_of_day_include_wishlist = 1`. No row loss.
- [ ] **T004** [X] [Cassius] Extend `src/api/database/database_test.go` (or add `feature354_model_shape_test.go`) asserting the new columns are present and typed as spec'd.

### Implementation

- [X] **T005** [X] [Cassius] Update `src/api/models/deep_identification_job.go`: change `ExpiresAt time.Time` to `ExpiresAt *time.Time`. Update all internal callers to pointer-dereference safely (grep for `.ExpiresAt`). If GORM cannot relax the column in place on SQLite, fall back to the sentinel `9999-12-31T00:00:00Z` strategy (plan D1) and keep the field non-pointer; document the choice in the file header.
- [X] **T006** [X] [Cassius] Update `src/api/models/featured_coin.go`: add `SourceType string \`gorm:"type:varchar(16);not null;default:'owned'" json:"sourceType"\``.
- [X] **T007** [X] [Cassius] Update `src/api/models/user.go`: add `CoinOfDayIncludeWishlist bool \`gorm:"not null;default:true" json:"coinOfDayIncludeWishlist"\``.
- [X] **T008** [X] [Cassius] Confirm `src/api/database/database.go` AutoMigrate order needs no changes (both are additive on existing tables). Run T003–T004 tests green.

---

## Phase 3: Backend — Deep-Identification Retention & Delete (US1, US3, US6)

### Tests first

- [X] **T009** [P] [US6] [Cassius] Extend `src/api/repository/deep_identification_repository_test.go` covering:
  - `SettleTerminal` with `clearExpiry=true` sets `ExpiresAt` to NULL (or sentinel) for `completed` / `partial`; leaves it untouched for `failed` / `cancelled`.
  - `ListExpiredJobIDs` returns rows only when `expires_at IS NOT NULL AND expires_at <= now`.
- [X] **T010** [P] [US3] [Cassius] Add repository test `DeleteJob(userID, jobID)`:
  - Deletes job + all provider-runs + all events + all artifact rows in one tx.
  - Rejects non-owner (returns `gorm.ErrRecordNotFound`).
  - Refuses non-terminal jobs (returns a sentinel `ErrDeepJobNotTerminal`).
  - Does NOT delete the linked `Coin` in `applied_coin_id`.
- [X] **T011** [P] [US3] [Cassius] Add handler test `src/api/handlers/deep_identification_test.go` covering:
  - `DELETE /deep-identification/jobs/{id}` → 204 for owner + terminal job.
  - 409 for non-terminal.
  - 404 for non-owner and for missing.
- [X] **T012** [P] [US6] [Cassius] Add janitor test in `src/api/services/deep_identification_janitor_test.go` (or extend existing) confirming `runRetentionSweep` never deletes a `completed`/`partial` job regardless of expires_at value.

### Implementation

- [X] **T013** [US6] [Cassius] Update `DeepIdentificationRepository.SettleTerminal` signature to accept a `clearExpiry bool`; pass `true` from the pipeline runner when settling to `completed` / `partial`.
- [X] **T014** [US6] [Cassius] Update `DeepIdentificationRepository.ListExpiredJobIDs` to add `expires_at IS NOT NULL` guard.
- [X] **T015** [US3] [Cassius] Implement `DeepIdentificationRepository.DeleteJob(userID, jobID uint) error` — single tx over provider-runs, events, artifacts, then job row. Return `ErrDeepJobNotTerminal` for non-terminal state.
- [X] **T016** [US3] [Cassius] Implement `DeepIdentificationService.DeleteJob(userID, jobID uint) error` — orchestrates the repo tx then calls `artifactStore.DeleteJobArtifacts(id)` best-effort (existing helper). File-unlink errors log-only.
- [X] **T017** [US3] [Cassius] Add `DeepIdentificationHandler.DeleteJob` gin handler + Swagger annotations. Route: `protected.DELETE("/deep-identification/jobs/:id", writeRateLimit, deepIdentificationHandler.DeleteJob)` in `src/api/main.go`.

---

## Phase 4: Backend — Re-apply Idempotency (US2)

### Tests first

- [X] **T018** [P] [US2] [Cassius] Extend `src/api/services/deep_identification_proposal_test.go`:
  - Re-apply with same target and linked coin present → returns existing coin id, no new coin.
  - Re-apply with same target but linked coin was deleted → creates new coin and refreshes linkage.
  - Re-apply with different target than initial (only where source/target coupling permits) → creates new linkage of new target.
  - Existing `ErrDeepProposalTargetMismatch` semantics preserved.
- [X] **T019** [P] [US2] [Cassius] Extend `deep_identification_repository_test.go`: `ApplyJob` (or its successor `RelinkAppliedJob`) supports re-writing `applied_coin_id`/`applied_draft_id`/`applied_at`.

### Implementation

- [X] **T020** [US2] [Cassius] Loosen `DeepIdentificationRepository.ApplyJob` to always update within `WHERE id = ? AND user_id = ?` (drop the `applied_at IS NULL` guard). Keep the return `(rowsAffected, err)` shape.
- [X] **T021** [US2] [Cassius] Add helper `DeepIdentificationProposalService.resolveExistingLinkage(job, target)` that returns `(coinID, exists bool, err)` by checking `Coin.FindByID(job.AppliedCoinID, userID)` for the coin/wishlist path and the draft repo for the draft path.
- [X] **T022** [US2] [Cassius] Update `DeepIdentificationProposalService.Apply` to consult `resolveExistingLinkage` first: return the existing linkage on match; otherwise fall through to the existing write path and record the new linkage via the loosened `ApplyJob`. Remove `ErrDeepProposalAlreadyApplied` from the return set.
- [X] **T023** [US2] [Cassius] Update handler `ApplyProposal` in `src/api/handlers/deep_identification.go` to always return 200 with the resolved linkage (no 409 for re-apply). Update Swagger annotations.

---

## Phase 5: Backend — List Badge (`appliedCoinExists`)

### Tests first

- [X] **T024** [P] [US4] [Cassius] Extend `deep_identification_repository_test.go` for `ListJobs`: returned rows include `AppliedCoinExists` = true when the applied coin resolves to an existing owned non-deleted row; false when the coin was deleted or when `AppliedCoinID` is nil.

### Implementation

- [X] **T025** [US4] [Cassius] Add a computed field on the repository result (either as a projected column via `Select` + `EXISTS` subquery, or a lightweight post-fetch batch check). Prefer the SQL path to keep it O(1) per row.
- [X] **T026** [US4] [Cassius] Extend `deepJobDTO` in `src/api/handlers/deep_identification.go` with `AppliedCoinExists bool \`json:"appliedCoinExists"\``.

---

## Phase 6: Backend — Featured Coin Wishlist Eligibility (US5)

### Tests first

- [X] **T027** [P] [US5] [Cassius] Extend `src/api/repository/featured_coin_repository_test.go`:
  - `PickNextCoinID(userID, includeWishlist=true)` returns wishlist coins in the fair cycle (never-shown-first).
  - `PickNextCoinID(userID, includeWishlist=false)` matches today's byte-identical owned-only behavior.
  - Sold coins remain excluded in both modes.
- [X] **T028** [P] [US5] [Cassius] Extend `src/api/services/coin_of_day_scheduler_test.go`:
  - Owned-source pick creates `FeaturedCoin{SourceType: "owned"}` and does NOT call the agent proxy.
  - Wishlist-source pick creates `FeaturedCoin{SourceType: "wishlist"}` and DOES call `agentProxy.WishlistFeaturedSummary`.
  - Agent error → fallback summary applied with fixed preamble; `FeaturedCoin` is still persisted; notification still fires.
  - User with `CoinOfDayIncludeWishlist=false` skips wishlist coins entirely.

### Implementation

- [X] **T029** [US5] [Cassius] Update `FeaturedCoinRepository.PickNextCoinID` to accept `includeWishlist bool`; update the `WHERE` clause per plan D6.
- [X] **T030** [US5] [Cassius] Extend `UserRepository.ListCoinOfDayEnabled` (or a peer method) so the scheduler can read each user's `CoinOfDayIncludeWishlist` per iteration. Prefer adding the flag onto the returned `User` struct rather than a second query.
- [X] **T031** [US5] [Cassius] Update `CoinOfDayScheduler.runCycleWithTrigger` to:
  1. Call the widened `PickNextCoinID(user.ID, user.CoinOfDayIncludeWishlist)`.
  2. If picked coin `IsWishlist`, call `agentProxy.WishlistFeaturedSummary(ctx, req)` with a 10s timeout.
  3. On agent success, use returned summary. On any error/timeout/empty, use `buildWishlistFallbackSummary(coin)` (new helper prefixing `buildCoinSummary` with `"From your wishlist — "`).
  4. Persist `FeaturedCoin{SourceType: "wishlist"|"owned", Summary: ...}`.
- [X] **T032** [US5] [Cassius] Add `WishlistFeaturedSummary(ctx, req) (string, error)` to `services/agent_proxy.go` (or an extension file) that POSTs to the new agent route and returns the trimmed summary.

---

## Phase 7: Python Agent — Wishlist Rationale Endpoint (US5)

### Tests first

- [X] **T033** [P] [US5] [Brutus] Add `src/agent/tests/test_wishlist_featured_summary.py`:
  - Anthropic mocked → returns bounded summary ≤ 500 chars, no leaked provider metadata, non-empty.
  - Ollama mocked → same shape.
  - Empty model response → route returns 502 with generic message (Go side treats as fallback).
  - Missing required coin fields → 400 validation.
  - Summary is trimmed of newlines and truncated to 500 chars if the model over-produces.
  - (Additive, in a dedicated new `test_wishlist_featured_summary_regression.py` file to avoid
    collision: Anthropic/Ollama parity of response shape, whitespace-only→502, strict-schema
    rejection of unknown fields, missing internal-service-token→401, no-DB-access sanity check.)

### Implementation

- [ ] **T034** [US5] [Brutus] Add `WishlistFeaturedSummaryRequest` / `WishlistFeaturedSummaryResponse` Pydantic models in `src/agent/app/models/`.
- [ ] **T035** [US5] [Brutus] Add route `POST /collection/wishlist-featured-summary` in `src/agent/app/routes/` (new file or existing `collection.py` if present) using `get_chat_model(provider, model)` per Aurearia's provider abstraction.
- [ ] **T036** [US5] [Brutus] System prompt: brief, factual, one-paragraph rationale; explicit "do not invent facts." Register route in `src/agent/app/main.py`.

---

## Phase 8: Frontend — Run History Page + Detail Actions (US1, US2, US3, US4)

### Tests first

- [X] **T037** [P] [US4] [Aurelia] Add `src/web/src/pages/__tests__/DeepAnalysisHistoryPage.test.ts`:
  - Renders list with status pill, source badge, applied-linkage badge.
  - Cursor pagination advances on "Load more".
  - Row click routes to `/deep-analysis?jobId=NN`.
- [X] **T038** [P] [US3] [Aurelia] Extend `src/web/src/pages/__tests__/DeepAnalysisPage.test.ts`:
  - Delete button visible only for terminal-state jobs.
  - Confirm dialog required; on confirm calls `DELETE /deep-identification/jobs/{id}`.
- [X] **T039** [P] [US2] [Aurelia] Extend `DeepAnalysisPage.test.ts`:
  - Save-to-Wishlist button dispatches apply, disables while pending.
  - Repeated click returns existing linkage (mocked 200), UI shows "Already saved" state.
  - Deleted-coin case (mocked new-coin-id return) refreshes linkage state.

### Implementation

- [X] **T040** [US4] [Aurelia] Add `src/web/src/pages/DeepAnalysisHistoryPage.vue` reverse-chrono list per plan §"Frontend Changes". Reuse `.chip`, `.badge`, `.btn` classes.
- [X] **T041** [US4] [Aurelia] Register `/deep-analysis/history` route in the router; add nav entry alongside existing Deep Analysis link.
- [X] **T042** [US3] [Aurelia] Add delete button + confirm modal to `DeepAnalysisPage.vue`; wire to `deleteDeepJob(id)` in `src/web/src/api/client.ts`.
- [X] **T043** [US2] [Aurelia] Add / adjust Save-to-Wishlist / Save-to-Collection action in `DeepAnalysisPage.vue` to reflect FR-007…011 idempotency (button label/state based on `job.appliedCoinId` + `job.appliedCoinExists`).
- [X] **T044** [US2, US3] [Aurelia] Extend `src/web/src/api/client.ts` with `deleteDeepJob(id)` and confirm existing `applyDeepProposal(...)` handles the new "already-linked" 200 shape without treating it as an error.

---

## Phase 9: Frontend — Featured Coin Modal & Settings (US5)

### Tests first

- [X] **T045** [P] [US5] [Aurelia] Extend `src/web/src/components/__tests__/FeaturedCoinModal.test.ts`:
  - `sourceType === "wishlist"` renders "Wishlist" badge and "Move to Collection" primary action.
  - "Move to Collection" click dispatches existing coin-update endpoint with `{ isWishlist: false }`.
  - `sourceType === "owned"` renders exactly as today.

### Implementation

- [X] **T046** [US5] [Aurelia] Update `FeaturedCoinModal.vue` per plan D9.
- [X] **T047** [US5] [Aurelia] Add `CoinOfDayIncludeWishlist` toggle to Settings → Account (mirror existing `CoinOfDayEnabled` toggle); wire to user-update API.

---

## Phase 10: Docs & Polish

- [X] **T048** [X] [Cassius] Regenerate Swagger docs (`swag init` or existing task); confirm new/modified endpoints appear.
- [ ] **T049** [X] [Maximus] Update `docs/prd.md` §"Deep Identification" and §"Coin of the Day" with the new persistence, delete, re-apply, and wishlist-source behaviors. Cite spec 354.
- [ ] **T050** [X] [Maximus] Add a short entry to `.squad/decisions.md` referencing this feature's inbox file once merged.
- [X] **T051** [X] [Brutus] Full-suite validation: `go test ./...`, `npm run type-check` + `npx vitest run`, `pytest tests/`. Attach summary to PR.

---

## Parallel Execution Boundaries

- **Cassius** owns everything in `src/api/`. Safe to parallelize across phases 3 (retention/delete), 4 (re-apply), 5 (list badge), 6 (featured coin) — each set of files is disjoint (`deep_identification_*.go` vs. `featured_coin_repository.go` vs. `coin_of_day_scheduler.go`).
- **Brutus** owns everything in `src/agent/` (phase 7). Fully independent of Cassius's Go changes; only integration seam is the Go proxy call in T032, which stubs the HTTP shape.
- **Aurelia** owns everything in `src/web/` (phases 8 + 9). Depends on Cassius's API contract stubs (delete route, `appliedCoinExists`, `sourceType`) — those can be typed against `src/web/src/types/index.ts` early so Aurelia can start against mocks while Cassius implements.
- **Maximus** reviews architecture/DoD on every PR and owns the PRD update in T049.

## Dependency Notes

- T005–T008 (data model) block T009…T032 (they all depend on the model shape).
- T032 (Go proxy method) can be developed against a mocked HTTP client independently of T033–T036; wire-up test happens once Brutus lands T035.
- T044 (frontend client changes) can be implemented against a locally mocked axios adapter before Cassius merges T017/T023/T026.
- T048 depends on T017, T023, T026 being merged.
