# Implementation Plan: Museum Coin Tray View

**Branch**: `224-museum-tray-view-rerun` | **Date**: 2026-06-18 | **Spec**: `specs/224-museum-tray-view/spec.md`
**Merge target**: `beta`
**Last Updated**: 2026-06-18 (tightened for accuracy)

---

## Update History

| Date | Change |
|------|--------|
| 2026-06-18 | Corrected store name to `useCoinsStore()`; updated router navigation to use `/coin/:id` route name; removed Zod/Pydantic validation mentions; simplified responsive description to avoid false claims about specific class names; aligned test locations to repo conventions (`__tests__/` subdirectories). |

## Summary

Add a read-only frontend-only Tray display mode that renders the current collection as physical coins seated in felt-lined wells. The tray is responsive, paginates large collections, scales coins by `diameterMm` within safe bounds, offers felt color themes, and routes coin interaction to detail pages. No API changes or schema additions required.

---

## Technical Context

**Language/Version**: Vue 3 with TypeScript (Composition API), Vite
**Primary Dependencies**: Vue Router, Pinia coins store, existing `Coin` and `CoinImage` types, CSS Grid/Flexbox
**Storage**: LocalStorage for felt color theme preference (UI state only, non-sensitive)
**Testing**: Vitest component and utility tests; `npm run type-check`; `npm run build`
**Target Platform**: Mobile PWA (375px+), tablet (768px+), desktop (1024px+)
**Project Type**: Frontend-only collection display feature
**Performance Goals**: Render 50 visible tray wells without jank (60fps); lazy-load coin images as needed
**Constraints**: No API changes, no schema additions, no drag-and-drop, no editing, no saved layouts
**Scale/Scope**: Current loaded filtered/sorted result set for v1; cross-page tray is a future enhancement

---

## Constitution Check

**Principle I (Clear Layered Architecture)**: Tray components use Pinia store (no direct backend calls); layout math isolated in `utils/trayLayout.ts` for testability.

**Principle III (Strict Types and Explicit Contracts)**: All props, composables, and utilities typed; nullable `diameterMm` handled without casts.

**Principle IV (Simple Complete Changes)**: Flat top-down CSS tray only (no 3D, no perspective, no saved drag layouts); direct mapping of current collection data to well grid.

**Principle VI (Consistent UX)**: Design tokens for all colors/spacing; existing global CSS classes (`btn`, `chip`); dark theme default; no emojis; PWA-compatible interaction.

**§17 Quality Gate**: Tests must cover diameter scaling, missing diameter fallback, drawer pagination, responsive grid, and action routing; `type-check`, `build` must pass.

**§21 DoD**: Component tests, utility tests, type safety, responsive design tested, integration with collection headers verified.

---

## Project Structure

```text
specs/224-museum-tray-view/
├── spec.md                              # Feature specification (user stories, requirements)
└── plan.md                              # This file

src/web/src/
├── pages/
│   ├── CollectionPage.vue               # Existing; add Tray button to headers
│   └── TrayViewPage.vue                 # NEW: Main tray view page
├── components/
│   ├── collection/
│   │   ├── DesktopCollectionHeader.vue  # Existing; add Tray button
│   │   └── PwaCollectionHeader.vue      # Existing; add Tray button
│   ├── tray/                            # NEW: Tray-specific components
│   │   ├── MuseumTray.vue               # NEW: Grid container with felt background
│   │   ├── MuseumTrayWell.vue           # NEW: Single coin well component
│   │   └── TrayControls.vue             # NEW: Drawer pagination + felt theme chips
│   └── __tests__/
│       ├── MuseumTrayWell.test.ts       # NEW: Test rendering, click/keyboard
│       ├── MuseumTray.test.ts           # NEW: Test grid, theme, responsive
├── utils/
│   ├── trayLayout.ts                    # NEW: Diameter scaling, grid layout helpers
│   └── __tests__/
│       └── trayLayout.test.ts           # NEW: Test diameter scaling, pagination
├── composables/
│   ├── useTrayPreference.ts             # NEW: LocalStorage felt color persistence
│   └── __tests__/
│       └── useTrayPreference.test.ts    # NEW: Test localStorage persistence
└── router/
    └── index.ts                         # Existing; add /tray route
```

**Structure Decision**: All tray-specific presentation under `components/tray/`. Layout math in `utils/trayLayout.ts` for independent testing and reuse. Composable for preference logic. Tests follow repo convention in `__tests__/` subdirectories at component/utils/composables level.

---

## Tray Layout Contracts

### Type: `TrayCoin`

```typescript
interface TrayCoin {
  id: number
  name: string
  diameterMm: number | null
  images: readonly CoinImage[]
}
```

### Type: `TrayLayoutOptions`

```typescript
interface TrayLayoutOptions {
  minCoinPx: number           // e.g., 40
  maxCoinPx: number           // e.g., 120
  defaultDiameterMm: number   // e.g., 20 (fallback when missing)
}
```

