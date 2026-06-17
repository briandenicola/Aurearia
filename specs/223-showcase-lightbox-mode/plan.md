# Implementation Plan: Table / Lightbox Showcase Mode

**Branch**: `223-showcase-lightbox-mode` | **Date**: 2026-06-17 | **Spec**: `specs/223-showcase-lightbox-mode/spec.md`  
**Merge target**: `beta`

## Summary

Add a fullscreen Present mode that displays one coin at a time on a dark, edge-lit background with swipe/keyboard navigation, tap-to-toggle metadata, obverse/reverse access, Fullscreen API support, and Wake Lock cleanup. It should launch from the authenticated gallery first and be designed so public showcases can adopt the same presentation component later.

This is priority 3 because it is a visible presentation feature that builds on existing collection data and can optionally reuse the 3D viewer branch after it merges.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Composition API  
**Primary Dependencies**: Vue Router, Pinia coins store, browser Fullscreen/Wake Lock APIs, pointer/touch events  
**Storage**: No new persistent storage; use route query/session state for return position if needed  
**Testing**: Vitest component/unit tests; `npm run type-check`; `npm run build`; browser smoke test if route behavior warrants it  
**Target Platform**: Mobile PWA, tablet, desktop browser  
**Project Type**: Frontend-only route/page feature  
**Performance Goals**: Smooth next/previous transitions for 50-coin page; no image preloading beyond adjacent coins for v1  
**Constraints**: Read-only; hides pricing/value; must preserve previous gallery filters/scroll when exiting; must release wake lock/fullscreen listeners  
**Scale/Scope**: Current collection page result set first; public showcase integration as optional follow-up unless simple

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Define a typed presentation coin shape and navigation contract; avoid broad `Coin` assumptions in reusable components where public showcase coins have fewer fields.
- **Principle IV (Simple Complete Changes)**: Ship gallery-launched Present mode first; defer slideshow, music, custom intervals, and complex source selection.
- **Principle VI (Consistent UX)**: Use design tokens, no pricing by default, dark theme, no emojis.
- **PWA/Mobile Interaction Rules**: Swipe/tap gestures must be one-handed, must not conflict with pull-to-refresh, and must respect reduced motion.
- **§17 Quality Gate / §21 DoD**: Test exit cleanup, metadata exclusion, keyboard navigation, and reduced-motion branches.

No constitution violations are expected.

## Resolved Scope Decisions

1. **Route shape**: Add authenticated route `/present` for collection presentation. Use query parameters `start`, `page`, and existing collection filters only if needed; prefer Pinia store state when launching from the loaded collection.
2. **Source set**: v1 presents the current loaded `store.coins` page/result set. Full cross-page streaming and selectable showcases are follow-ups.
3. **Public showcase**: Reuse the presentational component with a narrower coin interface, but do not make `/s/:slug` integration a blocker.
4. **Obverse/reverse**: If `222-3d-coin-viewer` is merged first, use `<CoinViewer3D>`; otherwise implement a small local face toggle and plan a follow-up refactor.
5. **Wake Lock**: Treat Wake Lock as progressive enhancement; failure should not block Present mode.

## Project Structure

```text
specs/223-showcase-lightbox-mode/
├── spec.md
└── plan.md

src/web/src/
├── router/index.ts                         # add /present route
├── pages/
│   ├── CollectionPage.vue                  # launch action with current collection state
│   ├── PresentModePage.vue                 # fullscreen route shell
│   └── PublicShowcasePage.vue              # optional later reuse
├── components/
│   ├── collection/DesktopCollectionHeader.vue
│   ├── collection/PwaCollectionHeader.vue
│   └── presentation/PresentCoinViewer.vue  # reusable one-coin display
├── composables/
│   ├── useFullscreen.ts
│   ├── useWakeLock.ts
│   └── useSwipeNavigation.ts               # only if not simple enough inline
└── components/**/__tests__/
```

**Structure Decision**: Put browser API lifecycle in composables and keep `PresentCoinViewer` stateless. `PresentModePage` owns route/store integration and exit behavior.

## Presentation Coin Contract

```ts
interface PresentCoin {
  id: number
  name?: string | null
  ruler?: string | null
  denomination?: string | null
  era?: string | null
  material?: string | null
  grade?: string | null
  images?: readonly CoinImage[]
}
```

Explicit exclusion: no `purchasePrice`, `currentValue`, purchase fields, notes, AI analysis, listing state, owner ids, or private admin metadata in overlay.

## Implementation Phases

### Phase 1: Browser API Composables

1. Add `useFullscreen(targetRef)` with `enter`, `exit`, `isFullscreen`, and cleanup on unmount.
2. Add `useWakeLock()` with `request`, `release`, and visibility-change reacquire logic.
3. Add tests for unsupported API, successful acquire/release, and unmount cleanup.

### Phase 2: Presentational Component

1. Create `components/presentation/PresentCoinViewer.vue`.
2. Render one coin centered with edge-lit/radial background and object-fit contain image.
3. Add overlay metadata with an explicit safe allowlist.
4. Add tap/click overlay toggle and reduced-motion-safe transitions.
5. Add obverse/reverse control; use `<CoinViewer3D>` if available from branch 222, otherwise use a simple local image toggle.
6. Add accessible exit/next/previous controls that auto-hide visually but remain keyboard usable.

### Phase 3: Route and Gallery Launch

1. Add route `/present` in `src/web/src/router/index.ts` with `requiresAuth`.
2. Add Present button to desktop and PWA collection headers using existing button/chip classes.
3. Pass/derive start index from current `store.galleryIndex` or selected card index.
4. On `PresentModePage` mount, use `store.coins`; if empty, fetch the current collection page with existing filters or redirect to `/`.
5. On exit, return to `/` and preserve scroll/filter state through existing store/composable state.

### Phase 4: Navigation

1. Implement touch swipe left/right for next/previous.
2. Implement ArrowLeft/ArrowRight/Escape keyboard controls.
3. Preload adjacent images by creating `Image` objects for previous/next only.
4. Prevent default browser gestures only inside the presentation container.

### Phase 5: Tests and Validation

1. Test `PresentCoinViewer` metadata overlay hides price/value fields.
2. Test tap toggles metadata overlay.
3. Test keyboard navigation emits next/previous/exit.
4. Test `PresentModePage` releases wake lock/fullscreen on exit/unmount.
5. Test collection header Present action routes to `/present`.

Run from `src/web`:

```powershell
npm.cmd test -- PresentCoinViewer PresentMode useWakeLock useFullscreen --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Wake Lock unsupported or denied | Progressive enhancement; log/display no blocking error, always release if acquired. |
| Store loses collection set on direct route load | Fetch default collection page or redirect with friendly empty state. |
| Metadata leaks value fields | Use typed allowlist and tests. |
| Gesture conflict with browser/PWA shell | Bind gestures only to presentation container; use Escape/visible exit button as fallback. |

## Out of Scope

- Auto-advance slideshow.
- Music or kiosk playlists.
- Editing coins.
- Full public showcase integration unless it falls out naturally from `PresentCoinViewer`.
