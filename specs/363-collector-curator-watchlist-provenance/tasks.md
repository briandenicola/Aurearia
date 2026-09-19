# Tasks: Coin Copilot Collector Curator

**Input**: Design documents in `specs/363-collector-curator-watchlist-provenance/`
**Branch constraint**: Work directly in the existing `beta` worktree; do not create or switch branches.
**Tests**: Mandatory and test-first. Add each phase's focused tests, run them to prove the intended assertions fail, and only then implement that phase.
**Scope constraint**: Add one `collector_profiles` table, bounded read-only curator context, and the existing UI-owned dealer-result wishlist flow only. Do not add an action lifecycle, another provider/orchestrator/browser/persistence system, or any auction implementation change.

## Format: `[ID] [P?] [Story] Description`

- **[P]** marks tasks that touch different files and can proceed concurrently after their shared prerequisites.
- **[US1]** is dealer-result Add to Wishlist, **[US2]** is private collector context, and **[US3]** is read-only curator guidance, matching `spec.md`.
- Every implementation phase starts with tests. Generated OpenAPI files are changed only by `task openapi`.

---

## Phase 1: Lightweight Private Owner-Scoped Collector Profile (US2)

**Goal**: Give each authenticated owner zero or one bounded private collector profile in one table and a compact Settings section.

**Independent Test**: Owner A can load neutral defaults, save/reload/edit/clear a profile, and receive atomic field errors; owner B, a public user, an invited friend, and an administrator outside A's session cannot read or overwrite A's values, including by supplying a foreign owner identifier.

### Tests for User Story 2 — write and run first

- [X] T001 [P] [US2] Add failing model and migration tests for the single `collector_profiles` table, unique owner row, JSON-array round trips, neutral absence without a row, and owner deletion cleanup in `src/api/models/collector_profile_test.go` and `src/api/database/database_test.go`
- [X] T002 [P] [US2] Add failing repository/service tamper tests for owner A/B isolation, same-owner replacement, foreign-owner lookup attempts, finite budget and ordering bounds, normalized duplicate arrays, atomic invalid-update rollback, and unchanged global settings in `src/api/repository/collector_profile_repository_test.go` and `src/api/services/collector_profile_service_test.go`
- [X] T003 [P] [US2] Add failing GET/PUT contract tests for authentication, neutral defaults, unknown-field rejection including forged `userId`/`ownerId`, body limits, sanitized field errors, and administrator/public/follower non-disclosure in `src/api/handlers/collector_profile_test.go`
- [X] T004 [P] [US2] Add failing compact-form tests for load/save/reload/edit/clear, server validation display, no `localStorage`/admin-setting persistence, keyboard use, 44px controls, dark theme, and 320px/PWA layout in `src/web/src/components/settings/__tests__/CollectorProfileSection.test.ts`

### Implementation for User Story 2

- [X] T005 [US2] Define the one-row `CollectorProfile` model with nullable budgets/currency, bounded JSON arrays, timestamps, and unique `UserID` ownership in `src/api/models/collector_profile.go`
- [X] T006 [US2] Register only `CollectorProfile` in the existing additive migration and account-deletion model set in `src/api/database/database.go`
- [X] T007 [US2] Implement owner-scoped get and transactional full replacement without accepting an owner from caller data in `src/api/repository/collector_profile_repository.go`
- [X] T008 [US2] Implement neutral defaults, whitespace normalization, exact list/string/number bounds, case-insensitive duplicate detection, atomic validation, and a bounded internal context projection in `src/api/services/collector_profile_service.go`
- [X] T009 [US2] Implement authenticated GET/PUT handlers with strict DTO decoding, server-derived owner identity, sanitized errors, and Swagger annotations, then wire them through `src/api/handlers/collector_profile.go`, `src/api/routes_protected.go`, and `src/api/main.go`
- [X] T010 [P] [US2] Add strict browser profile types and typed GET/PUT calls through the existing Go API client in `src/web/src/types/collectorProfile.ts` and `src/web/src/api/endpoints/collectorProfile.ts`
- [X] T011 [US2] Build the compact private profile form and mount it without extending global admin settings in `src/web/src/components/settings/CollectorProfileSection.vue` and `src/web/src/pages/SettingsPage.vue`

