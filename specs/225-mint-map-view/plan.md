# Implementation Plan: Mint Map View

**Branch**: `225-mint-map-view` | **Date**: 2026-06-18 | **Spec**: `specs/225-mint-map-view/spec.md`
**Merge target**: `beta`

## Summary

Pivot the beta Mint Map from a stylized inline SVG approximation to a real Leaflet map using OpenStreetMap tiles. Pins must be rendered from actual mint latitude/longitude data. The feature moves under Stats as `/stats/mint-map`; legacy `/mint-map` redirects there. Stats becomes a landing page with cards for Mint Map, Timeline, and Collection Distribution subviews.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Composition API
**Primary Dependencies**: Vue Router, Pinia coins store, static TypeScript mint reference data, `leaflet`, Leaflet CSS, OpenStreetMap tile layer
**Storage**: Static frontend dataset only; no database table in this follow-up
**Testing**: Vitest unit/component tests; `npm run type-check`; `npm run build`
**Target Platform**: Mobile PWA, tablet, desktop browser
**Project Type**: Frontend-only Stats visualization/navigation correction
**Performance Goals**: Aggregate collection mints quickly; render tens to hundreds of markers responsively; load tiles only when map subview opens
**Constraints**: Real geographic map required; OSM tile network requests allowed; no geocoding; no AI inference; no Go API/schema changes; no collection-data leakage to tile URLs
**Scale/Scope**: Existing active collection data source first unless an already-safe all-active-coin path exists

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Strongly type mint reference entries, grouped marker data, and route names.
- **Principle IV (Simple Complete Changes)**: Replace the wrong map workflow directly and clean directly related Stats/legacy navigation paths; do not broaden into backend mint management.
- **Principle V (Security/Auth/Privacy)**: Keep map authenticated; document external OSM tile requests; never send private collection data to tile providers.
- **Principle VI (Consistent UX)**: Use design tokens around the map, preserve PWA/mobile navigation, no Collection-header launch.
- **§17 Quality Gate / §21 DoD**: Tests must cover the exact route/navigation and Leaflet marker workflow regressions introduced by the beta mismatch.

No constitution waiver is expected.

## Resolved Scope Decisions

1. **Dataset location**: Static frontend TypeScript dataset in `src/web/src/data/ancientMints.ts` remains acceptable.
2. **Map renderer**: Leaflet with OpenStreetMap tiles replaces the inline SVG renderer.
3. **Coordinates**: Marker placement uses actual `lat`/`lng` from the dataset; remove SVG projection from the active rendering path.
4. **Feature placement**: Mint Map is a Stats subview only: `/stats/mint-map`.
5. **Legacy route strategy**: `/mint-map` and `/timeline` redirect to `/stats/mint-map` and `/stats/timeline`.
6. **Stats IA**: `/stats` is a card-based landing page; Timeline and Collection Distribution are sibling subviews.
7. **Tap behavior**: Marker tap opens the in-map/page coin list drawer for that mint; gallery query bridge remains optional only if an explicit tested route contract exists.
8. **Aliases**: Alias names map to one canonical mint for this follow-up; era-specific splitting remains out of scope.

## Project Structure

```text
specs/225-mint-map-view/
├── spec.md
├── plan.md
└── tasks.md

src/web/src/
├── router/index.ts
├── pages/
│   ├── StatsPage.vue                # landing cards
│   ├── MintMapPage.vue              # /stats/mint-map
│   ├── TimelinePage.vue             # /stats/timeline, if split from existing stats code
│   └── CollectionDistributionPage.vue # /stats/distribution
├── components/
│   └── map/
│       ├── MintMapLeaflet.vue       # replaces active MintMapSvg usage
│       ├── MintCoinDrawer.vue
│       └── UnattributedMintBucket.vue
├── data/
│   └── ancientMints.ts
├── utils/
│   └── mintMap.ts
└── components/**/__tests__/
```

**Structure Decision**: Keep reference data (`data/ancientMints.ts`) separate from aggregation (`utils/mintMap.ts`) and Leaflet rendering (`components/map/MintMapLeaflet.vue`). The beta `MintMapSvg.vue` should be deleted or disconnected once Leaflet replaces it.

