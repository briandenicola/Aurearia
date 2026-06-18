# Implementation Plan: Mint Map View

**Branch**: `225-mint-map-view` | **Date**: 2026-06-17 | **Spec**: `specs/225-mint-map-view/spec.md`  
**Merge target**: `beta`

## Summary

Add a collection Mint Map view that projects known mint names from the user's coins onto a themed inline SVG map of the ancient Mediterranean world. Pins show mint counts, tapping a pin filters or lists matching coins, and unmatched/unknown mints are surfaced explicitly instead of being dropped.

This is priority 5 because it is compelling but requires careful reference-data curation and normalization. v1 should avoid API/schema changes and heavy map dependencies.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Composition API  
**Primary Dependencies**: Vue Router, Pinia coins store, static TypeScript reference data, inline SVG/pointer events  
**Storage**: Static frontend dataset only; no database table in v1  
**Testing**: Vitest unit/component tests; `npm run type-check`; `npm run build`  
**Target Platform**: Mobile PWA, tablet, desktop browser  
**Project Type**: Frontend-only collection visualization feature  
**Performance Goals**: Aggregate and render pins for a 50-coin page instantly; no tile/network dependency; SVG pan/zoom stays responsive  
**Constraints**: No Leaflet/Mapbox dependency; no geocoding; no AI inference; no silent dropping of unmatched mints  
**Scale/Scope**: Current loaded collection/page first unless existing API/filtering can safely fetch all active coins without violating limits

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Strongly type mint reference entries, normalized groups, and unmatched buckets.
- **Principle IV (Simple Complete Changes)**: Static dataset and inline SVG only; defer editable mints/API table.
- **Principle VI (Consistent UX)**: Use design tokens, dark museum palette, accessible pin controls, no emojis.
- **Principle V (Security/Auth/Privacy)**: Authenticated collection map only; no public exposure of private collection geography unless public showcase integration is separately specified.
- **§17 Quality Gate / §21 DoD**: Tests must cover normalization aliases, unmatched bucket, pin counts, and tap-to-filter/list behavior.

No constitution violations are expected.

## Resolved Scope Decisions

1. **Dataset location**: Static frontend TypeScript dataset in `src/web/src/data/ancientMints.ts`.
2. **Map renderer**: Inline SVG, not map tiles. Approximate geography is acceptable for v1.
3. **Default extent**: Mediterranean/ancient world extent. Non-ancient or unknown mints go to unmatched/other affordance.
4. **Tap behavior**: Pin tap opens a drawer/list of coins for that mint in the map view, with a link/action to apply a collection search/filter if existing filter contracts support it.
5. **Aliases**: Alias names map to one canonical mint for v1, e.g. Byzantium/Constantinople/Istanbul can canonicalize based on dataset policy rather than era-splitting.

## Project Structure

```text
specs/225-mint-map-view/
├── spec.md
└── plan.md

src/web/src/
├── router/index.ts
├── pages/
│   ├── CollectionPage.vue
│   ├── StatsPage.vue
│   └── MintMapPage.vue
├── components/
│   ├── collection/DesktopCollectionHeader.vue
│   ├── collection/PwaCollectionHeader.vue
│   └── map/
│       ├── MintMapSvg.vue
│       ├── MintPin.vue
│       ├── MintCoinDrawer.vue
│       └── UnattributedMintBucket.vue
├── data/
│   └── ancientMints.ts
├── utils/
│   └── mintMap.ts
└── components/**/__tests__/
```

**Structure Decision**: Keep reference data (`data/ancientMints.ts`) separate from matching/aggregation logic (`utils/mintMap.ts`) and map rendering components (`components/map/*`).

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

Dataset v1 should include at least the fixture/common mints already present in the codebase and likely collection values: Rome, Athens, Constantinople, Alexandria, Antioch, Syracuse, Trier, Lugdunum/Lyon, Siscia, Nicomedia, Cyzicus, Carthage, Thessalonica, Heraclea, Aquileia, Arles/Arelate, and Ephesus.

## Implementation Phases

### Phase 1: Reference Data and Matching

1. Create `data/ancientMints.ts` with canonical mints and aliases.
2. Create `utils/mintMap.ts`:
   - `normalizeMintName(value: string): string`
   - `findMintReference(value: string): MintReference | null`
   - `groupCoinsByMint(coins: Coin[]): { matched: MintGroup[]; unmatched: UnmatchedMintGroup[]; unknown: Coin[] }`
   - `projectLatLngToViewBox(lat, lng)` for the chosen SVG extent.
3. Add tests for case-insensitive matching, punctuation/spacing, aliases, unknown empty mints, and unmatched non-empty mints.

### Phase 2: SVG Map Components

1. Create `MintMapSvg.vue` with a dark, stylized Mediterranean/ancient-world SVG backdrop.
2. Create `MintPin.vue` as an accessible button with count badge and active state.
3. Implement simple pan/zoom:
   - wheel/trackpad zoom on desktop.
   - pointer drag pan.
   - mobile pinch only if straightforward; otherwise provide zoom buttons for v1.
4. Use token-based colors and minimum tap target sizes.

### Phase 3: Page and Navigation

1. Add `/mint-map` route requiring auth.
2. Add Map launch action in collection headers and a Stats page card/link near existing heat map.
3. `MintMapPage.vue` uses `store.coins`; if empty, fetch default active collection.
4. Show summary counts: matched mints, unmatched mint names, unknown mint coins.
5. Pin tap opens `MintCoinDrawer` listing coins for that mint with links to `/coin/:id`.
6. Unattributed affordance opens `UnattributedMintBucket` with unknown and unmatched groups.

### Phase 4: Optional Collection Filtering Bridge

1. If current collection filters can accept a mint search without new API changes, add "View in Gallery" that routes to `/` with search set to the mint name.
2. If filter state cannot be reliably set through route/query today, keep drawer-only behavior and document query-filter as follow-up.

**Implementation note (Aurelia, 2026-06-17):** Existing collection query handling does not initialize the gallery search filter from route query parameters, so v1 keeps pin results inside the drawer. A safe gallery bridge should first add an explicit, tested collection route/query contract for search prefill.

### Phase 5: Tests and Validation

1. `mintMap.test.ts`: matching/aggregation/projection.
2. `MintMapSvg.test.ts`: renders pins and emits selected mint.
3. `MintMapPage.test.ts`: unmatched bucket appears and coin drawer lists only selected mint coins.
4. Header/Stats launch tests if buttons are added.

Run from `src/web`:

```powershell
npm.cmd test -- mintMap MintMap --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Reference dataset is incomplete | Ship unmatched bucket prominently and add tests for known common mints; dataset can grow incrementally. |
| Alias mapping is historically ambiguous | Canonicalize aliases for v1 and document era-specific splitting as follow-up. |
| SVG geography is too detailed to maintain | Use a stylized bounded map, not precise GIS/tile data. |
| Current loaded page hides coins from other pages | Label v1 source clearly; only expand to all active coins if existing API limits make it safe. |

## Out of Scope

- Go API `mints` table.
- User-editable mint coordinates.
- Reverse geocoding or AI mint inference.
- Trade routes or animated geography.
