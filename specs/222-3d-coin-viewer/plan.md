# Implementation Plan: 3D Flip / Gyroscope Coin Viewer

**Branch**: `222-3d-coin-viewer` | **Date**: 2026-06-17 | **Spec**: `specs/222-3d-coin-viewer/spec.md`  
**Merge target**: `beta`

## Summary

Create a reusable `<CoinViewer3D>` component that renders obverse and reverse coin images as a circular, beveled disc with tap/click flip behavior. Add progressive gyroscope tilt and glint only when supported and permitted, and respect `prefers-reduced-motion`. Integrate the shared component into the mobile swipe gallery and the coin detail hero without changing upload, edit, or grid behavior.

This is priority 2 because it establishes the shared presentation primitive that can later improve Showcase Lightbox and Museum Tray without duplicating flip logic.

## Technical Context

**Language/Version**: Vue 3 with TypeScript, Composition API  
**Primary Dependencies**: Vue, existing Coin/Image types, browser `DeviceOrientationEvent`, CSS transforms  
**Storage**: N/A  
**Testing**: Vitest component/unit tests; `npm run type-check`; `npm run build`  
**Target Platform**: Mobile PWA and desktop browser  
**Project Type**: Frontend-only component feature  
**Performance Goals**: Maintain 60fps for CSS flip; throttle orientation updates with `requestAnimationFrame`; no layout thrash during tilt  
**Constraints**: No WebGL/three.js dependency; no API/schema changes; no regression to grid view or image lightbox; support reduced-motion users  
**Scale/Scope**: One visible coin component at a time in swipe/detail hero contexts

## Constitution Check

- **Principle III (Strict Types and Explicit Contracts)**: Typed props for obverse/reverse images and options; typed guards around optional `DeviceOrientationEvent.requestPermission`.
- **Principle IV (Simple Complete Changes)**: CSS 3D baseline first; gyro is progressive enhancement; no WebGL tier in this branch.
- **Principle VI (Consistent UX)**: Uses design tokens, lucide icons, no emojis, dark theme compatibility.
- **Principle VI / PWA Mobile**: Touch behavior must not break swipe-gallery drag/tap behavior or existing PWA navigation.
- **§17 Quality Gate / §21 DoD**: Tests must cover flip, single-image fallback, permission denied behavior, and reduced-motion behavior.

No constitution violations are expected.

## Resolved Scope Decisions

1. **Default placement**: Replace the swipe gallery's current scaleX flip implementation with `<CoinViewer3D>`, and add the component to the detail-page hero as an enhanced dual-side viewer while retaining lightbox access.
2. **Tap semantics in swipe gallery**: Keep current swipe-card tap-to-detail behavior. The viewer flip should be triggered by a dedicated flip control or a tap area that does not conflict with drag/tap navigation.
3. **Single-image coins**: Render a static circular disc and disable flip control unless a second face is present.
4. **Gyroscope permission**: Request iOS motion permission only from a user gesture. Denial leaves flip fully functional and does not show repeated prompts.
5. **Reduced motion**: Disable gyro and use instant/cross-fade face swap.

## Project Structure

```text
specs/222-3d-coin-viewer/
├── spec.md
└── plan.md

src/web/src/
├── components/
│   ├── coin/CoinViewer3D.vue              # reusable viewer
│   ├── SwipeGallery.vue                   # use viewer for active/next cards
│   └── coin/CoinDetailHeaderActions.vue   # unchanged unless a viewer control is needed
├── composables/
│   ├── useDeviceOrientation.ts            # permission, subscription, RAF throttle
│   └── useReducedMotion.ts                # shared media-query helper if not already present
├── pages/
│   └── CoinDetailPage.vue                 # replace dual flat hero slots with shared viewer section
└── components/**/__tests__/
    ├── CoinViewer3D.test.ts
    └── SwipeGallery.test.ts updates
```

