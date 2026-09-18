# Tasks: Collector Curator, Watchlist, and Provenance

**Input**: Design artifacts in `specs/363-collector-curator-watchlist-provenance/`
**Branch constraint**: Work only on the existing `beta` branch; do not create or switch branches.
**Tests**: Mandatory. In every phase, create the listed tests and tamper fixtures first, run the focused test to prove the new assertion fails for the intended reason, and only then implement the guard or behavior.

## Format: `[ID] [P?] [Story] Description`

- **[P]** marks work that changes different files and has no dependency on another incomplete task in the same phase.
- **[US1]`–`[US5]** map directly to the five user stories in `spec.md`.
- Generated OpenAPI files are changed only by `task openapi`, never by hand.

## Phase 1: Shared Contracts and Adversarial Fixtures

**Purpose**: Establish one strict, cross-language contract corpus before implementing any workflow.

- [ ] T001 Create canonical valid profile, curator, watchlist, risk, stage, and confirmation payloads with stable SHA-256 snapshot digests in `specs/363-collector-curator-watchlist-provenance/contracts/fixtures/valid.json`
- [ ] T002 [P] Create unknown-field/enum, count/length/amount, non-finite number, stale version, malformed digest, and truncation cases in `specs/363-collector-curator-watchlist-provenance/contracts/fixtures/boundary-tamper.json`
- [ ] T003 [P] Create owner/foreign/unknown IDs, private-coin leakage, prompt-injection/token-shaped text, unsafe URL/redirect/citation, raw-provider-key, and replay-digest tamper cases in `specs/363-collector-curator-watchlist-provenance/contracts/fixtures/security-tamper.json`
- [ ] T004 [P] Create low-confidence, banned accusatory/forensic language, missing evidence, duplicate-image basis, broken-link, and Feature 362 available/unavailable cases in `specs/363-collector-curator-watchlist-provenance/contracts/fixtures/risk-tamper.json`
- [ ] T005 [P] Create immutable-stage, expiry, cancellation, stale listing/result, altered provider/provenance/price/currency/mapping, idempotency mismatch, and concurrent-confirm cases in `specs/363-collector-curator-watchlist-provenance/contracts/fixtures/wishlist-action-tamper.json`
- [ ] T006 Add a Go fixture-loader and digest snapshot test that fails on contract drift or non-canonical JSON in `src/api/services/collector_contract_fixtures_test.go`
- [ ] T007 [P] Add a Pydantic fixture-loader and digest snapshot test that fails on unknown fields, open enums, bound drift, or non-canonical JSON in `src/agent/tests/test_collector_contract_fixtures.py`
- [ ] T008 [P] Add a TypeScript fixture-loader and digest snapshot test that fails when discriminated unions, closed enums, bounds, or snapshots drift in `src/web/src/types/__tests__/collectorContracts.test.ts`
- [ ] T009 Add public route/schema drift tests for the profile and wishlist-action excerpt, including absence of a Python host and internal callback paths, in `src/api/openapi_feature363_test.go`
- [ ] T010 Add typed shared Go DTOs, closed enums, validation bounds, canonical digest helpers, URL/citation allowlists, and strict JSON decoding needed to satisfy T006 and T009 in `src/api/services/collector_contract.go`

**Checkpoint**: The same valid and adversarial fixtures are enforced by Go, Python, Vue, and OpenAPI tests.

---

## Phase 2: Additive Schema and Persistence Foundation

**Purpose**: Add the four private owner-scoped tables in dependency order without enabling any feature.

**⚠️ CRITICAL**: Complete this phase before any user-story implementation.

