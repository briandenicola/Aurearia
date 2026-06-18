# Tasks: Museum Coin Tray View (Feature 224)

**Input**: Design documents from `specs/224-museum-tray-view/`
**Prerequisites**: spec.md ✅, plan.md ✅
**Last Updated**: 2026-06-18 (updated navigation to sidebar per Brian's decision)
**Total Tasks**: 25
**Phases**: 1 (Setup) + 2 (Utilities & Composable) + 3 (Components) + 4 (Page & Route) + 5 (Coin Interaction) + 6 (Tests) + 7 (Polish)

---

## Update History

| Date | Change |
|------|--------|
| 2026-06-18 | Updated navigation decision: Tray submenu item under Collection in sidebar (App.vue), not collection header buttons. Removed DesktopCollectionHeader and PwaCollectionHeader tasks; added App.vue sidebar navigation task. Total tasks reduced from 26 to 25. |
| 2026-06-18 | Fixed test locations to follow repo convention (`src/web/src/utils/__tests__/`, `src/web/src/composables/__tests__/`, `src/web/src/components/__tests__/`); corrected store reference to `useCoinsStore()` in TrayViewPage task; removed false claims about specific class names (e.g., `tray-felt-red`, `tray-cols-mobile`); clarified responsive testing as viewport-based, not class-based. |

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare directory structure and confirm no blocking issues.

- [x] **T001** [P] Create `src/web/src/components/tray/` directory
- [x] **T002** [P] Verify `src/web/src/utils/` directory exists; create if missing
- [x] **T003** [P] Verify `src/web/src/composables/` directory exists; create if missing

**Checkpoint**: Directories ready. No blocking issues.

---

## Phase 2: Layout Utilities & Composable (No Dependencies)

**Purpose**: Core layout logic for diameter scaling, pagination, and theme persistence.

### Tests First (Write Failing, Then Implement)

- [x] **T004** [P] Write `src/web/src/utils/__tests__/trayLayout.test.ts`:
  - Test `normalizeDiameterMm()`: null → default, zero → default, negative → default, positive → return as-is.
  - Test `getCoinRenderSizePx()`: small (8mm) renders smaller than large (35mm), both clamped to [40px, 120px], missing diameter uses default.
  - Test `getDrawerCoins()`: slice coins correctly, boundary index returns empty.
  - Test `getTotalDrawers()`: 100 coins, 50 per drawer → 2 drawers; 101 coins → 3 drawers.
  - Verify all tests FAIL before implementing.

- [x] **T005** [P] Write `src/web/src/composables/__tests__/useTrayPreference.test.ts`:
  - Test read/write felt color from localStorage.
  - Test default color ('red') when localStorage is empty.
  - Test reactive updates when preference changes.
  - Verify all tests FAIL before implementing.

### Implementation

- [x] **T006** [P] Implement `src/web/src/utils/trayLayout.ts`:
  - Export `normalizeDiameterMm(diameterMm: number | null | undefined, defaultDiameterMm: number): number`.
  - Export `getCoinRenderSizePx(diameterMm: number, allDiameters: number[], options: TrayLayoutOptions): number`.
  - Export `getDrawerCoins(coins: TrayCoin[], drawerIndex: number, coinsPerDrawer: number): TrayCoin[]`.
  - Export `getTotalDrawers(totalCoins: number, coinsPerDrawer: number): number`.
  - Implement types: `TrayCoin`, `TrayLayoutOptions`.

- [x] **T007** [P] Implement `src/web/src/composables/useTrayPreference.ts`:
  - Export composable `useTrayPreference()`.
  - Return reactive `feltColor` ref (type: 'red' | 'green' | 'navy').
  - Implement getter from localStorage key 'tray:feltColor'.
  - Implement setter to localStorage on change.
  - Default to 'red' if localStorage empty.

- [x] **T008** Verify T004, T005 tests now PASS: `npm run test -- trayLayout useTrayPreference --run`

**Checkpoint**: Layout math and preference logic testable and working. Ready for components.

---

## Phase 3: Core Tray Components

**Purpose**: Render tray UI (well, grid, controls).

### Tests First (Write Failing, Then Implement)

- [x] **T009** [P] Write `src/web/src/components/__tests__/MuseumTrayWell.test.ts`:
  - Test coin image rendering (happy path with valid image).
  - Test placeholder rendering when `coin.images` is empty.
  - Test click event emission.
  - Test keyboard Enter event emission.
  - Test accessibility: aria-label = coin name.
  - Verify all tests FAIL before implementing.

- [x] **T010** [P] Write `src/web/src/components/__tests__/MuseumTray.test.ts`:
  - Test well grid renders all coins.
  - Test felt theme applied (class-binding or style check).
  - Test coin-clicked event emitted on click.
  - Test responsive layout: verify grid adjusts column count based on viewport simulation.
  - Verify all tests FAIL before implementing.

### Implementation

- [x] **T011** [P] Implement `src/web/src/components/tray/MuseumTrayWell.vue`:
  - Props: `coin: TrayCoin`, `renderSizePx: number`.
  - Render circular well container (CSS border-radius 50%).
  - Inner recessed shadow (darker inner circle using box-shadow or border).
  - Coin image centered (`object-fit: cover`); use `renderSizePx` for size.
  - Placeholder icon (Coins lucide icon) if no images.
  - Contact shadow below coin (soft drop-shadow CSS).
  - Aria-label with coin name.
  - Click handler → emit 'coin-clicked' with coin.id.
  - Keyboard handler: Enter key → emit 'coin-clicked'.
  - Use only design tokens for colors (shadows, backgrounds).
  - Respect `prefers-reduced-motion` (no hover transitions if reduced).

- [x] **T012** [P] Implement `src/web/src/components/tray/MuseumTray.vue`:
  - Props: `coins: TrayCoin[]`, `feltTheme: 'red' | 'green' | 'navy'` (default 'red').
  - Root div with felt background (CSS gradient/texture effect per theme).
  - CSS Grid with responsive columns:
    - 2–3 columns on mobile (< 576px).
    - 4–5 columns on tablet (576px–768px).
    - 6–8 columns on desktop (> 768px).
  - Apply theme to root (class-binding or style for felt color).
  - Render `MuseumTrayWell` for each coin.
  - Calculate render size for each coin using `trayLayout.getCoinRenderSizePx()`.
  - Emit 'coin-clicked' event from child wells.
  - Use design tokens for gap, padding, shadows.

- [x] **T013** [P] Implement `src/web/src/components/tray/TrayControls.vue`:
  - Props: `drawerIndex: number`, `totalDrawers: number`, `feltTheme: 'red' | 'green' | 'navy'`.
  - Render:
    - Text: "Drawer {{ drawerIndex + 1 }} of {{ totalDrawers }}".
    - Button "Previous" (disabled if drawerIndex === 0).
    - Button "Next" (disabled if drawerIndex === totalDrawers - 1).
    - Chip buttons for felt colors ('red', 'green', 'navy').
    - Active chip: use `active` class from existing global CSS.
  - Emit 'prev' event on Previous click.
  - Emit 'next' event on Next click.
  - Emit 'update:feltTheme' event on felt color chip click.
  - Use `.btn-sm` and `.chip` global classes.

- [x] **T014** Verify T009, T010 tests now PASS: `npm run test -- MuseumTrayWell MuseumTray --run`

**Checkpoint**: All tray components render correctly and interact properly. Ready for page/route.

---

## Phase 4: Page and Route

**Purpose**: Wire tray into app, make it reachable via sidebar.

- [x] **T015** Create `src/web/src/pages/TrayViewPage.vue`:
  - Setup:
    - Import `useCoinsStore()`, `useRouter()`, `useTrayPreference()`.
    - Local state: `drawerIndex = ref(0)`, `coinsPerDrawer = 50`.
  - Computed: `currentDrawerCoins` = `getDrawerCoins(store.coins, drawerIndex.value, coinsPerDrawer)`.
  - Computed: `totalDrawers` = `getTotalDrawers(store.coins.length, coinsPerDrawer)`.
  - Render:
    - If `store.coins.length === 0`: empty state (message + link to `/add` or back to `/`).
    - Else: `TrayControls`, `MuseumTray`, handle events.
  - Event handlers:
    - `@coin-clicked` → `router.push({ name: 'coin-detail', params: { id: coinId } })`.
    - `@prev` → `drawerIndex.value = Math.max(0, drawerIndex.value - 1)`.
    - `@next` → `drawerIndex.value = Math.min(totalDrawers - 1, drawerIndex.value + 1)`.
    - `@update:feltTheme` → `feltTheme.value = newTheme`.
  - Style: simple container, flex layout, use design tokens.

- [x] **T016** Update `src/web/src/router/index.ts`:
  - Add route: `{ path: '/tray', name: 'tray', component: TrayViewPage, meta: { requiresAuth: true } }`.
  - Verify no route name conflicts.

- [x] **T017** [P] Update `src/web/src/App.vue`:
  - Add "Tray" submenu item under Collection in the sidebar navigation.
  - Follow the same submenu structure as Stats (parent expandable/collapsible with sub-items).
  - Submenu items should include Gallery (main collection view) and Tray (new feature).
  - Click handler or router-link: navigate to `/tray`.
  - Ensure submenu expand/collapse works consistently with existing navigation patterns.

**Checkpoint**: Tray reachable from sidebar. Coin click routes to detail. Drawer controls work.

---

## Phase 5: Coin Interaction & Return Path

**Purpose**: Ensure seamless navigation to coin detail and back to tray with position preserved; verify sidebar navigation.

- [ ] **T018** Test coin detail return and sidebar navigation:
  - From tray drawer 2, click a coin → coin detail page opens.
  - In coin detail page (or browser back button) → return to `/tray`.
  - Verify still on drawer 2 (component state preserved, not cleared).
  - Verify keyboard navigation: Tab focus on wells, Enter opens detail.
  - Verify Collection submenu in sidebar expands/collapses correctly.
  - Verify both Gallery and Tray submenu items navigate to their respective routes.

**Checkpoint**: Coin interaction complete and intuitive. Sidebar navigation working.

---

## Phase 6: Type Safety & Build Verification

**Purpose**: Ensure no TypeScript errors, build succeeds.

- [x] **T019** [P] Run type-check: `cd src/web && npm run type-check`
  - Fix any errors related to tray components.
  - Verify `TrayCoin`, `TrayLayoutOptions`, prop types are correct.

- [x] **T020** [P] Run build: `cd src/web && npm run build`
  - Verify no build errors or warnings.
  - Check bundle size impact (should be minimal, < 10KB gzipped for tray feature).

**Checkpoint**: Code type-safe and builds successfully.

---

## Phase 7: Polish & Documentation

**Purpose**: Final integration, responsive design check, documentation.

- [ ] **T021** [P] Responsive viewport testing:
  - Mobile (375px): Tray renders 2–3 columns, wells remain tappable, pagination controls visible.
  - Tablet (768px): Tray renders 4–5 columns.
  - Desktop (1024px+): Tray renders 6–8 columns.
  - Use browser DevTools or responsive design mode.
  - Verify no horizontal scroll, no layout jank.

- [ ] **T022** [P] Felt color theme testing:
  - Click each felt color chip (red, green, navy).
  - Verify background changes immediately.
  - Reload page → verify selected theme persists.
  - Test on mobile and desktop.

- [ ] **T023** [P] Accessibility testing:
  - Verify `prefers-reduced-motion: reduce` disables animations (no hover transitions).
  - Verify keyboard navigation (Tab, Enter on wells).
  - Verify aria-labels on wells.

- [ ] **T024** [P] Verify no design token violations:
  - Inspect all colors, spacing, shadows in rendered tray.
  - Confirm all use design tokens (--accent-gold, --bg-card, --radius-sm, etc.).
  - No hardcoded hex colors, px values for spacing.

- [ ] **T025** [P] Clean up and finalize:
  - Remove any console.log or debug code.
  - Verify git status: only spec + src/web files modified.
  - Verify no unintended file changes.

**Checkpoint**: Feature complete, polished, tested, documented.

---

## Dependencies & Execution Order

### Critical Path

1. **Phase 1 (Setup)**: T001–T003 (parallel)
2. **Phase 2 (Utilities)**: T004–T005 (tests, parallel) → T006–T007 (implementation, parallel) → T008 (verify)
3. **Phase 3 (Components)**: T009–T010 (tests, parallel) → T011–T013 (implementation, parallel) → T014 (verify)
4. **Phase 4 (Page/Route)**: T015–T017 (parallel for route and sidebar)
5. **Phase 5 (Interaction)**: T018 (manual test)
6. **Phase 6 (Build)**: T019–T020 (parallel)
7. **Phase 7 (Polish)**: T021–T025 (parallel for different concerns)

### Parallelization Opportunities

- **Within Phase 1**: All 3 tasks [P] can run in parallel (directory creation).
- **Within Phase 2**: Tests T004–T005 [P] can run in parallel; implementation T006–T007 [P] can run in parallel.
- **Within Phase 3**: Tests T009–T010 [P] can run in parallel; implementation T011–T013 [P] can run in parallel.
- **Within Phase 4**: T015–T017 [P] can run in parallel (page creation, route addition, sidebar update).
- **Within Phase 6**: T019–T020 [P] can run in parallel.
- **Within Phase 7**: All T021–T025 [P] can run in parallel.

### No Task Blocks Another (Within Phase)

Each task is self-contained and modifies different files. Once Phase 2 is complete, Phase 3 and Phase 4 can proceed in parallel if staffed.

---

## Test-First Workflow

For each component/utility:

1. **Write tests first**: FAIL tests before implementation (T004, T005, T009, T010).
2. **Implement**: Make tests PASS (T006, T007, T011–T013).
3. **Verify**: `npm run test -- [name] --run`.
4. **Type-check & build**: Ensure no errors (T019, T020).

---

## Acceptance Criteria per Task

| Task | Success Criteria |
|------|------------------|
| T001–T003 | Directories exist and are empty. |
| T004–T005 | Tests written, all FAIL before implementation. |
| T006–T007 | Tests PASS; utilities/composables export correct types and functions. |
| T008 | `npm run test -- trayLayout useTrayPreference --run` shows all PASS. |
| T009–T010 | Tests written, all FAIL before implementation. |
| T011–T013 | Tests PASS; components render correctly, events emit as expected. |
| T014 | `npm run test -- MuseumTrayWell MuseumTray --run` shows all PASS. |
| T015–T017 | Tray page reachable via sidebar; route added; sidebar navigation updated; no errors in console. |
| T018 | Open tray, click coin, verify detail page opens; return via back, verify drawer position preserved; sidebar submenu works. |
| T019–T020 | `npm run type-check` and `npm run build` both exit with code 0. |
| T021–T025 | Manual testing on multiple viewports passes; all design tokens used; no console errors. |

---

## Notes

- All tasks use exact file paths; no ambiguity.
- Tests are written first, ensuring implementation is correct before deployment.
- [P] tasks are safe to parallelize (different files, no interdependencies).
- Phase 2 must complete before Phase 3; Phase 3 before Phase 4; all phases before Phase 5 sign-off.
- Avoid: vague task titles, same-file conflicts, cross-story dependencies.
- Commit after each phase checkpoint (or after 2–3 related tasks) to keep history clean.
- Navigation moved from collection page headers to sidebar (App.vue) per Brian's decision to follow Stats submenu pattern.
- DesktopCollectionHeader.vue and PwaCollectionHeader.vue are NOT modified; no header buttons needed.