**Checkpoint**: US2 is independently usable; profile data is private, optional, bounded, and stored in exactly one new table.

---

## Phase 2: Read-Only Curator Context in Existing Coin Copilot (US3)

**Goal**: Supply one bounded profile snapshot to the existing Coin Copilot run and compose only `collection_summary`, `portfolio_review`, and `gap_analysis`.

**Independent Test**: A controlled curator request with populated, empty, sparse, and contradictory inputs identifies facts, suggestions, profile influences, and limitations, invokes exactly the three allowed existing capabilities, and changes no collection, wishlist, draft, profile, or application-setting state.

### Tests for User Story 3 — write and run first

- [ ] T012 [P] [US3] Add failing Go contract/worker tests for an optional bounded `collector_context`, one owner-scoped snapshot per run, neutral absence, omission of IDs/credentials/action URLs, typed dealer-field projection compatibility, and no profile text in logs in `src/api/services/coin_copilot_contract_test.go` and `src/api/services/coin_copilot_worker_test.go`
- [ ] T013 [P] [US3] Add failing Python contract and harness tests for `extra="forbid"`, valid/empty bounded context, inert prompt-like profile text, exactly `collection_summary` + `portfolio_review` + `gap_analysis`, no invented preferences, and no create/save/wishlist callback or tool in `src/agent/tests/test_coin_copilot_contract.py`, `src/agent/tests/test_coin_copilot_harness.py`, and `src/agent/tests/test_coin_copilot_security.py`
- [ ] T014 [P] [US3] Add failing read-only row-set tests covering start, replay, resume, cancel, and fallback so only existing Coin Copilot run/checkpoint/event durability may change while coin, wishlist, draft, profile, and setting rows remain unchanged in `src/api/integration/coin_copilot_seam_test.go`

### Implementation for User Story 3

- [ ] T015 [US3] Extend the existing Go execution contract with the optional bounded collector context and preserve backward-compatible specialist evidence projection in `src/api/services/coin_copilot_contract.go`
- [ ] T016 [US3] Load the authenticated owner's profile once, capture one immutable context value for the run, and pass it through the existing worker without adding persistence or write authority in `src/api/services/coin_copilot_worker.go`
- [ ] T017 [P] [US3] Add the optional strict `collector_context` request model with the profile's existing bounds and no owner/action fields in `src/agent/app/models/requests.py`
- [ ] T018 [US3] Update the existing planner/supervisor instructions to use only the three shipped analysis capabilities and to separate observed facts, suggestions, explicit profile influences, conflicts, and limitations in `src/agent/app/teams/coin_copilot.py`

**Checkpoint**: US3 is independently testable through the existing Coin Copilot transport and remains read-only.

---

## Phase 3: Restore Dealer-Only Add to Wishlist (US1) 🎯 MVP Workflow

**Goal**: Restore the existing UI-owned wishlist action only for typed, verified, currently available `dealer_listing` evidence returned by `market_search`.

**Independent Test**: An explicit click on an eligible card creates exactly one owner-scoped wishlist coin through canonical `POST /api/coins`; all ineligible or tampered cards, passive rendering/replay, cancelled confirmation, repeated clicks, and duplicate URLs create nothing.

### Tests for User Story 1 — write and run first

