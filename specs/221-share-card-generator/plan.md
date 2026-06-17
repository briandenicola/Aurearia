# Implementation Plan: Share Card Generator

**Branch**: `221-share-card-generator` | **Date**: 2026-06-17 | **Spec**: `specs/221-share-card-generator/spec.md`  
**Merge target**: `beta`

## Summary

Add a client-side share-card flow for a single coin. The implementation should add a Share action on coin detail, compose a branded PNG from existing coin images and non-sensitive metadata in the browser, invoke the Web Share API with a file when supported, and provide a download fallback when native file sharing is unavailable.

This is the first priority because it is high-value, self-contained, and does not require API, database, or dependency changes.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Vite PWA frontend  
**Primary Dependencies**: Vue, lucide-vue-next, existing browser Canvas/Web Share APIs  
**Storage**: N/A; generated images are transient browser blobs  
**Testing**: Vitest component/unit tests; `npm run type-check`; `npm run build`  
**Target Platform**: Installed/mobile PWA first, desktop browser fallback  
**Project Type**: Frontend-only web application feature  
**Performance Goals**: Generate a card for a normal coin image in under 1 second on a modern phone; keep the UI responsive while images load  
**Constraints**: Offline-capable v1; no value/pricing on generated card; no server dependency; no new npm dependency unless canvas/SVG limitations prove it necessary  
**Scale/Scope**: One coin at a time from detail page; optional gallery-card entry is deferred

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Use typed helper inputs/outputs for card data and share results; no `any` browser API casts beyond narrow feature-detection guards.
- **Principle IV (Simple Complete Changes)**: Ship one fixed branded template and one fallback path; defer editable layouts, multi-coin collages, and server-side OG rendering.
- **Principle VI (Consistent UX)**: Use existing button classes, dark theme, design tokens, and no emojis.
- **Principle V (Security, Auth, Privacy)**: Do not include `purchasePrice`, `currentValue`, purchase location, notes, private fields, AI analysis, or owner-sensitive metadata in the image.
- **§17 Quality Gate / §21 DoD**: Add targeted tests for metadata exclusion, Web Share supported path, and fallback path; run frontend build/type-check.

No constitution violations are expected.

## Resolved Scope Decisions

1. **Template**: v1 ships a single obverse-first branded template. If both obverse and reverse exist, the plan may include a small reverse thumbnail only if implementation remains simple; otherwise defer two-up layout.
2. **Entry point**: Add the action to `CoinDetailHeaderActions.vue` and handle orchestration in `CoinDetailPage.vue` so the action has access to the loaded `Coin`.
3. **Rendering**: Use an offscreen `<canvas>` helper first. If cross-origin/image loading blocks canvas export for app-served uploads, fix the image-loading path rather than moving rendering server-side.
4. **Fallback**: Download PNG via an object URL. Clipboard image copy is a follow-up unless browser support is trivial and testable.
5. **Server OG endpoint**: Explicitly out of scope for this branch.

## Project Structure

```text
specs/221-share-card-generator/
├── spec.md
└── plan.md

src/web/src/
├── components/
│   ├── coin/CoinDetailHeaderActions.vue      # add Share button and event
│   └── share/CoinShareFallbackModal.vue      # optional small fallback modal if direct download is not enough
├── composables/
│   └── useCoinShareCard.ts                   # share/download orchestration
├── utils/
│   └── coinShareCard.ts                      # canvas rendering + metadata shaping
└── pages/
    └── CoinDetailPage.vue                    # wire action to current coin

src/web/src/**/__tests__/
└── coinShareCard tests near the helper/composable
```

**Structure Decision**: Keep rendering logic out of Vue components. Components own user interaction; `useCoinShareCard` owns browser capability detection; `coinShareCard.ts` owns deterministic canvas layout and privacy-safe metadata selection.

## Data and Contracts

No API contract changes. Define frontend-only types:

```ts
interface CoinShareCardInput {
  coin: Coin
  imageUrl: string | null
  appName: string
}

interface CoinShareResult {
  mode: 'shared' | 'downloaded' | 'unsupported'
}
```

Privacy contract: generated cards may include only `name`, `ruler`, `denomination`, `era`, `mint`, `material`, `grade`, `category`, and public coin image. Pricing/value, notes, purchase fields, owner ids, private flags, AI analysis, listing status, tags, and sets are excluded by default.

## Implementation Phases

### Phase 1: Rendering Helper

1. Create `src/web/src/utils/coinShareCard.ts`.
2. Add `getShareCardMetadata(coin)` that returns a strict allowlist of display fields.
3. Add `getPreferredShareImage(coin)` that chooses obverse, then primary, then first image.
4. Add `renderCoinShareCard(input): Promise<Blob>` using canvas dimensions suitable for social sharing, e.g. 1200x1600 or 1080x1350.
5. Use design-token color values mirrored as constants only where canvas cannot read CSS variables directly; keep them named after the token.
6. Handle missing images by rendering a branded placeholder, not by failing silently.

### Phase 2: Share/Download Composable

1. Create `src/web/src/composables/useCoinShareCard.ts`.
2. Convert PNG blob to a `File`.
3. Feature-detect `navigator.share` and `navigator.canShare?.({ files })`.
4. If supported, call native share with `{ files, title, text }`.
5. If unsupported, create an object URL and download `coin-name-share-card.png`.
6. Surface errors through the existing dialog pattern from `useDialog`; do not swallow generation/share failures.
7. Revoke object URLs after download or modal close.

### Phase 3: Coin Detail Integration

1. Update `CoinDetailHeaderActions.vue` to accept a Share action button using existing `.btn`, `.btn-secondary`, `.btn-xs` classes and a lucide `Share2` icon.
2. Emit `share` without adding business logic to the header component.
3. Update `CoinDetailPage.vue` to call the composable with `coin.value`.
4. Disable or show a loading state while the share card is being generated.
5. Keep Sell/Edit/Delete behavior unchanged.

### Phase 4: Tests

1. Add unit tests for `getShareCardMetadata()` proving prices, values, notes, AI analysis, and purchase fields are excluded.
2. Add a render test with mocked image/canvas APIs proving a Blob is requested and missing image fallback does not throw.
3. Add composable tests for native share path and download fallback path.
4. Add or update a `CoinDetailHeaderActions` component test proving Share emits and existing actions remain available.

### Phase 5: Validation

Run from `src/web`:

```powershell
npm.cmd test -- coinShareCard --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Canvas tainted by image origin | Use same-origin `/uploads/...` URLs and set `crossOrigin` only if needed; do not fetch third-party images. |
| Web Share file support varies | Gate with `canShare`; always provide a download fallback. |
| Privacy leak | Metadata helper uses an allowlist and has explicit exclusion tests. |
| Canvas layout hardcodes theme values | Keep canvas-only constants named after design tokens and align visually with `variables.css`. |

## Out of Scope

- Server-side OG image endpoint.
- Multi-coin collages.
- User-editable templates.
- Sharing private/public links.