- [ ] T011 Add failing model tests for closed enums, immutable stage/outcome fields, default versions, and JSON bounds in `src/api/models/collector_workflows_test.go`
- [ ] T012 Add failing migration tests for exact order `collector_profiles` → `collecting_goals` → `wishlist_action_stages` → `wishlist_action_outcomes`, all owner/FK/unique/expiry/source indexes, no destructive backfill, and all five flags default-off in `src/api/database/feature363_migration_order_test.go`
- [ ] T013 Add failing rollback/compatibility tests proving disable-first rollback leaves all four tables inert and recoverable, virtual profile defaults require no rows, and no columns/tables are dropped in `src/api/database/feature363_rollback_test.go`
- [ ] T014 [P] Add failing repository tamper tests for owner scoping, foreign-equals-unknown lookup behavior, atomic profile/goal replacement, optimistic conflicts, 50-goal cap, and transaction rollback in `src/api/repository/collector_profile_repository_test.go`
- [ ] T015 [P] Add failing repository tamper tests for immutable stages/outcomes, hashed idempotency keys, exact unique indexes, expiry/source lookups, owner isolation, and transaction rollback in `src/api/repository/wishlist_action_repository_test.go`
- [ ] T016 Define `CollectorProfile`, `CollectingGoal`, `WishlistActionStage`, and `WishlistActionOutcome` with the exact fields, ownership, defaults, closed enums, and indexes from `data-model.md` in `src/api/models/collector_workflows.go`
- [ ] T017 Register the four models in the tested additive order without data backfill or automatic flag enablement in `src/api/database/database.go`
- [ ] T018 Implement owner-scoped profile/goal reads and atomic optimistic replacement with `WithTx` variants in `src/api/repository/collector_profile_repository.go`
- [ ] T019 Implement immutable stage/outcome create, owner/source/idempotency lookup, cleanup eligibility, and shared transaction boundaries in `src/api/repository/wishlist_action_repository.go`

**Checkpoint**: Fresh and upgraded databases have correct ownership/indexes/defaults; rollback is disable-only and transaction failures leave no partial rows.

---

## Phase 3: User Story 1 — Private Collector Profile Foundation (Priority: P1) 🎯 MVP

**Goal**: Let one owner atomically configure a bounded private collector profile and goals in Settings, with neutral default-off fallback.

**Independent Test**: With owner A, owner B, and an administrator, create/read/update/clear/restore A's profile, race stale updates, disable the flag, and verify B/admin/unknown IDs are indistinguishable and global settings never change.

### Tests for User Story 1 — write and run first

- [ ] T020 [P] [US1] Add service validation tests for finite `0..100000000` budgets, min/max ordering, supported currency, normalized unique `20/20/50` preference bounds, 50 goals, title/description lengths, goal enums/versions, and atomic rejection in `src/api/services/collector_profile_service_test.go`
- [ ] T021 [P] [US1] Add service tamper tests for owner-derived scope, private snapshots, v0 neutral currency/defaults, stable snapshot digest, concurrent update linearization, and inert prompt-injection goal text in `src/api/services/collector_profile_security_test.go`
- [ ] T022 [P] [US1] Add handler tests for auth, 1 MiB/body-specific caps, unknown fields, sanitized field errors, stale `409` without profile disclosure, and owner/foreign/unknown/admin indistinguishability in `src/api/handlers/collector_profile_test.go`
- [ ] T023 [P] [US1] Add Vue tests for boundary errors, full atomic save, clear-to-default, version-conflict refresh, disabled/offline fallback, no admin-setting mutation, and no browser storage of profile values in `src/web/src/components/settings/__tests__/CollectorProfile.test.ts`
- [ ] T024 [P] [US1] Add desktop/320px/PWA accessibility tests for labels, explanations, keyboard operation, 44px targets, focus/error announcements, dark/high-contrast/reduced-motion, and local design tokens/Lucide reuse in `src/web/src/components/settings/__tests__/CollectorProfile.accessibility.test.ts`

### Implementation for User Story 1

