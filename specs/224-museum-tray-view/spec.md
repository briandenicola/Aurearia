# Spec: Museum Coin Tray View

**Status:** Draft
**Area:** Frontend (Vue 3 / TS) — Collection display
**Depends on:** none required; *enhanced by* size-accurate rendering (diameter field)
**Related:** Showcase / Lightbox Mode, Swipe Gallery, Collection Showcase

---

## Summary

A display mode that lays the collection out as physical coins resting in a museum
exhibit tray — a textured red felt (or configurable) surface with recessed coin
wells, soft drop shadows, and coins seated in a grid the way you'd see them in a
museum drawer or a dealer's case. Distinct from the single-coin lightbox: this is
the "see them all as objects" view rather than "spotlight one."

## Motivation

The existing/planned Showcase (Lightbox) mode presents one coin at a time. There is
no view that conveys the collection as a set of physical artifacts arranged
together. A felt-tray layout is immediately evocative, makes a collection feel
tangible and curated, and is a strong presentation surface for showing the whole
collection (or a curated subset) to a guest at a glance.

## User Stories

- As a collector, I open a "Tray" view and see my coins seated in felt-lined wells
  like a museum drawer, so the collection reads as physical objects, not data cards.
- As a collector, I tap a coin in the tray to flip it or open its detail.
- As a collector with a large collection, the tray paginates into multiple "drawers"
  I can move between.
- As a collector, coins are sized relative to each other (a small obol vs a large
  sestertius), so the tray honestly reflects their physical scale.

## Scope

### In scope
- A "Tray" display mode launched from the gallery (and optionally from a showcase or
  a multi-selected subset).
- A textured felt surface with recessed/wellled coin seats, soft shadows, and coins
  composited to look seated in the tray.
- A responsive grid of wells that reflows for phone / tablet / desktop.
- Pagination into multiple "drawers" for large collections.
- Tap a coin → flip in place (reuse `<CoinViewer3D>` if available) or open detail.
- Size-accurate seating: scale each coin by its `diameter` relative to others, with
  a sensible min/max clamp and a graceful default when diameter is missing.
- Felt color options (e.g. classic red, museum green, navy) themed to the palette.

### Out of scope
- Drag-and-drop rearranging / saving custom tray layouts (note as a later card).
- Editing coins from the tray (read-only display).
- 3D/perspective tray rendering (flat top-down view for v1).
- Public sharing of a tray — that remains the existing Collection Showcase feature
  (though Tray may launch *from* a showcase set).

## Design / Approach

**Layout**
- Top-down flat tray: a felt-textured background (CSS texture/gradient or a
  lightweight tiled asset) with a grid of circular wells.
- Each well = a slight inset shadow / darker recess; the coin image sits in it with
  a soft contact shadow so it reads as resting in felt.
- Grid reflows by viewport; wells keep finger-friendly tap targets on mobile.

**Size-accurate seating (enhancement)**
- If `diameter` is present, scale coin render size proportionally across the tray,
  clamped to a min/max so a 9mm coin is still tappable and a large medallion doesn't
  dominate. Coins lacking diameter fall back to a neutral default size (optionally
  flagged subtly).
- This is the same mechanic proposed in the earlier "size-accurate rendering" idea;
  the tray is its most natural home.

**Interaction**
- Tap a coin → either flip in place (reuse the 3D viewer card if shipped) or open the
  coin detail page; pick one default, make the other a long-press (open question).
- Pagination control for "drawers" (e.g. N coins per tray); preserve position on
  return.
- Optional: pinch-zoom the tray to inspect a cluster more closely.

**Theming**
- Felt color selectable (red default); use existing design tokens for chrome.
- Respect `prefers-reduced-motion` for any seat/flip transitions.

**Data**
- Reuses existing coin images + metadata and the `diameter` field. No new endpoints.

## Acceptance Criteria

- [ ] A "Tray" view is reachable from the gallery and renders coins seated in
      felt-lined wells on a textured tray background.
- [ ] Coins read as resting in the tray (recessed wells + contact shadows), not as
      flat cards on a color block.
- [ ] The tray grid reflows sensibly across phone, tablet, and desktop widths.
- [ ] Large collections paginate into multiple drawers with navigation, and position
      is preserved when returning from a coin.
- [ ] Tapping a coin flips it in place or opens its detail (per chosen default).
- [ ] When `diameter` is present, coins are sized relative to one another within
      clamped min/max bounds; coins without diameter use a graceful default.
- [ ] Felt color can be changed among the provided options.
- [ ] `prefers-reduced-motion` is respected; `npm run type-check` passes; existing
      design tokens used.

## Open Questions

- Default tap action: flip in place vs open detail (and is the other a long-press)?
- Should the tray source be the full collection, the current filtered set, or a
  chosen showcase / multi-selected subset?
- Coins per drawer — fixed, or adaptive to viewport size?
- Is drag-to-arrange + saved custom layouts a desirable follow-up card?
- Felt texture: pure CSS (lighter, themeable) or a small image asset (richer look)?
- Relationship to Lightbox mode — should tapping "present" from the tray hand off to
  the single-coin lightbox for a selected coin?
