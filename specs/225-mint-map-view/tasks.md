# Tasks: Mint Map View

**Input**: Design documents from `/specs/225-mint-map-view/`  
**Prerequisites**: `spec.md`, `plan.md`

**Scope guardrail**: Frontend-only v1. Do not add Go API routes, database schema, migrations, geocoding, map tiles, Leaflet/Mapbox, AI inference, or editable mint coordinates.

**Tests**: Add Vitest unit/component coverage for mint normalization, alias matching, unmatched/unknown buckets, pin rendering/selection, drawer filtering, and navigation entry points. Final validation runs from `src/web`: `npm.cmd test -- mintMap MintMap --run`, `npm.cmd run type-check`, and `npm.cmd run build`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after its dependency phase is complete.
- **[Story]**: User story label (`[US1]`, `[US2]`, `[US3]`).
- All task descriptions include concrete file paths.

---

## Phase 1: Setup and Scope Lock

**Purpose**: Establish the frontend-only surface and prevent accidental backend/dependency expansion.

- [x] T001 Confirm no Go/API/schema/package changes are needed by reviewing existing collection fields and route/filter behavior in `src/web/src/types/index.ts`, `src/web/src/stores/coins.ts`, `src/web/src/composables/useCollectionFilters.ts`, and `src/web/src/router/index.ts`
- [x] T002 [P] Create map feature folder placeholders through first real files only under `src/web/src/data/`, `src/web/src/utils/`, `src/web/src/components/map/`, and `src/web/src/pages/`
- [x] T003 [P] Add or extend frontend fixture mint cases in `src/web/src/test/fixtures/coins.ts` for matched, alias, unmatched, and empty/unknown mint values

**Checkpoint**: Implementers have verified the v1 boundary: static frontend data + inline SVG only.

---

## Phase 2: Reference Data and Matching Foundation

**Purpose**: Make mint identity, aliases, unmatched names, and projection deterministic before UI work.

**⚠️ CRITICAL**: Complete this phase before map components or page integration.

- [x] T004 [US1] Define `MintReference`, `MintGroup`, `UnmatchedMintGroup`, and aggregate result types in `src/web/src/utils/mintMap.ts`
- [x] T005 [US1] Create canonical static mint dataset with aliases in `src/web/src/data/ancientMints.ts`, including Rome, Athens, Constantinople, Alexandria, Antioch, Syracuse, Trier, Lugdunum/Lyon, Siscia, Nicomedia, Cyzicus, Carthage, Thessalonica, Heraclea, Aquileia, Arles/Arelate, and Ephesus
- [x] T006 [US1] Implement `normalizeMintName(value: string): string` in `src/web/src/utils/mintMap.ts` with case-insensitive, punctuation-insensitive, whitespace-normalized matching
- [x] T007 [US1] Implement `findMintReference(value: string): MintReference | null` in `src/web/src/utils/mintMap.ts` using canonical names and aliases from `src/web/src/data/ancientMints.ts`
- [x] T008 [US1] Implement `groupCoinsByMint(coins: Coin[]): { matched: MintGroup[]; unmatched: UnmatchedMintGroup[]; unknown: Coin[] }` in `src/web/src/utils/mintMap.ts`
- [x] T009 [US1] Implement `projectLatLngToViewBox(lat: number, lng: number)` for the chosen Mediterranean/ancient-world SVG extent in `src/web/src/utils/mintMap.ts`
- [x] T010 [P] [US1] Add `src/web/src/utils/__tests__/mintMap.test.ts` coverage for normalization, aliases, duplicate canonical grouping, count sorting, empty unknown mints, non-empty unmatched mints, and projection bounds

**Checkpoint**: Mint grouping is deterministic and tested without rendering any UI.

---

## Phase 3: Inline SVG Map Components

**Purpose**: Render an accessible, token-styled ancient-world map without external map dependencies.

