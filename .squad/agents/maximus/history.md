# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go/Gin/GORM/SQLite API, Vue 3/TypeScript frontend, Python FastAPI/LangGraph agent
- **Architecture:** Layered Handler → Service → Repository → Database; governed by `.specify/memory/constitution.md`.

## Core Context

- Maximus owns architecture, governance, feature design, and cross-artifact consistency. Durable governance outcomes: Constitution v2.0.0 with §0 hierarchy, §17 Quality Gate, §18 AI operating rules, §21 DoD, §22 amendment process; `.squad/log/` + `.squad/decisions.md` are the handoff surface, not `SESSION-NOTES.md` or `.copilot-state.md`.
- Governance scaffolding established: `specs/` workflow, retroactive `specs/001-foundation`, backlog cards F001-F007, PRD as product source of truth, ADR practice in `docs/adr/`, trimmed README, security-doc split, references/gitleaks/pre-commit support.
- Architecture baselines: Go API uses strict layered architecture and DI via `main.go`; Python agent remains stateless; Vue must use design tokens/global classes. Material auth/security/service-boundary choices require ADRs per §22.
- Important prior designs/reviews: #208 health scorecard audit identified scoring/test blockers; #216 camera-first intake was functionally sound but blocked on Principle V token violations until remediated; #217/#218 collection tools pivoted from route-based intent to composable LangChain tools with Go-owned internal/external adapters; Foundry rewrite spike was NO-GO except as a future strategic reconsideration.
- Brian's governance preference: copy cross-repo discipline (hierarchy, DoD, Quality Gate, ADRs, specs) without importing stack-specific .NET/Svelte rules.

## Recent Updates

- **2026-05-31:** Feature #219 validation gates defined for dual-side media, metadata tables, dedicated section pages, PWA behavior, and design-token compliance. Brutus later approved implementation.
- **2026-05-31:** Designed #217/#218 shared collection tool layer: 6 operations (`search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`, `propose_update`, `commit_update`), Go service ownership, Python ReAct tools, external adapter deferred/then implemented.
- **2026-05-31:** Re-reviewed #216 token remediation and approved: all originally flagged hardcoded colors moved to tokens except explicitly accepted contrast-safe black/white uses.
- **2026-06-01:** Storage Location design investigation completed. Recommended per-user `StorageLocation` lookup table with nullable `Coin.StorageLocationID`, single-select semantics, settings-style management, rename updates shared lookup row, duplicates rejected case-insensitively per user, and delete-while-in-use blocked by default pending Brian's final decision.

- **2026-06-01:** SQLite nullable-FK convention: for new nullable `Coin` lookup associations added after launch, keep the scalar `*_id` and preload association but use `constraint:-` to avoid destructive SQLite table rebuilds; enforce validity in service/repository code unless an explicit safe rebuild migration exists.
- **2026-06-01:** "Assign Location" bulk action feature — Cassius backend + Aurelia frontend parallel implementation; extends bulk endpoint with assign-location case, DI wiring, and frontend modal/button integration; validates ownership and handles nil-safe NULL updates correctly.