- [ ] T025 [US1] Implement strict normalization, validation, neutral defaults, atomic save, optimistic versioning, snapshot capture/digest, and privacy-safe events in `src/api/services/collector_profile_service.go`
- [ ] T026 [US1] Implement authenticated GET/PUT profile handlers with server-derived owner, strict DTO decoding, body caps, sanitized errors, and Swagger annotations in `src/api/handlers/collector_profile.go`
- [ ] T027 [US1] Register protected `/api/collector/profile` routes and wire handler → service → repository dependencies in `src/api/routes_protected.go`
- [ ] T028 [US1] Add `CollectorProfileEnabled` as an existing-style global setting defaulting false, with disabled reads returning neutral defaults and stored rows remaining untouched, in `src/api/services/settings_service.go`
- [ ] T029 [P] [US1] Define strict profile/goal request, response, validation-error, and conflict types in `src/web/src/types/collectorProfile.ts`
- [ ] T030 [P] [US1] Add typed GET/PUT profile calls through the existing Go API client only in `src/web/src/api/endpoints/collectorProfile.ts`
- [ ] T031 [US1] Implement the bounded budget/preferences/goals form, atomic save/clear, version recovery, safe fallback, and design-system accessibility in `src/web/src/components/settings/CollectorProfileSection.vue`
- [ ] T032 [US1] Integrate the dedicated section without extending admin settings or unrelated user columns in `src/web/src/pages/SettingsPage.vue`
- [ ] T033 [US1] Add account export/deletion and public/follower DTO regression tests for all Feature 363 private profile/goal records in `src/api/services/collector_profile_privacy_test.go`

**Checkpoint**: US1 ships independently with only `CollectorProfileEnabled`; all other slices remain off.

---

## Phase 4: User Story 2 — Read-Only Curator Recommendations (Priority: P1)

**Goal**: Compose existing owner-scoped collection, portfolio, and gap facts into bounded recommendations without adding any mutation path.

**Independent Test**: Run against controlled collection/profile fixtures while editing the profile; results use exactly one snapshot and show strengths, themes, gaps, acquisition ideas, evidence, confidence, why, profile effects, and limitations while data rows remain unchanged except existing run durability.

### Tests for User Story 2 — write and run first

- [ ] T034 [P] [US2] Add Go contract/tamper tests for `collector_curator`, 20-item/10-evidence/text bounds, closed outcomes/kinds/effects/confidence, stable profile digest, unknown fields, unsafe citations, and explicit truncation in `src/api/services/collector_curator_contract_test.go`
- [ ] T035 [P] [US2] Add Go service tests proving reuse of collection summary/portfolio/gap services, one profile snapshot, budget/preference/goal ranking effects, sparse/contradictory unknowns, and no invented preferences in `src/api/services/collector_workflow_service_test.go`
- [ ] T036 [P] [US2] Add database row-set tests proving view/rerun/replay/dismiss/cancel mutate nothing outside existing Coin Copilot run/checkpoint/event durability in `src/api/integration/collector_curator_readonly_test.go`
- [ ] T037 [P] [US2] Add Python request/result and prompt-injection tests for facts-versus-recommendations, evidence/confidence/why/limitations, bounded truncation, cancellation, and strict rejection in `src/agent/tests/test_collector_workflows.py`
- [ ] T038 [P] [US2] Add architecture tamper tests proving no create/save/stage/confirm/apply/bid/buy tool, callback route, approval DTO, database import, or generic credential exists in Python in `src/agent/tests/test_coin_copilot_architecture.py`
- [ ] T039 [P] [US2] Add Vue tests for typed curator cards, observed/recommended separation, profile effects, confidence, evidence, why, limitations, empty-profile invitation, replay/cancel, and unavailable fallback in `src/web/src/components/chat/__tests__/CopilotRunProgress.collector.test.ts`

### Implementation for User Story 2

- [ ] T040 [US2] Add bounded internal collector workflow request/result Pydantic models with `extra="forbid"` and inert untrusted text handling in `src/agent/app/models/collector_workflows.py`
- [ ] T041 [US2] Implement the fixed read-only curator synthesis over Go-supplied facts without database, provider, or write access in `src/agent/app/teams/coin_copilot.py`
- [ ] T042 [US2] Implement Go profile snapshot plus existing collection/portfolio/gap composition, cancellation checkpoints, replay-safe bounded result persistence, and privacy-safe logs in `src/api/services/collector_workflow_service.go`
- [ ] T043 [US2] Register only the fixed read-only curator capability and strict Go↔Python contract in `src/api/services/coin_copilot_contract.go`
- [ ] T044 [US2] Add curator result unions and render accessible responsive recommendation cards in `src/web/src/types/agent.ts` and `src/web/src/components/chat/CopilotRunProgress.vue`