**Structure Decision**: Keep device-orientation logic in a composable and the visual transform state in the viewer. `SwipeGallery` should not know gyroscope details.

## Component Contract

```ts
interface CoinViewer3DProps {
  obverseSrc?: string | null
  reverseSrc?: string | null
  obverseAlt?: string
  reverseAlt?: string
  size?: 'card' | 'hero' | 'tray'
  interactive?: boolean
  enableTilt?: boolean
}
```

Events:

- `flip`: emitted after user requests a face change.
- `open-image`: emitted when the current face should open the existing lightbox from detail page.

The component must not fetch coin data or mutate store state.

## Implementation Phases

### Phase 1: Foundation Helpers

1. Add `useReducedMotion()` composable using `window.matchMedia('(prefers-reduced-motion: reduce)')` with cleanup.
2. Add `useDeviceOrientation()` composable:
   - exposes `supported`, `permissionState`, `requestPermission()`, `start()`, `stop()`, and clamped `tilt`.
   - handles iOS static `DeviceOrientationEvent.requestPermission`.
   - throttles updates with `requestAnimationFrame`.
   - unregisters listeners on unmount.
3. Add unit tests with mocked `matchMedia`, `DeviceOrientationEvent`, and event listeners.

### Phase 2: `<CoinViewer3D>`

1. Create the component with two circular faces using `perspective`, `transform-style: preserve-3d`, and `backface-visibility`.
2. Render rim/bevel using token-based shadows/gradients and circular masks.
3. Add a small accessible flip button using lucide icon text/aria-label, not an emoji.
4. Apply gyroscope tilt/glint only when `enableTilt` and reduced motion is false.
5. Add disabled/static rendering for one-image or no-image coins.
6. Add keyboard support: Enter/Space triggers flip when focused.

### Phase 3: Swipe Gallery Integration

1. Replace `SwipeGallery.vue` active-card image block with `<CoinViewer3D>`.
2. Preserve existing drag/swipe pagination and tap-to-detail behavior.
3. Remove the current emoji flip button and scaleX animation.
4. Keep next-card rendering lightweight; use static image or non-tilting viewer for the next card.
5. Update `SwipeGallery.test.ts` to assert flip control exists and page-change behavior remains intact.

### Phase 4: Coin Detail Hero Integration

1. In `CoinDetailPage.vue`, replace the two-slot hero grid with a hero-sized `<CoinViewer3D>` when either obverse or reverse image exists.
2. Preserve placeholders for missing images and existing `ImageLightbox` open/save behavior.
3. Keep wishlist purchase CTA unchanged.
4. Ensure desktop detail layout and mobile layout remain responsive.

### Phase 5: Tests and Validation

1. `CoinViewer3D.test.ts`: two-sided flip, one-sided disabled flip, missing image placeholder, reduced-motion class/behavior.
2. `useDeviceOrientation.test.ts`: permission granted, permission denied, unsupported browser, cleanup.
3. `SwipeGallery.test.ts`: drag navigation still emits page changes; flip interaction does not navigate to detail.
4. Manual mobile validation on PWA if possible for iOS permission prompt and denial path.

Run from `src/web`:

```powershell
npm.cmd test -- CoinViewer3D SwipeGallery useDeviceOrientation --run
npm.cmd run type-check
npm.cmd run build
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Flip tap conflicts with swipe tap-to-detail | Use a dedicated flip control and stop propagation; preserve card tap for navigation. |
| iOS permission prompt cannot be automated | Unit-test guards and document manual validation; keep denial path safe. |
| Too much motion in PWA | Respect reduced motion and clamp tilt to ±15°. |
| Detail hero loses side-by-side comparison | Provide clear flip control and keep lightbox access; side-by-side can remain if implementation shows regression risk. |

## Out of Scope

- WebGL/three.js metallic shader.
- Edge-photo mapping.
- Image editing/upload changes.
- Saved user setting for default 3D behavior.
