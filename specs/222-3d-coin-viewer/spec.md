# Spec: 3D Flip / Gyroscope Coin Viewer

**Status:** Draft
**Area:** Frontend (Vue 3 / TS) — Collection display
**Depends on:** none (no API or schema changes)
**Related:** Swipe Gallery, Coin Detail page, Face Toggle (obverse/reverse)

---

## Summary

Replace the flat obverse/reverse face toggle with an interactive 3D coin that
flips on tap and (on supported mobile devices) subtly tilts and catches light as
the device is angled. Goal is to make browsing the collection feel like physically
handling a struck coin rather than swapping two photos.

## Motivation

The app already stores obverse and reverse images and exposes a face toggle, but
the experience is a flat image swap. A 3D flip plus gyroscope-driven lighting turns
each coin into a tactile object, which is the single highest-impact "delight"
upgrade for the mobile/PWA experience and a natural showpiece when demoing the app.

## User Stories

- As a collector browsing on my phone, I tap a coin and it flips with a 3D rotation
  to reveal the reverse, so it feels like turning the coin over in my hand.
- As a collector, when I tilt my phone, light glints across the coin's relief so I
  can appreciate the strike and surfaces.
- As a user who prefers reduced motion, the viewer respects my OS setting and falls
  back to a simple cross-fade or instant face swap.

## Scope

### In scope
- A reusable `<CoinViewer3D>` component used in (a) the swipe gallery card renderer
  and (b) the coin detail page hero.
- Tap / click to flip between obverse and reverse using a 3D `rotateY` transition.
- Edge treatment so the disc reads as a struck coin, not a flipping rectangle
  (subtle bevel via box-shadow / gradient on the coin rim).
- Optional gyroscope tilt + moving specular highlight on devices that support
  `DeviceOrientationEvent`, gated behind the iOS 13+ permission prompt.
- `prefers-reduced-motion` fallback.

### Out of scope
- WebGL / three.js metallic shader (note as a possible Tier-3 follow-up).
- Editing or capturing images (existing upload flow unchanged).
- 3D rendering of the coin edge photo (edge image, if present, not mapped onto rim).

## Design / Approach

**Tier 1 — CSS 3D flip (baseline, ships first)**
- Container with `perspective`; inner element with `transform-style: preserve-3d`.
- Two faces: front = obverse image, back = reverse image (`backface-visibility:
  hidden`, back pre-rotated `rotateY(180deg)`).
- Tap toggles a `flipped` class that animates `rotateY(0 -> 180deg)`.
- Circular mask (`border-radius: 50%`) + rim shadow/gradient for the struck-disc read.

**Tier 2 — Gyroscope tilt + glint (progressive enhancement)**
- On first interaction, if `DeviceOrientationEvent.requestPermission` exists (iOS),
  request permission from within the tap handler. If granted (or not required),
  subscribe to `deviceorientation`.
- Map `beta` (front-back) and `gamma` (left-right) to a small `rotateX` / `rotateY`,
  clamped to ±15° so the design is never hidden.
- Drive a radial-gradient "glint" overlay position from the same angles.
- Throttle updates to `requestAnimationFrame`; unsubscribe on unmount / when offscreen.

**Reduced motion**
- If `prefers-reduced-motion: reduce`, disable tilt, replace flip with an instant
  swap or 150ms cross-fade.

## Acceptance Criteria

- [ ] Tapping a coin in the swipe gallery flips it in 3D to show the reverse, and
      tapping again flips back.
- [ ] The coin renders as a circular disc with a visible rim/bevel, not a rectangle.
- [ ] On an iOS device, the first tilt interaction triggers the motion-permission
      prompt; granting it enables tilt, denying it leaves flip working with no errors.
- [ ] On a supported device, tilting the phone moves a light highlight across the
      coin and applies a clamped (±15°) parallax tilt.
- [ ] With `prefers-reduced-motion: reduce` set, no tilt occurs and the face change
      is an instant/cross-fade swap.
- [ ] Component is used by both the swipe gallery and the detail page hero from a
      single shared implementation.
- [ ] No regression to existing grid view or face toggle for users who don't open
      the 3D viewer.
- [ ] `npm run type-check` passes; respects existing design tokens (ADR 0004).

## Open Questions

- Should the 3D viewer fully replace the existing face toggle, or sit behind a
  setting / be the default only in swipe mode?
- Coins with only one image — show a static disc, or a placeholder back face?
- Is a WebGL metallic-shader tier worth a separate backlog card later?