**Checkpoint**: US2 is independently releasable behind `CollectorCuratorEnabled` and introduces no write semantics.

---

## Phase 5: User Story 3 — Read-Only Watchlist Evaluation (Priority: P2)

**Goal**: Explainably rank only owned wishlist coins, active goals, and retained eligible dealer/auction results against goals, budget, preferences, duplicates, coverage, and authoritative listing evidence.

**Independent Test**: Evaluate controlled duplicates, missing prices, incomparable currencies, stale/conflicting listings, foreign IDs, direct URLs, and raw payloads; verify closed typed outcomes, no provider fan-out, and no domain mutation.

### Tests for User Story 3 — write and run first

- [ ] T045 [P] [US3] Add Go contract/tamper tests for all three input unions, positive IDs, 20-input bound, 64-char digests, deferred direct URL/saved-search surfaces, raw payload rejection, closed evaluation statuses, and unknown fields in `src/api/services/collector_watchlist_contract_test.go`
- [ ] T046 [P] [US3] Add service tests for owner-resolved wishlist/goals/results, one profile snapshot, active-goal versioning, budget/currency comparability, preferences/dealers, owned/wishlist duplicates, collection coverage, source quality, and explicit unknowns in `src/api/services/collector_watchlist_service_test.go`
- [ ] T047 [P] [US3] Add foreign/unknown coin/goal/run/result indistinguishability, private-coin isolation, altered digest, prompt injection, stale/cancelled result, and no provider-call replay tamper tests in `src/api/services/collector_watchlist_security_test.go`
- [ ] T048 [P] [US3] Add integration row-set tests proving availability/auction records remain authoritative and evaluation/replay/cancel makes no wishlist/listing/auction/profile/settings mutations in `src/api/integration/collector_watchlist_readonly_test.go`
- [ ] T049 [P] [US3] Add Python strict evaluation tests for input origins, all ten checks, evidence/confidence/why/limitations, missing/stale/contradictory facts, no currency invention, and bounded truncation in `src/agent/tests/test_collector_workflows.py`
- [ ] T050 [P] [US3] Add Vue tests for input origin, duplicate relationships, unknown/incomparable facts, stale state, authoritative links, confidence/why/limitations, and disabled/offline presentation in `src/web/src/components/chat/__tests__/CopilotRunProgress.watchlist.test.ts`

### Implementation for User Story 3

- [ ] T051 [US3] Implement owner-scoped input resolution by repository/service APIs, authoritative availability/auction state linking, duplicate/coverage checks, and zero new provider work in `src/api/services/collector_workflow_service.go`
- [ ] T052 [US3] Add strict watchlist input/output Pydantic unions and read-only bounded ranking in `src/agent/app/models/collector_workflows.py` and `src/agent/app/teams/coin_copilot.py`
- [ ] T053 [US3] Extend the fixed Go Coin Copilot workflow allowlist with `watchlist_evaluation`, cancellation/replay controls, and `CollectorWatchlistEvaluationEnabled` fallback in `src/api/services/coin_copilot_contract.go`
- [ ] T054 [US3] Add discriminated watchlist evaluation types and request construction from typed retained IDs rather than card JSON in `src/web/src/types/agent.ts` and `src/web/src/composables/useCoinCopilot.ts`
- [ ] T055 [US3] Render responsive accessible watchlist evaluation cards without mutating or replacing authoritative lifecycle state in `src/web/src/components/chat/CopilotRunProgress.vue`

**Checkpoint**: US3 ships behind `CollectorWatchlistEvaluationEnabled`; unsupported input surfaces fail as deferred and replay triggers no provider work.

---

## Phase 6: User Story 4 — Tiered Provenance and Documentation Findings (Priority: P3)

**Goal**: Surface cautious baseline findings before Feature 362 and attribution-rich findings only when its validated projection is available.

**Independent Test**: Exercise every allowed kind/tier and adversarial language fixture; every accepted finding visibly says “needs review,” preserves evidence/claims/conflicts/confidence/why/limitations, and never makes forensic, fraud, seller, or authenticity assertions.

### Tests for User Story 4 — write and run first