- **2026-06-07:** Coin Lookup Feature Architecture Scope + Decision
  - **MVP Scope:** Numista-only (NGC deferred to post-MVP; no public API available). Stateless Python agent + Go proxy pattern. 1-2 photos (obverse/reverse preferred) → camera capture → AI analysis + Numista search → results display + quick actions (Add to Wishlist/Collection).
  - **Infrastructure Reuse:** 90%+ exists: AI Intake Draft (#216), Numista proxy, image analysis, agent proxy, catalog references. Recommended: extend intake draft with Go-only Numista enrichment service (low effort, 2-3 days).
  - **Open Decisions (Resolved for MVP):** NGC integration deferred; lookup history deferred (ephemeral); offline behavior fails gracefully (network required); spec-first workflow YES (create `specs/221-coin-lookup/`).
  - **Implementation Sequence:** Increment 1 (Core Lookup: Python team + Go endpoint), Increment 2 (Frontend: `/lookup` route), Increment 3 (Quick Actions), Increment 4 (Polish).
  - **Hard Blocker:** Spec #216 (AI Intake Draft) must land before Coin Lookup begins.
  - **Decision Owner:** Maximus. Review required from Brian (product), Brutus (Python feasibility), Aurelia (UX/PWA). Next: Brian confirms NGC/offline decisions, Maximus creates spec scaffold.

## 2026-06-09 — F013 promotion learning

- Promoting Agentic Excellence work works best when F013 owns the deterministic baseline first: typed coin mutation contracts, golden fixtures, and scripted critical browser workflows. Keep LLM-driven exploratory browser testing in F011 until F013 provides stable fixtures and workflow names.

- **F013 Regressions & Fixtures Batch Approved**
   - Reviewed and approved backend regression batch (T011/T012/T013): typed mutation coverage matches F013 risks, stale storage preload fix narrowly scoped, architecture boundaries maintained
   - Reviewed and approved frontend fixture batch (T015): fixtures typed and deterministic, suitable for phase 3/4 workflows
   - All builds/tests/lints passed; batch ready for coordinate validation and commit
   - Orchestration log: `.squad/orchestration-log/2026-06-09T12-51-39Z-maximus.md`

- **2026-06-09 (13:09:16):** F013 Phase 3 golden fixtures complete — approved T014 (Cassius Go fixtures) and T016/T017 (Brutus docs + frontend fixtures). Verdict: testutil package decoupled, frontend coverage checked, browser tooling deferred to Phase 4 as specified. Principle IV respected. Orchestration logs: `.squad/orchestration-log/2026-06-09T13-09-16-maximus-lead-{t014,t016-t017}.md`. F013 T006–T017 now complete; Phase 4 browser workflows pending.

- **2026-06-09:** F013 Phase 4 Browser Workflow Infrastructure (T018–T021, APPROVED)
  - **Initial Review:** Aurelia delivered Playwright suite (playwright.config.ts, fixtures, auth/coin-form workflows, test:browser script)
  - **BLOCKED on hygiene:** Identified `.gitignore` missing Playwright output paths, generated `src/web/test-results/` present, stale browser E2E TODO in docs
  - **Escalation:** Per Strict Lockout (§18.2), delegated hygiene fix to Brutus (QA)
  - **Re-review APPROVED:** Brutus remediated all issues — `.gitignore` entries added, generated outputs removed, docs cleaned, `npm run test:browser` passing, `git diff --check` clean
  - **Coordinator Validation:** Full test suite + diff check verified before hygiene review
  - **Principle Compliance:** Principle VI (Testing Infrastructure), Principle VIII (CI/CD & Build Hygiene)
  - **Next Phase 4 Tasks:** T022–T028 (edit workflows, image upload, search/filter, mobile viewport, Taskfile, docs)

- **2026-06-09 (13:32:43):** F013 Phase 4 Storage Location & Tags/Sets Workflows (T022–T023, APPROVED)
  - **Deliverables:** Aurelia delivered two E2E workflows (storage location edit, tags/sets edit) as Playwright tests with route-level mocks and fixture-backed test data
  - **Coordinator Pre-Validation:** npm type-check (99 Vitest ✅), npm test (99 ✅), npm run test:browser (6 Playwright ✅), git diff --check (✅)
  - **Review Assessment:**
    - ✅ Workflows deterministic and golden-fixture-backed (no random/flaky behavior)
    - ✅ Mock API coverage proportional to test scope (route-level mocks only; no over-engineering)
    - ✅ Isolated from live backend data (test fixtures self-contained)
    - ✅ Scope boundaries respected (storage/tags/sets only; did not attempt T024–T028)
    - ✅ Principle IV (Proportional Scope), Principle VI (Testing Infrastructure), Principle IX (Critical Workflows Deterministic)
  - **Verdict:** ✅ **APPROVED** — Completion mark justified. All Quality Gate §17 checks pass (type-check, lint, diff-check, no architecture violations)
  - **Phase 4 Progress:** T018–T021 + T022–T023 = 6 of 11 tasks complete. Remaining: T024–T028 (edit form validation, image upload, search/filter, mobile viewport, Taskfile, docs)
  - **Orchestration Log:** `.squad/orchestration-log/2026-06-09T13-32-43Z-maximus-t022-t023-approved.md`
  - **Session Log:** `.squad/log/2026-06-09T13-32-43Z-scribe-session-t022-t023.md`

- **2026-06-09 (13:45:22–13:45:33):** F013 Phase 4 Final Review Batch — T024–T026 Workflows + Docs Remediation APPROVED
  - **Initial Review (13:45:22):** Aurelia delivered T024–T026 (image upload, search/filter, mobile viewport) Playwright workflows
    - T024: Manual add/edit image upload/delete → form save → verify persisted in detail page; deterministic File fixtures
    - T025: Collection list filter by category/era/material/name → fixture-backed deterministic result filtering
    - T026: Mobile viewport (375px) edit workflow → responsive form layout, no scroll issues
    - Coordinator Pre-Validation: npm type-check ✅ (99 tests), npm test ✅ (99 tests), npm run test:browser ✅ (9 Playwright), git diff-check ✅
    - **Initial Verdict:** ✅ **APPROVED** — Workflows deterministic, fixture-backed, mocks proportional, scope strict (no T027–T028 creep), Principle VI + Principle IX satisfied

  - **Strict Lockout Block + Remediation (13:45:07):** Docs coverage misalignment discovered
    - Block Reason: `docs/testing.md` listed T024–T026 under "Remaining" instead of reflecting actual Playwright suite coverage
    - §18.2 Delegation: Maximus delegated docs fix to Brutus (QA)
    - Brutus Remediation: Moved T024–T026 → "Current Coverage", removed stale TODO, git diff-check ✅
    - Lockout Clearance: ✅ COMPLETE

  - **Final Re-Review (13:45:33):** Maximus approved revised state with docs alignment
    - ✅ T024–T026 workflows + docs now internally consistent
    - ✅ All Quality Gate §17 checks passed (type-check, lint, diff-check, tests, architecture)
    - ✅ Principle VIII (CI/CD & Build Hygiene) satisfied
    - ✅ Constitution §18.2 Strict Lockout process honored

  - **F013 Phase 4 Status:** T018–T023 + T024–T026 = 9 of 11 tasks complete. T027–T028 (Taskfile root command + legacy docs cleanup) are administrative completions, not feature risk
  - **Orchestration Logs:**
    - Coordinator: `.squad/orchestration-log/2026-06-09T13-44-58Z-coordinator.md`
    - Aurelia T024–T026: `.squad/orchestration-log/2026-06-09T13-45-22Z-aurelia.md`
    - Brutus Docs: `.squad/orchestration-log/2026-06-09T13-45-07Z-brutus.md`
    - Maximus Final: `.squad/orchestration-log/2026-06-09T13-45-33Z-maximus.md`
  - **Session Log:** `.squad/log/2026-06-09T13-45-44Z-scribe-f013-final-review.md`
  - **Decision:** F013 Critical Workflow Hardening implementation and validation COMPLETE. Ready for merge after session handoff.

## Learnings

- **2026-06-19 — Docker release gate (#312):** Docker publish workflows must be downstream of `Quality Gate` via `workflow_run`, restricted to successful push runs, and must checkout/tag `github.event.workflow_run.head_sha`; repository branch protection/rulesets remain a GitHub settings blocker, not a repo-file change.

- **2026-06-10 — Collection count contract review (64 vs 65, PWA shows 50):**
  - Canonical "collection count" = ActiveCollection scope (owned AND NOT wishlist AND NOT sold), defined in src/api/repository/scopes.go. Wishlist/Sold are separate buckets.
  - Invariant: /coins?wishlist=false&sold=false total == /coins/stats totalCoins == collection_summary tool totalCoins. All three already share the SAME SQL predicate, so predicates are not the bug.
  - "PWA shows 50" = page-based pagination, COINS_PER_PAGE=50 in src/web/src/pages/CollectionPage.vue; CollectionPagination + store.total. Working as designed; UX clarity issue only.
  - PWA collection list always sends wishlist:'false'/sold:'false' via src/web/src/composables/useCollectionFilters.ts (loadCoins), so its total matches stats. A 1-off divergence => either agent fidelity bug (AI narrates a number != tool totalCoins) or a data anomaly (one active coin user doesn't count).
  - Latent contract weakness: default /coins (no filters) total includes wishlist+sold, unlike stats.totalCoins. Don't change default silently (Wishlist/Sold pages rely on filtered totals); document that total reflects applied filter, not collection size.
  - Key files: src/api/repository/coin_repository.go (List ~L179, GetStats ~L495), src/api/services/collection_tools_service.go (CollectionSummary), src/api/handlers/internal_tools.go (CollectionSummary handler), src/web/src/composables/useCollectionFilters.ts, src/web/src/pages/CollectionPage.vue.
  - Decision recorded: .squad/decisions/inbox/maximus-collection-count-contract.md.

- **2026-06-19 — Agent safe outbound client (#310 lockout revision):**
  - Caller/model-provided Python agent URLs must use a shared outbound fetch helper that validates the initial URL and each redirect target before any follow-up request.
  - Public internet fetches may allow arbitrary public origins, but local/private targets are only allowed when the exact origin is configured as trusted and `AGENT_ALLOW_LOCAL_OUTBOUND=true`; metadata IPs stay blocked even in local-dev mode.
  - Regressions should assert the HTTP client is not constructed/called for unsafe initial URLs and that redirect-to-private/metadata stops after the safe public hop.
- **2026-06-19 — Security scan gate hardening (#323):**
  - Blanket `continue-on-error` on security scanners undermines Constitution Principle VII / §17; prefer removing it entirely unless a scanner is explicitly advisory and documented.
  - `npm audit --audit-level=high` gives the desired high/critical threshold. `pip-audit` has no portable severity threshold, so the practical fail-closed posture is to block on any Python vulnerability and require documented narrow exceptions.
  - Branch/release implication belongs in deployment docs: PR security jobs should be required branch checks on `main`/`beta`, and Docker publish workflows inherit that gate because they run only after protected-branch pushes.

- **2026-06-19 — Toolchain/base-image pinning (#320):** Keep Go on the active 1.26 line but move module/setup-go resolution and Docker API builder to the fixed patch (`go 1.26.4`, `golang:1.26.4-alpine`). CI-installed Go tools are reviewed pins (`swag v1.16.6`, `govulncheck v1.4.0`) and Docker production bases use tag-plus-OCI-index-digest references to preserve multi-arch while making builds reproducible. Running pinned `task openapi` exposed unrelated OpenAPI route drift from existing handler changes, so generated artifacts were reverted and #316 remains the coordination point.

- **2026-06-19 — Issue #314 Resolution: Frontend Modularization Guardrail (Closed, Deferred Extraction)**
  - **Finding:** Five frontend/API modules exceed safe review thresholds (AddCoinPage.vue 1,307 lines, AdminSchedulesSection.vue 1,134, CoinLookupPage.vue 1,097, App.vue 819, client.ts 780).
  - **Decision:** Close #314 with documented guardrail. Defer extraction; each module will be refactored only when touched for product/security/UX work. Extraction without a driver workflow violates Principle IV (Simple, Complete, Proportional).
  - **Rationale:** Pre-emptive extraction is low-signal refactoring. Each extraction requires new unit tests (5 modules = 5 independent test suites = high complexity). Tight coupling to active workflows makes casual changes risky.
  - **Action:** 
    1. Created `docs/frontend-modularity.md` with inventory and safe seams
    2. Updated PR template item 15a to trigger checklist reminder when these files are touched
    3. Defined safe extraction seams per module (composables, subcomponents, API groups) with conditions
  - **Safe Seams Summary:**
    - AddCoinPage.vue: Camera logic → useAddCoinCamera, form state → useAddCoinForm (trigger: camera UX work)
    - AdminSchedulesSection.vue: Scheduler table → SchedulerRunsTable.vue, run detail → SchedulerRunDetail.vue (trigger: admin dashboard redesign)
    - CoinLookupPage.vue: Image preview grid → ImagePreviewGrid.vue (trigger: lookup feature enhancement)
    - App.vue: Sidebar reorder → useSidebarReorder.ts (trigger: nav UX work)
    - client.ts: Domain groups → api/coin.ts, api/admin.ts, api/agent.ts (trigger: API versioning, multi-domain refactoring)
  - **Key Principle:** Extract only when actively working on the owned workflow. Guard against "fixing" module size pre-emptively.
  - **Next Steps:** When a future issue touches these files, apply safe seams + regression tests before proposing extraction.
