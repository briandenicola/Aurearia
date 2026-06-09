# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins QA/testing across Go API, Vue frontend, and Python agent
- **Architecture:** Tests enforce constitution-backed layer boundaries, auth/ownership guarantees, and feature acceptance criteria.

## Core Context

- Brutus owns testing and review. Durable test patterns: in-memory SQLite + httptest/Gin for Go handlers/repos/services, Vitest with axios/localStorage mocks for frontend, Ruff/pytest for Python, and architecture tests for import boundaries.
- Testing baseline expanded from minimal coverage to auth/security, API client, auth store, settings, parser, and component tests. `docs/testing.md` is canonical test strategy; known gaps include browser E2E, Go cross-process integration, and Python static type checking.
- Strict Lockout applies: when a reviewer marks BLOCK, the blocked implementer does not revise until the block is cleared by reviewer or delegated agent.
- Durable QA rules: verify actual code and data paths, not just claims; classify pre-existing failures separately; document non-blocking nits without blocking otherwise passing features.

## Recent Updates

- **2026-06-01:** QA contract note: coins now support nullable `storageLocationId` and optional `storageLocation`; storage-location CRUD is under protected `/api/storage-locations`; deleting a location in use returns 409 with a coin count. Settings Data now covers Tags + Storage Locations, while backups/imports/API keys moved to `Backups & Keys`.

- **2026-05-31:** Feature #219 QA approved: 12/12 functional requirements satisfied, route wiring/auth guards verified, vue type-check/build clean, no regressions except unrelated pre-existing test issues.
- **2026-05-31:** #216 Principle V token remediation executed under Strict Lockout: added design tokens and replaced flagged hardcoded colors; lint/build clean; Maximus later approved.
- **2026-06-01:** #218 polish validation approved: capability middleware tests added, Go build/vet/test clean, frontend build/lint clean, quickstart scenarios A/B/C and negative scenarios N1-N6 traced to code.
- **2026-06-01:** #218 BLOCK resolution applied: all Gin context type assertions in external tool handlers now use comma-ok guards returning 401/403 instead of risking panic; Go build/vet/test clean.
- **2026-06-01:** "Assign Location" bulk action feature — Cassius backend + Aurelia frontend parallel implementation verified aligned (POST /coins/bulk with action:assign-location); nil-safe NULL updates; BulkLocationPickerModal and BulkActionBar extension; all tests pass.
- **2026-06-03:** Aurelia refactor: per-coin value trend moved from Stats page to dedicated `/coin/:id/valuation` subpage. New route adds one additional coin-detail page (CoinDetailValuationPage.vue). Build/type-check/lint all passed. No backend changes; existing `/coins/:id/value-history` endpoint unchanged.

## Learnings

- **2026-06-07:** Era/Category + Coin Lookup QA Pass
  - **Era/Category Settings:** Backend 5 passing tests (CoinCategories/CoinEras defaults + customization). Frontend `options.ts` utility with 30-test spec (parse/format/roundtrip/edge cases). All tests pass: `npm test -- options.spec.ts` ✅ (30/30), `go test ./services/...` ✅. **AdminPage v-model claim was false alarm** — component uses correct prop/event pattern (lines 93-100 use `:category-options` + `@update:category-options`, no v-model).
  - **Coin Lookup State:** MVP implementation in progress. `CoinLookupPage.vue` exists with NGC cert display (line 107-121) but no normalization/extraction tests yet. NGC sample `823160-093` not found in codebase — likely deferred to Phase 2 per decisions.md (Numista-only MVP). Lookup endpoint (`POST /api/agent/coin-lookup`) contract not yet implemented (depends on Python `coin_lookup.py` team).
  - **Validation Status:** Go build ✅, Go vet ✅, Go tests 143 passing ✅. Frontend type-check ✅ (`vue-tsc --build` clean), build ✅ (11.06s, 105 PWA assets), lint ⚠️ 23 warnings (18 in CoinLookupPage.vue formatting, 5 pre-existing).
  - **Regression Coverage:** Era/category settings changes validated. Existing architecture tests pass (layered imports enforced). Frontend option parser fully tested (30 scenarios).
  - **QA Blockers:** None. Coin Lookup lint warnings are formatting-only (indentation, tag closing); type-check passed so no runtime risk. NGC extraction tests deferred until backend implements NGC API (post-MVP per architecture decision).

- **2026-06-09:** Coin Sets + Memberships Regression Coverage
  - **Regression:** User reported PUT /api/coins/8 failure: `coin_set_memberships.added_at` is NOT NULL but naive coin update only inserts `coin_id,set_id`.
  - **Root Cause Analysis:** `CoinRepository.Update` needed to omit relationship fields so a bound update payload cannot make GORM auto-sync many-to-many associations. `CoinSetMembership.AddedAt` is a required custom join-table field and must be populated through `SetRepository.AddCoinToSet`, not GORM's default association insert.
  - **Coverage Added:** Regression tests now cover both repository behavior and the real HTTP update path:
    1. `TestCoinRepository_Update_PreservesSets`: Proves that updating a coin with an existing set membership does NOT corrupt or recreate the membership. Verifies `AddedAt` remains non-zero and unchanged after update.
    2. `TestCoinRepository_Update_WithSetsField`: Proves that passing `coin.Sets` in update payload is safely ignored via `Omit("Sets")`. Ensures existing memberships are untouched and new sets aren't added.
    3. `TestCoinHandler_Update_WithSetsPayloadPreservesMemberships`: Proves `PUT /api/coins/:id` with a `sets` JSON payload returns 200, updates the coin, and preserves the original set membership and `AddedAt`.
  - **Test Infrastructure:** Added `CoinSet` and `CoinSetMembership` to `setupTestDB` AutoMigrate (previously missing from test schema).
  - **Test Results:** Targeted handler and repository regressions pass. Full Go API suite passes.
  - **Verdict:** The update path now has exact regression coverage for the production failure, not just repository-only coverage.