- [ ] T056 [P] [US4] Add Go contract tests for all six kinds, three tiers/confidences, required `needs_review=true`, safe labels, facts/claims/conflicts/recommendation separation, evidence bounds, and rejection of unknown fields/enums in `src/api/services/collector_risk_contract_test.go`
- [ ] T057 [P] [US4] Add tamper tests that reject fraud/inauthentic/accusatory claims, unsupported findings/citations, unsafe URLs, missing limitations, and attribution inference while preserving valid low-confidence `review_low` findings in `src/api/services/collector_risk_security_test.go`
- [ ] T058 [P] [US4] Add broken-link tests for approved HTTPS hosts, redirect revalidation, SSRF/private/link-local/metadata/DNS-rebinding rejection, bounded existing fetch behavior, and no new provider client in `src/api/services/collector_risk_link_test.go`
- [ ] T059 [P] [US4] Add duplicate-image tests requiring an existing comparison ID, method/basis, both digests, uncertainty/limitations, no new image fetch/comparison, and no claim about the original image in `src/api/services/collector_risk_image_test.go`
- [ ] T060 [P] [US4] Add Feature 362 handoff tests proving baseline operation without 362, exact unavailable gate, and byte-semantic preservation of validated confidence/conflicts/provider coverage/citations/limitations when enabled in `src/api/services/collector_risk_feature362_test.go`
- [ ] T061 [P] [US4] Add Python tamper tests for safe tiered vocabulary, prompt injection, low-confidence visibility, banned assertions, citation allowlists, duplicate-image evidence, and Feature 362 gating in `src/agent/tests/test_collector_workflows.py`
- [ ] T062 [P] [US4] Add Vue tests for visible “needs review,” tiers, uncertainty, facts versus claims, contrary evidence, broken-link/image limitations, and attribution-unavailable state in `src/web/src/components/chat/__tests__/CopilotRunProgress.risk.test.ts`

### Implementation for User Story 4

- [ ] T063 [US4] Implement baseline missing-provenance/documentation, approved broken-link, unverified/conflicting-claim, and defensible duplicate-image evidence composition without new browsing/image automation in `src/api/services/collector_workflow_service.go`
- [ ] T064 [US4] Implement strict risk Pydantic models and whole-result safety validation that rejects accusatory/forensic output but retains valid low-confidence findings in `src/agent/app/models/collector_workflows.py`
- [ ] T065 [US4] Add the read-only provenance review synthesis and safe vocabulary policy in `src/agent/app/teams/coin_copilot.py`
- [ ] T066 [US4] Gate only attribution-rich inputs on Feature 362 and preserve its validated persisted projection without reinterpretation in `src/api/services/collector_workflow_service.go`
- [ ] T067 [US4] Add risk result unions and accessible responsive tiered cards behind `CollectorRiskReviewEnabled` in `src/web/src/types/agent.ts` and `src/web/src/components/chat/CopilotRunProgress.vue`

**Checkpoint**: Baseline US4 ships pre-362; only attribution-rich evidence depends on Feature 362.

---

## Phase 7: User Story 5 — Explicit Confirmed Add to Wishlist (Priority: P4)

**Goal**: Start from an explicit eligible result-card control, create an immutable expiring Go stage, require a separate confirmation, and atomically create at most one wishlist coin with durable audit linkage.

**Independent Test**: Stage dealer and auction results, review exact mappings, cancel/expire/tamper/replay/race confirmations, and verify only an exact valid confirmation creates one owner-scoped wishlist coin with preserved source evidence and no manual/collection-only fields.

### Tests for User Story 5 — write and run first