- [x] T011 [US1] Create `src/web/src/components/map/MintPin.vue` as an accessible button for one mint group with count badge, active state, keyboard activation, and no hardcoded token-covered colors
- [x] T012 [US1] Create `src/web/src/components/map/MintMapSvg.vue` with a dark inline SVG Mediterranean/ancient-world backdrop, projected pins from `MintGroup[]`, and `select-mint` event emission
- [x] T013 [US1] Add simple pan/zoom behavior in `src/web/src/components/map/MintMapSvg.vue`: wheel/trackpad zoom, pointer drag pan, and visible zoom controls for touch/mobile fallback
- [x] T014 [P] [US1] Add component styles for map container, controls, pins, badges, and focus states using existing design tokens in `src/web/src/components/map/MintMapSvg.vue` and `src/web/src/components/map/MintPin.vue`
- [x] T015 [P] [US1] Add `src/web/src/components/map/__tests__/MintMapSvg.test.ts` coverage for pin count rendering, selected state, keyboard/click selection, and absence of network/tile dependencies

**Checkpoint**: The map renders pins and supports selection independently of the page.

---

## Phase 4: Drawer and Unattributed Surfaces

**Purpose**: Make pin taps and unmatched/unknown mints useful without relying on new API behavior.

- [x] T016 [US2] Create `src/web/src/components/map/MintCoinDrawer.vue` listing only the selected mint group's coins with links to `/coin/:id`, accessible close behavior, and count summary
- [x] T017 [US2] Create `src/web/src/components/map/UnattributedMintBucket.vue` showing unknown coins separately from unmatched non-empty mint groups, including original mint names and coin links
- [x] T018 [P] [US2] Add `src/web/src/components/map/__tests__/MintCoinDrawer.test.ts` coverage proving selected mint drawer lists only that mint's coins
- [x] T019 [P] [US2] Add `src/web/src/components/map/__tests__/UnattributedMintBucket.test.ts` coverage proving unknown and unmatched groups are surfaced instead of dropped

**Checkpoint**: Every coin is either matched to a pin or visible in the unattributed affordance.

---

## Phase 5: Mint Map Page and Data Source

**Purpose**: Assemble the authenticated map view using the existing Pinia collection store.

- [x] T020 [US1] Create `src/web/src/pages/MintMapPage.vue` that reads `useCoinsStore().coins`, fetches the default active collection with existing `store.fetchCoins({ wishlist: 'false', sold: 'false' })` when empty, and does not add API contracts
- [x] T021 [US1] Wire `MintMapPage.vue` to `groupCoinsByMint()` and render summary counts for matched mints, unmatched mint names, and unknown mint coins
- [x] T022 [US2] Handle pin selection in `MintMapPage.vue` by opening `MintCoinDrawer.vue` for the selected `MintGroup`
- [x] T023 [US2] Handle unattributed selection in `MintMapPage.vue` by opening or expanding `UnattributedMintBucket.vue`
- [x] T024 [US2] Add empty/loading/error states in `MintMapPage.vue` for no coins, store loading, and fetch failure using existing tokenized UI patterns
- [x] T025 [P] [US2] Add `src/web/src/pages/__tests__/MintMapPage.test.ts` coverage for initial fetch when store is empty, summary counts, pin drawer filtering, and unmatched bucket visibility

**Checkpoint**: `/mint-map` can work from the currently loaded/default active collection with no backend changes.

---

## Phase 6: Navigation and Optional Gallery Bridge

**Purpose**: Make the map reachable and preserve the plan's cautious filtering bridge.

