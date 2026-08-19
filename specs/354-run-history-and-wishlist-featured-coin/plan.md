# Implementation Plan: Deep-Identification Run History & Wishlist-Eligible Coin of the Day

**Branch**: `354-run-history-and-wishlist-featured-coin` | **Date**: 2026-08-19 | **Spec**: `./spec.md`
**Input**: Feature specification from `specs/354-run-history-and-wishlist-featured-coin/spec.md`

## Summary

Two joined capabilities delivered as one feature because they share the "revisit persisted work later" theme and touch overlapping subsystems (deep identification + Coin of the Day + notifications + Vue history UX):

1. **Persistent deep-identification run history**: retain terminal-completed / terminal-partial `DeepIdentificationJob` rows and their obverse/reverse artifacts indefinitely; add a user-invoked `DELETE` endpoint; loosen `ApplyJob` to a "per (job, target, linked-coin-existence)" idempotency model so users can save after having changed their mind; add a frontend history route.
2. **Wishlist-eligible Coin of the Day**: extend `FeaturedCoin` with a `SourceType` column; widen `PickNextCoinID` to include wishlist coins; generate/cache a bounded "why this belongs in your collection" summary via the Python agent (Go proxied); reflect origin in the modal with a "Move to Collection" CTA. Owned-coin behavior remains byte-identical.

All Python-side AI work is stateless and per-request, matching the existing `agent_proxy.go` boundary. No changes to the Deep Identification pipeline, vision-first orchestration, structured results, availability observability, or Wishlist Search Alerts.

## Technical Context