- [ ] T019 [P] [US1] Add failing Go contract tests proving the public specialist projection exposes only the bounded typed dealer fields required by the browser, never derives them from `facts`, and remains backward compatible in `src/api/services/coin_copilot_contract_test.go`
- [ ] T020 [P] [US1] Add failing canonical-create tests for same-owner reference-URL duplicate conflict inside the transaction, concurrent retries producing at most one wishlist coin, cross-owner independence, wishlist-without-URL behavior, and unchanged ordinary collection creation in `src/api/services/coin_service_test.go` and `src/api/repository/coin_repository_test.go`
- [ ] T021 [P] [US1] Add a failing eligibility/tamper matrix for exact capability, kind, verification, availability, non-empty title/URL, typed-field-only decisions, auction-shaped evidence, misleading `facts`, explicit-click-only emission, keyboard activation, and hidden/disabled button behavior in `src/web/src/components/chat/__tests__/CopilotRunProgress.collector.test.ts`
- [ ] T022 [P] [US1] Add failing parent/composable tests for rejecting a forged ineligible child event, adapting only typed dealer fields, category/era cancellation, one `POST /api/coins` call, repeated-click guards, duplicate feedback, and non-fatal image failure after one create in `src/web/src/components/__tests__/CoinSearchChat.copilot.test.ts` and `src/web/src/composables/__tests__/useCoinSearchChat.test.ts`
- [ ] T023 [P] [US1] Add failing payload assertions that supported dealer values map only to existing wishlist fields while collection acquisition/invoice/storage/sold fields and quick-capture/deep-analysis draft fields remain empty or absent in `src/web/src/composables/__tests__/useCoinSearchChat.test.ts` and `src/api/handlers/coin_handler_test.go`

### Implementation for User Story 1

- [ ] T024 [US1] Project description, dealer name, listed price, currency, availability, ruler, denomination, era, and material from validated dealer evidence without parsing display facts in `src/api/services/coin_copilot_contract.go`
- [ ] T025 [P] [US1] Extend the existing discriminated specialist evidence type with the optional typed dealer fields and no action lifecycle types in `src/web/src/types/agent.ts`
- [ ] T026 [US1] Render an accessible **Add to Wishlist** button only when the exact typed predicate passes and emit only from its explicit click in `src/web/src/components/chat/CopilotRunProgress.vue`
- [ ] T027 [US1] Recheck the complete typed eligibility predicate against forged child events, adapt eligible evidence to the existing `CoinSuggestion`, and pass current `addingIdx`/`addedSet` state and callback props in `src/web/src/components/CoinSearchChat.vue`
- [ ] T028 [US1] Reuse `resolveCategoryAndEra`, `buildWishlistCoinPayload`, `createCoin`, and the existing best-effort scrape/proxy/upload sequence exactly once, preserving cancellation, supported-field allowlisting, duplicate status, and image-warning behavior in `src/web/src/composables/useCoinSearchChat.ts`
- [ ] T029 [US1] Apply the existing owner-scoped `FindWishlistByReferenceURL` check inside the canonical wishlist create transaction without changing collection or URL-less wishlist behavior in `src/api/repository/coin_repository.go` and `src/api/services/coin_service.go`

**Checkpoint**: US1 restores one explicit owner action; the AI, rendering, replay, tool output, and ineligible/tampered evidence cannot initiate a write.

---

## Phase 4: Focused Regression, Full Quality Gates, Evidence, and Later Release Audit

**Purpose**: Prove the three increments together, preserve existing boundaries, and hand off the separately tracked combined F014/F015 audit before v4.2.

