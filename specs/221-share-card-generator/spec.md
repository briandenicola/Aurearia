# Spec: Share Card Generator

**Status:** Draft
**Area:** Frontend (Vue 3 / TS), client-side rendering; optional Go API for OG links
**Depends on:** existing coin images + metadata; existing `og:image` plumbing
**Related:** Collection Showcase, Social Features, PWA share flows

---

## Summary

Generate a clean, branded shareable image of a single coin — the coin photo plus
key stats and the app's (Ed-Mar) branding — and hand it to the native share sheet
via the Web Share API. Makes it one tap to text a coin to a friend or post it,
without screenshotting and cropping.

## Motivation

Collectors naturally want to show off individual coins. Today that means a manual
screenshot. A purpose-built share card produces a consistent, attractive, branded
image and leverages the existing PWA share capabilities and `og:image` scraping
infrastructure already in the app.

## User Stories

- As a collector, I tap "Share" on a coin and get a polished image card (photo +
  name, ruler, denomination, era) that I can send via my phone's share sheet.
- As a collector, the card is branded consistently so shared coins look like they
  came from my collection app.
- As a privacy-conscious user, value/pricing is excluded from the card by default.

## Scope

### In scope
- A "Share" action on the coin detail page (and optionally on gallery cards).
- Client-side composition of a share card to a PNG: coin image (obverse, or a
  side-by-side obverse/reverse option), key metadata, app/Ed-Mar branding, on a
  themed background.
- Invoke `navigator.share()` with the generated image `File` where supported;
  fall back to a download / "copy image" when `navigator.share` (with files) is not.
- Sensible defaults: include name, ruler, denomination, era; exclude price/value.

### Out of scope
- Bulk/multi-coin collage cards (note as a possible later card).
- Editing the card layout in-app (ship one or two fixed templates first).
- Server-side rendering — see Open Questions (optional follow-up for OG link previews).

## Design / Approach

**Client-side render (preferred for a PWA)**
- Compose on an offscreen `<canvas>` (or render an SVG and rasterize): draw the
  themed background, the coin image (object-fit contain into a frame), a text block
  with the chosen metadata, and the branding mark.
- Export via `canvas.toBlob()` → `File` → `navigator.share({ files: [...] })`.
- Detect support with `navigator.canShare?.({ files: [file] })`; otherwise offer a
  download link or copy-to-clipboard image.
- Keep it offline-capable (no server round-trip needed to produce the card).

**Optional server-side variant (later)**
- A Go endpoint that renders the same card for a coin id and returns a PNG, usable
  as an `og:image` so shared *links* (e.g. to a public showcase coin) get rich
  previews. Reuses existing og:image patterns. Not required for v1.

**Templates**
- Ship one clean default template; optionally a second "obverse + reverse" layout.
- Use existing design tokens, category accent color, and typography for consistency.

## Acceptance Criteria

- [x] A "Share" action is available on the coin detail page.
- [x] Tapping it generates a branded PNG card containing the coin image and the
      default metadata (name, ruler, denomination, era), with no price/value shown.
- [x] On a device supporting Web Share with files, the native share sheet opens with
      the generated image attached.
- [x] On unsupported browsers, the user can still download or copy the generated
      image (no dead-end).
- [x] Card generation works offline (client-side, no server dependency for v1).
- [x] Card styling uses existing design tokens / category accent / branding.
- [x] `npm run type-check` passes.

## Open Questions

- Default to obverse-only, or offer an obverse+reverse two-up layout from the start?
- Do we also want the server-side OG-image endpoint so shared showcase *links*
  preview richly, or defer that to a separate card?
- Should the user be able to toggle which fields appear (e.g. opt-in to show grade)?
