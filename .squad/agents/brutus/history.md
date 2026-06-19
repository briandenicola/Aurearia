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

- **2026-06-09:** F013 Critical Workflow Regression Strategy
  - **Inventory:** Existing backend coverage already protects auth/ownership, basic create/update/delete, set preservation, storage-location update, custom/legacy era, value snapshots/history, and API client sanitization. Frontend coverage remains source/unit-level only for edit-era and list/wishlist rendering; no browser workflow suite exists.
  - **Coverage Added:** Added focused handler regressions for `storageLocationId: null` clearing and structured reference replacement, and aligned coin handler test wiring with production reference/storage-location services.
  - **Bug Found:** Explicit storage-location clearing did not persist NULL through `CoinRepository.UpdateStorageLocationID`; fixed the helper to update by explicit owned coin query and verified with handler tests.
  - **Strategy:** Promoted F013 plan now carries the Brutus QA matrix: backend handler/service/repository tests first, golden fixtures second, deterministic browser tests third, F011 AI exploration later/advisory.
  - **Validation:** `go test -v ./handlers ./repository ./services` ✅, `go test -v ./...` ✅, `go vet ./...` ✅ from `src/api`.
- **2026-06-09:** F013 BLOCK resolution: presence-aware coin updates now use explicit GORM Select fields so false booleans, empty strings, and numeric zeros persist while omitted fields remain unchanged. Added HTTP handler and repository regressions for zero-value persistence plus storage-location null clear coverage remains dedicated.
- **2026-06-09:** F013 backend regression completion: added HTTP coverage for typed DTO unknown/read-only/broad relationship fields, non-owned storage rejection, and manual current-value history/snapshot side effects; service coverage now proves storage ownership, reference normalization, and value snapshots; repository coverage proves `UpdateStorageLocationID(nil)` persists NULL and clears stale preloaded storage pointers. `go test -v ./...`, `go vet ./...`, and `git diff --check` passed from the required scopes.
- **2026-06-09:** F013 Backend Regression Batch Completed
   - Tasks T011, T012, T013 marked complete
   - Regression test pattern documented for custom join-table fields
   - Orchestration log: `.squad/orchestration-log/2026-06-09T12-51-39Z-brutus.md`

- **2026-06-09:** F013 fixture docs/coverage slice: backend fixtures already had formal trait/persistence checks; frontend fixtures now have a catalog regression. Keep fixture matrices documented in `docs/testing.md` and defer browser workflow commands until Phase 4.
- **2026-06-09 (13:09:16):** F013 Phase 3 golden fixtures complete (T016/T017). Updated `docs/testing.md` with F013 fixture matrix and usage patterns. Added `src/web/src/test/fixtures/coins.test.ts` covering fixture catalog/clone/association coverage. Approved by Maximus Lead Review. Go and web build/test/lint all pass. Orchestration log: `.squad/orchestration-log/2026-06-09T13-09-16-brutus.md`
- **2026-06-09:** Playwright hygiene matters: keep `src/web/test-results/` and `src/web/playwright-report/` ignored, and remove stale docs once browser smoke coverage exists.

- **2026-06-09:** F013 Phase 4 Browser Workflow Infrastructure Hygiene (T018–T021, APPROVED)
  - **Strict Lockout Remediation:** Aurelia's browser suite BLOCKED by Maximus on .gitignore gaps + generated outputs present + stale docs TODO
  - **Independent QA Fix:** Added `.gitignore` entries for Playwright output paths (`src/web/test-results/`, `src/web/playwright-report/`), removed generated `src/web/test-results/` directory, removed stale browser E2E TODO from `docs/testing.md`
  - **Validation:** `npm run test:browser` — 4 tests passing, `git diff --check` — no formatting violations, design token changes only to `.gitignore` and docs

- **2026-06-10:** Coin of the Day Pushover Link Review (Cycle 1 BLOCK → Cycle 2 APPROVED)
  - **Cycle 1:** Cassius initial implementation used relative `/coin/{coinID}` URLs in Pushover payloads. Issue: Pushover notifications open in system notification center outside app context; relative URLs fail to navigate. ISSUED BLOCK (STRICT LOCKOUT §18.2). Assigned revision to Aurelia.
  - **Cycle 2:** Aurelia added `PublicAppURL` admin setting; links now build as absolute `http(s)://host/coin/{coinID}` (trim trailing slashes, join host + path). When setting blank/invalid, Pushover alerts omit the `url` field and HTML link anchor. In-app notification `ReferenceID = FeaturedCoin.ID` behavior unchanged.
  - **Coverage:** Test assertions added for configured and unconfigured link behavior; backend `go test -v ./services` ✅; frontend `npm run type-check`, `npm run build` ✅.
  - **Verdict:** BLOCK CLEARED. Feature ready for merge. Pushover external-link workflow is now explicit, tested, and deployment-configurable per Principle V.
  - Orchestration log: `.squad/orchestration-log/2026-06-10T20-31-52Z-brutus.md`
  - **Principle Compliance:** Principle VI (Testing Infrastructure), Principle VIII (CI/CD & Build Hygiene), §18.2 (Strict Lockout)
  - **Review Outcome:** Maximus Lead Re-review APPROVED revised state with all issues resolved
  - **Key Learning:** Playwright generates deterministic artifacts (test-results/, playwright-report/, test data); must be `.gitignore`-excluded before integration into CI baseline

