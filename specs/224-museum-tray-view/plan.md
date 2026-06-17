# Implementation Plan: Museum Coin Tray View

**Branch**: `224-museum-tray-view` | **Date**: 2026-06-17 | **Spec**: `specs/224-museum-tray-view/spec.md`  
**Merge target**: `beta`

## Summary

Add a read-only Tray display mode that renders the current collection as physical coins seated in felt-lined wells. The view should be responsive, paginate large collections into drawers, scale coins by `diameterMm` within safe tap-target bounds, offer a small set of felt color themes, and let users open or flip a coin without editing from the tray.

This is priority 4 because it is a larger visual surface that benefits from the 3D viewer and presentation-mode primitives, but it can still ship as a frontend-only feature.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Composition API  
**Primary Dependencies**: Vue Router, Pinia coins store, existing `Coin`/`CoinImage` types, CSS grid  
**Storage**: Local UI preference only for felt color and drawer size if needed (`localStorage`)  
**Testing**: Vitest component/unit tests; `npm run type-check`; `npm run build`; optional Playwright smoke for responsive display  
**Target Platform**: Mobile PWA, tablet, desktop browser  
**Project Type**: Frontend-only collection display feature  
**Performance Goals**: Render 50 visible tray wells without jank; avoid loading more images than the current drawer  
**Constraints**: No drag-to-arrange, no saved layout, no editing from tray, no API/schema changes  
**Scale/Scope**: Current collection result set/page first; full collection cross-page tray can be follow-up if API pagination makes it expensive

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Typed tray layout helpers and coin display props; nullable `diameterMm` handled without casts.
- **Principle IV (Simple Complete Changes)**: Flat top-down CSS tray only; no 3D tray scene or saved drag layouts.
- **Principle VI (Consistent UX)**: Use design tokens and existing button/chip classes; dark theme; no emojis.
- **PWA/Mobile Interaction Rules**: Wells must remain finger-friendly; pagination controls compact and wrapped, not full-width clutter.
- **§17 Quality Gate / §21 DoD**: Tests must cover diameter scaling, missing diameter fallback, drawer pagination, and action routing.

No constitution violations are expected.

## Resolved Scope Decisions

1. **Source set**: v1 launches from the collection page and uses the current loaded filtered/sorted result set (`store.coins`) rather than fetching every matching coin.
2. **Route vs inline mode**: Add a route `/tray` for an immersive display while keeping launch controls in collection headers.
3. **Tap action**: Default tap opens coin detail. If branch 222 has landed, add a secondary inline flip control per well; otherwise defer flip.
4. **Coins per drawer**: Adaptive based on viewport and configured well size, but capped to the loaded page size. Provide Previous/Next drawer controls within the loaded set.
5. **Felt texture**: Use pure CSS gradients/noise-like layered backgrounds for v1; no image asset.

## Project Structure

```text
specs/224-museum-tray-view/
├── spec.md
└── plan.md

src/web/src/
├── router/index.ts
├── pages/
│   ├── CollectionPage.vue
│   └── TrayViewPage.vue
├── components/
│   ├── collection/DesktopCollectionHeader.vue
│   ├── collection/PwaCollectionHeader.vue
│   └── tray/
│       ├── MuseumTray.vue
│       ├── MuseumTrayWell.vue
│       └── TrayControls.vue
├── utils/
│   └── trayLayout.ts
└── components/**/__tests__/
```

**Structure Decision**: Put all tray-specific presentation under `components/tray/`. Keep layout math in `utils/trayLayout.ts` so it is easy to test independently.

## Tray Layout Contract

```ts
interface TrayCoin {
  id: number
  name: string
  diameterMm: number | null
  images: readonly CoinImage[]
}

interface TrayLayoutOptions {
  minCoinPx: number
  maxCoinPx: number
  defaultDiameterMm: number
}
```

Rules:

- `diameterMm <= 0` and `null` use `defaultDiameterMm`.
- Render size is clamped between finger-safe min and visual max.
- The well remains larger than the coin to show felt recess.
- Missing images render a coin placeholder inside the well.

## Implementation Phases

### Phase 1: Layout Utilities

1. Create `utils/trayLayout.ts`.
2. Add `normalizeDiameterMm()` and `getCoinRenderSizePx()`.
3. Add `getDrawerCoins(coins, drawerIndex, drawerSize)`.
4. Add tests for small obol, large sestertius/medallion, missing diameter, zero/negative diameter, and drawer boundaries.

### Phase 2: Tray Components

1. Create `MuseumTrayWell.vue` with a recessed circular well, coin image, contact shadow, and accessible label.
2. Create `MuseumTray.vue` with CSS-grid layout and felt theme class.
3. Create `TrayControls.vue` for drawer pagination and felt color chips.
4. Use only design tokens for chrome; CSS felt gradients may use token-derived colors per theme.
5. Respect `prefers-reduced-motion` for hover/flip effects.

### Phase 3: Route and Collection Launch

1. Add `/tray` route with `requiresAuth`.
2. Add Tray action to desktop and PWA collection headers.
3. `TrayViewPage.vue` uses `store.coins`; if empty, fetch default collection or return to `/` with a friendly empty state.
4. Preserve return path to collection and current filters through store/composable state.
5. Keep bulk selection disabled/irrelevant in tray mode.

### Phase 4: Coin Interaction

1. Default tap/click on a well routes to `/coin/:id`.
2. If `<CoinViewer3D>` exists, allow a compact flip control inside a well without changing default tap routing.
3. Add keyboard focus/Enter behavior for opening detail.
4. Avoid long-press as a required path because it is hard to discover and test.

### Phase 5: Tests and Validation

1. `trayLayout.test.ts`: scaling and pagination.
2. `MuseumTrayWell.test.ts`: placeholder, image rendering, click/keyboard event.
3. `MuseumTray.test.ts`: felt theme class, drawer controls, responsive-friendly class output.
4. Header test: Tray launch route appears in both desktop and PWA controls.

Run from `src/web`:

```powershell
npm.cmd test -- tray MuseumTray --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Tray tries to show more coins than current paginated data | Scope v1 to current loaded result set and label drawers accordingly. |
| Small coins become untappable | Clamp to a minimum visual/tap size and center in larger wells. |
| Visual design drifts from token system | Use tokens for controls, borders, text, and named felt theme variables. |
| Integration duplicates 3D flip code | Only add inline flip if branch 222 component exists; otherwise route to detail. |

## Out of Scope

- Drag-and-drop saved tray layout.
- Editable tray.
- Public/shared tray links.
- 3D perspective drawer rendering.
