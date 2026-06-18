# Tasks: Mint Map View

**Input**: Design documents from `/specs/225-mint-map-view/`
**Prerequisites**: `spec.md`, `plan.md`

**Scope guardrail**: Frontend-only follow-up from current beta. Do not add Go API routes, database schema, migrations, geocoding, AI inference, editable mint coordinates, Mapbox, custom tile hosting, or offline tile cache. Leaflet and OpenStreetMap tile requests are now explicitly in scope.

**Tests**: Add/adjust Vitest unit/component/router coverage for mint normalization, alias matching, unmatched/unknown buckets, Leaflet marker rendering/selection, Stats landing/subviews, legacy redirects, and navigation cleanup. Final validation runs from `src/web`: `npm.cmd test -- mintMap MintMap Stats router --run`, `npm.cmd run type-check`, and `npm.cmd run build`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after its dependency phase is complete.
- **[Story]**: User story label (`[US1]`, `[US2]`, `[US3]`, `[NAV]`).
- All task descriptions include concrete file paths.

---

## Current Beta Baseline — Completed but Superseded

The following tasks were completed in the beta implementation. They remain useful for reference data, grouping, drawers, and validation, but the inline SVG map and Collection-header navigation are superseded by Brian's 2026-06-18 clarification.

- [x] T001 Confirm no Go/API/schema/package changes are needed by reviewing existing collection fields and route/filter behavior in `src/web/src/types/index.ts`, `src/web/src/stores/coins.ts`, `src/web/src/composables/useCollectionFilters.ts`, and `src/web/src/router/index.ts`
- [x] T002 [P] Create map feature folder placeholders through first real files only under `src/web/src/data/`, `src/web/src/utils/`, `src/web/src/components/map/`, and `src/web/src/pages/`
- [x] T003 [P] Add or extend frontend fixture mint cases in `src/web/src/test/fixtures/coins.ts` for matched, alias, unmatched, and empty/unknown mint values
- [x] T004 [US1] Define `MintReference`, `MintGroup`, `UnmatchedMintGroup`, and aggregate result types in `src/web/src/utils/mintMap.ts`
- [x] T005 [US1] Create canonical static mint dataset with aliases in `src/web/src/data/ancientMints.ts`
- [x] T006 [US1] Implement `normalizeMintName(value: string): string` in `src/web/src/utils/mintMap.ts`
- [x] T007 [US1] Implement `findMintReference(value: string): MintReference | null` in `src/web/src/utils/mintMap.ts`
- [x] T008 [US1] Implement `groupCoinsByMint(coins: Coin[]): { matched: MintGroup[]; unmatched: UnmatchedMintGroup[]; unknown: Coin[] }` in `src/web/src/utils/mintMap.ts`
- [x] T009 [US1] Implement `projectLatLngToViewBox(lat: number, lng: number)` for the SVG extent in `src/web/src/utils/mintMap.ts` — **superseded; remove if no longer used**
- [x] T010 [P] [US1] Add `src/web/src/utils/__tests__/mintMap.test.ts` coverage for normalization, aliases, grouping, unknown/unmatched, and projection bounds — **revise projection assertions if projection is deleted**
- [x] T011-T015 [US1] Build and test `MintPin.vue` / `MintMapSvg.vue` inline SVG map — **superseded by Leaflet follow-up**
- [x] T016-T019 [US2] Build and test `MintCoinDrawer.vue` and `UnattributedMintBucket.vue` — **retain unless Leaflet page integration requires small adjustments**
- [x] T020-T025 [US1/US2] Build and test `MintMapPage.vue` with drawer/unattributed states — **update to consume Leaflet component**
- [x] T026 [US1] Add authenticated `/mint-map` route — **replace with redirect to `/stats/mint-map`**
- [x] T027-T028 [US1] Add Collection-header Mint Map launch actions — **remove**
- [x] T029 [US1] Add Mint Map link/card in `StatsPage.vue` — **replace with Stats landing-card design**
- [x] T030-T038 [US1/US2] Optional bridge, polish, tests, type-check, build, and no-backend confirmation — **rerun after follow-up changes**
- [ ] T034 [P] Manual mobile/PWA verification remains pending and must be redone against the Leaflet map.

---

## Follow-up Phase 8: Leaflet Dependency and Map Replacement

**Purpose**: Replace the wrong stylized SVG map with the real geographic map Brian requested.

- [x] T039 [US1] Add `leaflet` to `src/web/package.json`, update the lockfile from `src/web`, and import Leaflet CSS through the frontend build path
- [x] T040 [US1] Create `src/web/src/components/map/MintMapLeaflet.vue` that initializes/destroys a Leaflet map, adds an OpenStreetMap tile layer with attribution, and accepts typed `MintGroup[]`
- [x] T041 [US1] Render markers in `src/web/src/components/map/MintMapLeaflet.vue` from `group.mint.lat` and `group.mint.lng`; do not use SVG viewbox projection for marker placement
- [x] T042 [US3] Add marker count labels/badges and selected state for mints with one or more coins
- [x] T043 [US2] Emit marker selection from `MintMapLeaflet.vue` so `MintMapPage.vue` opens the existing `MintCoinDrawer.vue`
- [x] T044 [US1] Fit map bounds to matched markers and use a Mediterranean default center/zoom when no matched markers exist
- [x] T045 [P] [US1] Replace active `MintMapSvg.vue` usage in `src/web/src/pages/MintMapPage.vue` with `MintMapLeaflet.vue`
- [x] T046 [P] [US1] Delete `src/web/src/components/map/MintMapSvg.vue` and obsolete SVG-only tests/helpers, or leave a clear non-routed historical component only if deletion would cause unnecessary churn
- [x] T047 [P] [US1] Remove `projectLatLngToViewBox()` from `src/web/src/utils/mintMap.ts` and its tests if no remaining code uses it