- [ ] T068 [P] [US5] Add handler contract tests for auth, strict JSON/body caps, 16..128 idempotency keys, exact status codes, foreign-equals-unknown, sanitized errors, and Swagger coverage for stage/confirm in `src/api/handlers/wishlist_action_test.go`
- [ ] T069 [P] [US5] Add staging tamper tests for retained owner-bound run/execution/tool/result resolution, provider/HTTPS/citation allowlists, provenance/freshness/run state, altered digests/card JSON, stale listing/result, cancellation, 15-minute expiry, and immutable stage replay in `src/api/services/wishlist_action_stage_test.go`
- [ ] T070 [P] [US5] Add mapping tests for proven title/identity/reference/current-value/listing-status plus `IsWishlist=true`, paired listing price/currency, and preservation of provider/source/provenance/observation evidence in `src/api/services/wishlist_action_mapping_test.go`
- [ ] T071 [P] [US5] Add negative mapping tests proving no `PurchasePrice`, purchase date/location, invoice/SKU, sold/storage/visibility/images/notes, unsupported references, uncited provenance, raw provider fields, or updates to manual wishlist/collection coins in `src/api/services/wishlist_action_mapping_tamper_test.go`
- [ ] T072 [P] [US5] Add confirmation tamper tests for exact stage/version/fingerprint/literal, owner, expiry, cancellation, changed provider/URL/provenance/result/price/currency/mapping, source freshness, and zero partial writes in `src/api/services/wishlist_action_confirm_test.go`
- [ ] T073 [P] [US5] Add idempotency/concurrency tests for same-key replay, key/payload `409`, separate confirmation key, owner/source uniqueness, normalized URL/manual duplicate conflict, two-tab races, and at-most-one coin in `src/api/integration/wishlist_action_race_test.go`
- [ ] T074 [P] [US5] Add transaction tests proving coin creation, immutable outcome, assisted-create journal, and audit linkage commit together and roll back together on each injected failure in `src/api/integration/wishlist_action_transaction_test.go`
- [ ] T075 [P] [US5] Add success/failure audit tests proving closed outcome codes and identifiers/digests/counts only, with no profile/budget/facts/title/URL/evidence/prompt/provider payload/credential/stage JSON/raw error leakage in `src/api/services/wishlist_action_audit_test.go`
- [ ] T076 [P] [US5] Add internal-route/tool enumeration tests rejecting `wishlist_*`, `create_*`, `save_*`, `stage_*`, and `confirm_*` callbacks in `src/api/handlers/coin_copilot_internal_tools_test.go`
- [ ] T077 [P] [US5] Extend Python architecture tests to prove no wishlist stage/confirm request, response, route, tool, credential, persistence, or approval state exists in `src/agent/tests/test_coin_copilot_architecture.py`
- [ ] T078 [P] [US5] Add Vue card tests proving only validated retained dealer/auction cards expose an accessible explicit Add to Wishlist control and prose/non-market/stale/cancelled/unverified cards cannot call stage in `src/web/src/components/chat/__tests__/CopilotRunProgress.collector.test.ts`
- [ ] T079 [P] [US5] Add Vue review tests for immutable source/evidence/mapping/omissions/duplicates/limitations, separate confirmation, expiry/cancel/retry, tampered card inertness, focus trap/restore, ARIA/live state, keyboard/44px touch, 320px/PWA/offline/high-contrast/reduced-motion in `src/web/src/components/wishlist/__tests__/WishlistActionReview.test.ts`

### Implementation for User Story 5

- [ ] T080 [US5] Add a narrow transaction-aware wishlist creation seam that reuses canonical validation, references, journal, and value snapshot behavior without exposing the broad manual DTO in `src/api/services/coin_service.go`
- [ ] T081 [US5] Implement immutable staging, eligibility/freshness/cancellation rechecks, allowlisted mapping, duplicate warnings, hashed keys, canonical fingerprints, expiry, confirmation linearization, atomic coin/outcome/journal creation, stable replay, and privacy-safe audit events in `src/api/services/wishlist_action_service.go`
- [ ] T082 [US5] Implement thin authenticated stage/confirm handlers with server-derived owner, strict decoding, idempotency headers, sanitized errors, and Swagger annotations in `src/api/handlers/wishlist_action.go`
- [ ] T083 [US5] Register only the two public Go action routes and wire the shared repository transaction boundary in `src/api/routes_protected.go`
- [ ] T084 [P] [US5] Add closed stage/mapping/evidence/outcome TypeScript types that cannot represent manual or collection-only fields in `src/web/src/types/wishlist.ts`
- [ ] T085 [P] [US5] Add typed stage/confirm calls through the Go API client, generating independent stage and confirmation idempotency keys only after UI events, in `src/web/src/api/endpoints/wishlist.ts`
- [ ] T086 [US5] Add the explicit eligible-card control in `src/web/src/components/chat/CopilotRunProgress.vue` and implement the purpose-built immutable review/confirmation modal in `src/web/src/components/wishlist/WishlistActionReview.vue`

