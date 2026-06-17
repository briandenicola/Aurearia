# Spec: Table / Lightbox Showcase Mode

**Status:** Draft
**Area:** Frontend (Vue 3 / TS) — Collection display
**Depends on:** none (reuses existing images + metadata)
**Related:** Swipe Gallery, Collection Showcase (public `/s/:slug`)

---

## Summary

A full-bleed, dark, edge-lit presentation mode that shows one coin at a time —
large, swipeable, with metadata fading in on tap. A "kiosk / show-this-to-someone"
view, distinct in intent from the working grid: no filters, no chrome, screen kept
awake. Think a single coin tray displayed on a dark table under a spotlight.

## Motivation

Every current view (grid, swipe, stats) is a *working* view optimized for managing
the collection. There is no view optimized purely for *presenting* it. When showing
the collection to another collector or a guest, a clean, immersive, distraction-free
mode makes the coins the entire focus.

## User Stories

- As a collector showing a friend my collection, I enter a fullscreen mode where one
  coin fills the screen and I swipe through, with no UI getting in the way.
- As a presenter, the screen stays awake while I talk through each coin.
- As a viewer, I tap to reveal name/ruler/denomination/era, and tap again to hide it
  and just appreciate the coin.

## Scope

### In scope
- A "Showcase / Present" mode launched from the gallery (and optionally from a
  showcase or a multi-select set).
- One coin per screen, maximized, on the museum-dark background with an edge-lit /
  spotlight treatment.
- Swipe (and arrow-key on desktop) to move between coins.
- Tap to toggle a minimal metadata overlay (name, ruler, denomination, era,
  optionally material/grade); pricing hidden by default to match showcase ethos.
- Use the Fullscreen API where available and a Wake Lock so the screen doesn't dim.
- Obverse/reverse access (tap-and-hold, a small flip control, or integrate the
  `<CoinViewer3D>` component if that ships).

### Out of scope
- Editing coins from this mode (read-only by design).
- Public sharing — that's the existing Collection Showcase feature; this is a local
  presentation mode (though it may launch *from* a showcase).
- Slideshow auto-advance / music (note as optional later).

## Design / Approach

- New route/overlay, e.g. `/present` or a fullscreen modal layer over the gallery,
  seeded with the current filtered/sorted coin set or a chosen showcase.
- Layout: centered large coin image, generous negative space, subtle radial
  vignette / edge light to lift the coin off the background.
- Controls auto-hide; a single tap toggles the metadata overlay, swipe changes coin.
- Request fullscreen on enter (where supported); acquire `navigator.wakeLock` and
  release it on exit / visibility change.
- Respect `prefers-reduced-motion` for transitions between coins.
- Reuse existing image gallery data; no new endpoints.

## Acceptance Criteria

- [ ] A Present/Showcase mode can be launched from the gallery and fills the screen
      with one coin at a time on the dark theme.
- [ ] Swiping (touch) and arrow keys (desktop) move between coins in the current set.
- [ ] Tapping toggles a minimal metadata overlay; pricing/value is not shown.
- [ ] Obverse and reverse are both viewable in this mode.
- [ ] On supported browsers the view goes fullscreen and the screen does not dim
      while the mode is active (Wake Lock), releasing correctly on exit.
- [ ] Exiting returns to the previous gallery state (scroll/filter preserved).
- [ ] `prefers-reduced-motion` is respected for coin-to-coin transitions.
- [ ] `npm run type-check` passes; uses existing design tokens.

## Open Questions

- Source set: always the current filtered gallery, or can the user pick a showcase /
  a multi-selected subset to present?
- Should this reuse `<CoinViewer3D>` for obverse/reverse (dependency on that card),
  or ship a simpler flip independent of it?
- Optional auto-advance slideshow with a configurable interval — v1 or later?