- [ ] T030 Add profile-route/OpenAPI drift assertions for only GET/PUT `/api/collector-profile`, absence of wishlist-action/Python callback routes, optional dealer projection fields, and unchanged existing contracts in `src/api/openapi_feature363_test.go`
- [ ] T031 Regenerate and verify Swagger/OpenAPI for the profile handlers and additive dealer fields using `task openapi`, updating only `src/api/docs/docs.go`, `src/api/docs/swagger.json`, `src/api/docs/swagger.yaml`, and `docs/openapi.json`
- [ ] T032 Run the focused Go, Python, and Vue suites from `specs/363-collector-curator-watchlist-provenance/quickstart.md`; record exact commands/results plus controlled profile-isolation, curator-read-only, eligibility-tamper, field-separation, duplicate, cancellation, and image-failure evidence in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T033 Run the full Go build/vet/test/architecture, Python ruff/pytest, Vue lint/type-check/test/build, and OpenAPI cleanliness gates from `specs/363-collector-curator-watchlist-provenance/quickstart.md`; inventory the final diff to prove no excluded subsystem or new platform/action lifecycle was added, and record results in `specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md`
- [ ] T034 After T033 and before the v4.2 release, execute the separately tracked combined F014/F015 engineering audit, resolve every release-blocking finding, and record scope, findings, dispositions, and release decision in `specs/363-collector-curator-watchlist-provenance/qc-audit.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (US2 profile)** starts immediately and establishes the only new persistence plus the owner-scoped context source.
- **Phase 2 (US3 curator)** depends on T008 and T009 for bounded owner-context capture; it does not depend on wishlist work.
- **Phase 3 (US1 wishlist)** may begin after the Phase 1 tests establish ownership conventions, and can run in parallel with Phase 2; it does not require profile values.
- **Phase 4 (quality/evidence)** depends on all selected implementation phases. T031 depends on T009, T024, and T030; T032 depends on T031; T033 depends on T032; T034 is a later release gate and depends on T033.

### User Story Dependencies

- **US2 (P2 in `spec.md`)**: Independent profile increment and prerequisite only for profile-influenced US3 behavior.
- **US3 (P3 in `spec.md`)**: Depends on the US2 context projection but remains functional with neutral empty context.
- **US1 (P1 in `spec.md`)**: Functionally independent of US2/US3 and remains the MVP user workflow; it reuses existing coin creation rather than profile or curator code.

### Within Each Phase

1. Write the listed tests and tamper cases first.
2. Run the narrow test selection and confirm failure for the intended missing behavior.
3. Implement model/contract changes before repositories/services, then handlers/UI integration.
4. Rerun the narrow tests before advancing to the checkpoint.

---

## Parallel Execution Examples

### Phase 1 / US2

After agreeing the profile contract, T001–T004 can be authored in parallel because they cover separate Go persistence/service/handler and Vue surfaces. After T009 defines the API shape, T010 can proceed alongside backend implementation before T011 integrates the form.

### Phase 2 / US3

T012, T013, and T014 can be written in parallel. Once their failure modes are established, T017 can proceed in parallel with T015 while T016 and T018 wait for their respective request contracts.

### Phase 3 / US1

T019–T023 are parallel test-first work across Go contracts, Go persistence, and Vue behavior. After T024, T025 can proceed independently; T026 then provides the card event, T027 provides the defensive parent guard/adapter, and T028 reuses the canonical UI mutation flow while T029 independently completes server-side duplicate protection.

---

## Implementation Strategy

### Smallest Safe Delivery

1. Complete Phase 1 to establish private owner context.
2. Complete Phase 2 to add read-only curator value without writes.
3. Complete Phase 3 to restore the explicit dealer-card wishlist workflow.
4. Complete focused and full gates before treating Feature 363 as complete.
5. Perform the combined F014/F015 engineering audit later as a distinct v4.2 release gate.

### MVP Validation

US1 is the product-priority MVP workflow even though its implementation phase follows the profile/context foundation requested for this worktree. It is independently demonstrable with a typed fixture and canonical `POST /api/coins`.

---

## Guardrails

- No task may modify an auction subsystem model, repository, service, handler, route, provider/configuration, or UI file. Auction-shaped evidence appears only as an ineligibility fixture in existing Coin Copilot tests.
- Do not add watchlist evaluation/ranking, provenance-risk, suspicious-listing, duplicate-image, forensic, purchase, bid, or autonomous action behavior.
- Do not add wishlist action endpoints, stages, revisions, revoke/confirm operations, action/audit tables, or a parallel lifecycle.
- Do not add another agent, graph, provider, browser, scheduler, or persistence system.
- Preserve the distinct collection, wishlist, and quick-capture/deep-analysis draft field and lifecycle rules in T020 and T023.