**Checkpoint**: `/stats/mint-map` can render a real Leaflet/OSM map with markers placed by actual latitude/longitude.

---

## Follow-up Phase 9: Stats Subviews and Route Strategy

**Purpose**: Move Mint Map into Stats-only information architecture and preserve legacy URLs through redirects.

- [x] T048 [NAV] Update `src/web/src/router/index.ts` so `/stats/mint-map` loads `src/web/src/pages/MintMapPage.vue`
- [x] T049 [NAV] Update `src/web/src/router/index.ts` so `/mint-map` redirects to `/stats/mint-map`
- [x] T050 [NAV] Update `src/web/src/router/index.ts` so `/stats/timeline` loads the Timeline subview/page and `/timeline` redirects to `/stats/timeline`
- [x] T051 [NAV] Create or extract `src/web/src/pages/CollectionDistributionPage.vue` for the current Collection Distribution stats content
- [x] T052 [NAV] Update `src/web/src/router/index.ts` so `/stats/distribution` loads `CollectionDistributionPage.vue`
- [x] T053 [NAV] Refactor `src/web/src/pages/StatsPage.vue` into a landing page with cards linking to `/stats/mint-map`, `/stats/timeline`, and `/stats/distribution`
- [x] T054 [NAV] Remove Mint Map launch actions from `src/web/src/components/collection/DesktopCollectionHeader.vue`
- [x] T055 [NAV] Remove Mint Map launch actions from `src/web/src/components/collection/PwaCollectionHeader.vue`

**Checkpoint**: Stats owns the visualization subviews; Collection headers no longer expose Mint Map.

---

## Follow-up Phase 10: Privacy, UX, and Accessibility Polish

**Purpose**: Make the external-tile and mobile map behavior explicit and safe.

- [x] T056 [P] [US1] Ensure OSM tile URL configuration in `MintMapLeaflet.vue` contains no coin IDs, user IDs, JWTs, mint names, or collection-derived query parameters
- [x] T057 [P] [US1] Preserve OpenStreetMap attribution visibility in the map UI
- [x] T058 [P] [US2] Verify marker controls have accessible labels including mint name and coin count
- [x] T059 [P] [US1] Audit `MintMapPage.vue`, `MintMapLeaflet.vue`, and Stats subview cards for design-token compliance around non-tile UI
- [ ] T060 [US1] Manually verify mobile/PWA map pan/zoom, marker tapping, drawer usability, and no horizontal page overflow at a phone viewport

---

## Follow-up Phase 11: Tests

**Purpose**: Prove the exact clarified workflows and prevent regression to the beta structure.

- [x] T061 [P] [US1] Add `src/web/src/components/map/__tests__/MintMapLeaflet.test.ts` coverage for OSM tile layer configuration with attribution, marker creation from `lat`/`lng`, marker count labels, default view, and selection emission; mock Leaflet/no live network
- [x] T062 [P] [US2] Update `src/web/src/pages/__tests__/MintMapPage.test.ts` to stub/use `MintMapLeaflet.vue` and prove selected marker opens only matching coins
- [x] T063 [P] [NAV] Add or update router tests for `/mint-map` → `/stats/mint-map` and `/timeline` → `/stats/timeline` redirects in `src/web/src/router/index.ts`
- [x] T064 [P] [NAV] Add or update `src/web/src/pages/__tests__/StatsPage.test.ts` coverage for Stats landing cards and subview links
- [x] T065 [P] [NAV] Add or update tests proving `DesktopCollectionHeader.vue` and `PwaCollectionHeader.vue` no longer render Mint Map actions
- [x] T066 [P] [NAV] Add or update tests for Collection Distribution subview routing/rendering

---

## Follow-up Phase 12: Validation

**Purpose**: Re-run the frontend quality gate after dependency, map, and route changes.

- [x] T067 Run targeted Vitest coverage from `src/web`: `npm.cmd test -- mintMap MintMap Stats router --run`
- [x] T068 Run strict frontend type checking from `src/web`: `npm.cmd run type-check`
- [x] T069 Run frontend production build from `src/web`: `npm.cmd run build`
- [x] T070 Confirm the final implementation diff contains no `src/api/`, database migration, geocoding, AI inference, Mapbox, custom tile-hosting, or private collection data in OSM tile URLs

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 8 (Leaflet Dependency and Map Replacement)**: Starts from current beta; blocks final Mint Map acceptance.
- **Phase 9 (Stats Subviews and Route Strategy)**: Can proceed in parallel with Phase 8 after route names are final.
- **Phase 10 (Privacy, UX, Accessibility)**: Depends on Leaflet and Stats UI surfaces existing.
- **Phase 11 (Tests)**: Can be written alongside Phases 8-9 with Leaflet mocked.
- **Phase 12 (Validation)**: Depends on all selected follow-up implementation tasks.

### Execution Graph

```text
Current beta
  ├─> Leaflet map replacement ──────┐
  ├─> Stats routes/subviews ────────┼─> Privacy/UX polish -> Validation
  └─> Tests for clarified workflows ─┘
```

## Implementation Strategy

1. Add Leaflet dependency and replace the active map renderer first.
2. Move navigation into Stats and add redirects before polishing links.
3. Delete or disconnect superseded SVG and Collection-header entry points.
4. Add tests that lock the clarified behavior.
5. Run targeted tests, type-check, and production build.