- **2026-06-09:** F013 T027/T028 command/docs slice: added root `task test-critical-workflows` as the stable ergonomic alias for the existing web `npm run test:browser` Playwright suite. Documented that the suite uses mocked `/api/*` routes, authenticated localStorage setup, and frontend golden fixtures; initial coverage was authenticated session setup, manual add, one-field edit, storage-location set/clear, and tags/sets; later superseded by the final T024-T026 workflow docs once those tests landed.
- **2026-06-09:** F013 final workflow docs BLOCK fix: keep `docs/testing.md` coverage bullets in lockstep with Playwright workflow additions so QA docs reflect the actual suite state.
- **2026-06-09:** F013 Browser split acceptance review: verified Playwright workflow coverage, root `task test-critical-workflows` docs, spec/plan F011 guardrails, and passed `git diff --check`; marked both remaining Browser split checklist items complete.

- **2026-06-09 (13:45:07):** F013 Docs Coverage Alignment — Strict Lockout Remediation APPROVED
   - **Initial Block:** Maximus blocked F013 Phase 4 signoff because `docs/testing.md` listed T024–T026 under "Remaining Workflows" instead of "Current Coverage"; stale browser E2E TODO also present
   - **Strict Lockout §18.2 Delegation:** Per constitution Strict Lockout process, Brutus delegated independent docs refresh
   - **Remediation Actions:** Moved T024–T026 from "Remaining" → "Current Coverage" section; removed stale browser E2E TODO; kept Firefox/Safari multi-browser note as explicit backlog item
   - **Validation:** git diff --check ✅ (no whitespace/line-ending issues); docs-only changes, no code impact
   - **Lockout Clearance:** ✅ COMPLETE — docs state now reflects actual Playwright suite (9 tests), no stale backlog items, §18.2 process honored
   - **Maximus Re-Review:** ✅ APPROVED — T024–T026 workflows + docs alignment internally consistent, Principle VIII (CI/CD & Build Hygiene) satisfied, all Quality Gate §17 checks passed
   - **Orchestration Log:** `.squad/orchestration-log/2026-06-09T13-45-07Z-brutus.md`

- **2026-06-18:** Mint location QA validation completed. Backend auth split, service validation, duplicate handling, seed idempotency, and architecture layering reviewed; Go tests/vet passed after a transient vet re-run. Frontend map now loads `/mint-locations` before active collection grouping and admin CRUD has client coordinate validation + comma/newline alias parsing. Fixed stale `MintMapPage.test.ts` mocks so regression tests cover backend-provided mint locations; full Vitest/type-check/build passed. `git diff --check` initially found trailing whitespace in Aurelia history; QA trimmed it.

- **2026-06-18:** Mint Map 50-coin cap fix QA APPROVED. Verified `MintMapPage.vue` now bypasses the paginated store cache and fetches all active collection pages directly with `wishlist=false`, `sold=false`, `page`, and `limit=100`. Regression test covers 120 active Rome coins across two API pages and asserts both page requests plus mapped count `120`; targeted `npm.cmd run test -- MintMapPage.test.ts` and `npm.cmd run build` passed from `src/web`. Decision merged to `decisions.md` as: "Mint Map Frontend 50-Coin Limit — Pagination Loop Implementation". Orchestration log: `.squad/orchestration-log/2026-06-18T21-14-02Z-brutus.md`
- **2026-06-18:** Biometric login missing-challenge regression coverage added for issue #299. Backend `LoginBegin` now has a contract test proving the response exposes browser request options directly at `options.challenge` and is not nested under `options.publicKey`; frontend auth-store tests prove it can consume both the fixed flat shape and legacy nested shape, trims begin/finish usernames, and fails before invoking browser biometrics if challenge data is absent. Targeted `npm.cmd run test -- auth.test.ts` and `go test -v .\handlers -run "TestWebAuthnHandlerLoginBeginReturnsRequestOptionsWithChallenge|TestWebAuthnHandlerLoginFinish"` passed.

- **2026-06-18:** WebAuthn Backup Eligible flag storage QA review completed for issue #299. User reported 401 POST /api/auth/webauthn/login/finish with "Backup Eligible flag inconsistency detected during login validation". Root cause: go-webauthn library (v0.17.4+) validates `credential.Flags.BackupEligible` matches authenticator flag during login. Implementation already complete by another agent: model now has nullable `BackupEligible` and `BackupState` fields, handler stores flags during registration and loads with legacy fallback, repository `UpdateCredentialAuthData` updates flags on each login. Added three regression tests: `TestWebAuthnHandlerLoadCredentialsRestoresBackupFlags` (stored flags), `TestWebAuthnHandlerLoadCredentialsBootstrapsLegacyBackupFlagsFromAssertion` (legacy fallback), `TestWebAuthnHandlerLoadCredentialsKeepsStoredBackupEligibleOverAssertion` (precedence). All tests pass: `go test -v ./handlers -run TestWebAuthnHandler` ✅, `go test -v ./...` ✅, `go vet ./...` ✅. Decision: `.squad/decisions/inbox/brutus-webauthn-backup-eligible.md`.