## Data Contract

```ts
interface MintReference {
  id: string
  displayName: string
  lat: number
  lng: number
  aliases: readonly string[]
  region?: string
}

interface MintGroup {
  mint: MintReference
  coins: Coin[]
  count: number
}

interface UnmatchedMintGroup {
  normalizedName: string
  originalNames: string[]
  coins: Coin[]
}
```

Remove active dependency on `projectLatLngToViewBox()` for map rendering. It may be deleted if no tests or code need it after `MintMapSvg.vue` is retired.

## Implementation Phases

### Phase 1: Dependency and Tile Policy

1. Add `leaflet` to `src/web/package.json` and install/update the lockfile.
2. Import Leaflet CSS from a stable frontend entry/component path.
3. Document in comments/tests only where needed that OSM tile URLs must contain no private collection data.
4. Mock Leaflet/tile behavior in unit tests; do not require live tile network in Vitest.

### Phase 2: Leaflet Map Component

1. Create `MintMapLeaflet.vue` (or rename `MintMapSvg.vue` only if history remains clear) to initialize/destroy a Leaflet map safely on mount/unmount.
2. Add an OSM tile layer using the standard attribution string.
3. Render markers from `MintGroup[].mint.lat/lng` with count labels or marker badges.
4. Emit selected mint/group on marker click and keyboard-accessible activation.
5. Fit bounds to matched markers; use a Mediterranean default center/zoom when no markers exist.
6. Keep map container sizing tokenized and mobile/PWA-safe.

### Phase 3: Mint Map Page Update

1. Replace active `MintMapSvg` usage in `MintMapPage.vue` with `MintMapLeaflet`.
2. Preserve matched/unmatched/unknown summaries and selected-mint drawer behavior.
3. Ensure empty/loading/error states still work without initializing broken map state.
4. Remove obsolete SVG projection UI/tests once no longer used.

### Phase 4: Stats Subviews and Redirects

1. Change `/stats` to a landing page with cards for Mint Map, Timeline, and Collection Distribution.
2. Add authenticated child/sibling routes for `/stats/mint-map`, `/stats/timeline`, and `/stats/distribution`.
3. Redirect `/mint-map` → `/stats/mint-map` and `/timeline` → `/stats/timeline`.
4. Remove Mint Map actions from `DesktopCollectionHeader.vue` and `PwaCollectionHeader.vue`.
5. Move existing Collection Distribution UI out of the landing page into its subview.

### Phase 5: Tests and Validation

1. `mintMap.test.ts`: keep normalization/aggregation tests; remove projection-only assertions if projection is deleted.
2. `MintMapLeaflet.test.ts`: verifies marker creation from lat/lng, count labels, selected-mint emission, default bounds/empty state, and OSM tile layer configuration without live network.
3. `MintMapPage.test.ts`: verifies drawer/unattributed behavior with the Leaflet component stubbed as needed.
4. Router tests: legacy redirects and authenticated Stats subview route targets.
5. Stats tests: landing cards link to subviews; Collection headers no longer expose Mint Map.

Run from `src/web`:

```powershell
npm.cmd install
npm.cmd test -- mintMap MintMap Stats router --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| External OSM tiles reveal network metadata | Document accepted tile requests; never encode collection data into tile URLs; auth-gate collection-derived markers. |
| Leaflet is browser/DOM-heavy in tests | Wrap in a focused component and mock/stub Leaflet in Vitest where needed. |
| CSS import breaks production build | Validate with `npm.cmd run build`, not only local type-check. |
| Stats route split breaks old bookmarks | Add explicit redirects for `/mint-map` and `/timeline`. |
| Collection headers retain stale entry points | Add tests/assertions proving header actions no longer include Mint Map. |
| Reference dataset remains incomplete | Keep unmatched bucket prominent and grow aliases incrementally. |

## Out of Scope

- Go API `mints` table.
- User-editable mint coordinates.
- Reverse geocoding or AI mint inference.
- Custom tile server/offline tile cache.
- Trade routes or animated geography.
