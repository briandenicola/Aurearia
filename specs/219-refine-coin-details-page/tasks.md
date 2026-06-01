# Tasks: Refine Coin Details Page for PWA and Desktop

**Input**: Design documents from `/specs/219-refine-coin-details-page/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/coin-detail-ui.contract.yaml, quickstart.md

**Tests**: No new automated tests were explicitly requested in spec.md; validation is via quickstart scenarios and build/type checks.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (`[US1]`, `[US2]`, `[US3]`)
- All task descriptions include concrete file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Introduce shared constants/types/styles needed by all story phases.

- [x] T001 Create coin detail section constants in `src/web/src/constants/coinDetailSections.ts`
- [x] T002 [P] Add section link and metadata-row view types in `src/web/src/types/index.ts`
- [x] T003 [P] Add shared table-row and settings-link utility styles in `src/web/src/assets/styles/main.css`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared route/component/composable foundation that all user stories depend on.

**⚠️ CRITICAL**: Complete this phase before starting user story implementation.

- [x] T004 Add authenticated detail subsection routes (`/coin/:id/journal`, `/coin/:id/notes`, `/coin/:id/actions`, `/coin/:id/analysis`) in `src/web/src/router/index.ts`
- [x] T005 Create reusable section links component in `src/web/src/components/coin/CoinDetailSectionLinks.vue`
- [x] T006 Create reusable metadata table component in `src/web/src/components/coin/CoinDetailMetadataTable.vue`
- [x] T007 Create shared coin detail context composable for coin loading and section page reuse in `src/web/src/composables/useCoinDetailContext.ts`
- [x] T008 Create shared section page shell component in `src/web/src/components/coin/CoinDetailSectionPageShell.vue`

**Checkpoint**: Foundation complete — user story work can begin.

---

## Phase 3: User Story 1 - See both coin sides immediately in a cleaner overview (Priority: P1) 🎯 MVP

**Goal**: Redesign overview hero so obverse and reverse are displayed by default with robust fallback states and mode-safe layout.

**Independent Test**: Open `/coin/{id}` for coins with two images, one image, and no images; verify dual-slot default rendering and PWA-safe behavior.

- [x] T009 [US1] Refactor overview media area to always render obverse/reverse slots by default in `src/web/src/pages/CoinDetailPage.vue`
- [x] T010 [US1] Add deterministic missing-side placeholder logic for hero slots in `src/web/src/pages/CoinDetailPage.vue`
- [x] T011 [P] [US1] Add hero media grid and fallback style rules for desktop/PWA in `src/web/src/assets/styles/main.css`
- [x] T012 [US1] Apply new hero hierarchy classes (title/subtitle/ownership chip) in `src/web/src/pages/CoinDetailPage.vue`
- [x] T013 [US1] Preserve non-sticky PWA behavior while keeping desktop behavior intact in `src/web/src/pages/CoinDetailPage.vue`

**Checkpoint**: Overview defaults to dual-side media and is independently reviewable.

---

## Phase 4: User Story 2 - Read coin metadata in a consistent table format (Priority: P1)

**Goal**: Replace boxed detail cards with a standardized row/table metadata surface.

**Independent Test**: Open `/coin/{id}` and verify metadata displays as consistent label/value rows with proper empty-value handling across desktop and PWA.

- [x] T014 [P] [US2] Implement row schema rendering and label/value row structure in `src/web/src/components/coin/CoinDetailMetadataTable.vue`
- [x] T015 [P] [US2] Implement metadata row mapping/formatting helper (dates, units, fallbacks) in `src/web/src/composables/useCoinDetailMetadataRows.ts`
- [x] T016 [US2] Replace `CoinInfoGrid` usage with `CoinDetailMetadataTable` in `src/web/src/pages/CoinDetailPage.vue`
- [x] T017 [US2] Remove legacy boxed-info layout hooks from overview in `src/web/src/pages/CoinDetailPage.vue`
- [x] T018 [US2] Tune responsive spacing/typography for table rows in `src/web/src/assets/styles/main.css`

**Checkpoint**: Overview metadata is fully table-based and independently reviewable.

---

## Phase 5: User Story 3 - Navigate deep detail sections from overview links (Priority: P1)

**Goal**: Move Journal, Notes, Actions, and AI Analysis into dedicated pages and expose settings-style links from overview.

**Independent Test**: Navigate from `/coin/{id}` to each section page and direct-load each URL; verify section content works and back navigation returns to overview context.

- [x] T019 [US3] Replace inline section blocks with settings-style section links in `src/web/src/pages/CoinDetailPage.vue`
- [x] T020 [US3] Implement journal section page using existing journal capabilities in `src/web/src/pages/CoinDetailJournalPage.vue`
- [x] T021 [P] [US3] Implement notes section page with existing note behavior in `src/web/src/pages/CoinDetailNotesPage.vue`
- [x] T022 [P] [US3] Implement actions section page using `CoinActionsPanel` in `src/web/src/pages/CoinDetailActionsPage.vue`
- [x] T023 [P] [US3] Implement analysis section page using `CoinAIAnalysis` in `src/web/src/pages/CoinDetailAnalysisPage.vue`
- [x] T024 [US3] Wire section pages to shared coin detail context and route params in `src/web/src/composables/useCoinDetailContext.ts`
- [x] T025 [US3] Remove relocated section state/imports from overview page in `src/web/src/pages/CoinDetailPage.vue`

**Checkpoint**: Section pages are route-driven and independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, validation, and removal of preview-only scaffolding.

- [x] T026 [P] Remove preview-only routes/pages after approval in `src/web/src/router/index.ts`, `src/web/src/pages/CoinDetailPreviewPage.vue`, and `src/web/src/pages/CoinDetailPreviewSectionPage.vue`
- [x] T027 [P] Update validation walkthrough for final route surface in `specs/219-refine-coin-details-page/quickstart.md`
- [x] T028 Run frontend production build/type-check workflow defined in `src/web/package.json`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all user stories
- **Phase 3 (US1)**: Depends on Phase 2
- **Phase 4 (US2)**: Depends on Phase 2 (recommended after US1 for visual stability)
- **Phase 5 (US3)**: Depends on Phase 2 (recommended after US1/US2 to avoid repeated overview churn)
- **Phase 6 (Polish)**: Depends on completion of selected stories

### User Story Dependencies

- **US1**: Independent after foundational work; defines core overview hero structure (MVP).
- **US2**: Independent after foundational work; touches same overview surface as US1, so sequence after US1 is recommended.
- **US3**: Independent after foundational work at route level, but integration quality improves after US1/US2 overview stabilization.

---

## Parallel Execution Examples

### User Story 1

```bash
Task: "T009 [US1] Refactor overview media area in src/web/src/pages/CoinDetailPage.vue"
Task: "T011 [P] [US1] Add hero media grid styles in src/web/src/assets/styles/main.css"
```

### User Story 2

```bash
Task: "T014 [P] [US2] Implement CoinDetailMetadataTable.vue"
Task: "T015 [P] [US2] Implement useCoinDetailMetadataRows.ts"
```

### User Story 3

```bash
Task: "T021 [P] [US3] Implement CoinDetailNotesPage.vue"
Task: "T022 [P] [US3] Implement CoinDetailActionsPage.vue"
Task: "T023 [P] [US3] Implement CoinDetailAnalysisPage.vue"
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Phases 1-2.
2. Deliver Phase 3 (US1) for dual-side default overview.
3. Validate with quickstart Scenario 1 + responsive checks.
4. Demo/sign off before table and section split.

### Incremental Delivery

1. **Increment A**: US1 dual-side default overview.
2. **Increment B**: US2 metadata table conversion.
3. **Increment C**: US3 section-page routing and moved capabilities.
4. **Increment D**: Polish cleanup and final build validation.

### Team Parallelization

After foundational work:
- Dev A: US1 overview hero/media
- Dev B: US2 metadata table component/mapping
- Dev C: US3 section pages and route wiring

Integrate after each story checkpoint to preserve independent testability.