**Checkpoint**: US5 ships behind `CollectorWishlistActionEnabled`; staging is not a coin write, Python remains stateless/read-only, and only a separate exact confirmation can create one wishlist item.

---

## Phase 8: Operational Hardening, Documentation, and Release Evidence

**Purpose**: Prove safe rollout, compatibility, privacy, accessibility, retention, and full-repository quality without deploying.

- [ ] T087 Add architecture guards for Handler → Service → Repository → DB, no handler/service GORM or raw SQL, no browser-to-Python URL, no internal action callback, and no model-chosen approval/action URL in `src/api/architecture_test.go`
- [ ] T088 [P] Add five default-off flag, dependency preflight, unavailable/disabled neutral fallback, independent rollback, and legacy workflow regression tests in `src/api/services/collector_feature_flags_test.go`
- [ ] T089 [P] Add cancellation/replay/retention tests covering checks before capabilities, after awaits, before result persistence/staging/commit, late-frame discard, stable committed outcomes, expired-stage cleanup, and bounded checkpoint/audit retention in `src/api/services/collector_lifecycle_test.go`
- [ ] T090 [P] Add cross-surface privacy tests for public/follower DTOs, telemetry, logs, browser storage, exports/deletion, and private profile/coin/image/price/inference isolation in `src/api/integration/collector_privacy_test.go`
- [ ] T091 Add focused regression tests for collection create/update/portfolio/gap, wishlist availability and alert conversion, auctions, Deep Analysis/Feature 362, Coin Copilot specialists/replay/fallback, and legacy chat row sets in `src/api/integration/feature363_existing_workflows_test.go`
- [ ] T092 Regenerate Swagger/OpenAPI with `task openapi` from handler annotations and update only `src/api/docs/docs.go`, `src/api/docs/swagger.json`, `src/api/docs/swagger.yaml`, and `docs/openapi.json`
- [ ] T093 Create the semantic-migration and cross-service confirmation-boundary ADR, including disable-first rollback and Feature 362's narrow dependency, in `docs/decisions/0363-collector-workflows-confirmation-boundary.md`
- [ ] T094 [P] Document flag order, safe fallbacks, retention/export/deletion, privacy-safe observability, no hardcoded deployment address, and no deploy/push authorization in `docs/features/collector-curator-watchlist-provenance.md`
- [ ] T095 Execute and record the controlled desktop/mobile/PWA/browser scenarios, tamper matrix, accessibility evidence, and unchanged provider-call counts in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T096 Run the full Go build/vet/tests/race gates from `specs/363-collector-curator-watchlist-provenance/quickstart.md` and record command results in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T097 Run the full Python sync/ruff/pytest and Vue clean-install/lint/type-check/test/build/browser gates from `specs/363-collector-curator-watchlist-provenance/quickstart.md` and record results in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T098 Verify OpenAPI generation is clean, all actions remain SHA-pinned, no dependency/provider was added, and CI/security-scan/Gitleaks/Govulncheck/npm-audit/pip-audit/agent-image checks are green; record evidence in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T099 Verify repository CodeQL/default code-scanning, `docker-publish-beta.yml`, release-targeted image checks without push/deploy, and `.github/workflows/ai-browser-exploration.yml` with privacy-safe artifacts; record evidence in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T100 Run the `post-major-work-qc-audit` skill after implementation, resolve every High/Critical finding, and document lower-severity dispositions in `specs/363-collector-curator-watchlist-provenance/qc-audit.md`

**Checkpoint**: All automated and controlled-browser evidence is green; no commit, push, deployment, or unrelated file change is part of this task list.

---

## Dependencies and Execution Order

### Phase dependencies

