# Spec: Mint Map View

**Status:** Draft
**Area:** Frontend (Vue 3 / TS) + small reference dataset (optionally Go API)
**Depends on:** existing mint-mark field on coins
**Related:** Era/Region Heat Map, Collection Statistics, category filtering

---

## Summary

A map of the ancient world with pins at the mints where coins in the collection
were struck. Tapping a mint filters to the coins from that mint. Turns the
collection into a geography and complements the existing era/region heat map
(the "where" to the heat map's "when").

## Motivation

The app captures mint mark per coin but never visualizes origin spatially. A map
view exposes the geographic shape of a collection at a glance — concentrations
(e.g. Rome, Antioch, Constantinople) and blank regions — in a way no table or
chart currently does, and it's a strong, novel mobile display mode.

## User Stories

- As a collector, I open a map and see pins where my coins were minted, so I
  understand the geographic spread of my collection.
- As a collector, I tap a mint pin and see only the coins struck there.
- As a collector with multiple coins from one mint, the pin reflects the count.

## Scope

### In scope
- A `MintMap` view reachable from the collection / stats navigation.
- A mint → coordinates reference lookup (name, lat, lng, optional alt names).
- Pins placed per mint present in the user's collection, sized or badged by count.
- Tap a pin → filtered coin list (reuse existing gallery/list filtering).
- Themed to the museum-dark palette; usable one-handed on mobile (pan/zoom).
- Graceful handling of coins whose mint is unknown or unmatched (an
  "unattributed" bucket, not dropped silently).

### Out of scope
- Editing mint coordinates in the UI (first version ships a static dataset).
- Routing/connections between mints, animation of trade routes.
- Reverse-geocoding or AI inference of mint from images.

## Design / Approach

**Mint reference data**
- Start with a static, shipped lookup table (mint name → lat/lng) bundled with the
  frontend. The set of ancient mints is bounded and knowable, so this avoids a
  schema change and ships fast.
- Normalize on lookup (case-insensitive, handle common alternate spellings).
- *Alternative (later):* promote to a `mints` reference table in the Go API if the
  data needs to be queryable or user-editable. Note as an open question, not v1.

**Map rendering**
- Prefer a lightweight inline SVG map of the Mediterranean / ancient world over a
  full tile stack (Leaflet/Mapbox): no tile licensing, no heavy dependency,
  trivially themeable to the dark palette, and fine for a bounded region.
- Project mint lat/lng to the SVG viewbox; render pins as themed markers with an
  optional count badge.
- Pan/zoom via pointer + pinch; keep tap targets finger-friendly.

**Interaction**
- Tap pin → navigate to (or open a drawer with) the existing coin list filtered to
  that mint.
- Show an "unattributed / unknown mint" affordance for coins with no matched mint.

## Acceptance Criteria

- [ ] A map view is reachable from the collection or stats navigation.
- [ ] Each distinct mint represented in the collection shows a pin at the correct
      approximate location.
- [ ] A mint with multiple coins is visually distinguished (size or count badge).
- [ ] Tapping a pin shows only the coins struck at that mint.
- [ ] Coins with an unknown/unmatched mint are surfaced in an "unattributed"
      affordance rather than silently omitted.
- [ ] Map pans and zooms smoothly and is operable one-handed on a phone.
- [ ] Styling uses existing design tokens and the museum-dark theme.
- [ ] `npm run type-check` passes.

## Open Questions

- Ship mint coordinates as a static frontend dataset, or as a Go API `mints` table
  from the start?
- How are alternate/historical mint names reconciled (e.g. Byzantium vs
  Constantinople vs Istanbul) — do we map them to one pin or distinct pins by era?
- Should the map default to the Mediterranean and expand only if a collection has
  coins outside it (e.g. Modern category)?