- [x] T026 [US1] Add authenticated `/mint-map` route in `src/web/src/router/index.ts` with lazy import of `src/web/src/pages/MintMapPage.vue`
- [x] T027 [US1] Add a Map launch action to `src/web/src/components/collection/DesktopCollectionHeader.vue`
- [x] T028 [US1] Add a Map launch action to `src/web/src/components/collection/PwaCollectionHeader.vue` that remains one-handed/PWA-safe
- [x] T029 [US1] Add a Mint Map card or link near the existing heat map area in `src/web/src/pages/StatsPage.vue`
- [x] T030 [US2] Evaluate whether existing route/query behavior can reliably prefill collection search from a mint; if yes, add a "View in Gallery" action using existing collection search contracts only; if no, keep drawer-only behavior and leave query-filtering as a follow-up in `specs/225-mint-map-view/plan.md`
- [x] T031 [P] [US1] Add or extend header/navigation tests in `src/web/src/components/__tests__/PwaCollectionHeader.test.ts`, a new desktop header test if needed, and `src/web/src/pages/__tests__/MintMapPage.test.ts`

**Checkpoint**: Users can reach the map from collection and stats navigation; gallery filtering is added only if the existing contract supports it safely.

---

## Phase 7: Polish, Accessibility, and Validation

**Purpose**: Prove the feature meets acceptance criteria and constitution gates without scope creep.

- [x] T032 [P] Run a design-token audit for all new map/page styles in `src/web/src/components/map/*.vue` and `src/web/src/pages/MintMapPage.vue`; replace any token-covered raw color/radius/spacing values
- [x] T033 [P] Verify keyboard and screen-reader labels for map pins, drawer close controls, zoom buttons, and unattributed affordance in `src/web/src/components/map/*.vue` and `src/web/src/pages/MintMapPage.vue`
- [ ] T034 [P] Verify mobile/PWA behavior manually at a phone viewport: one-handed navigation, tap targets, pan/zoom controls, drawer usability, and no horizontal page overflow
- [x] T035 Run targeted Vitest coverage from `src/web`: `npm.cmd test -- mintMap MintMap --run`
- [x] T036 Run strict frontend type checking from `src/web`: `npm.cmd run type-check`
- [x] T037 Run frontend production build from `src/web`: `npm.cmd run build`
- [x] T038 Confirm the final diff contains no `src/api/`, database migration, package dependency, or map tile provider changes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup and Scope Lock)**: No dependencies.
- **Phase 2 (Reference Data and Matching Foundation)**: Depends on Phase 1 and blocks all UI phases.
- **Phase 3 (Inline SVG Map Components)**: Depends on Phase 2.
- **Phase 4 (Drawer and Unattributed Surfaces)**: Depends on Phase 2; can proceed in parallel with Phase 3 after shared types are stable.
- **Phase 5 (Mint Map Page and Data Source)**: Depends on Phases 3 and 4.
- **Phase 6 (Navigation and Optional Gallery Bridge)**: Depends on Phase 5 for page route target; header/stat links can be prepared after route naming is fixed.
- **Phase 7 (Polish, Accessibility, and Validation)**: Depends on selected implementation tasks being complete.

### Execution Graph

```text
Setup -> Reference Data/Matching -> SVG Map Components ------\
                              \-> Drawer/Unattributed UI -----> Page/Data Source -> Navigation -> Validation
```

### Parallel Execution Examples

```bash
Task: "T010 [P] Add mintMap utility tests in src/web/src/utils/__tests__/mintMap.test.ts"
Task: "T014 [P] Add map component tokenized styles in src/web/src/components/map/"
Task: "T018 [P] Add MintCoinDrawer tests in src/web/src/components/map/__tests__/"
Task: "T019 [P] Add UnattributedMintBucket tests in src/web/src/components/map/__tests__/"
```

## Implementation Strategy

### MVP First

1. Complete Phases 1-2.
2. Deliver Phases 3-5 so pins, counts, selected-mint drawer, and unattributed bucket work.
3. Add route and one navigation entry point.
4. Run targeted tests, type-check, and build before adding optional gallery bridge.

### Incremental Delivery

1. **Increment A**: Static dataset + normalization/aggregation tests.
2. **Increment B**: Inline SVG map + accessible pin selection.
3. **Increment C**: Page assembly + drawer/unattributed surfaces.
4. **Increment D**: Navigation, optional safe gallery bridge, and final validation.
