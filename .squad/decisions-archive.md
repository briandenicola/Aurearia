# Squad Decisions Archive

Older entries moved from decisions.md (entries before 2026-07-18).

**Date:** 2026-06-30
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

iOS/PWA users should not see a camera permission prompt just by opening Add Coin agentic mode or Identify Coin. The app still needs to preserve the guided live camera experience once the user intentionally starts capture.

## Decision

`src/web/src/pages/AddCoinPage.vue` and `src/web/src/pages/CoinLookupPage.vue` no longer start camera streams from page mount, agentic mode entry, or Identify Coin retake. Both pages show a clear "Start Camera" placeholder action that calls `startCamera()` only from a user tap. Existing upload-library actions remain available, shutter buttons stay disabled until `cameraReady`, and Add Coin continues stopping active streams when leaving agentic mode.

## Validation

- `npm.cmd run test -- src/pages/__tests__/CoinLookupPage.test.ts src/__tests__/ui-patterns.test.ts`
- `npm.cmd run type-check`

## Alignment

- Principle III: Vue strict type-check passed.
- Principle IV: Simple complete change across both affected camera entry points.
- Principle VI: Preserves existing dark, token-based camera UI and upload fallback.

---

### Decision: Shared Typed HTTP Client for Numista Lookup