- **2026-06-18T22:59:00Z — WebAuthn Backup Eligible Storage Fix (Coordinated Session):** Completed team coordination for issue #299 fix. Cassius implemented backup flag persistence in model/handler/repository; Brutus added three regression tests covering persistence, legacy bootstrap, and precedence; Coordinator validated full suite and regenerated OpenAPI. Session log: `.squad/log/2026-06-18T22-59-00Z-webauthn-backup-eligible.md`. Orchestration logs: `.squad/orchestration-log/2026-06-18T22-59-00Z-{cassius,brutus,coordinator}.md`. All tests pass; ready for merge.

- **2026-06-18:** PWA page header normalization QA APPROVED. Reviewed CalendarPage, ShowcasesPage, NotesPage, SetsPage, FollowersPage, and CoinLookupPage for PWA header pattern compliance after Aurelia's normalization pass. All pages follow AuctionsPage baseline: title left in `.page-header`, `.pwa-actions` with `pwa-icon-btn` (icon size 22) on PWA, `.header-actions` with text buttons (icon size 16) on desktop. `usePwa` imports verified. No hardcoded design-token regressions found (only structural `margin-top` spacing for layout flow). `npm.cmd run build` passed per context. No behavior removed, no visual contract misses. Sets page had prior visual shell rewrite but preserved SetDashboardCard integration and creation wizard. Notes page removed summary stats block per context but kept core editing/display modes intact.
- **2026-06-19:** Issues #311/#312 QA review: targeted API auth tests passed for JWT-only admin routes, API-key admin denial, and exact capability parsing. Full `go test ./...`/`go vet ./...` remain blocked only by pre-existing `services.containsString` redeclaration (`collection_tools_service_test.go` vs `coin_service.go`), unrelated to #311. Workflow publish gating reviewed: Docker publish now follows successful `Quality Gate` push runs on exact `head_sha`; security scan includes push triggers for `main`/`beta`; `git diff --check` passed, actionlint unavailable locally.

- **2026-06-19:** Issue #313 upload-media auth QA BLOCKED. Backend removed static `/uploads` and added DB-backed auth handlers, but frontend still renders private/showcase/avatar media via raw `<img src="/uploads/...">`; browser image requests do not carry the JWT from localStorage, so authenticated collection images 401 and public showcase images miss the new `/api/showcase/:slug/uploads/*` route. Targeted backend/frontend tests pass, frontend build passes, full Go test/vet remain blocked only by pre-existing `services.containsString` redeclaration.
- **2026-06-19:** Issues #309/#310 QA review: #309 internal agent credential path approved (health/ready remain public; `/api/*`, `/logs`, `/log-level` require `X-Internal-Service-Token`; Go proxy attaches it; Compose exposes not publishes 8081). #310 blocked: `verify_url` and `fetch_dealer_page` still call arbitrary/loopback URLs directly, bypassing outbound allowlist controls; targeted Python tests/ruff passed, isolated AgentProxy tests passed, Go package test still blocked by pre-existing `services.containsString` redeclaration.
- **2026-06-19:** PR #310 revised outbound validation QA APPROVED after Maximus revision. Verified shared `safe_get()` validates initial URLs before creating `httpx.AsyncClient`, disables automatic redirects, validates each redirect target before the next GET, and rejects private/metadata/loopback targets without unsafe network calls. `verify_url`, `fetch_dealer_page`, SearXNG search, and NumisBids scraping/search now use `safe_get`; targeted `pytest tests\test_outbound_validation.py -v`, `pytest tests\test_availability.py -v`, and Ruff on outbound/search/numisbids/tests passed.
- **2026-06-19:** PR #313 revised media-flow QA BLOCKED on test hygiene after frontend route behavior review. Code review verified private renderers now use `AuthenticatedImage`/`useAuthenticatedMedia` and public showcase images use `/api/showcase/:slug/uploads/*`; PWA config no longer caches private uploads. Targeted Vitest exposed a stale share-card test: `renderCoinShareCard` now fetches private upload blobs through `/api/uploads/*`, so tests must stub `fetch`/media helper before rendering instead of relying on JSDOM/Node relative URL parsing.

- **2026-06-19:** #313 share-card authenticated media stale-test BLOCK CLEARED. Reviewed Maximus's test-only revision to `coinShareCard.test.ts`: render path now expects private `/uploads/*` share images to fetch via `/api/uploads/*`, create blob object URLs, draw loaded images, and revoke object URLs after render. Targeted Vitest commands passed: `npm.cmd run test -- coinShareCard.test.ts` and `npm.cmd run test -- coinShareCard.test.ts media.test.ts useCoinShareCard.test.ts`. Verdict: APPROVE; #313 can be closed.
