---
title: Progressive Web App and Offline
id: F006
status: promoted
priority: P1
effort: M
value: 5
risk: 2
owner: Aurelia
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Progressive Web App with install capability on iOS (via Safari Share → Add Home Screen), Android, and desktop. Offline-first caching: read collection, view cached coins, swipe gallery all work without network. Cache invalidation on collection mutation.

## Acceptance Criteria

- When user installs PWA on iOS/Android/desktop, app appears in home screen with custom icon
- When offline, user can browse collection gallery (swipe/grid), view coin details from cache
- When user is online and updates coin, service worker cache invalidates and refreshes
- When user pulls down gallery, refresh triggers online collection refetch
- When user takes photo via camera capture button (PWA mode only), image uploads to server when online

## Constitution Alignment

**Principle V (Design System):** PWA design respects dark theme and mobile viewports; hamburger menu compact for small screens.
**Principle XIII (PWA/Mobile Interaction Rules):** Service worker scope limited to `/` with `Cross-Origin` headers; offline boundaries respect authentication (users see only own coins).
**Principle X (Testing):** Architecture tests verify PWA service worker does not leak unauthed data.

## Implementation Notes

- Service worker: `src/web/public/service-worker.js` (caching strategy: cache-first for images, network-first for API)
- Manifest: `src/web/public/manifest.json` (app name, icons, start_url, display: standalone)
- Offline UI: graceful fallback (no "try again" on network error); pull-to-refresh for manual retry
- Camera capture: only in PWA context (detect via `navigator.standalone` or `display-mode: standalone`)
- Cache busting: version query param on build; coin mutations trigger `caches.delete('coins-v1')`

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Background removal via client-side ML (ONNX.js or TensorFlow.js) available on detail page for image enhancement.