**Language/Version**: Go 1.26.6 (API), TypeScript 5 / Vue 3 (web), Python 3.12 (agent).
**Primary Dependencies**: Gin, GORM, SQLite; existing `DeepIdentificationService` / `DeepIdentificationProposalService` / `DeepIdentificationRepository` / `deep_identification_janitor.go`; `CoinOfDayScheduler` / `FeaturedCoinRepository` / `NotificationService`; `agent_proxy.go`; FastAPI + LangChain on the agent side.
**Storage**: SQLite via GORM `AutoMigrate` only. Two additive DDL changes: (a) `deep_identification_jobs.expires_at` becomes nullable; (b) `featured_coins.source_type VARCHAR NOT NULL DEFAULT 'owned'`. Optional one-column addition on `users` for opt-out (`coin_of_day_include_wishlist BOOLEAN NOT NULL DEFAULT 1`).
**Testing**: `go test ./...` (architecture, unit, service, repository, handler), `npm run type-check` + `npx vitest` for `DeepAnalysisPage` / new history page / `FeaturedCoinModal`, `pytest tests/` for the new agent endpoint.
**Target Platform**: Linux server (single instance), PWA client.
**Project Type**: Web application (Go API + Vue SPA + stateless Python agent).
**Performance Goals**: No regression versus current job list P95; wishlist summary generation capped at 10s per pick with sequential per-user execution (matches today's `runCycleWithTrigger` loop).
**Constraints**: Additive DDL only. No destructive migration. No changes to Python `/deep_identify` contracts. No new AI providers.
**Scale/Scope**: ≤ 50 users, ≤ ~2 000 lifetime deep runs across all users, ≤ ~500 owned + wishlist coins per user.

## Constitution Check

- **Principle I (Layered Architecture)**: All backend changes remain in `handlers/` → `services/` → `repository/` → `models/`. The new delete endpoint lives on `DeepIdentificationHandler`; retention branching lives inside `DeepIdentificationRepository` and `deep_identification_janitor.go`; the `SourceType` widening lives in `FeaturedCoinRepository.PickNextCoinID`; the wishlist-summary call lives in `CoinOfDayScheduler` and calls `agent_proxy.WishlistFeaturedSummary(...)`. Enforced by `architecture_test.go`.
- **Principle IV (Simple Complete Proportional)**: We reuse the existing `DeepIdentificationJob` state machine, the existing artifact store, the existing agent proxy, and the existing `FeaturedCoin` / notification pipeline. No new subsystems. Two additive columns, one nullable column change, one new endpoint, one new Python route.
- **Principle V (Security, Privacy)**: All new endpoints are `protected` JWT-scoped; non-owner deletes return 404. Wishlist-summary Python endpoint receives no secrets and returns bounded text. No file paths, provider IDs, or upstream errors are surfaced to the client.
- **Principle VI (Consistent UX)**: Frontend uses existing chips, buttons, tokens; new history page mirrors the availability-runs history table (spec 353) rather than inventing a layout.
- **§17 Quality Gate**: Targeted `go test` on `services/`, `repository/`, `handlers/` deep-identification packages and coin-of-day packages; `vue-tsc` + affected `vitest` files; `pytest` for the new agent route; full `go test ./...` and `npm run build` before merge.
- **§21 Definition of Done**: See §"Definition of Done" checklist below.

No violations. Complexity waivers: none. No ADR required (no locked-file amendments; existing specs 344 / 351 / 352 / 001 are extended, not retroactively altered).

## Design Decisions

### D1. `ExpiresAt` becomes nullable; terminal-completed/partial set it to `NULL`

Rather than repurposing `EventsPrunedAt` or introducing a new `RetainedAt` column, we make the existing `deep_identification_jobs.expires_at` column nullable and clear it inside the same transaction that settles a run to `completed` or `partial`. `SettleTerminal` gains a bool flag "clear expiry"; the repository call site in `pipeline_runner` passes `true` for those two statuses. The janitor's `ListExpiredJobIDs` gains a `expires_at IS NOT NULL AND expires_at <= ?` guard. This preserves the existing `idx_deep_jobs_expires` and keeps all deletion machinery unchanged for failed/cancelled rows.

### D2. Re-apply idempotency = per (job, target, linked-coin-existence)

`DeepIdentificationProposalService.Apply` grows an "already-applied fast path": before invoking `applyToWishlist` / `applyToCoin` / `applyToDraft`, check `job.AppliedCoinID` / `AppliedDraftID` against the requested target. If the linked entity exists and is owned by the caller and matches the target, return the existing linkage in `DeepApplyResult` and skip the write path. Otherwise proceed as fresh, and let `ApplyJob` overwrite in place (loosen its `WHERE applied_at IS NULL` guard to unconditional per-owner update in a transaction). This gives users "click Save Twice" idempotency without duplicating coins, while allowing them to re-save after deleting the previously created coin.

### D3. Delete endpoint is transactional and owner-scoped

`DELETE /deep-identification/jobs/{id}` calls a new `DeepIdentificationService.DeleteJob(userID, jobID)` that (a) loads the job with owner scope and 404s on miss, (b) rejects non-terminal state with 409, (c) inside a single DB transaction deletes provider-run, event, artifact, and job rows, (d) after DB commit calls the artifact store's existing `DeleteJobArtifacts(id)` for best-effort file unlinking. A file-unlink error is logged but does not fail the request — the DB is the source of truth. On DB transaction failure, nothing is deleted.

### D4. List badge is server-computed via cheap `EXISTS`

`GET /deep-identification/jobs` grows a subquery-based `appliedCoinExists bool` field, computed once per list row via a correlated `EXISTS (SELECT 1 FROM coins WHERE id = j.applied_coin_id AND user_id = j.user_id AND deleted_at IS NULL)`. Free with the existing index; avoids N+1 fetches in the frontend.

### D5. `FeaturedCoin.SourceType` typed as string enum with default `"owned"`

The column is `VARCHAR(16) NOT NULL DEFAULT 'owned'` so existing rows migrate to `owned` on `AutoMigrate`. New picks set `owned` or `wishlist`. Frontend/API treat any unrecognized value as `owned` defensively.

### D6. Combined-pool cycle in one SQL

`PickNextCoinID` widens its `WHERE` to `AND is_sold = 0 AND (is_wishlist = 0 OR ? = 1)` where `?` is the caller's `IncludeWishlist` flag. The existing three-key sort (`(last_shown IS NULL) DESC, last_shown ASC, c.id ASC`) is preserved unchanged, so cycle fairness carries naturally.

### D7. Wishlist summary lives on Python; Go proxies

A new agent route (e.g. `POST /collection/wishlist-featured-summary`) accepts coin metadata + provider hint + optional user tone context and returns `{ summary: string }` (soft-capped to 500 chars server-side). `agent_proxy.go` gains a `WishlistFeaturedSummary(ctx, req)` method. `CoinOfDayScheduler` calls it when the picked coin has `IsWishlist == true`. All AI generation stays on the agent per Aurearia's canonical rule.

### D8. Deterministic fallback preserves the pick

`CoinOfDayScheduler.buildWishlistSummary(coin, agentErr)` returns a fixed-prefix static summary derived from `buildCoinSummary` when the agent errors, times out, is disabled, or returns an empty payload. Fallback is logged but never surfaced to the client. The pick, notification, and modal continue to work.

### D9. Modal "Move to Collection" reuses existing coin-update path

The Vue `FeaturedCoinModal` gains a conditional CTA that calls the existing coin-update endpoint with `{ isWishlist: false }`. No new backend endpoint; no new validation surface. Owner scoping and journal recording flow through existing `CoinService.UpdateCoinWithFields`.

### D10. Backward-compatible notification schema

`NotificationService.NotifyCoinOfDay` signature is unchanged. The `sourceType` field lives on the `FeaturedCoin` payload the modal fetches via `GET /featured-coins/{id}`. No consumer relying on `type="coin_of_day"` semantics regresses.

## Data Model Changes

### `deep_identification_jobs`

- `expires_at` becomes nullable (was `NOT NULL` behavior enforced at write-time). Additive schema change compatible with SQLite `ALTER COLUMN` avoidance via GORM's tolerant AutoMigrate on nullable relaxation. If AutoMigrate cannot relax nullability in place, `SettleTerminal` MAY instead set `expires_at` to a sentinel `9999-12-31T00:00:00Z`. Choice deferred to Cassius per compatibility test outcome.
- No new indexes. `idx_deep_jobs_expires` continues to serve the `IS NOT NULL AND <= ?` predicate.

### `featured_coins`

- New column `source_type VARCHAR(16) NOT NULL DEFAULT 'owned'`.
- No new indexes required — filtered queries on `source_type` are not in scope.

### `users`

- New column `coin_of_day_include_wishlist BOOLEAN NOT NULL DEFAULT 1`.
- Exposed as a boolean toggle in the account settings surface Aurelia already owns.

### `deep_identification_artifacts`, `deep_identification_events`, `deep_identification_provider_runs`

- **No schema change.** `DELETE /deep-identification/jobs/{id}` performs `WHERE job_id = ?` deletes on each in one transaction.

## API Changes

### New

- `DELETE /deep-identification/jobs/{id}` → 204 on success; 404 non-owner; 409 non-terminal.
- Python: `POST /collection/wishlist-featured-summary` (agent internal, called by Go only) →
  ```json
  {
    "summary": "This bronze follis of Diocletian would round out your Tetrarchic run..."
  }
  ```
  Input:
  ```json
  {
    "coin": {
      "name": "...", "era": "...", "category": "...", "denomination": "...",
      "ruler": "...", "mint": "...", "obverse_analysis": "...", "reverse_analysis": "...",
      "ai_analysis": "..."
    },
    "provider": "anthropic|ollama",
    "provider_model": "claude-3.5-sonnet",
    "user_display_name": "brian"
  }
  ```

### Modified

- `GET /deep-identification/jobs` response `deepJobDTO` gains `appliedCoinExists bool`.
- `POST /deep-identification/jobs/{id}/apply` no longer returns `ErrDeepProposalAlreadyApplied` under the FR-008/009 idempotency rule; instead returns 200 with the existing linkage or the new one.
- `GET /featured-coins/{id}` and `GET /featured-coins/latest` response payload gains `sourceType`.

### Unchanged

- All other deep-identification endpoints, SSE contract, Python `/deep_identify`, availability endpoints, Wishlist Search Alerts, notification schema.

## Frontend Changes

- **New page** `src/web/src/pages/DeepAnalysisHistoryPage.vue` (route `/deep-analysis/history`): reverse-chronological list of terminal runs, status pill, source badge, thumbnail from artifact endpoint, "Saved to Collection/Wishlist" badge driven by `appliedCoinExists` + `sourceType`, per-row delete action with confirm dialog.
- **`DeepAnalysisPage.vue`** gains: (a) a delete button on terminal runs, (b) a "Save to Wishlist" / "Save to Collection" button that behaves idempotently across repeated clicks per FR-007…011.
- **`FeaturedCoinModal.vue`** gains: conditional "Wishlist" badge and "Move to Collection" primary action when `featuredCoin.sourceType === 'wishlist'`.
- **Settings → Account**: existing `CoinOfDayEnabled` toggle joined by a new `CoinOfDayIncludeWishlist` toggle (default on).
- All new UI reuses existing tokens (`--accent-gold`, `--bg-card`, `.chip`, `.btn`, `.section-label`).

## Python Agent Changes

- New route in `app/routes/collection.py` (or `app/routes/wishlist.py` if it fits the layout) named `wishlist_featured_summary`.
- Uses `get_chat_model()` from `app/llm/provider.py` — **no web search required**; keep it fast and cheap.
- Prompt template is a short system message: "You write brief, factual, one-paragraph rationales for why a specific coin would be a good addition to a collector's collection. Do not invent facts not present in the input."
- Output post-processed to strip newlines, trim to ≤ 500 chars, and validated as non-empty before return.

## Risks & Mitigations

- **R1**: GORM/glebarez SQLite may not tolerate relaxing `NOT NULL` on `expires_at` in place → **Mitigation**: fallback to sentinel far-future timestamp (D1). Add a migration regression test parallel to `feature353_migration_order_regression_test.go`.
- **R2**: Users repeatedly clicking "Save" on a run whose linked coin was concurrently deleted from another tab → **Mitigation**: FR-009 explicitly handles this by falling through to fresh apply; `applyToWishlist` / `applyToCoin` remain in a single tx so the linked coin is never half-created.
- **R3**: Python agent slow response blocking daily cycle → **Mitigation**: NFR-003 hard timeout; NFR-004 sequential-only. Owned coins bypass the call entirely, so an outage never blocks the majority path.
- **R4**: Notification consumers assume owned-source semantics → **Mitigation**: `type="coin_of_day"` and `referenceId` are unchanged (FR-029). `sourceType` is a new, additive JSON field on the fetched `FeaturedCoin`.
- **R5**: Long-term retention grows the `deep_identification_jobs` table without bound → **Mitigation**: user-invoked delete is the primary pressure valve; observed usage does not warrant a global cap. If needed later, a simple age-based archive is a straightforward follow-up.

## Definition of Done Checklist (from §21)

- [ ] `go test ./...` green, including new architecture, repository, service, handler tests.
- [ ] `npm run type-check` green in `src/web/`; `npx vitest run` green for touched files.
- [ ] `pytest tests/` green for `src/agent/`.
- [ ] Migration regression test proves nullable `expires_at` (or sentinel fallback) works on real SQLite with legacy fixtures.
- [ ] `DELETE /deep-identification/jobs/{id}` documented in Swagger and covered by handler tests (204 / 404 / 409 / cross-user 404).
- [ ] `sourceType` visible in `FeaturedCoin` API responses; Vue `FeaturedCoinModal` visibly differentiates.
- [ ] Wishlist rationale summary end-to-end tested with Anthropic mocked and Ollama mocked; fallback path proved by an agent-error mock.
- [ ] PR description cites Principle I + Principle IV + §17 and §21.
- [ ] Manual sanity: two owned + two wishlist coins → run five scheduler cycles → observe fair interleave and distinct summaries.

## Complexity Tracking

None. All additions are proportional to concrete requirements. No waivers requested.
