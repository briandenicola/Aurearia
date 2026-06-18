# Spec: Mint Map View

**Status:** Draft — pivoted after beta review
**Area:** Frontend (Vue 3 / TS) + Leaflet/OpenStreetMap + static mint reference dataset
**Depends on:** existing mint field on coins
**Related:** Collection Statistics, Stats subviews, Collection Distribution, Timeline

---

## Summary

Show a real geographic mint map under Stats. The map uses Leaflet with OpenStreetMap tiles and places pins from actual mint latitude/longitude values in the shipped mint reference dataset. Tapping a mint pin shows the coins struck there, while unknown or unmatched mint values are surfaced explicitly.

The current beta stylized SVG approximation is not the intended product experience and must be replaced or retired.

## Motivation

The app captures mint information but does not visualize collection geography. A real slippy map makes the collection's spatial spread recognizable at a glance, with zoom/pan behavior users already understand from modern maps. This complements Stats subviews: Map answers "where", Timeline answers "when", Health answers "is the collection complete", and Value Trends answers "how value changes".

## User Stories

- As a collector, I open Stats and choose Mint Map, so I can see my collection on a real geographic map.
- As a collector, I tap a mint pin and see only the coins struck at that mint.
- As a collector with multiple coins from one mint, the pin reflects the count.
- As a collector, I can use existing `/mint-map` or `/timeline` links and land on the new Stats subview URLs.
- As a collector, I open Stats and see summary metrics as the landing page, with Stats subviews available only from the sidebar submenu.

## Scope

### In scope

- Mint Map lives only as the authenticated Stats subview `/stats/mint-map`.
- Existing `/mint-map` redirects to `/stats/mint-map`; existing `/timeline` redirects to `/stats/timeline`.
- Main `/stats` becomes the summary metrics landing page.
- Collection Distribution becomes its own Stats subview.
- Remove Mint Map launch actions from Collection headers/navigation.
- Replace the stylized SVG map experience with a Leaflet map using OpenStreetMap tile layers.
- Use actual `lat`/`lng` coordinates from the mint reference dataset for marker placement.
- Pins/markers show per-mint counts and support tap/click/keyboard selection.
- Gracefully handle coins whose mint is unknown or unmatched via an unattributed bucket.
- Preserve dark-theme/tokenized surrounding UI; accept that OSM tiles are externally styled imagery.

### Out of scope

- Editing mint coordinates in the UI.
- Go API `mints` table or database migrations.
- Reverse-geocoding or AI inference of mint from images.
- Custom tile hosting, offline tile cache, Mapbox, routing, trade routes, or map animations.
- Public/follower exposure of private collection geography.

## Design / Approach

**Mint reference data**

- Keep a static frontend mint lookup table with canonical name, `lat`, `lng`, aliases, and optional region.
- Normalize mint lookup case-insensitively and punctuation/spacing-insensitively.
- Alias collisions remain a dataset policy decision for v1; ambiguous names can canonicalize to one pin unless a later era-aware model is specified.

**Map rendering**

- Use Leaflet (`leaflet` npm package plus TypeScript-compatible imports) with OpenStreetMap raster tiles.
- Render markers directly from mint `lat`/`lng`; do not use SVG viewbox projection for the real map.
- Replace/remove `MintMapSvg.vue` or leave only as deleted historical code; active Mint Map rendering should be Leaflet-based.
- Use Leaflet pan/zoom controls and mobile touch support, with accessible marker labels and selected-mint state.

**Privacy / network**

- Opening `/stats/mint-map` may request OSM tile images from external OpenStreetMap tile servers. This reveals the user's IP address and approximate map viewport/zoom to the tile provider, but not collection contents unless marker data is encoded into external URLs, which must not happen.
- Do not send coin IDs, usernames, JWTs, mint names, or collection data to tile URLs or any third-party geocoding service.
- Auth remains required for the map because collection geography is private collection-derived data.

**Stats navigation**

- `/stats` is the summary metrics landing page previously represented by Summary Cards.
- Subviews use explicit routes: `/stats/mint-map`, `/stats/timeline`, `/stats/health`, and `/stats/value-trends`.
- The sidebar shows `Stats` as a collapsible parent item (collapsed by default) with exactly four indented submenu items: `Timeline` → `/stats/timeline`, `Map` → `/stats/mint-map`, `Health` → `/stats/health`, and `Value Trends` → `/stats/value-trends`.
- Clicking the Stats parent toggles the submenu expansion; if already expanded, clicking it navigates to `/stats` and closes the submenu.
- `Collection Distribution` remains available at `/stats/distribution` for distribution/heat map content but is not a sidebar submenu item.
- Legacy flat routes redirect with `router.replace`-style route records, not duplicate standalone pages.

**Mint Map summary display**

- Mint Map page shows a single-row summary bar centered on the page showing only the count of mapped coins, not a four-card grid.
- The summary uses design tokens (`--bg-card`, `--border-subtle`, `--radius-sm`, `--accent-gold`) and Cinzel font family for the count.
- The Leaflet map and unattributed bucket behavior remain unchanged.

## Acceptance Criteria

- [ ] `/stats` renders summary metrics/cards, not navigation cards.
- [ ] Sidebar Stats submenu starts collapsed and is expandable; contains exactly Timeline, Map, Health, and Value Trends; Timeline is not a top-level item.
- [ ] Health and Value Trends navigate to dedicated pages at `/stats/health` and `/stats/value-trends`, not hash anchors.
- [ ] `/stats/mint-map` renders a Leaflet map with OpenStreetMap tiles and a single-row mapped coin count summary.
- [ ] `/mint-map` redirects to `/stats/mint-map` and `/timeline` redirects to `/stats/timeline`.
- [ ] Mint Map is not launched from Collection headers.
- [ ] Collection Distribution is reachable as its own Stats subview without health/value trends sections.
- [ ] Each distinct matched mint in the collection shows a marker at its actual latitude/longitude.
- [ ] A mint with multiple coins is visually distinguished with a count.
- [ ] Tapping a marker shows only the coins struck at that mint.
- [ ] Coins with unknown/unmatched mint values are surfaced in an unattributed affordance.
- [ ] Map pan/zoom works on desktop and mobile/PWA viewports.
- [ ] Tests cover mint grouping, Leaflet marker rendering/selection, legacy redirects, Stats cards/subviews, collapsed/expandable Stats menu, dedicated Health and Value Trends pages, and navigation cleanup.
- [ ] `npm run type-check` and `npm run build` pass.

## Resolved Questions

- **Map background:** Leaflet with OpenStreetMap tiles, not a stylized inline SVG.
- **External tiles:** OSM tile requests are acceptable for this feature.
- **Feature placement:** Mint Map lives under Stats only.
- **Legacy URLs:** Keep compatibility via redirects to Stats subviews.
- **Stats structure:** Stats becomes a landing page; Collection Distribution is a subview.