### Function: `normalizeDiameterMm(diameterMm: number | null | undefined, defaultDiameterMm: number): number`

- Returns `diameterMm` if > 0; otherwise returns `defaultDiameterMm`.
- Used before any scaling calculation.

### Function: `getCoinRenderSizePx(diameterMm: number, allDiameters: number[], options: TrayLayoutOptions): number`

- Calculates render size by scaling proportionally relative to all coins' diameters.
- Formula: `minSize + (normalized / maxInSet) * (maxSize - minSize)`.
- Clamps to `[minCoinPx, maxCoinPx]`.
- Handles empty/missing diameter gracefully.

### Function: `getDrawerCoins(coins: TrayCoin[], drawerIndex: number, coinsPerDrawer: number): TrayCoin[]`

- Slices coins for a given drawer.
- Returns empty array if drawerIndex is out of bounds.
- Used for pagination.

### Function: `getTotalDrawers(totalCoins: number, coinsPerDrawer: number): number`

- Calculates drawer count: `Math.ceil(totalCoins / coinsPerDrawer)`.

---

## Implementation Phases

### Phase 1: Layout Utilities & Composable (Dependency: None)

**Purpose**: Core layout logic and preference persistence.

1. Create `src/web/src/utils/trayLayout.ts`:
   - Export `normalizeDiameterMm()`, `getCoinRenderSizePx()`, `getDrawerCoins()`, `getTotalDrawers()`.
   - Handle edge cases: empty diameter, negative diameter, single coin, all same diameter.

2. Create `src/web/src/composables/useTrayPreference.ts`:
   - Hook for reading/writing felt color theme from/to localStorage.
   - Reactive ref, computed getter.

3. Create `src/web/src/utils/__tests__/trayLayout.test.ts`:
   - Test `normalizeDiameterMm()`: small obol (8mm), large sestertius (35mm), zero/negative, null.
   - Test `getCoinRenderSizePx()`: proportional scaling, clamp bounds, missing diameter.
   - Test `getDrawerCoins()`: drawer slicing, boundary conditions.
   - Test `getTotalDrawers()`: various collection sizes.

---

### Phase 2: Core Tray Components (Dependency: Phase 1 complete)

**Purpose**: Render tray UI.

1. Create `src/web/src/components/tray/MuseumTrayWell.vue`:
   - Props: `coin: TrayCoin`, `renderSizePx: number`.
   - Render circular well with:
     - Recessed shadow (darker inner circle).
     - Coin image centered in well (or placeholder if missing).
     - Contact shadow below coin (soft shadow effect).
     - Accessible label (aria-label = coin name).
   - Handle click → emit event.
   - Handle keyboard focus (Enter key) → emit event.
   - Use design tokens for colors, shadows.
   - Respect `prefers-reduced-motion`.

2. Create `src/web/src/components/tray/MuseumTray.vue`:
   - Props: `coins: TrayCoin[]`, `feltTheme: 'red' | 'green' | 'navy'` (default 'red').
   - Render felt background with CSS texture/gradient.
   - CSS Grid layout with responsive columns: adjust column count based on viewport (mobile 2–3, tablet 4–5, desktop 6–8).
   - Render `MuseumTrayWell` for each coin, passing calculated `renderSizePx`.
   - Use `trayLayout.getCoinRenderSizePx()` to calculate sizes.
   - Emit `coin-clicked` event.
   - Apply theme to root (e.g., class-binding for felt color).
   - Use design tokens for gap, padding, shadows.

3. Create `src/web/src/components/tray/TrayControls.vue`:
   - Props: `drawerIndex: number`, `totalDrawers: number`, `feltTheme: 'red' | 'green' | 'navy'`.
   - Render:
     - "Drawer 1 of 3" label.
     - Previous/Next buttons.
     - Felt color chips (red, green, navy).
   - Emit `prev`, `next`, `update:feltTheme`.
   - Disable Previous on first drawer, Next on last drawer.

4. Create `src/web/src/components/__tests__/MuseumTrayWell.test.ts`:
   - Test coin image rendering, placeholder fallback.
   - Test click event emission.
   - Test keyboard (Enter) event emission.
   - Test accessibility (aria-label).

5. Create `src/web/src/components/__tests__/MuseumTray.test.ts`:
   - Test well grid rendering.
   - Test felt theme application (style or class-binding).
   - Test responsive layout behavior (viewport-based column adjustment).
   - Test coin-clicked event propagation.

6. Create `src/web/src/composables/__tests__/useTrayPreference.test.ts`:
   - Test reading/writing localStorage.
   - Test reactive updates.
   - Test default fallback.

---

### Phase 3: Page and Route (Dependency: Phase 2 complete)

**Purpose**: Wire tray into app and make it reachable.