1. **Phase 1 — Shared contracts/fixtures** has no implementation dependency.
2. **Phase 2 — Additive schema** depends on Phase 1 contract vocabulary and blocks persistence-backed stories.
3. **Phase 3 / US1 — Collector profile** depends on Phases 1–2.
4. **Phase 4 / US2 — Curator** depends on US1 snapshots and existing Feature 012 portfolio/gap services; it does **not** depend on Feature 362.
5. **Phase 5 / US3 — Watchlist evaluation** depends on US1, existing wishlist/availability/auction services, and retained Feature 361 market evidence; it does not depend on US2 or Feature 362.
6. **Phase 6 / US4 — Risk** follows the read-only contract foundation. Baseline findings do not depend on Feature 362; only T060/T066 attribution-rich handoff does.
7. **Phase 7 / US5 — Add to Wishlist** depends on Phases 1–2, retained Feature 361 market evidence, and the canonical Go coin/journal path. It does not depend on US2, US3, or Feature 362.
8. **Phase 8 — Hardening/release evidence** depends on all selected shipping phases.

### User-story dependency graph

```text
Shared contracts/fixtures
        |
Additive schema foundation
        |
       US1 Collector Profile
      /    |          | \
     v     v          v  v
   US2    US3      US4 baseline     US5
 Curator Watchlist      |
                        v
              US4 attribution-rich
                   Feature 362

US5 also requires retained Feature 361 result contracts, not Feature 362.
```

### Within each phase

1. Create fixture and test tasks first.
2. Run each focused test and confirm it fails for the missing guard/behavior, not because of fixture or compilation errors.
3. Implement models/repositories before services, services before handlers/routes, and contracts/API calls before UI integration.
4. Re-run focused tests after each guard and run the phase checkpoint before enabling its default-off flag.

## Parallel Execution Examples

### US1

After Phase 2, T020–T024 can be authored in parallel because they target separate service, handler, and Vue test files. After T025–T028 stabilize the API, T029 and T030 can run in parallel before T031–T032 integrate the UI.

### US2

T034–T039 are independent first-failing contract, service, integration, Python architecture, and Vue tests. T040 and the Go-side T042 can proceed in parallel against the shared fixtures, then converge in T043–T044.

### US3

T045–T050 can proceed in parallel. T051 and T052 can proceed in parallel after their tests fail, while T054 can define client-side unions before T055 renders them.

### US4

T056–T062 are parallel tamper/test streams. Baseline T063–T065 can ship without T060/T066; the attribution-rich sub-slice waits explicitly for Feature 362.

### US5

T068–T079 can be authored in parallel because each owns a distinct test file. After repository foundations exist, Go service work T080–T083 and client contract work T084–T085 can proceed in parallel, then converge in T086.

## Implementation Strategy

### MVP first

1. Complete shared contracts/fixtures and additive schema.
2. Complete US1 Collector Profile.
3. Run the US1 independent test and operational gates relevant to profile privacy/migration.
4. Ship only `CollectorProfileEnabled` to a controlled cohort; keep every later flag off.

### Incremental delivery

1. Add US2 read-only Curator and validate zero domain writes.
2. Add US3 read-only Watchlist Evaluation and validate authoritative-state reuse/no provider fan-out.
3. Add US4 baseline risk review; add the attribution-rich sub-slice only after Feature 362 is available.
4. Add US5 confirmed Wishlist Action last, after all test-first mutation guards pass.
5. Complete Phase 8 before merge/release consideration.

## Contradictions and Resolutions

- **No blocking product contradiction was found.**
- `plan.md` says the public mutation surface is “exactly” the two wishlist action routes, while `openapi-impact.md` also correctly defines `PUT /api/collector/profile` as a profile mutation. This task plan interprets the former as the **wishlist/coin mutation surface**; profile mutation remains the separately scoped US1 endpoint.
- `WishlistActionOutcome.outcome` is closed to `created|existing`, while the requested coverage includes success/failure audit outcomes. This plan keeps durable confirmation outcomes closed as designed and tests failures as privacy-safe closed audit/log events that create no `WishlistActionOutcome` row.
- The artifacts describe five user stories but the requested delivery shape includes shared contracts/schema and operational hardening. Those are non-story setup/foundational/final phases and therefore intentionally carry no `[US#]` labels.
