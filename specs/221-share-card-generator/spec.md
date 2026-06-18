# Spec: Share Card Generator

**Status:** Draft  
**Area:** Frontend (Vue 3 / TS), client-side rendering  
**Depends on:** existing coin images + metadata  
**Related:** Collection Showcase, Social Features, PWA share flows

---

## Summary

Generate a clean, branded shareable image of a single coin — the coin photo plus
key stats and the app's Ed-Mar branding — and hand it to the native share sheet
via the Web Share API. This rerun keeps the current beta/main coin detail display
unchanged and only adds the share action and card generator.

## Motivation

Collectors naturally want to show off individual coins. A purpose-built share
card produces a consistent, attractive, branded image without screenshotting and
cropping. Pricing, value, notes, and private analysis stay off the card.

## Scope

### In scope

- A "Share" action on the coin detail page.
- Client-side composition of a share card PNG.
- A spacious fixed template with separate image, title/category, metadata, and
  footer branding zones.
- Web Share API with file attachment where supported.
- Download fallback where Web Share files are unsupported.
- Preservation of the existing beta/main two-image coin detail display.

### Out of scope

- 3D coin viewer or Present mode changes.
- Bulk/multi-coin collage cards.
- Editing the card layout in-app.
- Server-side rendering or OG-image endpoints.

## Acceptance Criteria

- [x] A "Share" action is available on the coin detail page.
- [x] The coin detail hero remains the beta/main two-image display.
- [x] Tapping Share generates a branded PNG card containing the coin image and
      default public metadata, with no price/value shown.
- [x] The generated card keeps image, title/category, metadata, and footer text
      separated with no overlapping or pushed-together text.
- [x] On a device supporting Web Share with files, the native share sheet opens
      with the generated image attached.
- [x] On unsupported browsers, the generated image downloads as a PNG.
- [x] Card generation is client-side and requires no server endpoint.
- [x] `npm run type-check` passes.