1. Create `src/web/src/pages/TrayViewPage.vue`:
   - Use `useCoinsStore()` to access `store.coins`.
   - Use `useTrayPreference()` for felt color.
   - Local state: `drawerIndex`.
   - Render:
     - `MuseumTray` with current drawer coins.
     - `TrayControls` for pagination and theme.
     - Empty state if `store.coins.length === 0`.
   - Handle `coin-clicked` → route to `/coin/:id` using router.
   - Handle Previous/Next → update drawer index.
   - Preserve drawer position in component state (simplicity).

2. Update `src/web/src/router/index.ts`:
   - Add route: `{ path: '/tray', component: TrayViewPage, meta: { requiresAuth: true } }`.

3. Update `src/web/src/components/collection/DesktopCollectionHeader.vue`:
   - Add "Tray" chip/button in the toolbar-right area (alongside other view options).
   - Router link or click handler to navigate to `/tray`.

4. Update `src/web/src/components/collection/PwaCollectionHeader.vue`:
   - Add "Tray" chip/button in the View section of pwa-menu (alongside existing view toggles).
   - Router link or click handler to navigate to `/tray`.

---

### Phase 4: Coin Interaction (Dependency: Phase 3 complete)

**Purpose**: Handle user actions in tray.

1. Update `TrayViewPage.vue`:
   - On `coin-clicked` event, route to `/coin/:id` using `router.push({ name: 'coin-detail', params: { id: coin.id } })`.
   - Preserve return path so Back button returns to tray.

2. Test keyboard navigation:
   - Tab focus on wells.
   - Enter key opens coin detail.
   - Escape returns to tray (browser back).

3. Test drawer preservation:
   - Open a coin from drawer 2, return via browser back, verify on drawer 2 still.

---

### Phase 5: Tests and Validation (Dependency: Phase 4 complete)

**Purpose**: Verify end-to-end behavior.

1. Run component tests: `npm run test -- tray --run`.
2. Run type-check: `npm run type-check`.
3. Run build: `npm run build`.
4. Manual PWA/mobile viewport testing:
   - 375px (mobile): 2–3 columns.
   - 768px (tablet): 4–5 columns.
   - 1024px+ (desktop): 6–8 columns.
5. Manual interaction testing:
   - Click coin well → opens detail page.
   - Keyboard Enter on focused well → opens detail page.
   - Return from detail → tray drawer position preserved.
   - Felt color chip click → theme updates and persists.
6. Optional Playwright smoke test:
   - Navigate to `/tray` → verify coins render → click a coin → open detail → back → verify tray position.

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Flat top-down tray, not 3D perspective | Simpler CSS/rendering, works on all devices, aligns with Principle IV. |
| Use current loaded result set, not fetch all coins | Avoids API overload, aligns with collection filtering; future enhancement for cross-page tray. |
| Default tap opens detail, no required long-press | Discoverable, accessible, keyboard-friendly. Inline flip can be opt-in later. |
| Pure CSS felt texture | Lighter, themeable, no image asset overhead. |
| Fixed 50 coins per drawer (or page-based) | Simple, matches collection pagination; adaptive per-viewport is a future polish. |
| Prefer state-based drawer position, not URL param | Simpler component state, avoids URL clutter; position lost on page reload (acceptable for MVP). |
| Store felt color in localStorage | Non-sensitive UI preference; persistent across sessions without backend. |

---

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| **Tray shows more coins than loaded in store** | Scope v1 to `store.coins` only; label drawers clearly; future enhancement for full collection cross-page fetch. |
| **Small coins become untappable** | Clamp render size to minimum 40px; verify tap target in tests. |
| **Large coins dominate layout** | Clamp to maximum 120px; test with realistic diameter range (8mm–50mm). |
| **Visual drift from design tokens** | Use tokens for all colors, spacing, shadows; no hardcoded values. |
| **Felt color theme not persisting** | Test localStorage save/load in composable tests. |
| **Responsive grid breaks on edge viewports** | Test at exact breakpoints (375, 768, 1024) and intermediate widths. |
| **Drawer position lost on return** | Preserve in component state (not query param) for simplicity. |
| **Animation jank on low-end devices** | Respect `prefers-reduced-motion`, use CSS Grid (efficient), lazy-load images. |

---

## Out of Scope (Future Enhancements)

- Drag-and-drop tray rearrangement.
- Saved custom tray layouts.
- Full-collection cross-page tray (would require API pagination expansion).
- 3D perspective or flip animation within wells.
- Public/shared tray links.
- Editable tray (bulk actions, selection mode).
- Pinch-zoom or swipe navigation.

---

## Success Criteria (from Spec)

- ✅ Tray reachable from collection page in one click.
- ✅ Coins render in felt-lined wells with shadows.
- ✅ Grid reflows: 2–3 mobile, 4–5 tablet, 6–8 desktop.
- ✅ Large collections paginate with Previous/Next navigation.
- ✅ Tap/keyboard opens coin detail.
- ✅ Drawer position preserved on return.
- ✅ Diameter scaling works proportionally, fallback for missing.
- ✅ Felt color theme updates and persists.
- ✅ All tests pass; type-check, build succeed.
- ✅ `prefers-reduced-motion` respected.
